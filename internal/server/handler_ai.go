package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/AggroSec/dm-ai-backend/internal/ai"
	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

type MsgRequest struct {
	Message string `json:"message"`
}

type actionRequest struct {
	CampaignID        uuid.UUID  `json:"campaign_id"`
	CharacterID       uuid.UUID  `json:"character_id"`
	CombatID          *uuid.UUID `json:"combat_id,omitempty"`
	Message           string     `json:"message"`
	CharacterCreation bool       `json:"character_creation,omitempty"`
}

type actionResponse struct {
	Message     string     `json:"message"`
	CombatID    *uuid.UUID `json:"combat_id,omitempty"`
	CombatEnded bool       `json:"combat_ended,omitempty"`
}

func (s *Server) handlerAITest(w http.ResponseWriter, r *http.Request) {
	var req MsgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	msg := ai.Message{
		Role:    "user",
		Content: req.Message,
	}
	systemPrompt := ai.Message{
		Role:    "system",
		Content: "Respond with only 3-4 sentences, as if you are a DM narrating the beginning setting of a campaign. The user input will be a theme.",
	}
	resp, err := s.aiClient.Chat(r.Context(), []ai.Message{systemPrompt, msg})
	if err != nil {
		logAIError("AI chat error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logAIInfo("test chat successful")
	respondJSON(w, http.StatusOK, map[string]string{"response": resp})
}

func (s *Server) handlerAITestTools(w http.ResponseWriter, r *http.Request) {
	var req MsgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	msg := ai.Message{
		Role:    "user",
		Content: req.Message,
	}
	systemPrompt := ai.Message{
		Role:    "system",
		Content: "You are a test assistant. When the user asks you to roll a dice, call the request_roll tool",
	}
	resp, _, _, err := s.aiClient.ChatWithTools(r.Context(), s.aiClient.Model, s.aiClient.CombatModel, []ai.Message{systemPrompt, msg}, ai.GetToolDefinitions(ai.ModeNarrative), nil, nil)
	if err != nil {
		logAIError("AI chat error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logAIInfo("test chat successful")
	respondJSON(w, http.StatusOK, map[string]string{"response": resp})
}

func (s *Server) handlerAIAction(w http.ResponseWriter, r *http.Request) {
	var req actionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// combat uses SSE streaming, everything else uses regular JSON
	if req.CombatID != nil {
		s.handlerAIActionStream(w, r, req)
		return
	}
	s.handlerAIActionJSON(w, r, req)
}

func (s *Server) handlerAIActionJSON(w http.ResponseWriter, r *http.Request, req actionRequest) {
	playerMsg := ai.Message{
		Role:    "user",
		Content: req.Message,
	}

	msgSequence, err := s.db.GetNextSequence(r.Context(), req.CampaignID)
	if err != nil {
		logAIError("could not get next message sequence", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_, err = s.db.InsertMessage(r.Context(), database.InsertMessageParams{
		CampaignID: req.CampaignID,
		Role:       playerMsg.Role,
		Content:    playerMsg.Content,
		Sequence:   msgSequence,
		ToolCalls:  []byte("[]"),
	})
	if err != nil {
		logAIError("failed to add player message to db", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var mode ai.GameMode
	if req.CharacterCreation {
		mode = ai.ModeCharacterCreation
	} else {
		mode = ai.ModeNarrative
	}
	var aiContext []ai.Message
	if req.CharacterCreation {
		character, err := s.db.GetCharacterByID(r.Context(), req.CharacterID)
		if err != nil {
			logAIError("failed to retrieve character for character creation", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		aiBuild, err := ai.BuildCharacterCreationContext(s.cfg, r.Context(), s.db, character, req.CampaignID, playerMsg.Content)
		if err != nil {
			logAIError("failed to build character creation context", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		for _, msg := range aiBuild {
			aiContext = append(aiContext, msg)
		}
	} else {
		aiBuild, err := ai.BuildNarrativeContext(r.Context(), s.db, s.cfg, req.CampaignID, req.CharacterID, playerMsg.Content)
		if err != nil {
			logAIError("failed to build narrative context", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		for _, msg := range aiBuild {
			aiContext = append(aiContext, msg)
		}
	}

	dispatcher := ai.NewDispatcher(s.db, s.cfg, req.CampaignID)
	resp, newCombatID, combatEnded, err := s.aiClient.ChatWithTools(r.Context(), s.aiClient.Model, s.aiClient.CombatModel, aiContext, ai.GetToolDefinitions(mode), dispatcher, nil)
	if err != nil {
		logAIError("Chat call failed", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if strings.Contains(resp, game.CreationCompleteSignal) {
		_, err := s.db.SetCharacterCreationComplete(r.Context(), database.SetCharacterCreationCompleteParams{
			ID:                        req.CampaignID,
			CharacterCreationComplete: true,
		})
		if err != nil {
			logAIError("failed to set character creation complete", err)
		}
	}

	aiResponse := actionResponse{
		Message:     resp,
		CombatID:    newCombatID,
		CombatEnded: combatEnded,
	}

	logAIInfo(fmt.Sprintf("request successfully processed for character: %v", req.CharacterID))
	respondJSON(w, http.StatusOK, aiResponse)
	if !req.CharacterCreation {
		go func() {
			if err := ai.MaybeSummarizeCampaign(context.Background(), s.db, s.aiClient, req.CampaignID); err != nil {
				log.Printf(" | [Summarization] failed for campaign %v: %v", req.CampaignID, err)
			}
		}()
	}
}

func (s *Server) handlerAIActionStream(w http.ResponseWriter, r *http.Request, req actionRequest) {
	// Set SSE headers before writing anything
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	playerMsg := ai.Message{
		Role:    "user",
		Content: req.Message,
	}

	msgSequence, err := s.db.GetNextSequence(r.Context(), req.CampaignID)
	if err != nil {
		logAIError("could not get next message sequence", err)
		fmt.Fprintf(w, "event: error\ndata: internal server error\n\n")
		flusher.Flush()
		return
	}

	_, err = s.db.InsertMessage(r.Context(), database.InsertMessageParams{
		CampaignID: req.CampaignID,
		Role:       playerMsg.Role,
		Content:    playerMsg.Content,
		Sequence:   msgSequence,
		ToolCalls:  []byte("[]"),
	})
	if err != nil {
		logAIError("failed to add player message to db", err)
		fmt.Fprintf(w, "event: error\ndata: internal server error\n\n")
		flusher.Flush()
		return
	}

	aiBuild, err := ai.BuildCombatContext(r.Context(), s.db, s.cfg, req.CampaignID, req.CharacterID, *req.CombatID, playerMsg.Content)
	if err != nil {
		logAIError("failed to build combat context", err)
		fmt.Fprintf(w, "event: error\ndata: internal server error\n\n")
		flusher.Flush()
		return
	}

	var aiContext []ai.Message
	for _, msg := range aiBuild {
		aiContext = append(aiContext, msg)
	}

	// streamFn flushes each narrate_combat chunk to the client immediately
	streamFn := func(text string) {
		escaped := strings.ReplaceAll(text, "\n", "\\n")
		fmt.Fprintf(w, "data: %s\n\n", escaped)
		flusher.Flush()
	}

	dispatcher := ai.NewDispatcher(s.db, s.cfg, req.CampaignID)
	resp, newCombatID, combatEnded, err := s.aiClient.ChatWithTools(r.Context(), s.aiClient.CombatModel, s.aiClient.Model, aiContext, ai.GetToolDefinitions(ai.ModeCombat), dispatcher, streamFn)
	if err != nil {
		logAIError("Chat call failed", err)
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	// send metadata as final SSE event so CLI can update combat state
	responseCombatID := req.CombatID
	if newCombatID != nil {
		responseCombatID = newCombatID
	}
	if combatEnded {
		responseCombatID = nil
	}

	meta := actionResponse{
		Message:     resp,
		CombatID:    responseCombatID,
		CombatEnded: combatEnded,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		logAIError("failed to marshal meta response", err)
		fmt.Fprintf(w, "event: error\ndata: failed to marshal response\n\n")
		flusher.Flush()
		return
	}

	fmt.Fprintf(w, "event: meta\ndata: %s\n\n", string(metaJSON))
	flusher.Flush()

	logAIInfo(fmt.Sprintf("request successfully processed for character: %v", req.CharacterID))
}

func logAIError(msg string, err error) {
	log.Printf(" | [DM-AI] %s: %v", msg, err)
}

func logAIInfo(msg string) {
	log.Printf(" | [DM-AI] %s", msg)
}
