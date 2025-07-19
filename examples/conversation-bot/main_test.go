package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xostack/xollm"
	"github.com/xostack/xollm/config"
)

// mockClient implements xollm.Client for testing
type mockClient struct {
	generateFunc    func(ctx context.Context, prompt string) (string, error)
	providerNameVal string
	closeFunc       func() error
}

func (m *mockClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "Mock response to: " + prompt, nil
}

func (m *mockClient) ProviderName() string {
	if m.providerNameVal != "" {
		return m.providerNameVal
	}
	return "mock"
}

func (m *mockClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Mock factory function for testing
var originalGetClient = xollm.GetClient

func mockGetClient(cfg config.Config, debugMode bool) (xollm.Client, error) {
	return &mockClient{
		generateFunc: func(ctx context.Context, prompt string) (string, error) {
			if strings.Contains(prompt, "error") {
				return "", errors.New("mock generation error")
			}
			// Simulate conversation context awareness
			if strings.Contains(prompt, "previous") || strings.Contains(prompt, "earlier") {
				return "I remember our previous conversation about " + prompt, nil
			}
			return "Response to: " + prompt, nil
		},
		providerNameVal: cfg.DefaultProvider,
	}, nil
}

func TestNewConversation(t *testing.T) {
	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434", Model: "gemma:2b"},
	})

	conv := NewConversation(cfg, "test-bot")

	if conv == nil {
		t.Fatal("Expected conversation to be created")
	}

	if conv.GetBotName() != "test-bot" {
		t.Errorf("Expected bot name 'test-bot', got '%s'", conv.GetBotName())
	}

	if conv.GetMessageCount() != 0 {
		t.Errorf("Expected message count 0, got %d", conv.GetMessageCount())
	}

	if len(conv.GetHistory()) != 0 {
		t.Errorf("Expected empty history, got %d messages", len(conv.GetHistory()))
	}
}

func TestConversationSendMessage(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	conv := NewConversation(cfg, "test-bot")
	ctx := context.Background()

	response, err := conv.SendMessage(ctx, "Hello, bot!")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	// Check that message was added to history
	if conv.GetMessageCount() != 2 { // User message + bot response
		t.Errorf("Expected 2 messages in history, got %d", conv.GetMessageCount())
	}

	history := conv.GetHistory()
	if len(history) != 2 {
		t.Fatalf("Expected 2 messages in history, got %d", len(history))
	}

	// Check user message
	userMsg := history[0]
	if userMsg.Role != "user" {
		t.Errorf("Expected first message role 'user', got '%s'", userMsg.Role)
	}
	if userMsg.Content != "Hello, bot!" {
		t.Errorf("Expected first message content 'Hello, bot!', got '%s'", userMsg.Content)
	}

	// Check bot response
	botMsg := history[1]
	if botMsg.Role != "assistant" {
		t.Errorf("Expected second message role 'assistant', got '%s'", botMsg.Role)
	}
	if botMsg.Content != response {
		t.Errorf("Expected second message content '%s', got '%s'", response, botMsg.Content)
	}
}

func TestConversationMultipleTurns(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	conv := NewConversation(cfg, "chat-bot")
	ctx := context.Background()

	// First turn
	response1, err := conv.SendMessage(ctx, "What's your name?")
	if err != nil {
		t.Fatalf("Expected no error on first turn, got: %v", err)
	}

	// Second turn
	response2, err := conv.SendMessage(ctx, "Tell me about our previous conversation")
	if err != nil {
		t.Fatalf("Expected no error on second turn, got: %v", err)
	}

	// Check that both responses are different
	if response1 == response2 {
		t.Error("Expected different responses for different prompts")
	}

	// Check conversation state
	if conv.GetMessageCount() != 4 { // 2 user messages + 2 bot responses
		t.Errorf("Expected 4 messages in history, got %d", conv.GetMessageCount())
	}

	// Verify the conversation flow
	history := conv.GetHistory()
	expectedRoles := []string{"user", "assistant", "user", "assistant"}
	for i, msg := range history {
		if msg.Role != expectedRoles[i] {
			t.Errorf("Expected message %d role '%s', got '%s'", i, expectedRoles[i], msg.Role)
		}
	}
}

func TestConversationSystemPrompt(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	systemPrompt := "You are a helpful assistant specialized in Go programming."
	conv := NewConversationWithSystem(cfg, "go-bot", systemPrompt)

	if conv.GetSystemPrompt() != systemPrompt {
		t.Errorf("Expected system prompt '%s', got '%s'", systemPrompt, conv.GetSystemPrompt())
	}

	ctx := context.Background()
	_, err := conv.SendMessage(ctx, "Help me with Go")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// System prompt should not appear in regular history
	history := conv.GetHistory()
	for _, msg := range history {
		if msg.Role == "system" {
			t.Error("System prompt should not appear in user-visible history")
		}
	}
}

func TestConversationClearHistory(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	conv := NewConversation(cfg, "test-bot")
	ctx := context.Background()

	// Add some messages
	conv.SendMessage(ctx, "Hello")
	conv.SendMessage(ctx, "How are you?")

	if conv.GetMessageCount() != 4 {
		t.Errorf("Expected 4 messages before clear, got %d", conv.GetMessageCount())
	}

	conv.ClearHistory()

	if conv.GetMessageCount() != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", conv.GetMessageCount())
	}

	if len(conv.GetHistory()) != 0 {
		t.Errorf("Expected empty history after clear, got %d messages", len(conv.GetHistory()))
	}
}

