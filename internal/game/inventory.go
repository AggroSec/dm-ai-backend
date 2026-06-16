package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// GiveItem adds an item to a character's inventory and persists it to the DB.
func GiveItem(ctx context.Context, db *database.Queries, characterID uuid.UUID, item Item) (CharacterInfo, error) {
	dbChar, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to get character: %w", err)
	}

	var inventory []Item
	if len(dbChar.Inventory) > 0 {
		if err := json.Unmarshal(dbChar.Inventory, &inventory); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal inventory: %w", err)
		}
	}

	inventory = append(inventory, item)

	inventoryJSON, err := json.Marshal(inventory)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to marshal inventory: %w", err)
	}

	_, err = db.UpdateCharacterInventory(ctx, database.UpdateCharacterInventoryParams{
		ID:        characterID,
		Inventory: inventoryJSON,
	})
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to update inventory: %w", err)
	}

	return GetCharacterInfo(ctx, db, characterID)
}

// RemoveItem removes an item from a character's inventory by ID and persists to DB.
// Does not unequip — caller should unequip first if needed.
func RemoveItem(ctx context.Context, db *database.Queries, characterID uuid.UUID, itemID string) (CharacterInfo, error) {
	dbChar, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to get character: %w", err)
	}

	var inventory []Item
	if len(dbChar.Inventory) > 0 {
		if err := json.Unmarshal(dbChar.Inventory, &inventory); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal inventory: %w", err)
		}
	}

	found := false
	var updated []Item
	for _, item := range inventory {
		if item.ID == itemID {
			found = true
			continue
		}
		updated = append(updated, item)
	}
	if !found {
		return CharacterInfo{}, errors.New("item not found in inventory")
	}

	inventoryJSON, err := json.Marshal(updated)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to marshal inventory: %w", err)
	}

	_, err = db.UpdateCharacterInventory(ctx, database.UpdateCharacterInventoryParams{
		ID:        characterID,
		Inventory: inventoryJSON,
	})
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to update inventory: %w", err)
	}

	return GetCharacterInfo(ctx, db, characterID)
}

// EquipItem equips an item from the character's inventory to the specified slot and persists to DB.
func EquipItem(ctx context.Context, db *database.Queries, characterID uuid.UUID, slot, itemID string, combatID *uuid.UUID) (CharacterInfo, error) {
	dbChar, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to get character: %w", err)
	}

	var inventory []Item
	if len(dbChar.Inventory) > 0 {
		if err := json.Unmarshal(dbChar.Inventory, &inventory); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal inventory: %w", err)
		}
	}

	found := false
	for _, item := range inventory {
		if item.ID == itemID {
			found = true
			break
		}
	}
	if !found {
		return CharacterInfo{}, errors.New("item not found in inventory")
	}

	var equippedSlots EquippedSlots
	if dbChar.EquippedSlots.Valid {
		if err := json.Unmarshal(dbChar.EquippedSlots.RawMessage, &equippedSlots); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal equipped slots: %w", err)
		}
	}

	switch slot {
	case "head":
		equippedSlots.Head = itemID
	case "chest":
		equippedSlots.Chest = itemID
	case "legs":
		equippedSlots.Legs = itemID
	case "hands":
		equippedSlots.Hands = itemID
	case "weapon":
		equippedSlots.Weapon = itemID
	case "offhand":
		equippedSlots.Offhand = itemID
	case "accessory_1":
		equippedSlots.Accessory1 = itemID
	case "accessory_2":
		equippedSlots.Accessory2 = itemID
	default:
		return CharacterInfo{}, errors.New("invalid equipment slot")
	}

	slotsJSON, err := json.Marshal(equippedSlots)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to marshal equipped slots: %w", err)
	}

	_, err = db.UpdateCharacterEquippedSlots(ctx, database.UpdateCharacterEquippedSlotsParams{
		ID: characterID,
		EquippedSlots: pqtype.NullRawMessage{
			RawMessage: slotsJSON,
			Valid:      true,
		},
	})
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to update equipped slots: %w", err)
	}

	if combatID != nil {
		session, err := GetActiveCombatSession(ctx, db, *combatID)
		if err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to get combat session: %w", err)
		}
		for i, c := range session.Combatants {
			if c.ID == characterID {
				session.Combatants[i].EquippedSlots = equippedSlots
				break
			}
		}
		if err := saveCombatState(ctx, db, &session); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to save combat state: %w", err)
		}
	}

	return GetCharacterInfo(ctx, db, characterID)
}

// UnequipItem clears the specified equipment slot and persists to DB.
func UnequipItem(ctx context.Context, db *database.Queries, characterID uuid.UUID, slot string) (CharacterInfo, error) {
	dbChar, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to get character: %w", err)
	}

	var equippedSlots EquippedSlots
	if dbChar.EquippedSlots.Valid {
		if err := json.Unmarshal(dbChar.EquippedSlots.RawMessage, &equippedSlots); err != nil {
			return CharacterInfo{}, fmt.Errorf("failed to unmarshal equipped slots: %w", err)
		}
	}

	switch slot {
	case "head":
		equippedSlots.Head = ""
	case "chest":
		equippedSlots.Chest = ""
	case "legs":
		equippedSlots.Legs = ""
	case "hands":
		equippedSlots.Hands = ""
	case "weapon":
		equippedSlots.Weapon = ""
	case "offhand":
		equippedSlots.Offhand = ""
	case "accessory_1":
		equippedSlots.Accessory1 = ""
	case "accessory_2":
		equippedSlots.Accessory2 = ""
	default:
		return CharacterInfo{}, errors.New("invalid equipment slot")
	}

	slotsJSON, err := json.Marshal(equippedSlots)
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to marshal equipped slots: %w", err)
	}

	_, err = db.UpdateCharacterEquippedSlots(ctx, database.UpdateCharacterEquippedSlotsParams{
		ID: characterID,
		EquippedSlots: pqtype.NullRawMessage{
			RawMessage: slotsJSON,
			Valid:      true,
		},
	})
	if err != nil {
		return CharacterInfo{}, fmt.Errorf("failed to update equipped slots: %w", err)
	}

	return GetCharacterInfo(ctx, db, characterID)
}
