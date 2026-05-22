package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

func (s *Server) handlerStartCombat(w http.ResponseWriter, r *http.Request) {
	type NPCRequest struct {
		Name      string      `json:"name"`
		Type      string      `json:"type"`
		HP        int         `json:"hp"`
		MaxHP     int         `json:"max_hp"`
		WP        int         `json:"wp"`
		MaxWP     int         `json:"max_wp"`
		AP        int         `json:"ap"`
		Strength  int         `json:"strength"`
		Dexterity int         `json:"dexterity"`
		Fortitude int         `json:"fortitude"`
		Willpower int         `json:"willpower"`
		Alacrity  int         `json:"alacrity"`
		Wisdom    int         `json:"wisdom"`
		Inventory []game.Item `json:"inventory"`
	}

	type PartyMemberRequest struct {
		CharacterID uuid.UUID `json:"character_id"`
		UserID      uuid.UUID `json:"user_id"`
	}

	type CombatSessionRequest struct {
		Party []PartyMemberRequest `json:"party"`
		NPCs  []NPCRequest         `json:"npcs"`
	}

	var req CombatSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logCombatHandlerError("failed to decode json request", err)
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
}

func logCombatHandlerError(msg string, err error) {
	log.Printf(" | [CombatHandler] %s: %v", msg, err)
}