func TestConversationErrorHandling(t *testing.T) {
	// Mock the factory function with error
	xollm.GetClient = func(cfg config.Config, debugMode bool) (xollm.Client, error) {
		return &mockClient{
			generateFunc: func(ctx context.Context, prompt string) (string, error) {
				return "", errors.New("mock error")
			},
			providerNameVal: cfg.DefaultProvider,
		}, nil
	}
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	conv := NewConversation(cfg, "error-bot")
	ctx := context.Background()

	response, err := conv.SendMessage(ctx, "This will error")
	if err == nil {
		t.Fatal("Expected error, got none")
	}

	if response != "" {
		t.Errorf("Expected empty response on error, got '%s'", response)
	}

	// Error should not add messages to history
	if conv.GetMessageCount() != 0 {
		t.Errorf("Expected 0 messages after error, got %d", conv.GetMessageCount())
	}
}

func TestConversationMaxHistoryLength(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	maxHistory := 4 // 2 conversation turns
	conv := NewConversationWithMaxHistory(cfg, "limited-bot", maxHistory)
	ctx := context.Background()

	// Add more messages than the limit
	for i := 0; i < 5; i++ {
		conv.SendMessage(ctx, fmt.Sprintf("Message %d", i))
	}

	// Should maintain only the most recent messages within the limit
	if conv.GetMessageCount() != maxHistory {
		t.Errorf("Expected %d messages due to limit, got %d", maxHistory, conv.GetMessageCount())
	}

	history := conv.GetHistory()
	if len(history) != maxHistory {
		t.Errorf("Expected history length %d, got %d", maxHistory, len(history))
	}

	// Should contain the most recent messages
	lastMsg := history[len(history)-1]
	if !strings.Contains(lastMsg.Content, "Message 4") && lastMsg.Role != "assistant" {
		t.Error("Expected most recent messages to be preserved")
	}
}

func TestConversationContextAwareness(t *testing.T) {
	// Mock the factory function with context awareness
	xollm.GetClient = func(cfg config.Config, debugMode bool) (xollm.Client, error) {
		return &mockClient{
			generateFunc: func(ctx context.Context, prompt string) (string, error) {
				// Check if the prompt includes conversation history
				if strings.Contains(prompt, "User:") && strings.Contains(prompt, "Assistant:") {
					return "I can see our conversation history in the prompt", nil
				}
				return "No conversation context found", nil
			},
			providerNameVal: cfg.DefaultProvider,
		}, nil
	}
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	conv := NewConversation(cfg, "context-bot")
	ctx := context.Background()

	// First message
	conv.SendMessage(ctx, "Remember this: my favorite color is blue")

	// Second message should include context
	response, err := conv.SendMessage(ctx, "What's my favorite color?")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The mock should detect conversation history in the prompt
	if !strings.Contains(response, "conversation history") {
		t.Error("Expected response to indicate conversation context was provided")
	}
}

func TestFormatConversationHistory(t *testing.T) {
	messages := []ConversationMessage{
		{Role: "user", Content: "Hello", Timestamp: time.Now()},
		{Role: "assistant", Content: "Hi there!", Timestamp: time.Now()},
		{Role: "user", Content: "How are you?", Timestamp: time.Now()},
		{Role: "assistant", Content: "I'm doing well, thanks!", Timestamp: time.Now()},
	}

	formatted := formatConversationHistory(messages)

	// Should contain user and assistant markers
	if !strings.Contains(formatted, "User:") {
		t.Error("Expected formatted history to contain 'User:' markers")
	}

	if !strings.Contains(formatted, "Assistant:") {
		t.Error("Expected formatted history to contain 'Assistant:' markers")
	}

	// Should contain the message content
	for _, msg := range messages {
		if !strings.Contains(formatted, msg.Content) {
			t.Errorf("Expected formatted history to contain '%s'", msg.Content)
		}
	}

	// Should be properly structured
	lines := strings.Split(formatted, "\n")
	if len(lines) < len(messages) {
		t.Error("Expected formatted history to have proper line structure")
	}
}

func TestCreateBotPersonality(t *testing.T) {
	personality := createBotPersonality("helpful")

	if personality == "" {
		t.Error("Expected non-empty personality")
	}

	if !strings.Contains(personality, "helpful") {
		t.Error("Expected personality to reference the requested type")
	}

	// Test different personality types
	personalities := []string{"friendly", "professional", "creative", "technical"}
	for _, p := range personalities {
		result := createBotPersonality(p)
		if result == "" {
			t.Errorf("Expected non-empty personality for type '%s'", p)
		}
	}
}

func TestConversationStatistics(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	conv := NewConversation(cfg, "stats-bot")
	ctx := context.Background()

	// Add some messages
	conv.SendMessage(ctx, "Short")
	conv.SendMessage(ctx, "This is a longer message with more words")

	stats := conv.GetStatistics()

	if stats.TotalMessages != 4 { // 2 user + 2 assistant
		t.Errorf("Expected 4 total messages, got %d", stats.TotalMessages)
	}

	if stats.UserMessages != 2 {
		t.Errorf("Expected 2 user messages, got %d", stats.UserMessages)
	}

	if stats.AssistantMessages != 2 {
		t.Errorf("Expected 2 assistant messages, got %d", stats.AssistantMessages)
	}

	if stats.AverageMessageLength <= 0 {
		t.Error("Expected positive average message length")
	}

	if stats.ConversationDuration <= 0 {
		t.Error("Expected positive conversation duration")
	}
}
