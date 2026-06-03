package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sort"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
)

func CreateCombatSession(ctx context.Context, db *database.Queries, party []PartyMember, npcs []Combatant, campaignID *uuid.UUID) (CombatSession, error) {
	var session CombatSession
	for _, member := range party {
		dbPlayer, err := db.GetCharacterByID(ctx, member.CharacterID)
		if err != nil {
			return CombatSession{}, err
		}

		// unmarshal inventory from DB
		var inventory []Item
		if len(dbPlayer.Inventory) > 0 {
			if err := json.Unmarshal(dbPlayer.Inventory, &inventory); err != nil {
				return CombatSession{}, err
			}
		}

		// unmarshal equipped slots from DB
		var equippedSlots EquippedSlots
		if dbPlayer.EquippedSlots.Valid {
			if err := json.Unmarshal(dbPlayer.EquippedSlots.RawMessage, &equippedSlots); err != nil {
				return CombatSession{}, err
			}
		}

		session.Combatants = append(session.Combatants, Combatant{
			ID:            dbPlayer.ID,
			Name:          dbPlayer.Name,
			Type:          "player",
			HP:            int(dbPlayer.CurrentHp),
			MaxHP:         int(dbPlayer.MaxHp),
			WP:            int(dbPlayer.CurrentWp),
			MaxWP:         int(dbPlayer.MaxWp),
			AP:            int(dbPlayer.ActionPoints),
			MaxAP:         int(dbPlayer.MaxAp),
			OvercapAP:     int(dbPlayer.OvercapAp),
			Strength:      int(dbPlayer.Strength),
			Dexterity:     int(dbPlayer.Dexterity),
			Fortitude:     int(dbPlayer.Fortitude),
			Willpower:     int(dbPlayer.Willpower),
			Alacrity:      int(dbPlayer.Alacrity),
			Wisdom:        int(dbPlayer.Wisdom),
			IsAlive:       true,
			Inventory:     inventory,
			EquippedSlots: equippedSlots,
			IsDefending:   false,
		})
	}

	for _, npc := range npcs {
		npc.ID = uuid.New()
		npc.IsAlive = true
		npc.StatusEffects = []StatusEffect{}
		npc.IsDefending = false
		session.Combatants = append(session.Combatants, npc)
	}

	session.TurnOrder, session.Combatants = RollInitiative(session.Combatants)
	session.CurrentTurn = session.TurnOrder[0]
	session.Status = "active"
	session.Round = 1

	turnOrderJSON, err := json.Marshal(session.TurnOrder)
	if err != nil {
		return CombatSession{}, err
	}

	combatantsJSON, err := json.Marshal(session.Combatants)
	if err != nil {
		return CombatSession{}, err
	}

	dbCombatSession, err := db.CreateCombatSession(ctx, database.CreateCombatSessionParams{
		CurrentTurn: session.CurrentTurn,
		TurnOrder:   turnOrderJSON,
		Combatants:  combatantsJSON,
		CampaignID:  uuid.NullUUID{Valid: false},
	})
	if err != nil {
		return CombatSession{}, err
	}

	session.ID = dbCombatSession.ID

	return session, nil
}

func GetActiveCombatSession(ctx context.Context, db *database.Queries, sessionID uuid.UUID) (CombatSession, error) {
	dbCombatSession, err := db.GetActiveCombatBySession(ctx, sessionID)
	if err != nil {
		return CombatSession{}, err
	}

	var session CombatSession
	err = json.Unmarshal(dbCombatSession.TurnOrder, &session.TurnOrder)
	if err != nil {
		return CombatSession{}, err
	}

	err = json.Unmarshal(dbCombatSession.Combatants, &session.Combatants)
	if err != nil {
		return CombatSession{}, err
	}

	session.ID = dbCombatSession.ID
	session.CurrentTurn = dbCombatSession.CurrentTurn
	session.Status = string(dbCombatSession.Status)
	session.Round = int(dbCombatSession.Round)
	session.CampaignID = dbCombatSession.CampaignID.UUID

	return session, nil
}

