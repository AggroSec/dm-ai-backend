package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
		Content: CombatSystemPrompt(campaign.Theme),
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
		if message.Role != "tool" && message.ToolCalls != nil {
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
		Content: NarrativeSystemPrompt(campaign.Theme),
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
		if message.Role != "tool" && message.ToolCalls != nil {
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

func BuildCharacterCreationContext(cfg *config.Config, ctx context.Context, db *database.Queries, character database.Character, campaignID uuid.UUID, playerMessage string) ([]Message, error) {
	classReference, err := BuildClassReferenceContext(cfg.DataDir)
	if err != nil {
		return []Message{}, err
	}

	classDomain := getClassDomain(character.Class)
	fatesReference, err := BuildFatesReferenceContext(cfg.DataDir, classDomain)
	if err != nil {
		return []Message{}, err
	}

	unsummarizedMessages, err := db.GetMessagesAfterSequence(ctx, database.GetMessagesAfterSequenceParams{
		CampaignID: campaignID,
		Sequence:   0,
	})
	if err != nil {
		return []Message{}, err
	}

	var msgs []Message
	msgs = append(msgs, Message{
		Role:    "system",
		Content: CharacterCreationSystemPrompt(),
	})
	msgs = append(msgs, Message{
		Role:    "system",
		Content: CreateCharacterContext(character),
	})
	msgs = append(msgs, Message{
		Role:    "system",
		Content: classReference,
	})
	msgs = append(msgs, Message{
		Role:    "system",
		Content: fatesReference,
	})

	for _, message := range unsummarizedMessages {
		var toolCalls []ToolCall
		if message.Role != "tool" && message.ToolCalls != nil {
			err = json.Unmarshal(message.ToolCalls, &toolCalls)
			if err != nil {
				return []Message{}, err
			}
		}
		msgs = append(msgs, Message{
			Role:       message.Role,
			Content:    message.Content,
			ToolCalls:  toolCalls,
			ToolCallID: message.ToolCallID,
		})
	}

	msgs = append(msgs, Message{
		Role:    "user",
		Content: playerMessage,
	})

	log.Printf(" | [DEBUG] class: %s domain: %s fates context: %s", character.Class, classDomain, fatesReference[:100])
	return msgs, nil
}

func BuildClassReferenceContext(dataDir string) (string, error) {
	classes := []string{"warrior", "runeblade", "seer"}
	var sb strings.Builder
	sb.WriteString("CLASS REFERENCE\n")

	for _, class := range classes {
		data, err := game.LoadClassBranch(dataDir, class, "base")
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "\n[%s] %s\n", strings.ToUpper(class), data.Description)
		sb.WriteString("Level 1 Skills:\n")
		for _, skill := range data.Skills {
			if skill.LevelRequired == 1 {
				fmt.Fprintf(&sb, "  - %s (AP: %d, WP: %d, HP: %d) — %s\n",
					skill.Name, skill.APCost, skill.WPCost, skill.HPCost, skill.Description)
			}
		}
	}

	return sb.String(), nil
}

func BuildFatesReferenceContext(dataDir, domain string) (string, error) {
	fates, err := game.LoadFatesForDomain(dataDir, domain)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "AVAILABLE FATES (%s domain)\n", strings.ToUpper(domain))
	sb.WriteString("Choose one Driving Fate and one Binding Fate. Both must be from this domain.\n\n")

	for _, fate := range fates {
		fmt.Fprintf(&sb, "[%s] %s\n  %s\n  Effect: %s\n\n",
			strings.ToUpper(fate.Type), fate.Name, fate.Description, fate.Effect)
	}

	return sb.String(), nil
}

func getClassDomain(class string) string {
	switch strings.ToLower(class) {
	case "warrior":
		return "physical"
	case "seer":
		return "magical"
	case "runeblade":
		return "hybrid"
	default:
		return "physical"
	}
}

func CreateCharacterContext(character database.Character) string {
	str, dex, fort, wil, alc, wis := game.GetEffectiveStats(character)
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
		str, dex, fort, wil, alc, wis,
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
