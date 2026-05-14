package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AggroSec/dm-ai-backend/internal/auth"
	"github.com/AggroSec/dm-ai-backend/internal/database"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID        string    `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type authResponse struct {
	JWTToken string `json:"token"`
	UserID   string `json:"user_id"`
}

func (s *Server) handlerRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	hashPass, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf(" | error hashing password: %v", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	registerInfo := database.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashPass,
	}

	createdUser, err := s.db.CreateUser(r.Context(), registerInfo)
	if err != nil {
		log.Printf(" | user was not created: %v", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	jsonResp := registerResponse{
		ID:        createdUser.ID.String(),
		Username:  createdUser.Username,
		CreatedAt: createdUser.CreatedAt,
	}
	respondJSON(w, http.StatusCreated, jsonResp)
	log.Printf(" | user register successfully: %v(%v)", jsonResp.Username, jsonResp.ID)
}

func (s *Server) handlerLoginUser(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	user, err := s.db.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		log.Printf(" | attempted login with wrong username: %v, err: %v", req.Username, err)
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	err = auth.VerifyPassword(req.Password, user.HashedPassword)
	if err != nil {
		log.Printf(" | invalid login attempt for user: %v(%v), err: %v", user.Username, user.ID, err)
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	jwtToken, err := auth.GenerateJWT(user.ID.String(), s.cfg.JWTSecret, s.cfg.JWTExpiry)
	if err != nil {
		log.Printf(" | failed to generate jwt for user: %v(%v), err: %v", user.Username, user.ID, err)
		respondError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	respondJSON(w, http.StatusOK, authResponse{
		JWTToken: jwtToken,
		UserID:   user.ID.String(),
	})
	log.Printf(" | jwt created successfully for: %v(%v)", user.Username, user.ID)
}
