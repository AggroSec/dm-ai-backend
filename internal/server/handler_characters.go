package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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
	UpdatedAt             time.Time       `json:"updated_at"`
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
	log.Printf(" | [CharacterCreation]character: %v(%v) was successfully created for %v(%v)", character.Name, character.ID, user.Username, user.ID)
	respondJSON(w, http.StatusCreated, dbCharacterToResponse(character))
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

	user, err := s.db.GetUserByID(r.Context(), userUUID)
	log.Printf(" | [CharacterInfo] retrieved character list of: %v(%v)", user.Username, user.ID)
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handlerGetCharacterByID(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.Context().Value("userID").(string))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid userID")
		log.Printf(" | [CharacterInfo] failed to extract userID: %v", err)
		return
	}
	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid characterID")
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

func (s *Server) handlerUpdateCharacter(w http.ResponseWriter, r *http.Request) {
	characterID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid characterID")
		log.Printf(" | [CharacterUpdate] failed to convert to UUID: %v", err)
		return
	}
	userID, err := uuid.Parse(r.Context().Value("userID").(string))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "invalid userID")
		log.Printf(" | [CharacterUpdate] failed to extract userID: %v", err)
		return
	}

	character, err := s.db.GetCharacterByID(r.Context(), database.GetCharacterByIDParams{
		UserID: userID,
		ID:     characterID,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "character not found")
		log.Printf(" | [CharacterUpdate] failed to retrieve character from db: %v", err)
		return
	}

	var req updateCharacterRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad request")
		log.Printf(" | [CharacterUpdate] failed to decode request: %v", err)
		return
	}

	applyCharacterUpdates(&character, req)

	updated, err := s.db.UpdateCharacter(r.Context(), database.UpdateCharacterParams{
		ID:                    characterID,
		UserID:                userID,
		Name:                  character.Name,
		Race:                  character.Race,
		Class:                 character.Class,
		Level:                 character.Level,
		Strength:              character.Strength,
		Dexterity:             character.Dexterity,
		Fortitude:             character.Fortitude,
		Willpower:             character.Willpower,
		Alacrity:              character.Alacrity,
		Wisdom:                character.Wisdom,
		CurrentHp:             character.CurrentHp,
		MaxHp:                 character.MaxHp,
		CurrentWp:             character.CurrentWp,
		MaxWp:                 character.MaxWp,
		DrivingFate:           character.DrivingFate,
		BindingFate:           character.BindingFate,
		TalentsInvested:       character.TalentsInvested,
		TalentPointsAvailable: character.TalentPointsAvailable,
		Inventory:             character.Inventory,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update character")
		log.Printf(" | [CharacterUpdate] failed to update character: %v", err)
		return
	}

	respondJSON(w, http.StatusOK, dbCharacterToResponse(updated))
	log.Printf(" | [CharacterUpdate] character updated: %v(%v)", updated.Name, updated.ID)
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
		UpdatedAt:             c.UpdatedAt,
	}
}

type updateCharacterRequest struct {
	Name                  *string         `json:"name"`
	Race                  *string         `json:"race"`
	Class                 *string         `json:"class"`
	Level                 *int32          `json:"level"`
	DrivingFate           *string         `json:"driving_fate"`
	BindingFate           *string         `json:"binding_fate"`
	Strength              *int32          `json:"strength"`
	Dexterity             *int32          `json:"dexterity"`
	Fortitude             *int32          `json:"fortitude"`
	Willpower             *int32          `json:"willpower"`
	Alacrity              *int32          `json:"alacrity"`
	Wisdom                *int32          `json:"wisdom"`
	MaxHP                 *int32          `json:"max_hp"`
	CurrentHP             *int32          `json:"current_hp"`
	MaxWP                 *int32          `json:"max_wp"`
	CurrentWP             *int32          `json:"current_wp"`
	TalentPointsAvailable *int32          `json:"talent_points_available"`
	TalentsInvested       json.RawMessage `json:"talents_invested"`
	Inventory             json.RawMessage `json:"inventory"`
}

func applyCharacterUpdates(c *database.Character, req updateCharacterRequest) {
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Race != nil {
		c.Race = *req.Race
	}
	if req.Class != nil {
		c.Class = *req.Class
	}
	if req.Level != nil {
		c.Level = *req.Level
	}
	if req.DrivingFate != nil {
		c.DrivingFate = *req.DrivingFate
	}
	if req.BindingFate != nil {
		c.BindingFate = *req.BindingFate
	}
	if req.Strength != nil {
		c.Strength = *req.Strength
	}
	if req.Dexterity != nil {
		c.Dexterity = *req.Dexterity
	}
	if req.Fortitude != nil {
		c.Fortitude = *req.Fortitude
	}
	if req.Willpower != nil {
		c.Willpower = *req.Willpower
	}
	if req.Alacrity != nil {
		c.Alacrity = *req.Alacrity
	}
	if req.Wisdom != nil {
		c.Wisdom = *req.Wisdom
	}
	if req.MaxHP != nil {
		c.MaxHp = *req.MaxHP
	}
	if req.CurrentHP != nil {
		c.CurrentHp = *req.CurrentHP
	}
	if req.MaxWP != nil {
		c.MaxWp = *req.MaxWP
	}
	if req.CurrentWP != nil {
		c.CurrentWp = *req.CurrentWP
	}
	if req.TalentPointsAvailable != nil {
		c.TalentPointsAvailable = *req.TalentPointsAvailable
	}
	if req.TalentsInvested != nil {
		c.TalentsInvested = req.TalentsInvested
	}
	if req.Inventory != nil {
		c.Inventory = req.Inventory
	}
}
