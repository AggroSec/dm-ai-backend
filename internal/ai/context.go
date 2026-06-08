package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AggroSec/dm-ai-backend/internal/config"
	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

func BuildCombatContext(ctx context.Context, db *database.Queries, cfg *config.Config, campaignID uuid.UUID, characterID uuid.UUID, combatID uuid.UUID, playerMessage string) ([]Message, error) {
	campaign, err := db.GetCampaign(ctx, campaignID)
	if err != nil {
		return []Message{}, err
	}

	character, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return []Message{}, err
	}

	var talents []game.TalentInvestment
	err = json.Unmarshal(character.TalentsInvested, &talents)
	if err != nil {
		return []Message{}, err
	}

	characterSkills, err := game.LoadSkillsForCharacter(cfg.DataDir, character.Class, talents)
	if err != nil {
		return []Message{}, err
	}

	combatSession, err := game.GetActiveCombatSession(ctx, db, combatID)
	if err != nil {
		return []Message{}, err
	}

	var msgs []Message
	msgs = append(msgs, Message{
		Role:    "system",
		Content: "This is a placeholder system prompt",
	})
	if campaign.NarrativeSummary != "" {
		msgs = append(msgs, Message{
			Role:    "system",
			Content: campaign.NarrativeSummary,
		})
	}
	characterContext := CreateCharacterContext(character)
	msgs = append(msgs, Message{
		Role:    "system",
		Content: characterContext,
	})
	characterSkillsContext := GetSkillsContext(characterSkills, character.ID)
	msgs = append(msgs, Message{
		Role:    "system",
		Content: characterSkillsContext,
	})
	combatState := GetCombatState(combatSession)
	msgs = append(msgs, Message{
		Role:    "system",
		Content: combatState,
	})

	unsummarizedMessages, err := db.GetMessagesAfterSequence(ctx, database.GetMessagesAfterSequenceParams{
		CampaignID: campaign.ID,
		Sequence:   campaign.SummarizedThrough,
	})
	if err != nil {
		return []Message{}, err
	}
	for _, message := range unsummarizedMessages {
		var toolCalls []ToolCall
		if message.ToolCalls != nil {
			err = json.Unmarshal(message.ToolCalls, &toolCalls)
			if err != nil {
				return []Message{}, err
			}
		}
		appendMsg := Message{
			Role:       message.Role,
			Content:    message.Content,
			ToolCalls:  toolCalls,
			ToolCallID: message.ToolCallID,
		}
		msgs = append(msgs, appendMsg)
	}

	msgs = append(msgs, Message{
		Role:    "user",
		Content: playerMessage,
	})

	return msgs, nil
}

func BuildNarrativeContext(ctx context.Context, db *database.Queries, cfg *config.Config, campaignID uuid.UUID, characterID uuid.UUID, playerMessage string) ([]Message, error) {
	campaign, err := db.GetCampaign(ctx, campaignID)
	if err != nil {
		return []Message{}, err
	}

	character, err := db.GetCharacterByID(ctx, characterID)
	if err != nil {
		return []Message{}, err
	}

	var msgs []Message
	msgs = append(msgs, Message{
		Role:    "system",
		Content: "placeholder narrative system prompt for now",
	})
	if campaign.NarrativeSummary != "" {
		msgs = append(msgs, Message{
			Role:    "system",
			Content: campaign.NarrativeSummary,
		})
	}
	characterContext := CreateCharacterContext(character)
	msgs = append(msgs, Message{
		Role:    "system",
		Content: characterContext,
	})
	unsummarizedMessages, err := db.GetMessagesAfterSequence(ctx, database.GetMessagesAfterSequenceParams{
		CampaignID: campaign.ID,
		Sequence:   campaign.SummarizedThrough,
	})
	if err != nil {
		return []Message{}, err
	}
	for _, message := range unsummarizedMessages {
		var toolCalls []ToolCall
		if message.ToolCalls != nil {
			err = json.Unmarshal(message.ToolCalls, &toolCalls)
			if err != nil {
				return []Message{}, err
			}
		}
		appendMsg := Message{
			Role:       message.Role,
			Content:    message.Content,
			ToolCalls:  toolCalls,
			ToolCallID: message.ToolCallID,
		}
		msgs = append(msgs, appendMsg)
	}
	msgs = append(msgs, Message{
		Role:    "user",
		Content: playerMessage,
	})

	return msgs, nil
}

func CreateCharacterContext(character database.Character) string {
	return fmt.Sprintf(
		"CHARACTER SHEET\n"+
			"ID: %s\n"+
			"Name: %s | Race: %s | Class: %s | Level: %d\n"+
			"Fates — Driving: %s | Binding: %s\n"+
			"Stats — STR: %d | DEX: %d | FOR: %d | WIL: %d | ALC: %d | WIS: %d\n"+
			"HP: %d/%d | WP: %d/%d | AP: %d (max: %d, overcap: %d)\n"+
			"XP: %d | Talent Points Available: %d",
		character.ID,
		character.Name, character.Race, character.Class, character.Level,
		character.DrivingFate, character.BindingFate,
		character.Strength, character.Dexterity, character.Fortitude,
		character.Willpower, character.Alacrity, character.Wisdom,
		character.CurrentHp, character.MaxHp,
		character.CurrentWp, character.MaxWp,
		character.ActionPoints, character.MaxAp, character.OvercapAp,
		character.Experience, character.TalentPointsAvailable,
	)
}

func GetCombatState(session game.CombatSession) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "COMBAT STATE\nRound: %d | Status: %s | Current Turn: %s\nTurn Order: ",
		session.Round, session.Status, session.CurrentTurn)

	for i, id := range session.TurnOrder {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(id.String())
	}
	sb.WriteString("\n\nCOMBATANTS\n")

	for _, c := range session.Combatants {
		alive := "alive"
		if !c.IsAlive {
			alive = "dead"
		}
		fmt.Fprintf(&sb,
			"[%s] %s (ID: %s) — %s\n"+
				"  HP: %d/%d | WP: %d/%d | AP: %d/%d (overcap: %d)\n"+
				"  STR: %d | DEX: %d | FOR: %d | WIL: %d | ALC: %d | WIS: %d\n",
			c.Type, c.Name, c.ID, alive,
			c.HP, c.MaxHP, c.WP, c.MaxWP, c.AP, c.MaxAP, c.OvercapAP,
			c.Strength, c.Dexterity, c.Fortitude, c.Willpower, c.Alacrity, c.Wisdom,
		)
		if len(c.StatusEffects) > 0 {
			sb.WriteString("  Status Effects:\n")
			for _, se := range c.StatusEffects {
				fmt.Fprintf(&sb, "    - %s (duration: %d, instruction: %s)\n",
					se.Effect, se.Duration, se.Instruction)
			}
		}
	}

	return sb.String()
}

func GetSkillsContext(skills []game.Skill, characterID uuid.UUID) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Skills for CharacterID: %v\n", characterID)
	for _, skill := range skills {
		fmt.Fprintf(&sb, "Name: %v, Instructions: %v, HP cost: %v, WP cost: %v, AP cost: %v, Description: %v\n", skill.Name, skill.AIInstruction, skill.HPCost, skill.WPCost, skill.APCost, skill.Description)
	}

	return sb.String()
}
