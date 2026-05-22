package game

import "github.com/google/uuid"

type StatusEffect struct {
	ID       uuid.UUID `json:"id"`
	Effect   string    `json:"effect"`
	Duration int       `json:"duration"`
	Persists bool      `json:"persists"`
	IsActive bool      `json:"is_active"`
}
