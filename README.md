# xollm

**XOStack LLM Abstractions for Go**

A unified Go library providing clean, consistent interfaces for interacting with multiple Large Language Model providers including Gemini, Groq, and Ollama.

## Overview

xollm abstracts away the differences between various LLM providers, offering a single, clean interface for text generation across cloud-based and self-hosted models. Originally extracted from another XOStack project, this library is being refactored into a standalone, reusable component for the XOStack ecosystem.

## Supported Providers

### Gemini (Google)
- **Model**: `gemma-3-27b-it` (default)
- **Auth**: API Key

### Groq
- **Model**: `gemma2-9b-it` (default)  
- **Auth**: API Key

### Ollama (Self-hosted)
- **Model**: `gemma:2b` (default)
- **URL**: `http://localhost:11434` (default)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/xostack/xollm"
    "github.com/xostack/xollm/config"
    "github.com/xostack/xollm/gemini"
)

func main() {
    // Direct provider instantiation
    client, err := gemini.NewClient(context.Background(), "your-api-key", "", false)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Or use factory with configuration
    cfg := config.NewConfig("gemini", 60, map[string]config.LLMConfig{
        "gemini": {APIKey: "your-api-key"},
    })
    
    client, err = xollm.GetClient(cfg, false)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    response, err := client.Generate(context.Background(), "Hello, world!")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(response)
}
```

## Dir Tree

```
xollm/
├── xollm.go          # Core interfaces
├── factory.go        # Client factory
├── config/           # Configuration management
├── gemini/           # Gemini provider
├── groq/             # Groq provider
├── ollama/           # Ollama provider
└── examples/         # Usage examples (planned)
```

## Configuration

Current configuration uses TOML format:

```toml
default_provider = "ollama"
request_timeout_seconds = 60

[llms.ollama]
base_url = "http://localhost:11434"
model = "gemma:2b"

[llms.gemini]
api_key = "your-gemini-api-key"
model = "gemma-3-27b-it"

[llms.groq]
api_key = "your-groq-api-key"
model = "gemma2-9b-it"
```

## Dependencies

- `github.com/BurntSushi/toml` - Configuration parsing
- `github.com/google/generative-ai-go` - Gemini API client
- Standard library for HTTP clients (Groq, Ollama)

## Contributing

This project follows XOStack development standards:
- Test-driven development
- Comprehensive documentation
- Idiomatic Go code
- Consistent error handling
- Clean, minimal APIs

## License

MIT License - see [LICENSE](./LICENSE) for details.

## Related Projects

Part of the XOStack ecosystem - a collection of Go-based tools and libraries for modern development workflows.
