# Multi-Provider Comparison Example

This example demonstrates how to compare responses from multiple LLM providers simultaneously, enabling side-by-side analysis of different models' capabilities, performance, and characteristics.

## What You'll Learn

- How to run the same prompt across multiple providers concurrently
- Performance comparison and timing analysis
- Error handling across different providers
- Statistical analysis of provider responses
- Formatted output and reporting

## Features

- **Concurrent Execution**: All providers are queried simultaneously for faster results
- **Performance Metrics**: Response time tracking and comparison
- **Error Resilience**: Continues comparison even if some providers fail
- **Statistical Analysis**: Fastest/slowest provider identification, averages, and response length analysis
- **Formatted Output**: Clean, readable comparison results

## Prerequisites

You'll need at least one LLM provider configured. The example supports:

- **Ollama** (local): Requires Ollama running at `http://localhost:11434`
- **Gemini** (cloud): Requires a Google AI API key
- **Groq** (cloud): Requires a Groq API key

## Running the Example

### Basic Usage

Compare all providers with a default prompt:
```bash
go run main.go
```

### Custom Providers

Compare only specific providers:
```bash
# Compare only Ollama and Gemini
go run main.go -providers=ollama,gemini

# Compare only Groq
go run main.go -providers=groq
```

### Custom Prompt

Use a custom prompt for comparison:
```bash
go run main.go -prompt="Explain quantum computing in simple terms"
```

### Complete Example

```bash
go run main.go \
  -providers=ollama,gemini,groq \
  -prompt="Write a haiku about artificial intelligence" \
  -timeout=45 \
  -debug
```

## Configuration

### Environment Variables

Set API keys via environment variables:
```bash
export GEMINI_API_KEY="your-gemini-api-key"
export GROQ_API_KEY="your-groq-api-key"

# Optional: customize Ollama
export OLLAMA_BASE_URL="http://localhost:11434"
export OLLAMA_MODEL="llama3"
```

### Command Line Options

- `-providers`: Comma-separated list of providers (default: "ollama,gemini,groq")
- `-prompt`: Prompt to send to all providers (default: "Explain artificial intelligence in one sentence.")
- `-timeout`: Request timeout in seconds (default: 30)
- `-debug`: Enable debug mode for additional information

## Example Output

```
Multi-Provider LLM Comparison
Providers: ollama, gemini, groq
Prompt: Explain artificial intelligence in one sentence.

Running comparison...

PROVIDER COMPARISON RESULTS
==========================

Individual Results:
------------------
✓ GEMINI: 1203ms
  Response: Artificial intelligence is the development of computer systems that can perform tasks that typically...

✓ GROQ: 856ms
  Response: Artificial intelligence (AI) refers to the simulation of human intelligence in machines that are...

✗ OLLAMA: FAILED
  Error: generation failed for ollama: context deadline exceeded

Summary Analysis:
----------------
Total Providers: 3
Successful: 2
Failed: 1

Performance Metrics:
-------------------
Fastest: groq (856ms)
Slowest: gemini (1203ms)
Average Duration: 1029ms
Response Length Range: 127 - 134 characters

Total comparison time: 1245ms
```

## Code Structure

### Core Functions

#### `compareProviders(providers, configs, prompt)`
Main comparison function that executes the same prompt across multiple providers concurrently.

#### `analyzeResults(results)`
Performs statistical analysis on comparison results, calculating performance metrics and response characteristics.

#### `formatResults(results, analysis)`
Creates formatted output displaying individual results and summary statistics.

#### `createProviderConfigs()`
Generates sample configurations for all supported providers, using environment variables when available.

### Data Structures

#### `ProviderResult`
```go
type ProviderResult struct {
    Provider string        // Provider name
    Response string        // Generated response
    Duration time.Duration // Response time
    Error    error         // Any error encountered
}
```

#### `ResultAnalysis`
```go
type ResultAnalysis struct {
    TotalProviders      int
    SuccessfulProviders int
    FailedProviders     int
    FastestProvider     string
    FastestDuration     time.Duration
    SlowestProvider     string
    SlowestDuration     time.Duration
    AverageDuration     time.Duration
    ShortestResponse    int
    LongestResponse     int
}
```

## Use Cases

### Model Evaluation
Compare how different models respond to the same prompt:
```bash
go run main.go -prompt="Explain the concept of recursion in programming"
```

### Performance Testing
Identify the fastest provider for your use case:
```bash
go run main.go -prompt="Hello" -timeout=10
```

### Quality Assessment
Analyze response quality across providers:
```bash
go run main.go -prompt="Write a professional email declining a meeting"
```

### Failure Analysis
Test provider reliability under various conditions:
```bash
go run main.go -prompt="Very long prompt..." -timeout=5
```

## Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test -v

# Run with coverage
go test -v -cover

# Run specific test
go test -v -run TestCompareProviders
```

The tests demonstrate:
- Concurrent provider comparison
- Error handling and resilience
- Statistical analysis accuracy
- Timeout and context cancellation
- Output formatting

## Key Concepts

### Concurrent Processing
All providers are queried simultaneously using goroutines, significantly reducing total comparison time compared to sequential processing.

### Error Isolation
Failures in one provider don't affect others - the comparison continues and reports what succeeded.

### Context Handling
Proper timeout and cancellation support ensures the comparison doesn't hang indefinitely.

### Performance Analysis
Detailed timing and statistical analysis helps identify the best provider for specific use cases.

## Next Steps

After mastering multi-provider comparison, explore other examples:

- [`basic-usage`](../basic-usage/) - Simple single-provider usage
- [`config-driven-cli`](../config-driven-cli/) - File-based configuration management  
- [`conversation-bot`](../conversation-bot/) - Multi-turn conversations
- [`batch-processing`](../batch-processing/) - Concurrent processing patterns
