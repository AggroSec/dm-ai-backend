package game

import (
	"context"
	"time"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
)

type StatusEffectResponse struct {
	ID                string    `json:"id"`
	AffectedCharacter string    `json:"character_id"`
	Effect            string    `json:"effect"`
	Duration          int       `json:"duration"`
	Persists          bool      `json:"persists"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Instruction       string    `json:"instruction"`
}

// ApplyStatusEffect adds a status effect to a character and returns the result.
// Called by both HTTP handler and AI tool handler.
func ApplyStatusEffect(ctx context.Context, db *database.Queries, characterID uuid.UUID, effect string, duration int, persists bool, ai_instruction string, combatSession *CombatSession) (StatusEffectResponse, error) {
	if combatSession != nil {
		for i, c := range combatSession.Combatants {
			if c.ID == characterID && c.Type == "npc" {
				statusEffect := StatusEffect{
					ID:          uuid.New(),
					Effect:      effect,
					Duration:    duration,
					Persists:    persists,
					IsActive:    true,
					Instruction: ai_instruction,
				}
				combatSession.Combatants[i].StatusEffects = append(combatSession.Combatants[i].StatusEffects, statusEffect)
				err := saveCombatState(ctx, db, combatSession)
				if err != nil {
					return StatusEffectResponse{}, err
				}
				return StatusEffectResponse{
					ID:                statusEffect.ID.String(),
					AffectedCharacter: characterID.String(),
					Effect:            statusEffect.Effect,
					Duration:          statusEffect.Duration,
					Persists:          statusEffect.Persists,
					IsActive:          statusEffect.IsActive,
					Instruction:       statusEffect.Instruction,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}, nil
			}
		}
	}

	dbResult, err := db.AddStatusEffect(ctx, database.AddStatusEffectParams{
		CharacterID: characterID,
		Effect:      effect,
		Duration:    int32(duration),
		Persists:    persists,
		Instruction: ai_instruction,
	})
	if err != nil {
		return StatusEffectResponse{}, err
	}
	return StatusEffectResponse{
		ID:                dbResult.ID.String(),
		AffectedCharacter: characterID.String(),
		Effect:            dbResult.Effect,
		Duration:          int(dbResult.Duration),
		Persists:          dbResult.Persists,
		IsActive:          dbResult.IsActive,
		Instruction:       dbResult.Instruction,
		CreatedAt:         dbResult.CreatedAt,
		UpdatedAt:         dbResult.UpdatedAt,
	}, nil
}

// GetStatusEffects returns all active status effects for a character.
// Called by both HTTP handler and AI tool handler.
func GetStatusEffects(ctx context.Context, db *database.Queries, characterID uuid.UUID) ([]StatusEffectResponse, error) {
	dbResults, err := db.GetStatusEffectsByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var effects []StatusEffectResponse
	for _, e := range dbResults {
		effects = append(effects, StatusEffectResponse{
			ID:                e.ID.String(),
			AffectedCharacter: e.CharacterID.String(),
			Effect:            e.Effect,
			Duration:          int(e.Duration),
			Persists:          e.Persists,
			IsActive:          e.IsActive,
			Instruction:       e.Instruction,
			CreatedAt:         e.CreatedAt,
			UpdatedAt:         e.UpdatedAt,
		})
	}
	return effects, nil
}

// TickStatusEffects decrements duration on all active effects, deactivates expired ones,
// purges inactive effects, and returns the remaining active effects.
// Called by both HTTP handler and AI tool handler.
func TickStatusEffects(ctx context.Context, db *database.Queries, characterID uuid.UUID) ([]StatusEffectResponse, error) {
	current, err := GetStatusEffects(ctx, db, characterID)
	if err != nil {
		return nil, err
	}

	for _, effect := range current {
		effect.Duration -= 1
		if effect.Duration == 0 {
			effect.IsActive = false
		}
		_, err := db.UpdateStatusDuration(ctx, database.UpdateStatusDurationParams{
			ID:          uuid.MustParse(effect.ID),
			Duration:    int32(effect.Duration),
			IsActive:    effect.IsActive,
			CharacterID: characterID,
		})
		if err != nil {
			return nil, err
		}
	}

	db.PurgeInactiveEffects(ctx, characterID)

	return GetStatusEffects(ctx, db, characterID)
}

func RemoveStatusEffect(ctx context.Context, db *database.Queries, characterID uuid.UUID, effectID uuid.UUID, combatSession *CombatSession) error {
	if combatSession != nil {
		for i, c := range combatSession.Combatants {
			if c.ID == characterID && c.Type == "npc" {
				var updatedEffects []StatusEffect
				for _, se := range combatSession.Combatants[i].StatusEffects {
					if se.ID != effectID {
						updatedEffects = append(updatedEffects, se)
					}
				}
				combatSession.Combatants[i].StatusEffects = updatedEffects
				err := saveCombatState(ctx, db, combatSession)
				if err != nil {
					return err
				}

			}
		}
	}

	return db.RemoveStatusEffect(ctx, database.RemoveStatusEffectParams{
		ID:          effectID,
		CharacterID: characterID,
	})
}
