package server

import (
	"encoding/json"
	"log"
	"net/http"
)

func (s *Server) handlerApplyStatus(w http.ResponseWriter, r *http.Request) {
	type applyStatusEffectRequest struct {
		Effect   string `json:"effect"`
		Duration int    `json:"duration"`
		Persists bool   `json:"persists"`
	}

	var req applyStatusEffectRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logStatusError("failed to decode json request", err)
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

}

func logStatusError(errMsg string, err error) {
	log.Printf(" | [EffectsManager] %v: %v", errMsg, err)
}
