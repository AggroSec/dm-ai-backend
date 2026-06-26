package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/AggroSec/dm-ai-backend/internal/config"
	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

type Dispatcher struct {
	db         *database.Queries
	cfg        *config.Config
	campaignID uuid.UUID
}

func NewDispatcher(db *database.Queries, cfg *config.Config, campaignID uuid.UUID) *Dispatcher {
	return &Dispatcher{db: db, cfg: cfg, campaignID: campaignID}
}

type toolHandlerFunc func(ctx context.Context, args json.RawMessage) (string, error)

func (d *Dispatcher) ExecuteToolCall(ctx context.Context, toolCall ToolCall) (string, error) {
	handlers := map[string]toolHandlerFunc{
		"request_roll":         d.handleRequestRoll,
		"start_combat":         d.handleStartCombat,
		"get_combat_state":     d.handleGetCombatState,
		"end_combat":           d.handleEndCombat,
		"skip_turn":            d.handleSkipTurn,
		"validate_action":      d.handleValidateAction,
		"apply_damage":         d.handleApplyDamage,
		"apply_heal":           d.handleApplyHeal,
		"apply_status_effect":  d.handleApplyStatusEffect,
		"remove_status_effect": d.handleRemoveStatusEffect,
		"action_failed":        d.handleActionFailed,
		"award_xp":             d.handleAwardXP,
		"get_character":        d.handleGetCharacter,
		"get_skills":           d.handleGetSkills,
		"update_character":     d.handleUpdateCharacter,
		"give_item":            d.handleGiveItem,
		"equip_item":           d.handleEquipItem,
		"rest":                 d.handleRest,
		"end_turn":             d.handleEndTurn,
		"narrate_combat":       d.handleNarrateCombat,
	}

	handler, ok := handlers[toolCall.Function.Name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}

	return handler(ctx, json.RawMessage(toolCall.Function.Arguments))
}

func (d *Dispatcher) handleRequestRoll(ctx context.Context, args json.RawMessage) (string, error) {
	type rollParams struct {
		Sides int `json:"sides"`
		Count int `json:"count"`
	}
	var params rollParams
	err := json.Unmarshal(args, &params)
	if err != nil {
		return "", err
	}

	rolls, total := game.DiceRoll(params.Sides, params.Count)
	log.Printf(" | [DiceRoll] sides:%d count:%d rolls:%v total:%d", params.Sides, params.Count, rolls, total)
	return fmt.Sprintf("Rolled %vd%v: %v, Total: %v", params.Count, params.Sides, rolls, total), nil
}

func (d *Dispatcher) handleStartCombat(ctx context.Context, args json.RawMessage) (string, error) {
	var npcData struct {
		NPCs []game.Combatant `json:"npcs"`
	}
	err := json.Unmarshal(args, &npcData)
	if err != nil {
		return "", err
	}
	npcs := npcData.NPCs

	campaign, err := d.db.GetCampaign(ctx, d.campaignID)
	if err != nil {
		return "", err
	}
	var party []game.PartyMember
	err = json.Unmarshal(campaign.Party, &party)
	if err != nil {
		return "", err
	}

	combatSession, err := game.CreateCombatSession(ctx, d.db, party, npcs, &campaign.ID)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Combat session created: %v | NPCs: %d | Party: %d", combatSession.ID, len(npcs), len(party)))

	jsonCombatSession, err := json.Marshal(combatSession)
	if err != nil {
		return "", err
	}

	return string(jsonCombatSession), nil
}

func (d *Dispatcher) handleGetCombatState(ctx context.Context, args json.RawMessage) (string, error) {
	type getCombatStateArgs struct {
		CombatID uuid.UUID `json:"combat_id"`
	}
	var toolArgs getCombatStateArgs
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	combatSession, err := game.GetActiveCombatSession(ctx, d.db, toolArgs.CombatID)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Combat state retrieved: %v | Round: %d | CurrentTurn: %v", toolArgs.CombatID, combatSession.Round, combatSession.CurrentTurn))

	jsonCombatSession, err := json.Marshal(combatSession)
	if err != nil {
		return "", err
	}

	return string(jsonCombatSession), nil
}

