package ai

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/google/uuid"
)

const (
	messageThresholdForSummarization = 60
	trailingWindow                   = 15
)

func (c *Client) SummarizeMessages(ctx context.Context, existingSummary string, msgs []Message) (string, error) {
	prompt := MessageSummaryPrompt()

	summaryLabel := "EXISTING SUMMARY: (none yet — this is the first summarization pass)"
	if existingSummary != "" {
		summaryLabel = "EXISTING SUMMARY:\n" + existingSummary
	}

	var eventsText strings.Builder
	eventsText.WriteString("NEW EVENTS:\n")
	for _, msg := range msgs {
		switch msg.Role {
		case "user":
			fmt.Fprintf(&eventsText, "Player: %s\n", msg.Content)
		case "assistant":
			if msg.Content != "" {
				fmt.Fprintf(&eventsText, "DM: %s\n", msg.Content)
			}
		case "tool":
			fmt.Fprintf(&eventsText, "Tool result: %s\n", msg.Content)
		}
	}

	fullContext := []Message{
		{Role: "system", Content: prompt},
		{Role: "system", Content: summaryLabel},
		{Role: "user", Content: eventsText.String()},
	}

	result, err := c.Chat(ctx, fullContext)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func MaybeSummarizeCampaign(ctx context.Context, db *database.Queries, aiClient *Client, campaignID uuid.UUID) error {
	campaign, err := db.GetCampaign(ctx, campaignID)
	if err != nil {
		return err
	}
	count, err := db.CountMessagesAfterSequence(ctx, database.CountMessagesAfterSequenceParams{
		CampaignID: campaignID,
		Sequence:   campaign.SummarizedThrough,
	})
	if err != nil {
		return err
	}
	if count < messageThresholdForSummarization {
		return nil
	}

	currentSummary := campaign.NarrativeSummary
	msgs, err := db.GetMessagesAfterSequence(ctx, database.GetMessagesAfterSequenceParams{
		CampaignID: campaignID,
		Sequence:   campaign.SummarizedThrough,
	})
	if err != nil {
		return err
	}

	if len(msgs) <= trailingWindow {
		return nil
	}

	toFold := msgs[:len(msgs)-trailingWindow]

	var aiMsgs []Message
	for _, msg := range toFold {
		aiMsgs = append(aiMsgs, dbToAIMessage(msg))
	}

	newSummary, err := aiClient.SummarizeMessages(ctx, currentSummary, aiMsgs)
	if err != nil {
		return err
	}

	_, err = db.UpdateNarrativeSummary(ctx, database.UpdateNarrativeSummaryParams{
		ID:                campaignID,
		NarrativeSummary:  newSummary,
		SummarizedThrough: toFold[len(toFold)-1].Sequence,
	})
	if err != nil {
		return err
	}

	logAISummarization(fmt.Sprintf("Campaign %s: updated narrative summary to %d characters", campaignID, len(newSummary)))
	logAISummarization(fmt.Sprintf("New Summary:\n%s", newSummary))
	return nil
}

func dbToAIMessage(dbMsg database.Message) Message {
	return Message{
		Role:    dbMsg.Role,
		Content: dbMsg.Content,
	}
}

func logAISummarization(msg string) {
	log.Printf(" | [AISummarization] %s", msg)
}
