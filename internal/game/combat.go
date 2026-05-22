package game

import (
	"context"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
)

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
}

type PartyMember struct {
	CharacterID uuid.UUID `json:"character_id"`
	UserID      uuid.UUID `json:"user_id"`
}

func CreateCombatSession(ctx context.Context, db *database.Queries, party []PartyMember, npcs []Combatant, campaignID uuid.UUID) (CombatSession, error) {
	var session CombatSession
	for _, member := range party {
		dbPlayer, err := db.GetCharacterByID(ctx, database.GetCharacterByIDParams{
			ID:     member.CharacterID,
			UserID: member.UserID,
		})
		if err != nil {
			return CombatSession{}, err
		}
		session.Combatants = append(session.Combatants, Combatant{
			ID:        dbPlayer.ID,
			Name:      dbPlayer.Name,
			Type:      "player",
			HP:        int(dbPlayer.CurrentHp),
			MaxHP:     int(dbPlayer.MaxHp),
			WP:        int(dbPlayer.CurrentWp),
			MaxWP:     int(dbPlayer.MaxWp),
			AP:        int(dbPlayer.ActionPoints),
			Strength:  int(dbPlayer.Strength),
			Dexterity: int(dbPlayer.Dexterity),
			Fortitude: int(dbPlayer.Fortitude),
			Willpower: int(dbPlayer.Willpower),
			Alacrity:  int(dbPlayer.Alacrity),
			Wisdom:    int(dbPlayer.Wisdom),
			IsAlive:   true,
			Inventory: []Item{}, // TODO: Fetch inventory items
		})
	}

	for _, npc := range npcs {
		npc.ID = uuid.New()
		npc.Type = "npc"
		npc.IsAlive = true
		npc.StatusEffects = []StatusEffect{}
		session.Combatants = append(session.Combatants, npc)
	}
	return session, nil
}

func RollInitiative(combatants []Combatant) []uuid.UUID {
	return nil
}

func GetModifier(stat int) int {
	return 0 //come back to write formula once rules are finalized
}
