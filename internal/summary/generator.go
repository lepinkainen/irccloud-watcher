package summary

import (
	"fmt"
	"os"
	"strings"
	"time"

	"irccloud-watcher/internal/storage"
)

// SummaryGenerator generates daily summaries of IRC messages.
type Generator struct{}

// NewSummaryGenerator creates a new SummaryGenerator.
func NewSummaryGenerator() *Generator {
	return &Generator{}
}

// GenerateDailySummary generates a summary of messages from the last 24 hours.
func (g *Generator) GenerateDailySummary(db *storage.DB, outputPath string) error {
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	messages, err := db.GetMessagesInTimeRange(startTime, endTime)
	if err != nil {
		return fmt.Errorf("could not get messages for time range %s to %s: %w", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), err)
	}

	if len(messages) == 0 {
		fmt.Printf("No messages found in the last 24 hours\n")
		return nil
	}

	summary := formatSummary(messages)

	err = os.WriteFile(outputPath, []byte(summary), 0o644)
	if err != nil {
		return fmt.Errorf("could not write summary to file %s: %w", outputPath, err)
	}

	fmt.Printf("Successfully generated summary for last 24 hours to %s\n", outputPath)
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
			sb.WriteString(fmt.Sprintf("[%s] <%s> %s\n", msg.Timestamp.Format(time.RFC3339), msg.Sender, msg.Message))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