func (d *Dispatcher) handleEndCombat(ctx context.Context, args json.RawMessage) (string, error) {
	type EndCombatParams struct {
		CombatID uuid.UUID `json:"combat_id"`
		Outcome  string    `json:"outcome"`
	}
	var toolArgs EndCombatParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	combatSession, err := game.EndCombat(ctx, d.db, toolArgs.CombatID, toolArgs.Outcome)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Combat ended: %v | Outcome: %s", toolArgs.CombatID, toolArgs.Outcome))

	jsonCombatSession, err := json.Marshal(combatSession)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Combat Ended Successfully: %s", string(jsonCombatSession)), nil
}

func (d *Dispatcher) handleSkipTurn(ctx context.Context, args json.RawMessage) (string, error) {
	type skipTurnParams struct {
		CombatID uuid.UUID `json:"combat_id"`
		Entity   uuid.UUID `json:"entity"`
		Reason   string    `json:"reason,omitempty"`
	}
	var toolArgs skipTurnParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	combatSession, err := game.GetActiveCombatSession(ctx, d.db, toolArgs.CombatID)
	if err != nil {
		return "", err
	}

	if toolArgs.Entity != combatSession.CurrentTurn {
		logAIDispatcher(fmt.Sprintf("skip_turn mismatch: supplied %v, current turn %v", toolArgs.Entity, combatSession.CurrentTurn))
		return fmt.Sprintf("skip_turn failed: entity %v is not the current turn holder, current turn is %v", toolArgs.Entity, combatSession.CurrentTurn), nil
	}

	combatSession, err = game.AdvanceTurn(ctx, toolArgs.CombatID, d.db)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Turn skipped: %v | Reason: %s | NextTurn: %v", toolArgs.Entity, toolArgs.Reason, combatSession.CurrentTurn))

	jsonCombatSession, err := json.Marshal(combatSession)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Combat Skip Turn Successful: %s", string(jsonCombatSession)), nil
}

func (d *Dispatcher) handleValidateAction(ctx context.Context, args json.RawMessage) (string, error) {
	type validateActionParams struct {
		Action      string    `json:"action"`
		HpCost      int       `json:"hp_cost"`
		WpCost      int       `json:"wp_cost"`
		ApCost      int       `json:"ap_cost"`
		CombatantID uuid.UUID `json:"combatant_id"`
		CombatID    uuid.UUID `json:"combat_id"`
	}
	var toolArgs validateActionParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	valid, err := game.ValidateAction(ctx, d.db, toolArgs.Action, toolArgs.HpCost, toolArgs.WpCost, toolArgs.ApCost, toolArgs.CombatantID, toolArgs.CombatID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Action: %s validity is: %v", toolArgs.Action, valid), nil
}

func (d *Dispatcher) handleApplyDamage(ctx context.Context, args json.RawMessage) (string, error) {
	type applyDamageArgs struct {
		Entity     uuid.UUID  `json:"entity"`
		Amount     int        `json:"amount"`
		DamageType string     `json:"type"`
		CombatID   *uuid.UUID `json:"combat_id,omitempty"`
		Source     string     `json:"source,omitempty"`
	}
	var toolArgs applyDamageArgs
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	err = game.ApplyDamage(ctx, d.db, toolArgs.Entity, toolArgs.Amount, toolArgs.DamageType, toolArgs.Source, toolArgs.CombatID)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("damage applied successfully to combatant: %v (-%d %s)", toolArgs.Entity, toolArgs.Amount, toolArgs.DamageType))
	return fmt.Sprintf("damage applied successfully to combatant: %v (-%d %s)", toolArgs.Entity, toolArgs.Amount, toolArgs.DamageType), nil
}

func (d *Dispatcher) handleApplyHeal(ctx context.Context, args json.RawMessage) (string, error) {
	type applyHealArgs struct {
		Entity   uuid.UUID  `json:"entity"`
		Amount   int        `json:"amount"`
		HealType string     `json:"type"`
		CombatID *uuid.UUID `json:"combat_id,omitempty"`
		Source   string     `json:"source,omitempty"`
	}
	var toolArgs applyHealArgs
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	err = game.ApplyHeal(ctx, d.db, toolArgs.Entity, toolArgs.Amount, toolArgs.HealType, toolArgs.Source, toolArgs.CombatID)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("heal applied successfully to combatant: %v (+%d %s)", toolArgs.Entity, toolArgs.Amount, toolArgs.HealType))
	return fmt.Sprintf("heal applied successfully to combatant: %v (+%d %s)", toolArgs.Entity, toolArgs.Amount, toolArgs.HealType), nil
}

