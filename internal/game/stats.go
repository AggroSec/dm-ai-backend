package game

const (
	modifierDivisor = 5
	resourceDivisor = 5
	baseMaxHP       = 50
	baseMaxWP       = 30
	baseMaxAP       = 3
	baseOvercapAP   = 5
	baseWPRegen     = 1
	hpPerLevel      = 3
	wpPerLevel      = 2
)

// GetModifier returns the stat modifier used in rolls and damage calculations.
func GetModifier(stat int) int {
	return stat / modifierDivisor
}

// GetEquippedItem returns a pointer to the item equipped in the given slot,
// or nil if the slot is empty or the item is not found in inventory.
func GetEquippedItem(combatant Combatant, slot string) *Item {
	var itemID string
	switch slot {
	case "head":
		itemID = combatant.EquippedSlots.Head
	case "chest":
		itemID = combatant.EquippedSlots.Chest
	case "legs":
		itemID = combatant.EquippedSlots.Legs
	case "hands":
		itemID = combatant.EquippedSlots.Hands
	case "weapon":
		itemID = combatant.EquippedSlots.Weapon
	case "offhand":
		itemID = combatant.EquippedSlots.Offhand
	case "accessory_1":
		itemID = combatant.EquippedSlots.Accessory1
	case "accessory_2":
		itemID = combatant.EquippedSlots.Accessory2
	default:
		return nil
	}
	if itemID == "" {
		return nil
	}
	for i, item := range combatant.Inventory {
		if item.ID == itemID {
			return &combatant.Inventory[i]
		}
	}
	return nil
}

// GetTotalStatBuffs sums all stat bonuses from equipped items.
// Returns an Item used as an accumulator — caller accesses .Strength, .Dexterity, etc.
func GetTotalStatBuffs(combatant Combatant) Item {
	slots := []string{"head", "chest", "legs", "hands", "weapon", "offhand", "accessory_1", "accessory_2"}
	var buffs Item
	for _, slot := range slots {
		item := GetEquippedItem(combatant, slot)
		if item == nil {
			continue
		}
		buffs.Strength += item.Strength
		buffs.Dexterity += item.Dexterity
		buffs.Fortitude += item.Fortitude
		buffs.Willpower += item.Willpower
		buffs.Alacrity += item.Alacrity
		buffs.Wisdom += item.Wisdom
	}
	return buffs
}

// DeriveAC returns the armor class based on fortitude and equipped chest armor.
func DeriveAC(combatant Combatant) int {
	buffs := GetTotalStatBuffs(combatant)
	ac := GetModifier(combatant.Fortitude + buffs.Fortitude)
	chest := GetEquippedItem(combatant, "chest")
	if chest != nil {
		ac += chest.BaseDefense
	}
	return ac
}

// DeriveMaxHP returns max HP based on fortitude, level, and gear buffs.
// Scales exponentially — investing in both fortitude and leveling compounds.
func DeriveMaxHP(fortitude, level int, buffs Item) int {
	return baseMaxHP + (level * hpPerLevel) + ((fortitude + buffs.Fortitude) * level / resourceDivisor)
}

// DeriveMaxWP returns max WP based on willpower, level, and gear buffs.
// Same exponential scaling as HP — high willpower blaster builds are intentional.
func DeriveMaxWP(willpower, level int, buffs Item) int {
	return baseMaxWP + (level * wpPerLevel) + ((willpower + buffs.Willpower) * level / resourceDivisor)
}

// DeriveMaxAP returns max AP based on dexterity and gear buffs.
// Flat scaling only — action economy shouldn't snowball with level.
func DeriveMaxAP(dexterity int, buffs Item) int {
	return baseMaxAP + ((dexterity + buffs.Dexterity) / resourceDivisor)
}

// DeriveOvercapAP returns the AP carry-over cap based on wisdom and gear buffs.
// Flat scaling only — tactical banking shouldn't compound with level.
func DeriveOvercapAP(wisdom int, buffs Item) int {
	return baseOvercapAP + ((wisdom + buffs.Wisdom) / resourceDivisor)
}

// DeriveWPRegen returns WP recovered at the start of each turn based on alacrity and gear buffs.
// Flat scaling only — regen shouldn't snowball with level.
func DeriveWPRegen(alacrity int, buffs Item) int {
	return baseWPRegen + ((alacrity + buffs.Alacrity) / resourceDivisor)
}
