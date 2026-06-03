package game

import "github.com/google/uuid"

type Item struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`       // weapon, armor, accessory
	ArmorType   string `json:"armor_type"` // light, medium, heavy, none
	BaseDefense int    `json:"base_defense"`
	APCost      int    `json:"ap_cost"`
	Strength    int    `json:"strength"`
	Dexterity   int    `json:"dexterity"`
	Fortitude   int    `json:"fortitude"`
	Willpower   int    `json:"willpower"`
	Alacrity    int    `json:"alacrity"`
	Wisdom      int    `json:"wisdom"`
}

type EquippedSlots struct {
	Head       string `json:"head"`
	Chest      string `json:"chest"`
	Legs       string `json:"legs"`
	Hands      string `json:"hands"`
	Weapon     string `json:"weapon"`
	Offhand    string `json:"offhand"`
	Accessory1 string `json:"accessory_1"`
	Accessory2 string `json:"accessory_2"`
}

type StatusEffect struct {
	ID          uuid.UUID `json:"id"`
	Effect      string    `json:"effect"`
	Duration    int       `json:"duration"`
	Persists    bool      `json:"persists"`
	IsActive    bool      `json:"is_active"`
	Instruction string    `json:"instruction"`
}

type CombatSession struct {
	ID          uuid.UUID   `json:"id"`
	CampaignID  uuid.UUID   `json:"campaign_id"`
	Status      string      `json:"status"`
	Round       int         `json:"round"`
	Combatants  []Combatant `json:"combatants"`
	CurrentTurn uuid.UUID   `json:"current_turn"`
	TurnOrder   []uuid.UUID `json:"turn_order"`
}

type Combatant struct {
	ID            uuid.UUID      `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	HP            int            `json:"hp"`
	MaxHP         int            `json:"max_hp"`
	WP            int            `json:"wp"`
	MaxWP         int            `json:"max_wp"`
	AP            int            `json:"ap"`
	MaxAP         int            `json:"max_ap"`
	OvercapAP     int            `json:"overcap_ap"`
	Initiative    int            `json:"initiative"`
	Strength      int            `json:"strength"`
	Dexterity     int            `json:"dexterity"`
	Fortitude     int            `json:"fortitude"`
	Willpower     int            `json:"willpower"`
	Alacrity      int            `json:"alacrity"`
	Wisdom        int            `json:"wisdom"`
	IsAlive       bool           `json:"is_alive"`
	StatusEffects []StatusEffect `json:"status_effects"`
	Inventory     []Item         `json:"inventory"`
	EquippedSlots EquippedSlots  `json:"equipped_slots"`
	IsDefending   bool           `json:"is_defending"`
}

type PartyMember struct {
	CharacterID uuid.UUID `json:"character_id"`
	UserID      uuid.UUID `json:"user_id"`
}
