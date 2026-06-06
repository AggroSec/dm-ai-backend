package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/ai"
)

type MsgRequest struct {
	Message string `json:"message"`
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
	resp, err := s.aiClient.ChatWithTools(r.Context(), []ai.Message{systemPrompt, msg}, ai.GetToolDefinitions())
	if err != nil {
		logAIError("AI chat error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logAIInfo("test chat successful")
	respondJSON(w, http.StatusOK, map[string]string{"response": resp})
}

func logAIError(msg string, err error) {
	log.Printf(" | [DM-AI] %s: %v", msg, err)
}

func logAIInfo(msg string) {
	log.Printf(" | [DM-AI] %s", msg)
}
