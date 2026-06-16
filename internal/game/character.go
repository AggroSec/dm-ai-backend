package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

type CharacterInfo struct {
	ID                    string             `json:"id"`
	UserID                string             `json:"user_id"`
	Name                  string             `json:"name"`
	Race                  string             `json:"race"`
	Class                 string             `json:"class"`
	Level                 int                `json:"level"`
	Experience            int                `json:"experience"`
	DrivingFate           string             `json:"driving_fate"`
	BindingFate           string             `json:"binding_fate"`
	Strength              int                `json:"strength"`
	Dexterity             int                `json:"dexterity"`
	Fortitude             int                `json:"fortitude"`
	Willpower             int                `json:"willpower"`
	Alacrity              int                `json:"alacrity"`
	Wisdom                int                `json:"wisdom"`
	CurrentHp             int                `json:"current_hp"`
	MaxHp                 int                `json:"max_hp"`
	CurrentWp             int                `json:"current_wp"`
	MaxWp                 int                `json:"max_wp"`
	ActionPoints          int                `json:"action_points"`
	MaxAp                 int                `json:"max_ap"`
	OvercapAp             int                `json:"overcap_ap"`
	TalentPointsAvailable int                `json:"talent_points_available"`
	TalentsInvested       []TalentInvestment `json:"talents_invested"`
	Inventory             []Item             `json:"inventory"`
	EquippedSlots         EquippedSlots      `json:"equipped_slots"`
}

type UpdateCharacterRequest struct {
	Name                  *string            `json:"name,omitempty"`
	Race                  *string            `json:"race,omitempty"`
	Class                 *string            `json:"class,omitempty"`
	Level                 *int               `json:"level,omitempty"`
	DrivingFate           *string            `json:"driving_fate,omitempty"`
	BindingFate           *string            `json:"binding_fate,omitempty"`
	Strength              *int               `json:"strength,omitempty"`
	Dexterity             *int               `json:"dexterity,omitempty"`
	Fortitude             *int               `json:"fortitude,omitempty"`
	Willpower             *int               `json:"willpower,omitempty"`
	Alacrity              *int               `json:"alacrity,omitempty"`
	Wisdom                *int               `json:"wisdom,omitempty"`
	CurrentHp             *int               `json:"current_hp,omitempty"`
	MaxHp                 *int               `json:"max_hp,omitempty"`
	CurrentWp             *int               `json:"current_wp,omitempty"`
	MaxWp                 *int               `json:"max_wp,omitempty"`
	ActionPoints          *int               `json:"action_points,omitempty"`
	MaxAp                 *int               `json:"max_ap,omitempty"`
	OvercapAp             *int               `json:"overcap_ap,omitempty"`
	TalentPointsAvailable *int               `json:"talent_points_available,omitempty"`
	TalentsInvested       []TalentInvestment `json:"talents_invested,omitempty"`
}

func GetCharacterInfo(ctx context.Context, db *database.Queries, characterID uuid.UUID) (CharacterInfo, error) {
	dbChar, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to get character: %w", err)
	}

	var talents []TalentInvestment
	if len(dbChar.TalentsInvested) > 0 {
		if err := json.Unmarshal(dbChar.TalentsInvested, &talents); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal talents: %w", err)
		}
	}

	var inventory []Item
	if len(dbChar.Inventory) > 0 {
		if err := json.Unmarshal(dbChar.Inventory, &inventory); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal inventory: %w", err)
		}
	}

	var equippedSlots EquippedSlots
	if dbChar.EquippedSlots.Valid {
		if err := json.Unmarshal(dbChar.EquippedSlots.RawMessage, &equippedSlots); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal equipped slots: %w", err)
		}
	}

	return CharacterInfo{
		ID:                    dbChar.ID.String(),
		UserID:                dbChar.UserID.String(),
		Name:                  dbChar.Name,
		Race:                  dbChar.Race,
		Class:                 dbChar.Class,
		Level:                 int(dbChar.Level),
		Experience:            int(dbChar.Experience),
		DrivingFate:           dbChar.DrivingFate,
		BindingFate:           dbChar.BindingFate,
		Strength:              int(dbChar.Strength),
		Dexterity:             int(dbChar.Dexterity),
		Fortitude:             int(dbChar.Fortitude),
		Willpower:             int(dbChar.Willpower),
		Alacrity:              int(dbChar.Alacrity),
		Wisdom:                int(dbChar.Wisdom),
		CurrentHp:             int(dbChar.CurrentHp),
		MaxHp:                 int(dbChar.MaxHp),
		CurrentWp:             int(dbChar.CurrentWp),
		MaxWp:                 int(dbChar.MaxWp),
		ActionPoints:          int(dbChar.ActionPoints),
		MaxAp:                 int(dbChar.MaxAp),
		OvercapAp:             int(dbChar.OvercapAp),
		TalentPointsAvailable: int(dbChar.TalentPointsAvailable),
		TalentsInvested:       talents,
		Inventory:             inventory,
		EquippedSlots:         equippedSlots,
	}, nil
}

