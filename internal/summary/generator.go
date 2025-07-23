package summary

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/llm"
	"irccloud-watcher/internal/storage"
)

// Generator generates daily summaries of IRC messages with LLM support.
type Generator struct {
	config   *config.Config
	provider llm.Provider
}

// MessageGroup represents a group of related messages.
type MessageGroup struct {
	Channel   string
	Topic     string
	Messages  []storage.Message
	StartTime time.Time
	EndTime   time.Time
}

// Template holds configurable prompt templates.
type Template struct {
	SystemPrompt string
	UserPrompt   string
}

// NewGenerator creates a new summary generator.
func NewGenerator(cfg *config.Config) *Generator {
	g := &Generator{
		config: cfg,
	}

	// Initialize LLM provider if configured
	if cfg.LLM.Provider != "" {
		g.initializeLLMProvider()
	}

	return g
}

// initializeLLMProvider initializes the LLM provider based on config.
func (g *Generator) initializeLLMProvider() {
	switch g.config.LLM.Provider {
	case "ollama":
		providerConfig := &llm.ProviderConfig{
			BaseURL:            g.config.LLM.BaseURL,
			DefaultModel:       g.config.LLM.Model,
			DefaultMaxTokens:   g.config.LLM.MaxTokens,
			DefaultTemperature: g.config.LLM.Temperature,
			Timeout:            30 * time.Second,
			RetryAttempts:      3,
			RetryDelay:         1 * time.Second,
			MaxRetryDelay:      10 * time.Second,
		}
		g.provider = llm.NewOllamaClient(providerConfig)
	default:
		log.Printf("⚠️ Unsupported LLM provider: %s, falling back to basic formatting", g.config.LLM.Provider)
	}
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

	// Preprocess messages
	filteredMessages := g.filterMessages(messages)
	groupedMessages := g.groupMessages(filteredMessages)

	var summary string

	// Try LLM generation first, fall back to basic formatting
	if g.provider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		llmSummary, llmErr := g.generateLLMSummary(ctx, groupedMessages)
		if llmErr != nil {
			log.Printf("⚠️ LLM summary generation failed: %v, falling back to basic formatting", llmErr)
			summary = g.formatSummary(filteredMessages)
		} else {
			summary = llmSummary
		}
	} else {
		summary = g.formatSummary(filteredMessages)
	}

	err = os.WriteFile(outputPath, []byte(summary), 0o644)
	if err != nil {
		return fmt.Errorf("could not write summary to file %s: %w", outputPath, err)
	}

	fmt.Printf("Successfully generated summary for last 24 hours to %s\n", outputPath)
	return nil
}

