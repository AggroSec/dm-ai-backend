package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

func (s *Server) handlerApplyStatusEffect(w http.ResponseWriter, r *http.Request) {
	type applyStatusEffectRequest struct {
		Effect      string `json:"effect"`
		Duration    int    `json:"duration"`
		Persists    bool   `json:"persists"`
		Instruction string `json:"instruction"`
	}

	var req applyStatusEffectRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logStatusError("failed to decode json request", err)
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	effectedCharacter, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logStatusError("failed to convert to UUID", err)
		respondError(w, http.StatusBadRequest, "invalid characterID")
		return
	}

	statusEffect, err := game.ApplyStatusEffect(r.Context(), s.db, effectedCharacter, req.Effect, req.Duration, req.Persists, req.Instruction, nil)
	if err != nil {
		logStatusError("failed to apply status effect", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logStatusEffect(fmt.Sprintf("%v(%v) effect added to %v", statusEffect.Effect, statusEffect.ID, statusEffect.AffectedCharacter))
	respondJSON(w, http.StatusCreated, statusEffect)
}

func (s *Server) handlerGetStatusEffectsByID(w http.ResponseWriter, r *http.Request) {
	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logStatusError("failed to convert to UUID", err)
		respondError(w, http.StatusBadRequest, "invalid characterID")
		return
	}
	effectsListResponse, err := game.GetStatusEffects(r.Context(), s.db, characterID)
	if err != nil {
		logStatusError("Database query failed", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, effectsListResponse)
}

func (s *Server) handlerStatusEffectsTick(w http.ResponseWriter, r *http.Request) {
	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logStatusError("failed to convert to UUID", err)
		respondError(w, http.StatusBadRequest, "invalid characterID")
		return
	}

	effectsListResponse, err := game.TickStatusEffects(r.Context(), s.db, characterID)
	if err != nil {
		logStatusError("Database query failed", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logStatusEffect("ticker successfully ran")
	respondJSON(w, http.StatusOK, effectsListResponse)
}

func logStatusError(errMsg string, err error) {
	log.Printf(" | [EffectsManager] %v: %v", errMsg, err)
}

func logStatusEffect(msg string) {
	log.Printf(" | [EffectsManager] %v", msg)
}
