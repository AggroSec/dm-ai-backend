package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
)

type StatusEffectResponse struct {
	ID                string    `json:"id"`
	AffectedCharacter string    `json:"characterID"`
	Effect            string    `json:"effect"`
	Duration          int       `json:"duration"`
	Persists          bool      `json:"persists"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (s *Server) handlerApplyStatusEffect(w http.ResponseWriter, r *http.Request) {
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

	effectedCharacter, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logStatusError("failed to convert to UUID", err)
		respondError(w, http.StatusBadRequest, "invalid characterID")
		return
	}

	dbreq := database.AddStatusEffectParams{
		CharacterID: effectedCharacter,
		Effect:      req.Effect,
		Duration:    int32(req.Duration),
		Persists:    req.Persists,
	}
	dbresults, err := s.db.AddStatusEffect(r.Context(), dbreq)
	if err != nil {
		logStatusError("Database query failed", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	statusEffect := StatusEffectResponse{
		ID:                dbresults.ID.String(),
		AffectedCharacter: dbreq.CharacterID.String(),
		Effect:            dbresults.Effect,
		Duration:          int(dbresults.Duration),
		Persists:          dbreq.Persists,
		IsActive:          dbresults.IsActive,
		CreatedAt:         dbresults.CreatedAt,
		UpdatedAt:         dbresults.UpdatedAt,
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
	effectsListResponse, err := s.getStatusEffects(characterID)
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

	effectsListResponse, err := s.getStatusEffects(characterID)
	if err != nil {
		logStatusError("Database query failed", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updatedEffectdList := []StatusEffectResponse{}

	for _, effect := range effectsListResponse {
		effect.Duration -= 1
		if effect.Duration == 0 {
			effect.IsActive = false
		}
		updatedEffect, err := s.db.UpdateStatusDuration(r.Context(), database.UpdateStatusDurationParams{
			ID:          uuid.MustParse(effect.ID),
			Duration:    int32(effect.Duration),
			IsActive:    effect.IsActive,
			CharacterID: uuid.MustParse(effect.AffectedCharacter),
		})
		if err != nil {
			logStatusError("Database query failed", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		logStatusEffect(fmt.Sprintf("%v(%v) duration has been updated for %v", updatedEffect.Effect, updatedEffect.ID, updatedEffect.CharacterID))
	}

	s.db.PurgeInactiveEffects(r.Context(), characterID)

	dbResults, err := s.db.GetStatusEffectsByID(r.Context(), characterID)
	if err != nil {
		logStatusError("Database query failed", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	for _, effect := range dbResults {
		result := StatusEffectResponse{
			ID:                effect.ID.String(),
			AffectedCharacter: effect.CharacterID.String(),
			Effect:            effect.Effect,
			Duration:          int(effect.Duration),
			Persists:          effect.Persists,
			IsActive:          effect.IsActive,
			CreatedAt:         effect.CreatedAt,
			UpdatedAt:         effect.UpdatedAt,
		}
		updatedEffectdList = append(updatedEffectdList, result)
	}

	logStatusEffect("ticker successfully ran")
	respondJSON(w, http.StatusOK, updatedEffectdList)
}

func logStatusError(errMsg string, err error) {
	log.Printf(" | [EffectsManager] %v: %v", errMsg, err)
}

func (s *Server) getStatusEffects(characterID uuid.UUID) ([]StatusEffectResponse, error) {
	dbresults, err := s.db.GetStatusEffectsByID(context.Background(), characterID)
	if err != nil {
		return nil, err
	}

	effectsListResponse := []StatusEffectResponse{}
	for _, effect := range dbresults {
		result := StatusEffectResponse{
			ID:                effect.ID.String(),
			AffectedCharacter: effect.CharacterID.String(),
			Effect:            effect.Effect,
			Duration:          int(effect.Duration),
			Persists:          effect.Persists,
			IsActive:          effect.IsActive,
			CreatedAt:         effect.CreatedAt,
			UpdatedAt:         effect.UpdatedAt,
		}
		effectsListResponse = append(effectsListResponse, result)
	}
	return effectsListResponse, nil
}

func logStatusEffect(msg string) {
	log.Printf(" | [EffectsManager] %v", msg)
}
