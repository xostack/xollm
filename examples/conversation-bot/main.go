package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/xostack/xollm"
	"github.com/xostack/xollm/config"
)

// ConversationMessage represents a single message in a conversation
type ConversationMessage struct {
	Role      string    // "user", "assistant", or "system"
	Content   string    // The message content
	Timestamp time.Time // When the message was created
}

// ConversationStatistics holds statistics about a conversation
type ConversationStatistics struct {
	TotalMessages        int           // Total number of messages
	UserMessages         int           // Number of user messages
	AssistantMessages    int           // Number of assistant messages
	AverageMessageLength float64       // Average length of all messages
	ConversationDuration time.Duration // Duration since first message
	StartTime            time.Time     // When the conversation started
}

// Conversation manages a stateful conversation with an LLM
type Conversation struct {
	config       config.Config         // LLM configuration
	client       xollm.Client          // LLM client instance
	botName      string                // Name of the bot
	systemPrompt string                // System prompt for the bot
	messages     []ConversationMessage // Conversation history
	maxHistory   int                   // Maximum number of messages to keep (0 = unlimited)
	startTime    time.Time             // When the conversation started
	mutex        sync.RWMutex          // For thread safety
}

// NewConversation creates a new conversation with default settings
func NewConversation(cfg config.Config, botName string) *Conversation {
	return &Conversation{
		config:     cfg,
		botName:    botName,
		messages:   make([]ConversationMessage, 0),
		maxHistory: 0, // Unlimited by default
		startTime:  time.Now(),
	}
}

// NewConversationWithSystem creates a new conversation with a system prompt
func NewConversationWithSystem(cfg config.Config, botName, systemPrompt string) *Conversation {
	conv := NewConversation(cfg, botName)
	conv.systemPrompt = systemPrompt
	return conv
}

// NewConversationWithMaxHistory creates a new conversation with a maximum history limit
func NewConversationWithMaxHistory(cfg config.Config, botName string, maxHistory int) *Conversation {
	conv := NewConversation(cfg, botName)
	conv.maxHistory = maxHistory
	return conv
}

// GetBotName returns the bot's name
func (c *Conversation) GetBotName() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.botName
}

// GetSystemPrompt returns the system prompt
func (c *Conversation) GetSystemPrompt() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.systemPrompt
}

// GetMessageCount returns the total number of messages in the conversation
func (c *Conversation) GetMessageCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.messages)
}

// GetHistory returns a copy of the conversation history
func (c *Conversation) GetHistory() []ConversationMessage {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Return a copy to prevent external modification
	history := make([]ConversationMessage, len(c.messages))
	copy(history, c.messages)
	return history
}

// ClearHistory clears the conversation history
func (c *Conversation) ClearHistory() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.messages = make([]ConversationMessage, 0)
}

// SendMessage sends a message to the LLM and returns the response
func (c *Conversation) SendMessage(ctx context.Context, userMessage string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Create client if not already created
	if c.client == nil {
		client, err := xollm.GetClient(c.config, false)
		if err != nil {
			return "", fmt.Errorf("failed to create LLM client: %w", err)
		}
		c.client = client
	}

	// Build the full prompt with conversation context
	prompt := c.buildPrompt(userMessage)

	// Generate response
	response, err := c.client.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	// Add user message to history
	userMsg := ConversationMessage{
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}
	c.messages = append(c.messages, userMsg)

	// Add assistant response to history
	assistantMsg := ConversationMessage{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	}
	c.messages = append(c.messages, assistantMsg)

	// Trim history if needed
	c.trimHistoryIfNeeded()

	return response, nil
}

// buildPrompt constructs the full prompt including system prompt and conversation history
func (c *Conversation) buildPrompt(userMessage string) string {
	var prompt strings.Builder

	// Add system prompt if present
	if c.systemPrompt != "" {
		prompt.WriteString("System: ")
		prompt.WriteString(c.systemPrompt)
		prompt.WriteString("\n\n")
	}

	// Add conversation history if present
	if len(c.messages) > 0 {
		prompt.WriteString("Previous conversation:\n")
		prompt.WriteString(formatConversationHistory(c.messages))
		prompt.WriteString("\n")
	}

	// Add current user message
	prompt.WriteString("User: ")
	prompt.WriteString(userMessage)
	prompt.WriteString("\nAssistant:")

	return prompt.String()
}

// trimHistoryIfNeeded removes old messages if the history exceeds the maximum limit
func (c *Conversation) trimHistoryIfNeeded() {
	if c.maxHistory <= 0 || len(c.messages) <= c.maxHistory {
		return
	}

	// Remove oldest messages, keeping the most recent ones
	toRemove := len(c.messages) - c.maxHistory
	c.messages = c.messages[toRemove:]
}