// filterMessages filters out noise like joins/parts, bot messages, etc.
func (g *Generator) filterMessages(messages []storage.Message) []storage.Message {
	var filtered []storage.Message

	// Patterns for noise filtering
	joinPartRegex := regexp.MustCompile(`^(-->|<--|\*{3})\s*(.*?)\s+(has joined|has left|has quit|joined|left|quit)`)
	modeChangeRegex := regexp.MustCompile(`^(-->|<--|\*{3})\s*.*?\s+(sets mode|was kicked|was banned)`)
	nickChangeRegex := regexp.MustCompile(`^(-->|<--|\*{3})\s*.*?\s+is now known as`)
	topicChangeRegex := regexp.MustCompile(`^(-->|<--|\*{3})\s*.*?\s+(changed the topic|set the topic)`)

	// Bot patterns (common bot names and patterns)
	botPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(bot|github|travis|ci|deploy|monitor|alert|status|feed|rss)`),
		regexp.MustCompile(`(?i)bot$`),
	}

	for _, msg := range messages {
		// Skip empty messages
		if strings.TrimSpace(msg.Message) == "" {
			continue
		}

		// Skip join/part/quit messages
		if joinPartRegex.MatchString(msg.Message) {
			continue
		}

		// Skip mode changes
		if modeChangeRegex.MatchString(msg.Message) {
			continue
		}

		// Skip nick changes
		if nickChangeRegex.MatchString(msg.Message) {
			continue
		}

		// Skip topic changes (unless it's substantial)
		if topicChangeRegex.MatchString(msg.Message) && len(msg.Message) < 100 {
			continue
		}

		// Skip likely bot messages
		isBot := false
		for _, pattern := range botPatterns {
			if pattern.MatchString(msg.Sender) {
				isBot = true
				break
			}
		}
		if isBot {
			continue
		}

		// Skip very short messages (likely not meaningful)
		if len(strings.TrimSpace(msg.Message)) < 10 {
			continue
		}

		filtered = append(filtered, msg)
	}

	return filtered
}

// groupMessages groups messages by channel and conversation topics.
func (g *Generator) groupMessages(messages []storage.Message) []MessageGroup {
	// Group by channel first
	channelGroups := make(map[string][]storage.Message)
	for _, msg := range messages {
		channelGroups[msg.Channel] = append(channelGroups[msg.Channel], msg)
	}

	var groups []MessageGroup

	for channel, msgs := range channelGroups {
		// Sort messages by timestamp
		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].Timestamp.Before(msgs[j].Timestamp)
		})

		// Group into conversation topics based on time gaps and content similarity
		currentGroup := MessageGroup{
			Channel:   channel,
			Messages:  []storage.Message{},
			StartTime: time.Now(),
			EndTime:   time.Time{},
		}

		for i, msg := range msgs {
			// Start new group if there's a significant time gap (>1 hour) or topic change
			if len(currentGroup.Messages) > 0 {
				lastMsg := currentGroup.Messages[len(currentGroup.Messages)-1]
				timeDiff := msg.Timestamp.Sub(lastMsg.Timestamp)

				// Create new group on significant time gap or when group gets too large
				if timeDiff > time.Hour || len(currentGroup.Messages) > 20 {
					if len(currentGroup.Messages) > 0 {
						currentGroup.EndTime = currentGroup.Messages[len(currentGroup.Messages)-1].Timestamp
						currentGroup.Topic = g.extractTopic(currentGroup.Messages)
						groups = append(groups, currentGroup)
					}

					currentGroup = MessageGroup{
						Channel:   channel,
						Messages:  []storage.Message{},
						StartTime: msg.Timestamp,
					}
				}
			}

			if len(currentGroup.Messages) == 0 {
				currentGroup.StartTime = msg.Timestamp
			}

			currentGroup.Messages = append(currentGroup.Messages, msg)

			// Add final group
			if i == len(msgs)-1 && len(currentGroup.Messages) > 0 {
				currentGroup.EndTime = msg.Timestamp
				currentGroup.Topic = g.extractTopic(currentGroup.Messages)
				groups = append(groups, currentGroup)
			}
		}
	}

	return groups
}

// extractTopic attempts to extract a topic from a group of messages.
func (g *Generator) extractTopic(messages []storage.Message) string {
	if len(messages) == 0 {
		return "General Discussion"
	}

	// Look for common keywords and topics
	wordCount := make(map[string]int)
	totalWords := 0

	for _, msg := range messages {
		words := strings.Fields(strings.ToLower(msg.Message))
		for _, word := range words {
			// Clean word of punctuation
			word = regexp.MustCompile(`\W`).ReplaceAllString(word, "")
			if len(word) > 3 && !isStopWord(word) {
				wordCount[word]++
				totalWords++
			}
		}
	}

	// Find most common meaningful words
	type wordFreq struct {
		word  string
		count int
	}

	var frequencies []wordFreq
	for word, count := range wordCount {
		if count > 1 && float64(count)/float64(totalWords) > 0.05 {
			frequencies = append(frequencies, wordFreq{word, count})
		}
	}

	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].count > frequencies[j].count
	})

	if len(frequencies) > 0 {
		topic := strings.ToUpper(string(frequencies[0].word[0])) + frequencies[0].word[1:]
		if len(frequencies) > 1 {
			topic += " & " + strings.ToUpper(string(frequencies[1].word[0])) + frequencies[1].word[1:]
		}
		return topic + " Discussion"
	}

	return "General Discussion"
}

// isStopWord checks if a word is a common stop word.
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true, "not": true,
		"you": true, "all": true, "can": true, "had": true, "her": true, "was": true,
		"one": true, "our": true, "out": true, "day": true, "get": true, "has": true,
		"him": true, "how": true, "its": true, "may": true, "new": true, "now": true,
		"old": true, "see": true, "two": true, "who": true, "boy": true, "did": true,
		"she": true, "use": true, "way": true, "oil": true, "sit": true, "set": true,
		"say": true, "run": true, "eat": true, "far": true, "sea": true, "eye": true,
		"ask": true, "own": true, "under": true, "think": true, "also": true, "back": true,
		"after": true, "first": true, "well": true, "year": true, "work": true, "such": true,
		"make": true, "even": true, "here": true, "good": true, "this": true, "that": true,
		"with": true, "have": true, "from": true, "they": true, "know": true, "want": true,
		"been": true, "much": true, "some": true, "time": true, "very": true, "when": true,
		"come": true, "just": true, "like": true, "long": true, "many": true, "over": true,
		"take": true, "than": true, "them": true, "were": true, "will": true,
	}
	return stopWords[word]
}

// generateLLMSummary generates a summary using the configured LLM provider.
func (g *Generator) generateLLMSummary(ctx context.Context, groups []MessageGroup) (string, error) {
	if g.provider == nil {
		return "", fmt.Errorf("no LLM provider configured")
	}

	// Check provider health first
	if err := g.provider.Health(ctx); err != nil {
		return "", fmt.Errorf("LLM provider health check failed: %w", err)
	}

	template := g.getPromptTemplate()
	prompt := g.buildPrompt(template, groups)

	req := &llm.GenerateRequest{
		Model:       g.config.LLM.Model,
		Prompt:      prompt,
		MaxTokens:   g.config.LLM.MaxTokens,
		Temperature: g.config.LLM.Temperature,
	}

	resp, err := g.provider.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %w", err)
	}

	// Add metadata
	summary := fmt.Sprintf("# Daily IRC Summary - %s\n\n", time.Now().Format("January 2, 2006"))
	summary += fmt.Sprintf("*Generated using %s (%s) - %d tokens*\n\n", g.provider.Name(), resp.Model, resp.TokensUsed)
	summary += resp.Text

	return summary, nil
}

// getPromptTemplate returns the prompt template for summary generation.
func (g *Generator) getPromptTemplate() Template {
	return Template{
		SystemPrompt: `You are an intelligent IRC conversation summarizer. Your task is to create concise, informative daily summaries of IRC channel discussions.

Guidelines:
- Focus on key discussions, decisions, and important information
- Group related topics together
- Highlight actionable items, announcements, and decisions
- Ignore noise (joins/parts, bot messages, short exchanges)
- Use clear, readable formatting with headers and bullet points
- Keep summaries concise but comprehensive
- Preserve important technical details and links
- Note any questions that were asked but not answered`,

		UserPrompt: `Please create a daily summary of the following IRC conversations. The messages are grouped by channel and topic. Focus on the most important discussions and key takeaways.

IRC Conversations:
%s

Please provide a well-structured summary with:
1. An overview of the day's activity
2. Key discussions by channel/topic
3. Important decisions or announcements
4. Technical discussions and solutions
5. Outstanding questions or action items

Format the summary in clear markdown with appropriate headers and structure.`,
	}
}

// buildPrompt builds the complete prompt for LLM generation.
func (g *Generator) buildPrompt(template Template, groups []MessageGroup) string {
	var conversationText strings.Builder

	for _, group := range groups {
		if len(group.Messages) == 0 {
			continue
		}

		conversationText.WriteString(fmt.Sprintf("\n## %s - %s\n", group.Channel, group.Topic))
		conversationText.WriteString(fmt.Sprintf("*Time: %s to %s*\n\n",
			group.StartTime.Format("15:04"), group.EndTime.Format("15:04")))

		for _, msg := range group.Messages {
			conversationText.WriteString(fmt.Sprintf("[%s] <%s> %s\n",
				msg.Timestamp.Format("15:04"), msg.Sender, strings.TrimSpace(msg.Message)))
		}
		conversationText.WriteString("\n")
	}

	// Combine system prompt and user prompt
	fullPrompt := template.SystemPrompt + "\n\n" + fmt.Sprintf(template.UserPrompt, conversationText.String())
	return fullPrompt
}

// formatSummary formats messages using the original basic formatting (fallback).
func (g *Generator) formatSummary(messages []storage.Message) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Daily IRC Summary - %s\n\n", time.Now().Format("January 2, 2006")))
	sb.WriteString("*Generated using basic text formatting*\n\n")

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