func (d *Dispatcher) handleApplyStatusEffect(ctx context.Context, args json.RawMessage) (string, error) {
	type statusEffectParams struct {
		CombatantID uuid.UUID  `json:"combatant_id"`
		CombatID    *uuid.UUID `json:"combat_id"`
		Effect      string     `json:"effect"`
		Duration    int        `json:"duration"`
		Persists    bool       `json:"persists"`
		Instruction string     `json:"instruction"`
	}
	var toolArgs statusEffectParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	var combatSession *game.CombatSession
	if toolArgs.CombatID != nil {
		session, err := game.GetActiveCombatSession(ctx, d.db, *toolArgs.CombatID)
		if err != nil {
			return "", err
		}
		combatSession = &session
	}

	statusEffect, err := game.ApplyStatusEffect(ctx, d.db, toolArgs.CombatantID, toolArgs.Effect, toolArgs.Duration, toolArgs.Persists, toolArgs.Instruction, combatSession)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Status effect applied: %s | Target: %v | Duration: %d | Persists: %v", toolArgs.Effect, toolArgs.CombatantID, toolArgs.Duration, toolArgs.Persists))

	jsonEffect, err := json.Marshal(statusEffect)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("effect applied successfully: %s", string(jsonEffect)), nil
}

func (d *Dispatcher) handleRemoveStatusEffect(ctx context.Context, args json.RawMessage) (string, error) {
	type removeStatusParams struct {
		CharacterID uuid.UUID  `json:"character_id"`
		StatusID    uuid.UUID  `json:"status_id"`
		CombatID    *uuid.UUID `json:"combat_id"`
	}
	var toolArgs removeStatusParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	var combatSession *game.CombatSession
	if toolArgs.CombatID != nil {
		session, err := game.GetActiveCombatSession(ctx, d.db, *toolArgs.CombatID)
		if err != nil {
			return "", err
		}
		combatSession = &session
	}

	err = game.RemoveStatusEffect(ctx, d.db, toolArgs.CharacterID, toolArgs.StatusID, combatSession)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Status effect removed: %v | Target: %v", toolArgs.StatusID, toolArgs.CharacterID))

	return fmt.Sprintf("Status effect(%v) removed successfully.", toolArgs.StatusID), nil
}

func (d *Dispatcher) handleActionFailed(ctx context.Context, args json.RawMessage) (string, error) {
	type failedActionParams struct {
		CharacterID uuid.UUID `json:"character_id"`
		Reason      string    `json:"reason"`
	}
	var toolArgs failedActionParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Character[%v] action failed: %s", toolArgs.CharacterID, toolArgs.Reason))
	return fmt.Sprintf("Instructions: respond to the player. Reason action failed: %s", toolArgs.Reason), nil
}

func (d *Dispatcher) handleAwardXP(ctx context.Context, args json.RawMessage) (string, error) {
	type awardXPParams struct {
		CharacterID uuid.UUID `json:"character_id"`
		Amount      int       `json:"amount"`
	}
	var toolArgs awardXPParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	results, err := game.AwardXP(ctx, d.db, toolArgs.CharacterID, toolArgs.Amount)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("XP awarded: %d | Character: %v | LeveledUp: %v", toolArgs.Amount, toolArgs.CharacterID, results.LeveledUp))

	jsonData, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("XP awarded successfully: %v", string(jsonData)), nil
}

func (d *Dispatcher) handleGetCharacter(ctx context.Context, args json.RawMessage) (string, error) {
	type getCharacterParams struct {
		CharacterID uuid.UUID `json:"character_id"`
	}
	var toolArgs getCharacterParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	characterInfo, err := game.GetCharacterInfo(ctx, d.db, toolArgs.CharacterID)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(characterInfo)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("character %s(%v) info retrieved successfully", characterInfo.Name, characterInfo.ID))
	return string(jsonData), nil
}

func (d *Dispatcher) handleGetSkills(ctx context.Context, args json.RawMessage) (string, error) {
	type getSkillsParams struct {
		CharacterID  uuid.UUID `json:"character_id"`
		TalentPoints int       `json:"talent_points"` //will be zero unless leveling up
	}
	var toolArgs getSkillsParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	characterInfo, err := game.GetCharacterInfo(ctx, d.db, toolArgs.CharacterID)
	if err != nil {
		return "", err
	}

	skills, err := game.LoadAllSkillsForCharacter(d.cfg.DataDir, characterInfo.Class, characterInfo.TalentsInvested, toolArgs.TalentPoints)
	if err != nil {
		logAIDispatcher(fmt.Sprintf("ai could not retrieve skills, error: %v", err))
		return "", err
	}

	jsonData, err := json.Marshal(skills)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Character %s(%v) skills retrieves successfully", characterInfo.Name, characterInfo.ID))
	return string(jsonData), nil
}

