# Conversation Bot Example

This example demonstrates how to create stateful, multi-turn conversations with LLMs using the xollm library. It showcases conversation memory, context management, and interactive chat interfaces.

## What You'll Learn

- How to maintain conversation state and history
- Context-aware LLM interactions with memory
- Interactive chat interfaces and command handling
- Conversation statistics and history management
- Bot personality configuration with system prompts
- Thread-safe conversation management

## Features

- **Stateful Conversations**: Maintains conversation history and context
- **Interactive Chat**: Real-time conversation interface
- **Bot Personalities**: Configurable system prompts for different bot behaviors
- **Conversation Commands**: Built-in commands for managing conversations
- **History Management**: Configurable history limits and trimming
- **Statistics**: Track conversation metrics and analytics
- **Thread Safety**: Safe for concurrent access

## Prerequisites

You'll need at least one LLM provider configured:

- **Ollama** (local): Default for easy setup
- **Gemini** (cloud): Requires Google AI API key
- **Groq** (cloud): Requires Groq API key

## Quick Start

### Interactive Mode (Default)

Start an interactive conversation:
```bash
go run main.go
```

### Custom Bot Personality

```bash
go run main.go -personality=professional -bot-name="BusinessBot"
```

### Test Mode

Run a predefined test conversation:
```bash
go run main.go -test
```

## Usage Examples

### Basic Interactive Chat

```bash
# Start with default settings (Ollama)
go run main.go

# Use Gemini with custom personality
export GEMINI_API_KEY="your-api-key"
go run main.go -provider=gemini -personality=creative

# Professional assistant with history limit
go run main.go -personality=professional -max-history=20
```

### Single Message Mode

```bash
# Non-interactive mode
go run main.go -interactive=false "What is artificial intelligence?"
```

### Advanced Configuration

```bash
go run main.go \
  -provider=groq \
  -bot-name="TechExpert" \
  -personality=technical \
  -max-history=50 \
  -timeout=45 \
  -debug
```

## Bot Personalities

Choose from predefined personalities:

- **helpful** (default): Friendly and helpful assistant
- **professional**: Business-focused, formal communication
- **creative**: Imaginative and artistic responses
- **technical**: Precise technical expert
- **friendly**: Casual and conversational
- **concise**: Brief and to-the-point
- **educational**: Teaching-focused explanations

```bash
# Try different personalities
go run main.go -personality=creative
go run main.go -personality=technical
go run main.go -personality=educational
```

## Interactive Commands

During conversation, use these commands:

- `/help` - Show available commands
- `/stats` - Display conversation statistics
- `/history` - Show conversation history
- `/clear` - Clear conversation history
- `quit`, `exit`, `bye` - End conversation

Example session:
```
You: Hello, what can you help me with?
Assistant: Hello! I'm here to help you with questions, tasks, and conversations...

You: /stats
Conversation Statistics:
  Total messages: 2
  Your messages: 1
  Bot messages: 1
  Average message length: 45.5 characters
  Conversation duration: 2m15s
  Started at: 14:30:15

You: Tell me about quantum computing
Assistant: Quantum computing is a revolutionary computing paradigm...

You: /clear
Conversation history cleared.

You: quit
Goodbye!
```

## Environment Variables

Configure providers with environment variables:

```bash
# Ollama (default)
export OLLAMA_BASE_URL="http://localhost:11434"
export OLLAMA_MODEL="gemma:2b"

# Gemini
export GEMINI_API_KEY="your-gemini-api-key" 
export GEMINI_MODEL="gemini-1.5-flash-latest"

# Groq
export GROQ_API_KEY="your-groq-api-key"
export GROQ_MODEL="gemma2-9b-it"
```

## Command Line Options

- `-provider` - LLM provider (ollama, gemini, groq) [default: ollama]
- `-bot-name` - Name for the conversation bot [default: Assistant]
- `-personality` - Bot personality type [default: helpful]
- `-max-history` - Maximum messages in history (0 = unlimited) [default: 0]
- `-timeout` - Request timeout in seconds [default: 60]
- `-interactive` - Enable interactive mode [default: true]
- `-test` - Run predefined test conversation [default: false]
- `-debug` - Enable debug output [default: false]

## Programming Interface

### Basic Conversation

```go
package main

import (
    "context"
    "fmt"
    "github.com/xostack/xollm/config"
)

func main() {
    // Create configuration
    cfg := config.NewConfig("ollama", 60, map[string]config.LLMConfig{
        "ollama": {BaseURL: "http://localhost:11434", Model: "gemma:2b"},
    })
    
    // Create conversation
    conv := NewConversation(cfg, "MyBot")
    defer conv.Close()
    
    // Send messages
    ctx := context.Background()
    response, err := conv.SendMessage(ctx, "Hello!")
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Bot:", response)
    
    // Continue conversation
    response, err = conv.SendMessage(ctx, "What did I just say?")
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Bot:", response)
}
```

### Conversation with System Prompt

