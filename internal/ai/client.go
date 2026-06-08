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
)

const (
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	maxIterations    = 10
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

func (c *Client) ChatWithTools(ctx context.Context, msgs []Message, tools []Tool) (string, error) {
	messages := msgs

	for i := 0; i < maxIterations; i++ {
		chatRequest := ChatRequest{
			Model:     c.Model,
			Messages:  messages,
			MaxTokens: c.MaxTokens,
			Tools:     tools,
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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("openrouter returned status %d: %s", resp.StatusCode, string(body))
		}

		var chatResponse ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
			return "", err
		}

		resp.Body.Close()

		log.Printf(" | [DEBUG] finish_reason: %s", chatResponse.Choices[0].FinishReason)
		log.Printf(" | [DEBUG] tool_calls: %v", chatResponse.Choices[0].Message.ToolCalls)

		if chatResponse.Choices[0].FinishReason == "stop" {
			return chatResponse.Choices[0].Message.Content, nil
		} else if chatResponse.Choices[0].FinishReason == "tool_calls" {
			messages = append(messages, chatResponse.Choices[0].Message)
			for _, toolCall := range chatResponse.Choices[0].Message.ToolCalls {
				toolLog := fmt.Sprintf("AI called tool %v with %v parameters", toolCall.Function.Name, toolCall.Function.Arguments)
				logInternalAI(toolLog)
				result, err := ExecuteToolCall(ctx, c.db, toolCall)
				if err != nil {
					result = fmt.Sprintf("tool execution failed Name: %v, Error: %v", toolCall.Function.Name, err.Error())
				}
				messages = append(messages, Message{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Content:    result,
				})
			}
			continue
		}
	}
	return "", fmt.Errorf("too many AI iterations.")
}

func logInternalAI(msg string) {
	log.Printf(" | [InternalAILog] %v", msg)
}
