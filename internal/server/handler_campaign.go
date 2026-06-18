package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

func (s *Server) handlerCreateCampaign(w http.ResponseWriter, r *http.Request) {
	type newCampaign struct {
		Name          string `json:"name"`
		Theme         string `json:"theme"`
		CharacterName string `json:"character_name"`
	}
	var req newCampaign
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(r.Context().Value("userID").(string))
	if err != nil {
		logCampaignError("failed to parse uuid", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	campaign, err := s.db.CreateCampaign(r.Context(), database.CreateCampaignParams{
		Name:    req.Name,
		Theme:   req.Theme,
		OwnerID: userID,
	})
	if err != nil {
		logCampaignError("campaign was not created in db", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	character, err := s.db.CreateCharacter(r.Context(), database.CreateCharacterParams{
		Name:   req.CharacterName,
		Class:  "runeblade",
		UserID: userID,
	})
	if err != nil {
		logCampaignError("character creation in db failed", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	party := []game.PartyMember{{
		CharacterID: character.ID,
		UserID:      userID,
	}}
	marshalParty, err := json.Marshal(party)
	if err != nil {
		logCampaignError("unable to marshal party", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	updatedCampaign, err := s.db.UpdateCampaignParty(r.Context(), database.UpdateCampaignPartyParams{
		ID:    campaign.ID,
		Party: marshalParty,
	})
	if err != nil {
		logCampaignError("failed to update campaign party with new character", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logCampaign(fmt.Sprintf("Successfully created new campaign: %s(%v) with party: %v", updatedCampaign.Name, updatedCampaign.ID, string(marshalParty)))
	type newCampaignResponse struct {
		CampaignID  uuid.UUID `json:"campaign_id"`
		CharacterID uuid.UUID `json:"character_id"`
	}
	resp := newCampaignResponse{
		CampaignID:  updatedCampaign.ID,
		CharacterID: character.ID,
	}
	respondJSON(w, http.StatusCreated, resp)
}

func (s *Server) handlerGetCampaigns(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.Context().Value("userID").(string))
	if err != nil {
		logCampaignError("failed to parse uuid", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	dbResult, err := s.db.GetCampaignsByOwner(r.Context(), userID)
	if err != nil {
		logCampaignError(fmt.Sprintf("failed to retrieve campaigns for user - %v", userID), err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	type getCampaignResponse struct {
		ID     uuid.UUID          `json:"campaign_id"`
		Name   string             `json:"name"`
		Status string             `json:"status"`
		Party  []game.PartyMember `json:"party"`
		Theme  string             `json:"theme"`
	}
	var campaigns []getCampaignResponse
	for _, campaign := range dbResult {
		var party []game.PartyMember
		err = json.Unmarshal(campaign.Party, &party)
		if err != nil {
			logCampaignError("failed to deserialize json from db", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		campaignInfo := getCampaignResponse{
			ID:     campaign.ID,
			Name:   campaign.Name,
			Status: campaign.Status,
			Party:  party,
			Theme:  campaign.Theme,
		}
		campaigns = append(campaigns, campaignInfo)
	}

	logCampaign(fmt.Sprintf("list of campaigns retrieved successfully for user %v", userID))
	respondJSON(w, http.StatusOK, campaigns)
}

func logCampaignError(msg string, err error) {
	log.Printf(" | [CampaignErr] %s:%v", msg, err)
}

func logCampaign(msg string) {
	log.Printf(" | [CampaignInfo] %s", msg)
}
