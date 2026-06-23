package server

import (
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
	resp, _, _, err := s.aiClient.ChatWithTools(r.Context(), []ai.Message{systemPrompt, msg}, ai.GetToolDefinitions(), nil)
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

	var aiContext []ai.Message
	if req.CharacterCreation {
		character, err := s.db.GetCharacterByID(r.Context(), req.CharacterID)
		if err != nil {
			logAIError("failed to retrieve character for character creation", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
		}
		aiBuild, err := ai.BuildCharacterCreationContext(s.cfg, r.Context(), s.db, character, req.CampaignID, playerMsg.Content)
		if err != nil {
			logAIError("failed to built character creation context", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		for _, msg := range aiBuild {
			aiContext = append(aiContext, msg)
		}
	} else if req.CombatID != nil {
		aiBuild, err := ai.BuildCombatContext(r.Context(), s.db, s.cfg, req.CampaignID, req.CharacterID, *req.CombatID, playerMsg.Content)
		if err != nil {
			logAIError("failed to build combat context", err)
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

	responseCombatID := req.CombatID

	dispatcher := ai.NewDispatcher(s.db, s.cfg, req.CampaignID)
	resp, newCombatID, combatEnded, err := s.aiClient.ChatWithTools(r.Context(), aiContext, ai.GetToolDefinitions(), dispatcher)
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
	if newCombatID != nil {
		responseCombatID = newCombatID
	}

	var aiResponse actionResponse
	if combatEnded {
		aiResponse = actionResponse{
			Message:     resp,
			CombatID:    responseCombatID,
			CombatEnded: combatEnded,
		}
	} else {
		aiResponse = actionResponse{
			Message:  resp,
			CombatID: responseCombatID,
		}
	}

	logAIInfo(fmt.Sprintf("request successfully processed for character: %v", req.CharacterID))
	respondJSON(w, http.StatusOK, aiResponse)
}

func logAIError(msg string, err error) {
	log.Printf(" | [DM-AI] %s: %v", msg, err)
}

func logAIInfo(msg string) {
	log.Printf(" | [DM-AI] %s", msg)
}
