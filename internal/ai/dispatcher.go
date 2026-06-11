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

	//validate combatants match
	if toolArgs.Entity != combatSession.CurrentTurn {
		return "", fmt.Errorf("supplied entity ID(%v) does not match entity ID(%v) for current turn control.", toolArgs.Entity, combatSession.CurrentTurn)
	}

	combatSession, err = game.AdvanceTurn(ctx, toolArgs.CombatID, d.db)
	if err != nil {
		return "", err
	}

	jsonCombatSession, err := json.Marshal(combatSession)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Combat Skip Turn Successful: %s", string(jsonCombatSession)), nil
}

func (d *Dispatcher) handleValidateAction(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: validate_action")
}

func (d *Dispatcher) handleApplyDamage(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: apply_damage")
}

func (d *Dispatcher) handleApplyHeal(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: apply_heal")
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

	jsonEffect, err := json.Marshal(statusEffect)
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

	return fmt.Sprintf("Status effect(%v) removed successfully.", toolArgs.StatusID), nil
}

func (d *Dispatcher) handleActionFailed(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: action_failed")
}

func (d *Dispatcher) handleAwardXP(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: award_xp")
}

func (d *Dispatcher) handleGetCharacter(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: get_character")
}

func (d *Dispatcher) handleGetSkills(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: get_skills")
}

func (d *Dispatcher) handleUpdateCharacter(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: update_character")
}

func (d *Dispatcher) handleGiveItem(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: give_item")
}

func (d *Dispatcher) handleEquipItem(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: equip_item")
}

func (d *Dispatcher) handleRest(ctx context.Context, args json.RawMessage) (string, error) {
	return "", fmt.Errorf("not implemented: rest")
}
