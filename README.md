# xollm

**XOStack LLM Abstractions for Go**

A unified Go library providing clean, consistent interfaces for interacting with multiple Large Language Model providers including Gemini, Groq, and Ollama.

## Overview

xollm abstracts away the differences between various LLM providers, offering a single, clean interface for text generation across cloud-based and self-hosted models. Originally extracted from another XOStack project, this library is being refactored into a standalone, reusable component for the XOStack ecosystem.

## Current Status

⚠️ **Under Active Refactoring** - This library is being transformed from application-specific code into a general-purpose library suitable for import by other XOStack projects.

### What Works Now
- ✅ Core `Client` interface definition
- ✅ Complete Gemini provider implementation
- ✅ Complete Groq provider implementation  
- ✅ Complete Ollama provider implementation
- ✅ TOML-based configuration system
- ✅ Factory pattern for provider instantiation

### What Needs Work
- ❌ Import path issues in factory.go
- ❌ Legacy references in configuration
- ❌ Missing test coverage
- ❌ Missing build infrastructure (Makefile)
- ❌ Configuration system designed for CLI apps, not libraries

## Supported Providers

### Gemini (Google)
- **Model**: `gemini-1.5-flash-latest` (default)
- **Auth**: API Key
- **Features**: Safety filtering, content generation, model override

### Groq
- **Model**: `llama3-8b-8192` (default)  
- **Auth**: API Key
- **Features**: OpenAI-compatible API, retry logic, timeout handling

### Ollama (Self-hosted)
- **Model**: `llama3` (default)
- **URL**: `http://localhost:11434` (default)
- **Features**: Local deployment, model management, non-streaming responses

## Quick Start (Current API - Subject to Change)

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/xostack/xollm"
    "github.com/xostack/xollm/config"
)

func main() {
    // Load configuration
    cfg, err := config.Load(false)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create client using factory
    client, err := xollm.GetClient(cfg, false)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close() // If provider supports it
    
    // Generate text
    response, err := client.Generate(context.Background(), "Hello, world!")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Provider: %s\n", client.ProviderName())
    fmt.Printf("Response: %s\n", response)
}
```

## Planned API (Post-Refactoring)

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/xostack/xollm"
    "github.com/xostack/xollm/providers/gemini"
)

func main() {
    // Direct provider instantiation
    client, err := gemini.NewClient(context.Background(), "your-api-key", "", false)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Or use factory with configuration
    config := xollm.Config{
        DefaultProvider: "gemini",
        Providers: map[string]xollm.ProviderConfig{
            "gemini": {APIKey: "your-api-key"},
        },
    }
    
    client, err = xollm.NewClient(config)
    if err != nil {
        panic(err)
    }
    
    response, err := client.Generate(context.Background(), "Hello, world!")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(response)
}
```

## Architecture

### Core Interface

```go
type Client interface {
    Generate(ctx context.Context, prompt string) (string, error)
    ProviderName() string
}
```

### Provider Structure
```
xollm/
├── xollm.go           # Core interfaces
├── factory.go         # Client factory
├── config/            # Configuration management
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
model = "llama3"

[llms.gemini]
api_key = "your-gemini-api-key"
model = "gemini-1.5-flash-latest"

[llms.groq]
api_key = "your-groq-api-key"
model = "llama3-8b-8192"
```

## Development Status

See [CODEBASE_ANALYSIS.md](./CODEBASE_ANALYSIS.md) for detailed technical analysis and refactoring plan.

### Immediate TODOs
1. Fix import path in factory.go
2. Remove legacy references from config
3. Add comprehensive test suite
4. Create Makefile with standard targets
5. Design library-first configuration API

### Phase 1: Core Library (Target: Self-contained)
- [ ] Fix critical import issues
- [ ] Remove application-specific code
- [ ] Add unit tests for all providers
- [ ] Create proper build infrastructure
- [ ] Library-friendly configuration API

### Phase 2: Enhanced Features
- [ ] Streaming response support
- [ ] Provider plugin architecture
- [ ] Metrics and observability
- [ ] Advanced error handling
- [ ] Configuration validation

### Phase 3: Ecosystem Integration
- [ ] XOStack project integration examples
- [ ] CLI utility package (separate from core library)
- [ ] Provider capability discovery
- [ ] Performance optimization

## Dependencies

### Core Dependencies
- `github.com/BurntSushi/toml` - Configuration parsing
- `github.com/google/generative-ai-go` - Gemini API client
- Standard library for HTTP clients (Groq, Ollama)

### Development Dependencies (Planned)
- Testing framework
- Mocking libraries for API testing
- Linting and formatting tools

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
