package server

import "net/http"

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"env":    s.cfg.AppEnv,
	})
}
