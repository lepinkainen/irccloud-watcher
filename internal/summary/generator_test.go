package summary

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/llm"
	"irccloud-watcher/internal/storage"
)

func TestNewGenerator(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider:    "ollama",
			BaseURL:     "http://localhost:11434",
			Model:       "llama3.2",
			Temperature: 0.7,
			MaxTokens:   1000,
		},
	}

	generator := NewGenerator(cfg)
	if generator == nil {
		t.Fatal("Expected generator to be created, got nil")
	}

	if generator.config != cfg {
		t.Error("Expected generator config to match provided config")
	}
}

func TestNewGeneratorWithoutLLM(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{},
	}

	generator := NewGenerator(cfg)
	if generator == nil {
		t.Fatal("Expected generator to be created, got nil")
	}

	if generator.provider != nil {
		t.Error("Expected no LLM provider when LLM config is empty")
	}
}

func TestFilterMessages(t *testing.T) {
	cfg := &config.Config{}
	generator := NewGenerator(cfg)

	messages := []storage.Message{
		{ID: 1, Channel: "#test", Sender: "user1", Message: "Hello everyone, how are you doing today?", Timestamp: time.Now()},
		{ID: 2, Channel: "#test", Sender: "user2", Message: "*** user3 has joined #test", Timestamp: time.Now()},
		{ID: 3, Channel: "#test", Sender: "user1", Message: "Great discussion about Go programming", Timestamp: time.Now()},
		{ID: 4, Channel: "#test", Sender: "bot", Message: "Automated status update", Timestamp: time.Now()},
		{ID: 5, Channel: "#test", Sender: "user2", Message: "hi", Timestamp: time.Now()},
		{ID: 6, Channel: "#test", Sender: "user3", Message: "", Timestamp: time.Now()},
		{ID: 7, Channel: "#test", Sender: "user1", Message: "I think we should implement this feature", Timestamp: time.Now()},
	}

	filtered := generator.filterMessages(messages)

	// Should filter out: join message, bot message, short message, empty message
	expectedCount := 3
	if len(filtered) != expectedCount {
		t.Errorf("Expected %d filtered messages, got %d", expectedCount, len(filtered))
	}

	// Check specific messages that should remain
	expectedMessages := []string{
		"Hello everyone, how are you doing today?",
		"Great discussion about Go programming",
		"I think we should implement this feature",
	}

	for i, msg := range filtered {
		if i < len(expectedMessages) && msg.Message != expectedMessages[i] {
			t.Errorf("Expected message %d to be '%s', got '%s'", i, expectedMessages[i], msg.Message)
		}
	}
}

func TestGroupMessages(t *testing.T) {
	cfg := &config.Config{}
	generator := NewGenerator(cfg)

	baseTime := time.Now()
	messages := []storage.Message{
		{ID: 1, Channel: "#dev", Sender: "user1", Message: "Let's discuss the new feature", Timestamp: baseTime},
		{ID: 2, Channel: "#dev", Sender: "user2", Message: "I think we should use Go for this", Timestamp: baseTime.Add(5 * time.Minute)},
		{ID: 3, Channel: "#general", Sender: "user3", Message: "Anyone free for lunch?", Timestamp: baseTime.Add(10 * time.Minute)},
		{ID: 4, Channel: "#dev", Sender: "user1", Message: "Go sounds good", Timestamp: baseTime.Add(15 * time.Minute)},
		{ID: 5, Channel: "#dev", Sender: "user2", Message: "After two hours, let's continue", Timestamp: baseTime.Add(2*time.Hour + 20*time.Minute)},
	}

	groups := generator.groupMessages(messages)

	// Should have groups: #dev (first discussion), #general, #dev (after gap)
	expectedGroups := 3
	if len(groups) != expectedGroups {
		t.Errorf("Expected %d groups, got %d", expectedGroups, len(groups))
	}

	// Check channels are correctly grouped
	channels := make(map[string]int)
	for _, group := range groups {
		channels[group.Channel]++
	}

	if channels["#dev"] != 2 {
		t.Errorf("Expected 2 #dev groups (separated by time gap), got %d", channels["#dev"])
	}

	if channels["#general"] != 1 {
		t.Errorf("Expected 1 #general group, got %d", channels["#general"])
	}
}

func TestExtractTopic(t *testing.T) {
	cfg := &config.Config{}
	generator := NewGenerator(cfg)

	tests := []struct {
		name     string
		messages []storage.Message
		expected string
	}{
		{
			name:     "empty messages",
			messages: []storage.Message{},
			expected: "General Discussion",
		},
		{
			name: "programming discussion",
			messages: []storage.Message{
				{Message: "Let's talk about programming languages"},
				{Message: "I love programming in Go"},
				{Message: "Programming is fun when you solve problems"},
			},
			expected: "Programming Discussion",
		},
		{
			name: "docker and kubernetes",
			messages: []storage.Message{
				{Message: "We need to deploy using docker containers"},
				{Message: "Docker makes deployment easier"},
				{Message: "Should we use kubernetes for orchestration?"},
				{Message: "Kubernetes would help with scaling"},
			},
			expected: "Discussion", // Topic extraction order can vary, just check it contains Discussion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic := generator.extractTopic(tt.messages)
			if !strings.Contains(topic, tt.expected) {
				t.Errorf("Expected topic to contain '%s', got '%s'", tt.expected, topic)
			}
		})
	}
}

