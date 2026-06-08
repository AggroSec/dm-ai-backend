package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/game"
)

func ExecuteToolCall(ctx context.Context, db *database.Queries, toolCall ToolCall) (string, error) {
	switch toolCall.Function.Name {
	case "request_roll":
		type rollParams struct {
			Sides int `json:"sides"`
			Count int `json:"count"`
		}
		var params rollParams
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
		if err != nil {
			return "", err
		}

		rolls, total := game.DiceRoll(params.Sides, params.Count)
		log.Printf(" | [DiceRoll] sides:%d count:%d rolls:%v total:%d", params.Sides, params.Count, rolls, total)
		return fmt.Sprintf("Rolled %vd%v: %v, Total: %v", params.Count, params.Sides, rolls, total), nil
	default:
		return "", fmt.Errorf("Unknown tool name: %v", toolCall.Function.Name)
	}
}