// GetStatistics returns statistics about the conversation
func (c *Conversation) GetStatistics() ConversationStatistics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := ConversationStatistics{
		TotalMessages:        len(c.messages),
		ConversationDuration: time.Since(c.startTime),
		StartTime:            c.startTime,
	}

	if len(c.messages) == 0 {
		return stats
	}

	var totalLength int
	for _, msg := range c.messages {
		totalLength += len(msg.Content)
		switch msg.Role {
		case "user":
			stats.UserMessages++
		case "assistant":
			stats.AssistantMessages++
		}
	}

	stats.AverageMessageLength = float64(totalLength) / float64(len(c.messages))
	return stats
}

// Close cleans up the conversation resources
func (c *Conversation) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// formatConversationHistory formats a list of messages into a readable string
func formatConversationHistory(messages []ConversationMessage) string {
	var formatted strings.Builder

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			formatted.WriteString("User: ")
		case "assistant":
			formatted.WriteString("Assistant: ")
		case "system":
			continue // Skip system messages in history
		}
		formatted.WriteString(msg.Content)
		formatted.WriteString("\n")
	}

	return formatted.String()
}

// createBotPersonality returns a system prompt for different bot personalities
func createBotPersonality(personalityType string) string {
	personalities := map[string]string{
		"helpful":      "You are a helpful and friendly assistant. You provide clear, accurate, and useful responses while maintaining a warm and approachable tone.",
		"professional": "You are a professional assistant with expertise in business and formal communication. You provide detailed, well-structured responses in a formal tone.",
		"creative":     "You are a creative and imaginative assistant. You think outside the box and provide innovative, artistic, and original responses.",
		"technical":    "You are a technical expert assistant. You provide precise, detailed, and accurate technical information with clear explanations.",
		"friendly":     "You are a friendly and casual assistant. You communicate in a relaxed, conversational manner while being helpful and supportive.",
		"concise":      "You are a concise assistant. You provide brief, to-the-point responses that are accurate and efficient.",
		"educational":  "You are an educational assistant. You explain concepts clearly, provide context, and help users learn and understand topics thoroughly.",
	}

	if personality, exists := personalities[personalityType]; exists {
		return personality
	}

	// Default personality
	return personalities["helpful"]
}

// Interactive conversation loop
func runInteractiveConversation(conv *Conversation) error {
	fmt.Printf("Starting conversation with %s\n", conv.GetBotName())
	fmt.Println("Type 'quit', 'exit', or 'bye' to end the conversation")
	fmt.Println("Type '/help' for available commands")
	fmt.Println(strings.Repeat("-", 50))

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle special commands
		switch input {
		case "quit", "exit", "bye":
			fmt.Println("\nGoodbye!")
			return nil
		case "/help":
			printHelpCommands()
			continue
		case "/stats":
			printConversationStats(conv)
			continue
		case "/history":
			printConversationHistory(conv)
			continue
		case "/clear":
			conv.ClearHistory()
			fmt.Println("Conversation history cleared.")
			continue
		}

		// Send message to bot
		fmt.Printf("%s: ", conv.GetBotName())
		response, err := conv.SendMessage(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(response)
	}

	return scanner.Err()
}

// printHelpCommands prints available commands
func printHelpCommands() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  /help     - Show this help message")
	fmt.Println("  /stats    - Show conversation statistics")
	fmt.Println("  /history  - Show conversation history")
	fmt.Println("  /clear    - Clear conversation history")
	fmt.Println("  quit/exit/bye - End the conversation")
}

// printConversationStats prints conversation statistics
func printConversationStats(conv *Conversation) {
	stats := conv.GetStatistics()
	fmt.Printf("\nConversation Statistics:\n")
	fmt.Printf("  Total messages: %d\n", stats.TotalMessages)
	fmt.Printf("  Your messages: %d\n", stats.UserMessages)
	fmt.Printf("  Bot messages: %d\n", stats.AssistantMessages)
	fmt.Printf("  Average message length: %.1f characters\n", stats.AverageMessageLength)
	fmt.Printf("  Conversation duration: %v\n", stats.ConversationDuration.Round(time.Second))
	fmt.Printf("  Started at: %s\n", stats.StartTime.Format("15:04:05"))
}

// printConversationHistory prints the conversation history
func printConversationHistory(conv *Conversation) {
	history := conv.GetHistory()
	if len(history) == 0 {
		fmt.Println("\nNo conversation history.")
		return
	}

	fmt.Println("\nConversation History:")
	fmt.Println(strings.Repeat("-", 30))

	for _, msg := range history {
		timestamp := msg.Timestamp.Format("15:04:05")
		switch msg.Role {
		case "user":
			fmt.Printf("[%s] You: %s\n", timestamp, msg.Content)
		case "assistant":
			fmt.Printf("[%s] %s: %s\n", timestamp, conv.GetBotName(), msg.Content)
		}
	}
}

