package server

import (
	"encoding/json"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/config"
	"github.com/AggroSec/dm-ai-backend/internal/database"
)

type Server struct {
	cfg *config.Config
	db  *database.Queries
}

func New(cfg *config.Config, db *database.Queries) *Server {
	return &Server{
		cfg: cfg,
		db:  db,
	}
}

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /auth/register", s.handlerRegisterUser)
	mux.HandleFunc("POST /auth/login", s.handlerLoginUser)
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