func TestIsStopWord(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
	}{
		{"the", true},
		{"and", true},
		{"programming", false},
		{"docker", false},
		{"you", true},
		{"implementation", false},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			result := isStopWord(tt.word)
			if result != tt.expected {
				t.Errorf("Expected isStopWord('%s') to be %v, got %v", tt.word, tt.expected, result)
			}
		})
	}
}

func TestGetPromptTemplate(t *testing.T) {
	cfg := &config.Config{}
	generator := NewGenerator(cfg)

	template := generator.getPromptTemplate()

	if template.SystemPrompt == "" {
		t.Error("Expected system prompt to be non-empty")
	}

	if template.UserPrompt == "" {
		t.Error("Expected user prompt to be non-empty")
	}

	if !strings.Contains(template.SystemPrompt, "IRC conversation summarizer") {
		t.Error("Expected system prompt to mention IRC conversation summarizer")
	}

	if !strings.Contains(template.UserPrompt, "%s") {
		t.Error("Expected user prompt to contain placeholder for conversations")
	}
}

func TestBuildPrompt(t *testing.T) {
	cfg := &config.Config{}
	generator := NewGenerator(cfg)

	template := Template{
		SystemPrompt: "You are a test summarizer.",
		UserPrompt:   "Summarize this: %s",
	}

	baseTime := time.Now()
	groups := []MessageGroup{
		{
			Channel:   "#test",
			Topic:     "Test Discussion",
			StartTime: baseTime,
			EndTime:   baseTime.Add(30 * time.Minute),
			Messages: []storage.Message{
				{Sender: "user1", Message: "Hello", Timestamp: baseTime},
				{Sender: "user2", Message: "Hi there", Timestamp: baseTime.Add(5 * time.Minute)},
			},
		},
	}

	prompt := generator.buildPrompt(template, groups)

	if !strings.Contains(prompt, "You are a test summarizer.") {
		t.Error("Expected prompt to contain system prompt")
	}

	if !strings.Contains(prompt, "#test - Test Discussion") {
		t.Error("Expected prompt to contain channel and topic")
	}

	if !strings.Contains(prompt, "user1") {
		t.Error("Expected prompt to contain sender names")
	}

	if !strings.Contains(prompt, "Hello") {
		t.Error("Expected prompt to contain message content")
	}
}

func TestFormatSummary(t *testing.T) {
	cfg := &config.Config{}
	generator := NewGenerator(cfg)

	baseTime := time.Now()
	messages := []storage.Message{
		{
			Channel:   "#test",
			Sender:    "user1",
			Message:   "Hello world",
			Timestamp: baseTime,
		},
		{
			Channel:   "#dev",
			Sender:    "user2",
			Message:   "Let's code",
			Timestamp: baseTime.Add(10 * time.Minute),
		},
	}

	summary := generator.formatSummary(messages)

	expectedContent := []string{
		"# Daily IRC Summary",
		"*Generated using basic text formatting*",
		"## Summary for #test",
		"user1",
		"Hello world",
		"## Summary for #dev",
		"user2",
		"Let's code",
	}

	for _, content := range expectedContent {
		if !strings.Contains(summary, content) {
			t.Errorf("Expected summary to contain '%s', but it didn't.\nSummary: %s", content, summary)
		}
	}
}

func TestGenerateDailySummaryBasicFormatting(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := storage.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Insert test messages
	baseTime := time.Now().Add(-12 * time.Hour) // Within 24 hours
	testMessages := []storage.Message{
		{
			Channel:   "#test",
			Sender:    "user1",
			Message:   "This is a test message for summary generation",
			Timestamp: baseTime,
			Date:      baseTime.Format("2006-01-02"),
			EID:       12345,
		},
		{
			Channel:   "#dev",
			Sender:    "user2",
			Message:   "Let's discuss the implementation details and architecture for the new microservice",
			Timestamp: baseTime.Add(1 * time.Hour),
			Date:      baseTime.Add(1 * time.Hour).Format("2006-01-02"),
			EID:       12346,
		},
	}

	for _, msg := range testMessages {
		insertErr := db.InsertMessage(&msg)
		if insertErr != nil {
			t.Fatalf("Failed to insert test message: %v", insertErr)
		}
	}

	// Create generator (without LLM provider)
	cfg := &config.Config{}
	generator := NewGenerator(cfg)

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "summary-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp output file: %v", err)
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	// Generate summary
	err = generator.GenerateDailySummary(db, outputFile.Name())
	if err != nil {
		t.Fatalf("Failed to generate summary: %v", err)
	}

	// Read and verify summary
	summaryContent, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatalf("Failed to read summary file: %v", err)
	}

	summary := string(summaryContent)
	expectedContent := []string{
		"# Daily IRC Summary",
		"*Generated using basic text formatting*",
		"user1",
		"This is a test message for summary generation",
		"user2",
		"Let's discuss the implementation details and architecture for the new microservice",
	}

	for _, content := range expectedContent {
		if !strings.Contains(summary, content) {
			t.Errorf("Expected summary to contain '%s', but it didn't", content)
		}
	}
}

// MockLLMProvider for testing LLM functionality
type MockLLMProvider struct {
	shouldFail bool
	response   string
}

func (m *MockLLMProvider) Generate(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if m.shouldFail {
		return nil, errors.New("mock LLM failure")
	}

	return &llm.GenerateResponse{
		Text:         m.response,
		TokensUsed:   100,
		Model:        "mock-model",
		FinishReason: "stop",
	}, nil
}

func (m *MockLLMProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func (m *MockLLMProvider) Health(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("mock health check failure")
	}
	return nil
}

func (m *MockLLMProvider) Name() string {
	return "mock"
}

func (m *MockLLMProvider) Close() error {
	return nil
}
