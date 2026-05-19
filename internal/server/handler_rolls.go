package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/game"
)

func (s *Server) handlerDiceRolls(w http.ResponseWriter, r *http.Request) {
	var validDice = map[string]int{
		"d4":  4,
		"d6":  6,
		"d8":  8,
		"d10": 10,
		"d12": 12,
		"d20": 20,
	}

	type dicerollRequest struct {
		Die   string `json:"die"`
		Count int    `json:"count"`
	}

	type dicerollResponse struct {
		Die     string `json:"die"`
		Count   int    `json:"count"`
		Results []int  `json:"results"`
		Total   int    `json:"total"`
	}

	var diceRequest dicerollRequest
	err := json.NewDecoder(r.Body).Decode(&diceRequest)
	if err != nil {
		logRollerError("failed to decode json response, bad request", err)
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	sides, ok := validDice[diceRequest.Die]
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid die type")
		return
	}

	if diceRequest.Count < 1 {
		respondError(w, http.StatusBadRequest, "count must be greater than zero.")
		return
	}

	results, total := game.DiceRoll(sides, diceRequest.Count)

	diceResp := dicerollResponse{
		Die:     diceRequest.Die,
		Count:   diceRequest.Count,
		Results: results,
		Total:   total,
	}
	respondJSON(w, http.StatusOK, diceResp)
}

func logRollerError(errorMsg string, err error) {
	log.Printf(" | [DiceRoller] %v: %v", errorMsg, err)
}
