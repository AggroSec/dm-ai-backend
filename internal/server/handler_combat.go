package server

import (
	"encoding/json"
	"fmt"
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
		MaxAP     int         `json:"max_ap"`
		OvercapAP int         `json:"overcap_ap"`
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

	var npcs []game.Combatant
	for _, npc := range req.NPCs {
		npcs = append(npcs, game.Combatant{
			Name:      npc.Name,
			Type:      npc.Type,
			HP:        npc.HP,
			MaxHP:     npc.MaxHP,
			WP:        npc.WP,
			MaxWP:     npc.MaxWP,
			AP:        npc.AP,
			MaxAP:     npc.MaxAP,
			OvercapAP: npc.OvercapAP,
			Strength:  npc.Strength,
			Dexterity: npc.Dexterity,
			Fortitude: npc.Fortitude,
			Willpower: npc.Willpower,
			Alacrity:  npc.Alacrity,
			Wisdom:    npc.Wisdom,
			IsAlive:   true,
			Inventory: npc.Inventory,
		})
	}

	var party []game.PartyMember
	for _, member := range req.Party {
		party = append(party, game.PartyMember{
			CharacterID: member.CharacterID,
			UserID:      member.UserID,
		})
	}

	newSession, err := game.CreateCombatSession(r.Context(), s.db, party, npcs, uuid.Nil)
	if err != nil {
		logCombatHandlerError("failed to create combat session", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logCombatHandler(fmt.Sprintf("Combat Session created - Combatants: %v", newSession.Combatants))
	respondJSON(w, http.StatusCreated, newSession)
}

func (s *Server) handlerGetActiveCombatSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logCombatHandlerError("failed to convert to UUID", err)
		respondError(w, http.StatusBadRequest, "bad request")
		return
	}

	session, err := game.GetActiveCombatSession(r.Context(), s.db, sessionID)
	if err != nil {
		logCombatHandlerError("failed to get active combat session", err)
		respondError(w, http.StatusNotFound, "combat session not found")
		return
	}

	logCombatHandler(fmt.Sprintf("Active combat session found: %v", session.ID))
	respondJSON(w, http.StatusOK, session)
}

func (s *Server) handlerAdvanceTurn(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logCombatHandlerError("failed to convert to UUID", err)
		respondError(w, http.StatusBadRequest, "bad request")
		return
	}

	updatedSession, err := game.AdvanceTurn(r.Context(), sessionID, s.db)
	if err != nil {
		logCombatHandlerError("failed to advance turn", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logCombatHandler(fmt.Sprintf("Turn advanced for combat session: %v - Current Turn: %v", updatedSession.ID, updatedSession.CurrentTurn))
	respondJSON(w, http.StatusOK, updatedSession)
}

func (s *Server) handlerTakeAction(w http.ResponseWriter, r *http.Request) {
	type TakeActionRequest struct {
		ActorID    uuid.UUID `json:"actor_id"`
		ActionType string    `json:"action_type"`
		TargetID   uuid.UUID `json:"target_id"`
	}

	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logCombatHandlerError("invalid session ID", err)
		respondError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	var req TakeActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logCombatHandlerError("failed to decode request", err)
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	session, err := game.ProcessAction(r.Context(), s.db, sessionID, req.ActorID, req.ActionType, req.TargetID)
	if err != nil {
		logCombatHandlerError("failed to process action", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logCombatHandler(fmt.Sprintf("Action processed - type: %s actor: %s", req.ActionType, req.ActorID))
	respondJSON(w, http.StatusOK, session)
}

func (s *Server) handlerEndCombat(w http.ResponseWriter, r *http.Request) {
	type EndCombatRequest struct {
		Outcome string `json:"outcome"`
	}

	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		logCombatHandlerError("invalid session ID", err)
		respondError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	var req EndCombatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logCombatHandlerError("failed to decode request", err)
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.Outcome == "" {
		respondError(w, http.StatusBadRequest, "outcome is required")
		return
	}

	session, err := game.EndCombat(r.Context(), s.db, sessionID, req.Outcome)
	if err != nil {
		logCombatHandlerError("failed to end combat", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logCombatHandler(fmt.Sprintf("Combat session ended - outcome: %s", req.Outcome))
	respondJSON(w, http.StatusOK, session)
}

func logCombatHandlerError(msg string, err error) {
	log.Printf(" | [CombatHandler] %s: %v", msg, err)
}

func logCombatHandler(msg string) {
	log.Printf(" | [CombatHandler] %s", msg)
}
