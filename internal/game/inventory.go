package game

import "errors"

// EquipItem equips an item from the combatant's inventory to the specified slot.
// Called by the equip_item AI tool handler.
func EquipItem(combatant *Combatant, slot, itemID string) error {
	// Verify item exists in inventory
	found := false
	for _, item := range combatant.Inventory {
		if item.ID == itemID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("item not found in inventory")
	}

	// Assign to slot
	switch slot {
	case "head":
		combatant.EquippedSlots.Head = itemID
	case "chest":
		combatant.EquippedSlots.Chest = itemID
	case "legs":
		combatant.EquippedSlots.Legs = itemID
	case "hands":
		combatant.EquippedSlots.Hands = itemID
	case "weapon":
		combatant.EquippedSlots.Weapon = itemID
	case "offhand":
		combatant.EquippedSlots.Offhand = itemID
	case "accessory_1":
		combatant.EquippedSlots.Accessory1 = itemID
	case "accessory_2":
		combatant.EquippedSlots.Accessory2 = itemID
	default:
		return errors.New("invalid equipment slot")
	}
	return nil
}

// UnequipItem clears the specified equipment slot.
// Called by the equip_item AI tool handler when swapping gear.
func UnequipItem(combatant *Combatant, slot string) error {
	switch slot {
	case "head":
		combatant.EquippedSlots.Head = ""
	case "chest":
		combatant.EquippedSlots.Chest = ""
	case "legs":
		combatant.EquippedSlots.Legs = ""
	case "hands":
		combatant.EquippedSlots.Hands = ""
	case "weapon":
		combatant.EquippedSlots.Weapon = ""
	case "offhand":
		combatant.EquippedSlots.Offhand = ""
	case "accessory_1":
		combatant.EquippedSlots.Accessory1 = ""
	case "accessory_2":
		combatant.EquippedSlots.Accessory2 = ""
	default:
		return errors.New("invalid equipment slot")
	}
	return nil
}

// AddItem adds an item to the combatant's inventory.
// Called by the give_item AI tool handler.
func AddItem(combatant *Combatant, item Item) {
	combatant.Inventory = append(combatant.Inventory, item)
}

// RemoveItem removes an item from the combatant's inventory by ID.
// Does not unequip — caller should unequip first if needed.
func RemoveItem(combatant *Combatant, itemID string) error {
	for i, item := range combatant.Inventory {
		if item.ID == itemID {
			combatant.Inventory = append(combatant.Inventory[:i], combatant.Inventory[i+1:]...)
			return nil
		}
	}
	return errors.New("item not found in inventory")
}
