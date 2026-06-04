package server

import (
	"encoding/json"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/ai"
	"github.com/AggroSec/dm-ai-backend/internal/config"
	"github.com/AggroSec/dm-ai-backend/internal/database"
)

type Server struct {
	cfg      *config.Config
	db       *database.Queries
	aiClient *ai.Client
}

func New(cfg *config.Config, db *database.Queries) *Server {
	return &Server{
		cfg:      cfg,
		db:       db,
		aiClient: ai.NewClient(*cfg),
	}
}

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /auth/register", s.handlerRegisterUser)
	mux.HandleFunc("POST /auth/login", s.handlerLoginUser)
	mux.HandleFunc("POST /characters", s.requireAuth(s.handlerCreateCharacter))
	mux.HandleFunc("GET /characters", s.requireAuth(s.handlerGetUserCharacters))
	mux.HandleFunc("GET /characters/{id}", s.handlerGetCharacterByID)
	mux.HandleFunc("PUT /characters/{id}", s.requireInternal(s.handlerUpdateCharacter))
	mux.HandleFunc("DELETE /characters/{id}", s.requireAuth(s.handlerDeleteCharacter))
	mux.HandleFunc("POST /rolls", s.handlerDiceRolls)
	mux.HandleFunc("POST /characters/{id}/status_effects", s.requireInternal(s.handlerApplyStatusEffect))
	mux.HandleFunc("GET /characters/{id}/status_effects", s.requireAuth(s.handlerGetStatusEffectsByID))
	mux.HandleFunc("POST /characters/{id}/status_effects/tick", s.requireInternal(s.handlerStatusEffectsTick))
	mux.HandleFunc("POST /combat", s.requireInternal(s.handlerStartCombat))
	mux.HandleFunc("GET /combat/{id}", s.requireAuth(s.handlerGetActiveCombatSession))
	mux.HandleFunc("POST /combat/{id}/turn", s.requireInternal(s.handlerAdvanceTurn))
	mux.HandleFunc("POST /combat/{id}/action", s.requireInternal(s.handlerTakeAction))
	mux.HandleFunc("POST /combat/{id}/end", s.requireInternal(s.handlerEndCombat))
	mux.HandleFunc("POST /ai/test", s.requireInternal(s.handlerAITest))
	return mux
}

func respondJSON(w http.ResponseWriter, status int, val any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(val)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
