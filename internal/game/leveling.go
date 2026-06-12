package game

import (
	"context"
	"fmt"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
)

type XPAwardResult struct {
	NewXP               int    `json:"new_xp"`
	LeveledUp           bool   `json:"leveled_up"`
	NewLevel            int    `json:"new_level,omitempty"`
	TalentPointsAwarded int    `json:"talent_points_awarded,omitempty"`
	StatPointsAwarded   int    `json:"stat_points_awarded,omitempty"`
	LevelUpInstructions string `json:"level_up_instructions,omitempty"`
}

const (
	StatPointsPerLevel   = 5
	TalentPointsPerLevel = 1
	LevelUpInstructions  = `Character has leveled up. Take the following steps in order:
1. Call get_skills to retrieve the full skill tree and present available skills to the player organized by branch.
2. Wait for the player to choose how to invest their talent points. Call update_character with the updated talents_invested array once decided.
3. Inform the player they have %d stat points to distribute across: Strength, Dexterity, Fortitude, Willpower, Alacrity, Wisdom. Current stats are visible on the character sheet.
4. Wait for the player to allocate their stat points. Call update_character with the final stat values once decided.
5. Confirm level up is complete by presenting the full charactersheet to the player and continue the story.`
)

func AwardXP(ctx context.Context, db *database.Queries, characterID uuid.UUID, amount int) (XPAwardResult, error) {
	loadCharacter, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return XPAwardResult{}, err
	}

	currentLevel := loadCharacter.Level
	currentXP := loadCharacter.Experience
	newXP := currentXP + int32(amount)
	levelUp := false
	xpThreshold := int32(100 * currentLevel * (currentLevel + 1) / 2)
	if newXP >= xpThreshold {
		levelUp = true
	}

	_, err = db.UpdateCharacterXP(ctx, database.UpdateCharacterXPParams{
		ID:         characterID,
		Experience: newXP,
	})
	if err != nil {
		return XPAwardResult{}, err
	}

	if levelUp {
		_, err = db.UpdateCharacterLevel(ctx, database.UpdateCharacterLevelParams{
			ID:    characterID,
			Level: currentLevel + 1,
		})
		if err != nil {
			return XPAwardResult{}, err
		}
		results := XPAwardResult{
			NewXP:               int(newXP),
			LeveledUp:           true,
			NewLevel:            int(currentLevel) + 1,
			TalentPointsAwarded: TalentPointsPerLevel,
			StatPointsAwarded:   StatPointsPerLevel,
			LevelUpInstructions: fmt.Sprintf(LevelUpInstructions, StatPointsPerLevel),
		}
		return results, nil
	}

	results := XPAwardResult{
		NewXP:     int(newXP),
		LeveledUp: false,
	}
	return results, nil
}