```go
systemPrompt := "You are a helpful coding assistant specializing in Go programming."
conv := NewConversationWithSystem(cfg, "GoBot", systemPrompt)
defer conv.Close()

response, err := conv.SendMessage(ctx, "How do I create a slice in Go?")
```

### Limited History

```go
// Keep only last 10 messages
conv := NewConversationWithMaxHistory(cfg, "LimitedBot", 10)
defer conv.Close()

// Conversation will automatically trim old messages
for i := 0; i < 20; i++ {
    conv.SendMessage(ctx, fmt.Sprintf("Message %d", i))
}

// Only last 10 messages retained
fmt.Println("Message count:", conv.GetMessageCount()) // Will be 10
```

## Data Structures

### ConversationMessage

```go
type ConversationMessage struct {
    Role      string    // "user", "assistant", or "system"
    Content   string    // The message content  
    Timestamp time.Time // When the message was created
}
```

### ConversationStatistics

```go
type ConversationStatistics struct {
    TotalMessages        int           // Total number of messages
    UserMessages         int           // Number of user messages
    AssistantMessages    int           // Number of assistant messages
    AverageMessageLength float64       // Average length of all messages
    ConversationDuration time.Duration // Duration since first message
    StartTime            time.Time     // When the conversation started
}
```

### Conversation Methods

```go
// Core functionality
func (c *Conversation) SendMessage(ctx context.Context, message string) (string, error)
func (c *Conversation) GetHistory() []ConversationMessage
func (c *Conversation) GetMessageCount() int
func (c *Conversation) ClearHistory()
func (c *Conversation) Close() error

// Information
func (c *Conversation) GetBotName() string
func (c *Conversation) GetSystemPrompt() string
func (c *Conversation) GetStatistics() ConversationStatistics
```

## Example Use Cases

### Customer Support Bot

```bash
go run main.go \
  -bot-name="SupportBot" \
  -personality=professional \
  -max-history=100
```

### Creative Writing Assistant

```bash
go run main.go \
  -bot-name="CreativeWriter" \
  -personality=creative \
  -provider=gemini
```

### Technical Help Desk

```bash
go run main.go \
  -bot-name="TechSupport" \
  -personality=technical \
  -max-history=50
```

### Educational Tutor

```bash
go run main.go \
  -bot-name="Tutor" \
  -personality=educational \
  -timeout=90
```

## Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test -v

# Run with coverage
go test -v -cover

# Test specific functionality
go test -v -run TestConversation
```

The tests cover:
- Conversation creation and management
- Message sending and history tracking
- System prompt integration
- History limits and trimming
- Error handling and recovery
- Statistics calculation
- Thread safety

## Best Practices

### Memory Management

1. **Set History Limits**: Use `-max-history` for long conversations
2. **Clear History**: Use `/clear` command or `ClearHistory()` method
3. **Close Conversations**: Always call `Close()` when done
4. **Monitor Statistics**: Track conversation metrics

### Performance

1. **Choose Appropriate Providers**: Ollama for speed, cloud for quality
2. **Set Reasonable Timeouts**: Balance responsiveness with reliability
3. **Limit Context Size**: Avoid extremely long conversations
4. **Use System Prompts**: Set context once instead of repeating

### User Experience

1. **Provide Clear Commands**: Use `/help` to show available options
2. **Show Statistics**: Let users track conversation progress
3. **Handle Errors Gracefully**: Provide helpful error messages
4. **Enable Debug Mode**: Use `-debug` for troubleshooting

## Integration Examples

### Web Server Integration

```go
func chatHandler(w http.ResponseWriter, r *http.Request) {
    conv := getOrCreateConversation(sessionID)
    defer conv.Close()
    
    message := r.PostFormValue("message")
    response, err := conv.SendMessage(r.Context(), message)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]string{
        "response": response,
        "bot_name": conv.GetBotName(),
    })
}
```

### CLI Application

```go
func main() {
    conv := setupConversation()
    defer conv.Close()
    
    for {
        input := getUserInput()
        if input == "quit" {
            break
        }
        
        response, err := conv.SendMessage(context.Background(), input)
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        
        fmt.Printf("Bot: %s\n", response)
    }
}
```

## Troubleshooting

### Common Issues

1. **Memory Usage**: High memory with unlimited history
   - **Solution**: Set `-max-history` limit

2. **Slow Responses**: Long response times
   - **Solution**: Reduce `-timeout` or switch providers

3. **Context Loss**: Bot doesn't remember conversation
   - **Solution**: Check that history isn't being cleared

4. **API Errors**: Authentication or rate limiting
   - **Solution**: Verify API keys and check provider status

### Debug Mode

Enable debug mode for detailed information:
```bash
go run main.go -debug
```

Shows:
- Configuration details
- System prompt content
- Provider information
- Timing information

## Next Steps

After mastering conversation bots, explore other examples:

- [`basic-usage`](../basic-usage/) - Simple single-provider usage
- [`multi-provider-comparison`](../multi-provider-comparison/) - Provider comparison
- [`config-driven-cli`](../config-driven-cli/) - File-based configuration
- [`batch-processing`](../batch-processing/) - Concurrent processing patterns