func ApplyCharacterUpdates(ctx context.Context, db *database.Queries, characterID uuid.UUID, req UpdateCharacterRequest) (CharacterInfo, error) {
	dbChar, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to get character: %w", err)
	}

	if req.Name != nil {
		dbChar.Name = *req.Name
	}
	if req.Race != nil {
		dbChar.Race = *req.Race
	}
	if req.Class != nil {
		dbChar.Class = *req.Class
	}
	if req.Level != nil {
		dbChar.Level = int32(*req.Level)
	}
	if req.DrivingFate != nil {
		dbChar.DrivingFate = *req.DrivingFate
	}
	if req.BindingFate != nil {
		dbChar.BindingFate = *req.BindingFate
	}
	if req.Strength != nil {
		dbChar.Strength = int32(*req.Strength)
	}
	if req.Dexterity != nil {
		dbChar.Dexterity = int32(*req.Dexterity)
	}
	if req.Fortitude != nil {
		dbChar.Fortitude = int32(*req.Fortitude)
	}
	if req.Willpower != nil {
		dbChar.Willpower = int32(*req.Willpower)
	}
	if req.Alacrity != nil {
		dbChar.Alacrity = int32(*req.Alacrity)
	}
	if req.Wisdom != nil {
		dbChar.Wisdom = int32(*req.Wisdom)
	}
	if req.CurrentHp != nil {
		dbChar.CurrentHp = int32(*req.CurrentHp)
	}
	if req.MaxHp != nil {
		dbChar.MaxHp = int32(*req.MaxHp)
	}
	if req.CurrentWp != nil {
		dbChar.CurrentWp = int32(*req.CurrentWp)
	}
	if req.MaxWp != nil {
		dbChar.MaxWp = int32(*req.MaxWp)
	}
	if req.ActionPoints != nil {
		dbChar.ActionPoints = int32(*req.ActionPoints)
	}
	if req.MaxAp != nil {
		dbChar.MaxAp = int32(*req.MaxAp)
	}
	if req.OvercapAp != nil {
		dbChar.OvercapAp = int32(*req.OvercapAp)
	}
	if req.TalentPointsAvailable != nil {
		dbChar.TalentPointsAvailable = int32(*req.TalentPointsAvailable)
	}

	var talentsJSON []byte
	if req.TalentsInvested != nil {
		talentsJSON, err = json.Marshal(req.TalentsInvested)
		if err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to marshal talents: %w", err)
		}
		dbChar.TalentsInvested = talentsJSON
	}

	var equippedSlots pqtype.NullRawMessage
	if dbChar.EquippedSlots.Valid {
		equippedSlots = dbChar.EquippedSlots
	}

	_, err = db.UpdateCharacter(ctx, database.UpdateCharacterParams{
		ID:                    characterID,
		Name:                  dbChar.Name,
		Race:                  dbChar.Race,
		Class:                 dbChar.Class,
		Level:                 dbChar.Level,
		DrivingFate:           dbChar.DrivingFate,
		BindingFate:           dbChar.BindingFate,
		Strength:              dbChar.Strength,
		Dexterity:             dbChar.Dexterity,
		Fortitude:             dbChar.Fortitude,
		Willpower:             dbChar.Willpower,
		Alacrity:              dbChar.Alacrity,
		Wisdom:                dbChar.Wisdom,
		CurrentHp:             dbChar.CurrentHp,
		MaxHp:                 dbChar.MaxHp,
		CurrentWp:             dbChar.CurrentWp,
		MaxWp:                 dbChar.MaxWp,
		ActionPoints:          dbChar.ActionPoints,
		MaxAp:                 dbChar.MaxAp,
		OvercapAp:             dbChar.OvercapAp,
		TalentPointsAvailable: dbChar.TalentPointsAvailable,
		TalentsInvested:       dbChar.TalentsInvested,
		EquippedSlots:         equippedSlots,
	})
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to update character: %w", err)
	}

	return GetCharacterInfo(ctx, db, characterID)
}

func CharacterRest(ctx context.Context, db *database.Queries, characterID uuid.UUID, restType string) error {
	character, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return err
	}

	switch strings.ToLower(restType) {
	case "short":
		missingHP := character.MaxHp - character.CurrentHp
		missingWP := character.MaxWp - character.CurrentWp
		HPHealing := missingHP / 2
		WPHealing := missingWP / 2
		err = db.UpdateCharacterHP(ctx, database.UpdateCharacterHPParams{
			ID:        characterID,
			CurrentHp: character.CurrentHp + HPHealing,
		})
		if err != nil {
			return err
		}
		err = db.UpdateCharacterWP(ctx, database.UpdateCharacterWPParams{
			ID:        characterID,
			CurrentWp: character.CurrentWp + WPHealing,
		})
		if err != nil {
			return err
		}
		statusEffects, err := db.GetStatusEffectsByID(ctx, characterID)
		if err != nil {
			return err
		}
		for _, effect := range statusEffects {
			err = RemoveStatusEffect(ctx, db, characterID, effect.ID, nil)
			if err != nil {
				return err
			}
		}
		return nil
	case "long":
		err = db.UpdateCharacterHP(ctx, database.UpdateCharacterHPParams{
			ID:        characterID,
			CurrentHp: character.MaxHp,
		})
		if err != nil {
			return err
		}
		err = db.UpdateCharacterWP(ctx, database.UpdateCharacterWPParams{
			ID:        characterID,
			CurrentWp: character.MaxWp,
		})
		if err != nil {
			return err
		}
		statusEffects, err := db.GetStatusEffectsByID(ctx, characterID)
		if err != nil {
			return err
		}
		for _, effect := range statusEffects {
			err = RemoveStatusEffect(ctx, db, characterID, effect.ID, nil)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("not a valid rest type")
}
