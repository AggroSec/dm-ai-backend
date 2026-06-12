package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

type characterResponse struct {
	ID                    string          `json:"id"`
	UserID                string          `json:"user_id"`
	Name                  string          `json:"name"`
	Race                  string          `json:"race"`
	Class                 string          `json:"class"`
	Level                 int32           `json:"level"`
	Experience            int32           `json:"experience"`
	DrivingFate           string          `json:"driving_fate"`
	BindingFate           string          `json:"binding_fate"`
	Strength              int32           `json:"strength"`
	Dexterity             int32           `json:"dexterity"`
	Fortitude             int32           `json:"fortitude"`
	Willpower             int32           `json:"willpower"`
	Alacrity              int32           `json:"alacrity"`
	Wisdom                int32           `json:"wisdom"`
	MaxHP                 int32           `json:"max_hp"`
	CurrentHP             int32           `json:"current_hp"`
	MaxWP                 int32           `json:"max_wp"`
	CurrentWP             int32           `json:"current_wp"`
	CurrentAP             int32           `json:"current_ap"`
	MaxAP                 int32           `json:"max_ap"`
	OvercapAP             int32           `json:"overcap_ap"`
	TalentPointsAvailable int32           `json:"talent_points_available"`
	TalentsInvested       json.RawMessage `json:"talents_invested"`
	Inventory             json.RawMessage `json:"inventory"`
	EquippedSlots         json.RawMessage `json:"equipped_slots"`
	//StatusEffects         json.RawMessage `json:"status_effects"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Server) handlerCreateCharacter(w http.ResponseWriter, r *http.Request) {
	type createCharacterRequest struct {
		Name  string
		Class string
	}

	userID := r.Context().Value("userID").(string)
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid userID")
		log.Printf(" | [CharacterCreation]failed conversion to UUID: %v", err)
		return
	}

	var charReq createCharacterRequest
	err = json.NewDecoder(r.Body).Decode(&charReq)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad request")
		log.Printf(" | [CharacterCreation]failed to decode request: %v", err)
		return
	}

	dbreq := database.CreateCharacterParams{
		Name:   charReq.Name,
		Class:  charReq.Class,
		UserID: userUUID,
	}

	character, err := s.db.CreateCharacter(r.Context(), dbreq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "character creation failed.")
		log.Printf(" | [CharacterCreation]failed to add character to db: %v", err)
		return
	}

	user, err := s.db.GetUserByID(r.Context(), userUUID)
	if err != nil {
		log.Printf(" | internal server error - was not able to retrieve user: %v", err)
	}
	log.Printf(" | [CharacterCreation] character: %v(%v) was successfully created for %v(%v)", character.Name, character.ID, user.Username, user.ID)
	respondJSON(w, http.StatusCreated, "character created successfully")
}

func (s *Server) handlerGetUserCharacters(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid userID")
		log.Printf(" | [CharacterInfo]failed to convert to UUID: %v", err)
		return
	}

	characters, err := s.db.GetCharacterByUserID(r.Context(), userUUID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "no characters found")
		log.Printf(" | [CharacterInfo] no characters were found for %v, or connection to db failed: %v", userID, err)
		return
	}

	user, err := s.db.GetUserByID(r.Context(), userUUID)
	if err != nil {
		log.Printf(" | internal server error - was not able to retrieve user: %v", err)
	}
	log.Printf(" | [CharacterInfo] retrieved character list of: %v(%v)", user.Username, user.ID)
	respondJSON(w, http.StatusOK, characters) // this is ugly, can come back and fix later when needed.
}

func (s *Server) handlerGetCharacterByID(w http.ResponseWriter, r *http.Request) {
	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid characterID")
		log.Printf(" | [CharacterInfo] failed to convert to UUID: %v", err)
		return
	}

	character, err := game.GetCharacterInfo(r.Context(), s.db, characterID)
	if err != nil {
		log.Printf(" | [CharacterInfo] game function to get character info failed: %v", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, character)
	log.Printf(" | [CharacterInfo] character info successfully retrieved: %v(%v) - %v", character.Name, character.ID, character.UserID)
}

func (s *Server) handlerUpdateCharacter(w http.ResponseWriter, r *http.Request) {
	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid characterID")
		log.Printf(" | [CharacterUpdate] failed to convert to UUID: %v", err)
		return
	}

	var req game.UpdateCharacterRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad request")
		log.Printf(" | [CharacterUpdate] failed to decode request: %v", err)
		return
	}

	result, err := game.ApplyCharacterUpdates(r.Context(), s.db, characterID, req)
	if err != nil {
		log.Printf("| [CharacterUpdate] game update of character failed: %v", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, result)
	log.Printf(" | [CharacterUpdate] character updated: %v(%v)", result.Name, result.ID)
}

func (s *Server) handlerDeleteCharacter(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.Context().Value("userID").(string))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid userID")
		log.Printf(" | [CharacterDelete] failed to convert userID to UUID: %v", err)
		return
	}

	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid character id")
		log.Printf(" | [CharacterDelete] failed to convert characterID to UUID: %v", err)
		return
	}

	err = s.db.DeleteCharacter(r.Context(), database.DeleteCharacterParams{
		ID:     characterID,
		UserID: userID,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete character")
		log.Printf(" | [CharacterDelete] failed to delete character %v: %v", characterID, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "character deleted"})
	log.Printf(" | [CharacterDelete] character %v successfully deleted by user %v", characterID, userID)
}