// demonstrateConversationBot runs the main conversation bot demonstration
func demonstrateConversationBot() error {
	// Parse command line flags
	provider := flag.String("provider", "ollama", "LLM provider to use (ollama, gemini, groq)")
	botName := flag.String("bot-name", "Assistant", "Name for the conversation bot")
	personality := flag.String("personality", "helpful", "Bot personality (helpful, professional, creative, technical, friendly)")
	maxHistory := flag.Int("max-history", 0, "Maximum number of messages to keep in history (0 = unlimited)")
	timeout := flag.Int("timeout", 60, "Request timeout in seconds")
	interactive := flag.Bool("interactive", true, "Run in interactive mode")
	testMode := flag.Bool("test", false, "Run in test mode with predefined conversation")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	// Create configuration
	var cfg config.Config
	switch *provider {
	case "ollama":
		cfg = config.NewConfig("ollama", *timeout, map[string]config.LLMConfig{
			"ollama": {
				BaseURL: getEnvOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"),
				Model:   getEnvOrDefault("OLLAMA_MODEL", "llama3"),
			},
		})
	case "gemini":
		apiKey := getEnvOrDefault("GEMINI_API_KEY", "")
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY environment variable is required for Gemini provider")
		}
		cfg = config.NewConfig("gemini", *timeout, map[string]config.LLMConfig{
			"gemini": {
				APIKey: apiKey,
				Model:  getEnvOrDefault("GEMINI_MODEL", "gemini-1.5-flash-latest"),
			},
		})
	case "groq":
		apiKey := getEnvOrDefault("GROQ_API_KEY", "")
		if apiKey == "" {
			return fmt.Errorf("GROQ_API_KEY environment variable is required for Groq provider")
		}
		cfg = config.NewConfig("groq", *timeout, map[string]config.LLMConfig{
			"groq": {
				APIKey: apiKey,
				Model:  getEnvOrDefault("GROQ_MODEL", "llama3-8b-8192"),
			},
		})
	default:
		return fmt.Errorf("unsupported provider: %s", *provider)
	}

	// Create conversation with system prompt based on personality
	systemPrompt := createBotPersonality(*personality)
	var conv *Conversation
	if *maxHistory > 0 {
		conv = NewConversationWithMaxHistory(cfg, *botName, *maxHistory)
	} else {
		conv = NewConversation(cfg, *botName)
	}
	conv.systemPrompt = systemPrompt
	defer conv.Close()

	if *debug {
		fmt.Printf("Configuration:\n")
		fmt.Printf("  Provider: %s\n", cfg.DefaultProvider)
		fmt.Printf("  Bot Name: %s\n", *botName)
		fmt.Printf("  Personality: %s\n", *personality)
		fmt.Printf("  Max History: %d\n", *maxHistory)
		fmt.Printf("  Timeout: %ds\n", *timeout)
		fmt.Printf("  System Prompt: %s\n\n", systemPrompt)
	}

	if *testMode {
		return runTestConversation(conv)
	}

	if *interactive {
		return runInteractiveConversation(conv)
	}

	// Single message mode
	if len(flag.Args()) == 0 {
		return fmt.Errorf("no message provided in non-interactive mode")
	}

	message := strings.Join(flag.Args(), " ")
	ctx := context.Background()

	fmt.Printf("User: %s\n", message)
	response, err := conv.SendMessage(ctx, message)
	if err != nil {
		return fmt.Errorf("conversation failed: %w", err)
	}

	fmt.Printf("%s: %s\n", conv.GetBotName(), response)
	return nil
}

// runTestConversation runs a predefined test conversation
func runTestConversation(conv *Conversation) error {
	testMessages := []string{
		"Hello, what's your name?",
		"Can you remember what I just asked you?",
		"What's the capital of France?",
		"Thank you for the conversation!",
	}

	ctx := context.Background()
	fmt.Printf("Running test conversation with %s...\n\n", conv.GetBotName())

	for i, message := range testMessages {
		fmt.Printf("Turn %d\n", i+1)
		fmt.Printf("User: %s\n", message)

		response, err := conv.SendMessage(ctx, message)
		if err != nil {
			return fmt.Errorf("failed at turn %d: %w", i+1, err)
		}

		fmt.Printf("%s: %s\n\n", conv.GetBotName(), response)

		// Brief pause between messages
		time.Sleep(500 * time.Millisecond)
	}

	// Print final statistics
	fmt.Println("Test conversation completed!")
	printConversationStats(conv)

	return nil
}

// getEnvOrDefault returns the value of an environment variable or a default value if not set
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	if err := demonstrateConversationBot(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
