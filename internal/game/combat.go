package game

import "github.com/google/uuid"

type CombatSession struct {
}

type Combatant struct {
	ID            uuid.UUID
	Name          string
	Type          string
	HP            int
	MaxHP         int
	WP            int
	AP            int
	Initiative    int
	Strength      int
	Dexterity     int
	Fortitude     int
	Willpower     int
	Alacrity      int
	Wisdom        int
	IsAlive       bool
	StatusEffects []StatusEffect
}
