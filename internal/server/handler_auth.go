package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/AggroSec/dm-ai-backend/internal/auth"
	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
)

const MinPasswordLength = 15

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
	JWTToken     string `json:"token"`
	UserID       string `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
}

type requestToken struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handlerRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if utf8.RuneCountInString(req.Password) < MinPasswordLength {
		respondError(w, http.StatusBadRequest, "password too short")
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			log.Printf(" | registration attempted with duplicate username: %v", req.Username)
			respondError(w, http.StatusConflict, "username already taken")
			return
		}
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
		respondError(w, http.StatusBadRequest, "invalid request body")
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

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		log.Printf(" | failed to generate refresh token for user: %v(%v), err: %v", user.Username, user.ID, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	_, err = s.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(s.cfg.JWTRefreshExpiry),
	})

	respondJSON(w, http.StatusOK, authResponse{
		JWTToken:     jwtToken,
		UserID:       user.ID.String(),
		RefreshToken: refreshToken,
	})
	log.Printf(" | jwt created successfully for: %v(%v)", user.Username, user.ID)
}

func (s *Server) handlerRefreshJWT(w http.ResponseWriter, r *http.Request) {
	var req requestToken
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf(" | failed to decode json: %v", err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	refreshToken, err := s.db.GetRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		log.Printf(" | failed to retrieve refresh token from db: %v", err)
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		log.Print(" | token is expired")
		respondError(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	newJWT, err := auth.GenerateJWT(refreshToken.UserID.String(), s.cfg.JWTSecret, s.cfg.JWTExpiry)
	if err != nil {
		log.Printf(" | failed to generate jwt for user - err: %v", err)
		respondError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	type refreshResponse struct {
		NewJWT string `json:"jwt"`
	}
	respondJSON(w, http.StatusOK, refreshResponse{NewJWT: newJWT})
}

func (s *Server) HandlerLogout(w http.ResponseWriter, r *http.Request) {
	var req requestToken
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf(" | failed to decode json: %v", err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = s.db.DeleteRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		log.Printf(" | failed to delete refresh token: %v", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, "logged out successfully")
}
