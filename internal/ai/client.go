package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/config"
	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/game"
	"github.com/google/uuid"
)

const (
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	maxIterations    = 20
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Function struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	Tools     []Tool    `json:"tools,omitempty"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Client struct {
	HTTPClient *http.Client
	APIKey     string
	Model      string
	MaxTokens  int
	URL        string
	db         *database.Queries
}

type StreamCallback func(text string)

func NewClient(cfg config.Config, db *database.Queries) *Client {
	return &Client{
		HTTPClient: &http.Client{},
		APIKey:     cfg.OpenRouterAPIKey,
		Model:      cfg.OpenRouterModel,
		MaxTokens:  cfg.OpenRouterMaxTokens,
		URL:        openRouterAPIURL,
		db:         db,
	}
}

func (c *Client) Chat(ctx context.Context, msgs []Message) (string, error) {
	chatRequest := ChatRequest{
		Model:     c.Model,
		Messages:  msgs,
		MaxTokens: c.MaxTokens,
	}

	jsonData, err := json.Marshal(chatRequest)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("HTTP-Referer", "http://localhost:8080")
	req.Header.Set("X-Title", "DM-AI Backend")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openrouter returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResponse ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return "", err
	}

	if len(chatResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from openrouter")
	}

	return chatResponse.Choices[0].Message.Content, nil
}

func (c *Client) ChatWithTools(ctx context.Context, msgs []Message, tools []Tool, dispatcher *Dispatcher, streamFn StreamCallback) (string, *uuid.UUID, bool, error) {
	messages := msgs
	var combatID *uuid.UUID
	combatEnded := false

	for i := 0; i < maxIterations; i++ {
		chatRequest := ChatRequest{
			Model:     c.Model,
			Messages:  messages,
			MaxTokens: c.MaxTokens,
			Tools:     tools,
		}

		jsonData, err := json.Marshal(chatRequest)
		if err != nil {
			return "", nil, combatEnded, err
		}
		if dispatcher.cfg.DebugLogging {
			log.Printf(" | [DEBUG] full request: %s", string(jsonData))
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", nil, combatEnded, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("HTTP-Referer", "http://localhost:8080")
		req.Header.Set("X-Title", "DM-AI Backend")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return "", nil, combatEnded, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return "", nil, combatEnded, fmt.Errorf("openrouter returned status %d: %s", resp.StatusCode, string(body))
		}

		var rawResponse json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
			return "", nil, combatEnded, err
		}
		if dispatcher.cfg.DebugLogging {
			log.Printf(" | [DEBUG] full response: %s", string(rawResponse))
		}

		var chatResponse ChatResponse
		if err := json.Unmarshal(rawResponse, &chatResponse); err != nil {
			return "", nil, combatEnded, err
		}
		if len(chatResponse.Choices) == 0 {
			return "", nil, combatEnded, fmt.Errorf("openrouter returned empty choices — context may be full")
		}

		log.Printf(" | [DEBUG] finish_reason: %s", chatResponse.Choices[0].FinishReason)
		log.Printf(" | [DEBUG] tool_calls: %v", chatResponse.Choices[0].Message.ToolCalls)

		if chatResponse.Choices[0].FinishReason == "stop" {
			msg := chatResponse.Choices[0].Message.Content
			nextSequence, err := dispatcher.db.GetNextSequence(ctx, dispatcher.campaignID)
			if err != nil {
				return "", nil, combatEnded, fmt.Errorf("failed to get message sequence: %v", err)
			}
			dispatcher.db.InsertMessage(ctx, database.InsertMessageParams{
				CampaignID: dispatcher.campaignID,
				Role:       "assistant",
				Content:    msg,
				ToolCalls:  []byte("[]"),
				Sequence:   nextSequence,
			})
			if streamFn != nil {
				streamFn(msg)
			}
			return msg, combatID, combatEnded, nil
		} else if chatResponse.Choices[0].FinishReason == "tool_calls" {
			assistantMsg := chatResponse.Choices[0].Message
			jsonToolCalls, err := json.Marshal(assistantMsg.ToolCalls)
			if err != nil {
				return "", nil, combatEnded, err
			}
			nextSequence, err := dispatcher.db.GetNextSequence(ctx, dispatcher.campaignID)
			if err != nil {
				return "", nil, combatEnded, err
			}
			dispatcher.db.InsertMessage(ctx, database.InsertMessageParams{
				CampaignID: dispatcher.campaignID,
				Role:       "assistant",
				Content:    assistantMsg.Content,
				ToolCalls:  jsonToolCalls,
				Sequence:   nextSequence,
			})
			messages = append(messages, chatResponse.Choices[0].Message)
			for _, toolCall := range chatResponse.Choices[0].Message.ToolCalls {
				toolLog := fmt.Sprintf("AI called tool %v with %v parameters", toolCall.Function.Name, toolCall.Function.Arguments)
				logInternalAI(toolLog)
				result, err := dispatcher.ExecuteToolCall(ctx, toolCall)
				if err != nil {
					result = fmt.Sprintf("tool execution failed Name: %v, Error: %v", toolCall.Function.Name, err.Error())
				}
				resultMsg := Message{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Content:    result,
				}
				jsonToolCall, err := json.Marshal([]ToolCall{toolCall})
				if err != nil {
					result = fmt.Sprintf("failed to marshal tool call to json: %v", err)
				}
				nextSequence, err := dispatcher.db.GetNextSequence(ctx, dispatcher.campaignID)
				if err != nil {
					result = fmt.Sprintf("failed to get message sequence: %v", err)
				}
				_, err = dispatcher.db.InsertMessage(ctx, database.InsertMessageParams{
					CampaignID: dispatcher.campaignID,
					Role:       "tool",
					Content:    resultMsg.Content,
					ToolCalls:  jsonToolCall,
					ToolCallID: resultMsg.ToolCallID,
					Sequence:   nextSequence,
				})
				if err != nil {
					result = fmt.Sprintf("failed to add tool call message to db: %v", err)
				}
				if toolCall.Function.Name == "start_combat" {
					var combatSession game.CombatSession
					if err := json.Unmarshal([]byte(result), &combatSession); err == nil {
						combatID = &combatSession.ID
					} else {
						result = fmt.Sprintf("failed to unmarshal combat session: %v", err)
					}
				}
				if toolCall.Function.Name == "narrate_combat" && streamFn != nil {
					streamFn(result)
				}
				messages = append(messages, resultMsg)
				logInternalAI("tool call finished")
				if toolCall.Function.Name == "end_combat" {
					combatEnded = true
				}
			}
			continue
		}
	}
	return "", nil, combatEnded, fmt.Errorf("too many AI iterations.")
}

func logInternalAI(msg string) {
	log.Printf(" | [InternalAILog] %v", msg)
}
