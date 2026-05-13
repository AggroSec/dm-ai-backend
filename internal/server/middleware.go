package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/AggroSec/dm-ai-backend/internal/auth"
)

func (s *Server) requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := getBearerToken(r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "unathorized")
			return
		}

		userID, err := auth.VerifyJWT(token, s.cfg.JWTSecret)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), "userID", userID)
		handler(w, r.WithContext(ctx))
	}
}

func getBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header")
	}

	splitHeader := strings.Split(authHeader, " ")
	if len(splitHeader) != 2 || splitHeader[0] != "Bearer" {
		return "", errors.New("invalid authorization header")
	}

	return splitHeader[1], nil
}