func AdvanceTurn(ctx context.Context, sessionID uuid.UUID, db *database.Queries) (CombatSession, error) {
	dbCombatSession, err := db.GetActiveCombatBySession(ctx, sessionID)
	if err != nil {
		return CombatSession{}, err
	}

	var session CombatSession
	err = json.Unmarshal(dbCombatSession.TurnOrder, &session.TurnOrder)
	if err != nil {
		return CombatSession{}, err
	}
	err = json.Unmarshal(dbCombatSession.Combatants, &session.Combatants)
	if err != nil {
		return CombatSession{}, err
	}

	session.ID = dbCombatSession.ID
	session.CurrentTurn = dbCombatSession.CurrentTurn
	session.Status = string(dbCombatSession.Status)
	session.Round = int(dbCombatSession.Round)
	session.CampaignID = dbCombatSession.CampaignID.UUID

	turnIndex := 0
	for i, id := range session.TurnOrder {
		if id == session.CurrentTurn {
			turnIndex = i + 1
			if turnIndex >= len(session.TurnOrder) {
				turnIndex = 0
				session.Round++
			}
			break
		}
	}
	// Skip dead combatants
	for {
		idx := findCombatantIndex(session.Combatants, session.TurnOrder[turnIndex])
		if idx == -1 || session.Combatants[idx].IsAlive {
			break
		}
		turnIndex++
		if turnIndex >= len(session.TurnOrder) {
			turnIndex = 0
			session.Round++
		}
	}
	session.CurrentTurn = session.TurnOrder[turnIndex]

	StartTurn(&session)

	saveCombatState(ctx, db, &session)

	return session, nil
}

func ProcessAction(ctx context.Context, db *database.Queries, sessionID, actorID uuid.UUID, actionType string, target uuid.UUID) (CombatSession, error) {
	session, err := GetActiveCombatSession(ctx, db, sessionID)
	if err != nil {
		return CombatSession{}, err
	}
	actorIdx := findCombatantIndex(session.Combatants, actorID)
	if actorIdx == -1 {
		return CombatSession{}, errors.New("actor not found in combat session")
	}
	if session.CurrentTurn != actorID {
		return CombatSession{}, errors.New("not actor's turn")
	}
	if !session.Combatants[actorIdx].IsAlive {
		return CombatSession{}, errors.New("actor is not alive")
	}
	if session.Combatants[actorIdx].AP <= 0 {
		return CombatSession{}, errors.New("actor does not have enough AP")
	}
	switch actionType {
	case "attack":
		targetIdx := findCombatantIndex(session.Combatants, target)
		if targetIdx == -1 {
			return CombatSession{}, errors.New("target not found in combat session")
		}
		if !session.Combatants[targetIdx].IsAlive {
			return CombatSession{}, errors.New("target is not alive")
		}
		_, damage := DiceRoll(20, 1)
		damage += GetModifier(session.Combatants[actorIdx].Strength)
		if session.Combatants[targetIdx].IsDefending {
			damage /= 2
		}
		session.Combatants[targetIdx].HP -= damage
		if session.Combatants[targetIdx].HP <= 0 {
			session.Combatants[targetIdx].IsAlive = false
			session.Combatants[targetIdx].HP = 0
		}
		session.Combatants[actorIdx].AP -= 1
	case "defend":
		session.Combatants[actorIdx].IsDefending = true
		session.Combatants[actorIdx].AP -= 1
	case "flee":
		_, fleeRoll := DiceRoll(20, 1)
		fleeRoll += GetModifier(session.Combatants[actorIdx].Alacrity)
		if fleeRoll >= 15 {
			// TODO: replace with proper fled status, AI handles narrative
			session.Combatants[actorIdx].IsAlive = false
		}
		session.Combatants[actorIdx].AP -= 1
	default:
		return CombatSession{}, errors.New("invalid action type")
	}

	if outcome := CheckCombatEnd(&session); outcome != "" {
		err = saveCombatState(ctx, db, &session)
		if err != nil {
			return CombatSession{}, err
		}
		return EndCombat(ctx, db, session.ID, outcome)
	}

	err = saveCombatState(ctx, db, &session)
	if err != nil {
		return CombatSession{}, err
	}

	return session, nil
}

