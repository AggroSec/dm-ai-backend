package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/database"
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
	ActionPoints          int32           `json:"action_points"`
	TalentPointsAvailable int32           `json:"talent_points_available"`
	TalentsInvested       json.RawMessage `json:"talents_invested"`
	Inventory             json.RawMessage `json:"inventory"`
	StatusEffects         json.RawMessage `json:"status_effects"`
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

	respondJSON(w, http.StatusCreated, character)
	log.Printf(" | [CharacterCreation]character: %v(%v) was successfully created for %v(%v)", character.Name, character.ID, user.Username, user.ID)
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

	var response []characterResponse
	for _, c := range characters {
		response = append(response, dbCharacterToResponse(c))
	}
	respondJSON(w, http.StatusOK, response)
	user, err := s.db.GetUserByID(r.Context(), userUUID)
	log.Printf(" | [CharacterInfo] retrieved character list of: %v(%v)", user.Username, user.ID)
}

func (s *Server) handlerGetCharacterByID(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.Context().Value("userID").(string))
	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid userID")
		log.Printf(" | [CharacterInfo] failed to convert to UUID: %v", err)
		return
	}

	character, err := s.db.GetCharacterByID(r.Context(), database.GetCharacterByIDParams{
		UserID: userID,
		ID:     characterID,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "character not found")
		log.Printf(" | [CharacterInfo] failed to retrieve character from db: %v", err)
		return
	}

	respondJSON(w, http.StatusOK, dbCharacterToResponse(character))
	log.Printf(" | [CharacterInfo] character info successfully retrieved: %v(%v) - %v", character.Name, character.ID, character.UserID)
}

func dbCharacterToResponse(c database.Character) characterResponse {
	return characterResponse{
		ID:                    c.ID.String(),
		UserID:                c.UserID.String(),
		Name:                  c.Name,
		Race:                  c.Race,
		Class:                 c.Class,
		Level:                 c.Level,
		Experience:            c.Experience,
		DrivingFate:           c.DrivingFate,
		BindingFate:           c.BindingFate,
		Strength:              c.Strength,
		Dexterity:             c.Dexterity,
		Fortitude:             c.Fortitude,
		Willpower:             c.Willpower,
		Alacrity:              c.Alacrity,
		Wisdom:                c.Wisdom,
		MaxHP:                 c.MaxHp,
		CurrentHP:             c.CurrentHp,
		MaxWP:                 c.MaxWp,
		CurrentWP:             c.CurrentWp,
		ActionPoints:          c.ActionPoints,
		TalentPointsAvailable: c.TalentPointsAvailable,
		TalentsInvested:       c.TalentsInvested,
		Inventory:             c.Inventory,
		StatusEffects:         c.StatusEffects,
	}
}
