# Basic Usage Example

This example demonstrates the simplest possible usage of the xollm library, showing how to get started with LLM text generation in under 20 lines of code.

## What You'll Learn

- How to use the factory pattern with configuration
- How to create clients directly from providers
- Basic error handling patterns
- Proper resource cleanup with `defer client.Close()`
- Context usage for request management

## Prerequisites

You'll need at least one LLM provider configured. The example includes sample configurations for:

- **Ollama** (local): Requires Ollama running at `http://localhost:11434`
- **Gemini** (cloud): Requires a Google AI API key
- **Groq** (cloud): Requires a Groq API key

## Running the Example

### Option 1: Using Ollama (Recommended for Testing)

1. Install and start Ollama:
   ```bash
   # Install Ollama (if not already installed)
   curl -fsSL https://ollama.ai/install.sh | sh
   
   # Start Ollama service
   ollama serve &
   
   # Pull a model (if not already done)
   ollama pull gemma:2b
   ```

2. Run the example:
   ```bash
   go run main.go
   ```

### Option 2: Using Cloud Providers

1. Set your API key as an environment variable:
   ```bash
   # For Gemini
   export GEMINI_API_KEY="your-gemini-api-key"
   
   # For Groq  
   export GROQ_API_KEY="your-groq-api-key"
   ```

2. Run with provider selection:
   ```bash
   # Use Gemini
   go run main.go -provider=gemini
   
   # Use Groq
   go run main.go -provider=groq
   ```

## Example Output

```
$ go run main.go
Using provider: ollama
Prompt: Hello, world! Please introduce yourself.

Response: Hello! I'm a helpful AI assistant. I'm here to help answer questions, provide information, and assist with various tasks. Is there anything specific you'd like to know or discuss today?

Example completed successfully!
```

## Code Structure

The example demonstrates two approaches:

### 1. Factory Pattern (Recommended)
```go
// Create configuration programmatically
cfg := config.NewConfig("ollama", 60, map[string]config.LLMConfig{
    "ollama": {BaseURL: "http://localhost:11434", Model: "gemma:2b"},
})

// Get client from factory
client, err := xollm.GetClient(cfg, false)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close() // Important: cleanup resources
```

### 2. Direct Provider Instantiation
```go
// Create client directly (useful for simple cases)
client, err := ollama.NewClient("http://localhost:11434", "gemma:2b", 60, false)
if err != nil {
    log.Fatalf("Failed to create Ollama client: %v", err)
}
defer client.Close() // Important: cleanup resources
```

## Key Concepts

### Resource Management
Always call `client.Close()` when done with a client. Use `defer` to ensure cleanup even if errors occur:

```go
client, err := xollm.GetClient(cfg, false)
if err != nil {
    return err
}
defer client.Close() // Cleanup guaranteed
```

### Context Usage
Use context for timeout and cancellation control:

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Generate with context
response, err := client.Generate(ctx, prompt)
```

### Error Handling
Handle errors at multiple levels:

```go
// Client creation errors
client, err := xollm.GetClient(cfg, false)
if err != nil {
    return fmt.Errorf("failed to create client: %w", err)
}

// Generation errors  
response, err := client.Generate(ctx, prompt)
if err != nil {
    return fmt.Errorf("failed to generate response: %w", err)
}
```

## Testing

Run the included tests to verify functionality:

```bash
# Run all tests
go test -v

# Run with coverage
go test -v -cover
```

The tests demonstrate:
- Mocking LLM clients for unit testing
- Configuration validation
- Error handling scenarios
- Context cancellation behavior

## Next Steps

After mastering basic usage, explore other examples:

- [`multi-provider-comparison`](../multi-provider-comparison/) - Compare responses across providers
- [`config-driven-cli`](../config-driven-cli/) - File-based configuration management
- [`conversation-bot`](../conversation-bot/) - Multi-turn conversations
- [`batch-processing`](../batch-processing/) - Concurrent processing patterns