func EndCombat(ctx context.Context, db *database.Queries, sessionID uuid.UUID, outcome string) (CombatSession, error) {
	dbCombatSession, err := db.GetCombatSession(ctx, sessionID)
	if err != nil {
		return CombatSession{}, err
	}

	var session CombatSession
	err = json.Unmarshal(dbCombatSession.TurnOrder, &session.TurnOrder)
	if err != nil {
		return CombatSession{}, err
	}
	err = json.Unmarshal(dbCombatSession.Combatants, &session.Combatants)
	if err != nil {
		return CombatSession{}, err
	}
	session.ID = dbCombatSession.ID
	session.CurrentTurn = dbCombatSession.CurrentTurn
	session.Round = int(dbCombatSession.Round)
	session.CampaignID = dbCombatSession.CampaignID.UUID

	// Reset WP to max for all players and sync HP/WP to characters table
	for i, c := range session.Combatants {
		if c.Type == "player" {
			session.Combatants[i].WP = c.MaxWP
			if err := db.UpdateCharacterHP(ctx, database.UpdateCharacterHPParams{
				ID:        c.ID,
				CurrentHp: int32(c.HP),
			}); err != nil {
				log.Printf("failed to sync HP for character %s at end of combat: %v", c.ID, err)
			}
			if err := db.UpdateCharacterWP(ctx, database.UpdateCharacterWPParams{
				ID:        c.ID,
				CurrentWp: int32(c.MaxWP),
			}); err != nil {
				log.Printf("failed to sync WP for character %s at end of combat: %v", c.ID, err)
			}
			// Purge non-persisting effects at end of combat
			if err := db.PurgeNonPersistingEffects(ctx, c.ID); err != nil {
				log.Printf("failed to purge non-persisting effects for character %s at end of combat: %v", c.ID, err)
			}
		}
	}

	err = saveCombatState(ctx, db, &session)
	if err != nil {
		return CombatSession{}, err
	}

	dbEnded, err := db.EndCombatSession(ctx, session.ID)
	if err != nil {
		return CombatSession{}, err
	}

	session.Status = string(dbEnded.Status) + " - " + outcome
	return session, nil
}

func StartTurn(session *CombatSession) {
	charID := session.CurrentTurn
	for i, c := range session.Combatants {
		if c.ID == charID {
			if session.Round == 1 {
				session.Combatants[i].AP = c.MaxAP
			} else {
				session.Combatants[i].AP = min(c.AP+c.MaxAP, c.OvercapAP)
			}
			session.Combatants[i].IsDefending = false
			break
		}
	}
}

func RollInitiative(combatants []Combatant) ([]uuid.UUID, []Combatant) {
	type initiativeRoll struct {
		ID         uuid.UUID
		Initiative int
	}

	var rolls []initiativeRoll
	for i, c := range combatants {
		_, roll := DiceRoll(20, 1)
		roll += GetModifier(c.Dexterity)
		combatants[i].Initiative = roll
		rolls = append(rolls, initiativeRoll{
			ID:         c.ID,
			Initiative: roll,
		})
	}

	sort.Slice(rolls, func(i, j int) bool {
		return rolls[i].Initiative > rolls[j].Initiative
	})

	var turnOrder []uuid.UUID
	for _, r := range rolls {
		turnOrder = append(turnOrder, r.ID)
	}
	return turnOrder, combatants
}

func findCombatantIndex(combatants []Combatant, id uuid.UUID) int {
	for i, c := range combatants {
		if c.ID == id {
			return i
		}
	}
	return -1
}

func CheckCombatEnd(session *CombatSession) string {
	allEnemiesDead := true
	allPlayersDead := true
	for _, c := range session.Combatants {
		if c.Type == "enemy_npc" && c.IsAlive {
			allEnemiesDead = false
		}
		if c.Type == "player" && c.IsAlive {
			allPlayersDead = false
		}
	}
	if allEnemiesDead {
		return "victory"
	}
	if allPlayersDead {
		return "defeat"
	}
	return ""
}

// helper function for updating database with new combat state
func saveCombatState(ctx context.Context, db *database.Queries, session *CombatSession) error {
	turnOrderJSON, err := json.Marshal(session.TurnOrder)
	if err != nil {
		return err
	}
	combatantsJSON, err := json.Marshal(session.Combatants)
	if err != nil {
		return err
	}
	_, err = db.UpdateCombatState(ctx, database.UpdateCombatStateParams{
		ID:          session.ID,
		CurrentTurn: session.CurrentTurn,
		TurnOrder:   turnOrderJSON,
		Combatants:  combatantsJSON,
		Round:       int32(session.Round),
	})
	return err
}
