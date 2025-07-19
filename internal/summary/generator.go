package summary

import (
	"fmt"
	"os"
	"strings"
	"time"

	"irccloud-watcher/internal/storage"
)

// SummaryGenerator generates daily summaries of IRC messages.
type SummaryGenerator struct{}

// NewSummaryGenerator creates a new SummaryGenerator.
func NewSummaryGenerator() *SummaryGenerator {
	return &SummaryGenerator{}
}

// GenerateDailySummary generates a summary of messages from the previous day.
func (g *SummaryGenerator) GenerateDailySummary(db *storage.DB, outputPath string) error {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	messages, err := db.GetMessagesByDate(yesterday)
	if err != nil {
		return fmt.Errorf("could not get messages for date %s: %w", yesterday, err)
	}

	if len(messages) == 0 {
		fmt.Printf("No messages found for %s\n", yesterday)
		return nil
	}

	summary := formatSummary(messages)

	err = os.WriteFile(outputPath, []byte(summary), 0644)
	if err != nil {
		return fmt.Errorf("could not write summary to file %s: %w", outputPath, err)
	}

	fmt.Printf("Successfully generated summary for %s to %s\n", yesterday, outputPath)
	return nil
}

// formatSummary formats a slice of messages into a string.
func formatSummary(messages []storage.Message) string {
	var sb strings.Builder

	// Group messages by channel
	messagesByChannel := make(map[string][]storage.Message)
	for _, msg := range messages {
		messagesByChannel[msg.Channel] = append(messagesByChannel[msg.Channel], msg)
	}

	for channel, msgs := range messagesByChannel {
		sb.WriteString(fmt.Sprintf("## Summary for %s\n\n", channel))
		for _, msg := range msgs {
			sb.WriteString(fmt.Sprintf("[%s] <%s> %s\n", msg.Timestamp.Format("15:04"), msg.Sender, msg.Message))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