func (d *Dispatcher) handleUpdateCharacter(ctx context.Context, args json.RawMessage) (string, error) {
	type updateCharacterArgs struct {
		CharacterID uuid.UUID `json:"character_id"`
		game.UpdateCharacterRequest
	}
	var toolArgs updateCharacterArgs
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	updatedCharacter, err := game.ApplyCharacterUpdates(ctx, d.db, toolArgs.CharacterID, toolArgs.UpdateCharacterRequest)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(updatedCharacter)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Character %s(%v) was updated successfully", updatedCharacter.Name, updatedCharacter.ID))
	return fmt.Sprintf("character updated successfully: %s", string(jsonData)), nil
}

func (d *Dispatcher) handleGiveItem(ctx context.Context, args json.RawMessage) (string, error) {
	type giveItemParams struct {
		CharacterID uuid.UUID `json:"character_id"`
		Item        game.Item `json:"item"`
	}
	var toolArgs giveItemParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	characterInfo, err := game.GiveItem(ctx, d.db, toolArgs.CharacterID, toolArgs.Item)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(characterInfo)
	if err != nil {
		return "", err
	}
	logAIDispatcher(fmt.Sprintf("character %s(%v) has been given item: %v", characterInfo.Name, characterInfo.ID, toolArgs.Item))
	return fmt.Sprintf("item successfully given to character: %s", string(jsonData)), nil
}

func (d *Dispatcher) handleEquipItem(ctx context.Context, args json.RawMessage) (string, error) {
	type equipItemParams struct {
		CharacterID uuid.UUID  `json:"character_id"`
		ItemID      string     `json:"item_id"`
		Slot        string     `json:"slot"`
		CombatID    *uuid.UUID `json:"combat_id"`
	}
	var toolArgs equipItemParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	_, err = game.UnequipItem(ctx, d.db, toolArgs.CharacterID, toolArgs.Slot)
	if err != nil {
		return "", err
	}
	updatedCharacterInfo, err := game.EquipItem(ctx, d.db, toolArgs.CharacterID, toolArgs.Slot, toolArgs.ItemID, toolArgs.CombatID)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(updatedCharacterInfo)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Character %s(%v) successfully equipped item %s", updatedCharacterInfo.Name, updatedCharacterInfo.ID, toolArgs.ItemID))
	return fmt.Sprintf("Item equipped successfully: %s", string(jsonData)), nil
}

func (d *Dispatcher) handleRest(ctx context.Context, args json.RawMessage) (string, error) {
	type restParams struct {
		CharacterID uuid.UUID `json:"character_id"`
		RestType    string    `json:"rest_type"`
		Location    string    `json:"location"`
	}
	var toolArgs restParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	err = game.CharacterRest(ctx, d.db, toolArgs.CharacterID, toolArgs.RestType)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("Character %v has successfully had a %s rest at %s.", toolArgs.CharacterID, toolArgs.RestType, toolArgs.Location))
	return fmt.Sprintf("Character %v has successfully had a %s rest at %s.", toolArgs.CharacterID, toolArgs.RestType, toolArgs.Location), nil
}

func (d *Dispatcher) handleEndTurn(ctx context.Context, args json.RawMessage) (string, error) {
	type endTurnParams struct {
		CombatID    uuid.UUID `json:"combat_id"`
		CombatantID uuid.UUID `json:"combatant_id"`
	}
	var toolArgs endTurnParams
	err := json.Unmarshal(args, &toolArgs)
	if err != nil {
		return "", err
	}

	combatSession, err := game.AdvanceTurn(ctx, toolArgs.CombatID, d.db)
	if err != nil {
		return "", err
	}

	logAIDispatcher(fmt.Sprintf("turn advancement successfully completed: %v", combatSession))
	return fmt.Sprintf("Turn advancement successful: %v", combatSession), nil
}

func (d *Dispatcher) handleNarrateCombat(ctx context.Context, args json.RawMessage) (string, error) {
	var toolArgs struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(args, &toolArgs); err != nil {
		return "", fmt.Errorf("failed to unmarshal narrate args: %w", err)
	}
	return toolArgs.Message, nil
}

func logAIDispatcher(msg string) {
	log.Printf(" | [AIToolDispatcher] %s", msg)
}
