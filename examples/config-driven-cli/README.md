# Config-Driven CLI Example

This example demonstrates production-ready configuration management patterns for the xollm library, including file-based configuration, interactive setup, validation, and a complete CLI interface.

## What You'll Learn

- File-based configuration management with TOML
- Interactive configuration setup and validation
- Command-line interface patterns for LLM applications
- Configuration merging and overrides
- Production deployment best practices

## Features

- **TOML Configuration**: Human-readable configuration files
- **Interactive Setup**: Guided configuration creation
- **Validation**: Comprehensive configuration validation
- **CLI Interface**: Complete command-line tool with flags
- **Config Discovery**: Automatic configuration file location
- **Template Generation**: Generate configuration templates
- **Provider Management**: List and validate available providers

## Prerequisites

This example works with any supported LLM provider:

- **Ollama** (local): Default for easy setup
- **Gemini** (cloud): Requires Google AI API key  
- **Groq** (cloud): Requires Groq API key

## Quick Start

### 1. Create Configuration

Interactive setup (recommended for first time):
```bash
go run main.go -create-config -interactive
```

Or create a default configuration:
```bash
go run main.go -create-config
```

### 2. Run with Configuration

```bash
go run main.go -prompt="Hello, world!"
```

## Configuration File Format

The tool uses TOML configuration files. Here's a complete example:

```toml
# Default provider to use
default_provider = "ollama"

# Global timeout setting
request_timeout_seconds = 60

# Ollama configuration (self-hosted)
[llms.ollama]
base_url = "http://localhost:11434"
model = "gemma:2b"

# Google Gemini configuration  
[llms.gemini]
api_key = "your-gemini-api-key"
model = "gemini-1.5-flash-latest"

# Groq configuration
[llms.groq]
api_key = "your-groq-api-key"
model = "gemma:2b-8b-8192"
```

## Configuration File Locations

The tool searches for configuration files in this order:

1. Path specified with `-config` flag
2. `xollm.toml` in current directory
3. `.xollm.toml` in current directory
4. `~/.xollm.toml` in home directory
5. `~/.config/xollm.toml` in config directory

## Command Line Usage

### Basic Commands

```bash
# Use default configuration and prompt
go run main.go

# Specify custom prompt
go run main.go -prompt="Explain quantum computing"

# Use specific provider
go run main.go -provider=gemini -prompt="Hello"

# Use custom config file
go run main.go -config=/path/to/config.toml
```

### Configuration Management

```bash
# Create new configuration interactively
go run main.go -create-config -interactive

# Create default configuration
go run main.go -create-config

# Validate existing configuration
go run main.go -validate-config

# List available providers
go run main.go -list-providers
```

### Advanced Options

```bash
# Override timeout
go run main.go -timeout=45 -prompt="Complex question"

# Enable debug mode
go run main.go -debug -prompt="Test"

# Combine multiple options
go run main.go \
  -config=production.toml \
  -provider=groq \
  -timeout=30 \
  -debug \
  -prompt="Analyze this data"
```

## Environment Variables

The interactive setup can use environment variables for API keys:

```bash
export GEMINI_API_KEY="your-gemini-api-key"
export GROQ_API_KEY="your-groq-api-key"
export OLLAMA_BASE_URL="http://localhost:11434"
export OLLAMA_MODEL="gemma:2b"
```

## Example Workflows

### Initial Setup

1. Create configuration interactively:
   ```bash
   go run main.go -create-config -interactive
   ```

2. Follow the prompts to configure your preferred provider

3. Test the configuration:
   ```bash
   go run main.go -validate-config
   ```

4. Run your first query:
   ```bash
   go run main.go -prompt="Hello, world!"
   ```

### Development Workflow

1. Use Ollama for local development:
   ```bash
   go run main.go -provider=ollama -prompt="Test prompt"
   ```

2. Switch to cloud providers for production:
   ```bash
   go run main.go -provider=gemini -prompt="Production query"
   ```

### Production Deployment

1. Create production configuration:
   ```toml
   default_provider = "gemini"
   request_timeout_seconds = 30
   
   [llms.gemini]
   api_key = "${GEMINI_API_KEY}"
   model = "gemini-1.5-flash-latest"
   ```

2. Deploy with environment variables:
   ```bash
   export GEMINI_API_KEY="prod-api-key"
   ./your-app -config=production.toml
   ```

### Multi-Environment Setup

Development config (`dev.toml`):
```toml
default_provider = "ollama"
request_timeout_seconds = 60

[llms.ollama]
base_url = "http://localhost:11434"
model = "gemma:2b"
```

Production config (`prod.toml`):
```toml
default_provider = "gemini"
request_timeout_seconds = 30

[llms.gemini]
api_key = "${GEMINI_API_KEY}"
model = "gemini-1.5-flash-latest"
```

Usage:
```bash
# Development
go run main.go -config=dev.toml

# Production  
go run main.go -config=prod.toml
```

## Error Handling

The tool provides detailed error messages for common issues:

### Missing Configuration
```
Config file not found: xollm.toml
Run with -create-config to create a new configuration file.
```

### Invalid Configuration
```
Configuration validation failed: API key required for gemini provider
```

### Provider Errors
```
Generation failed: failed to create client: API key for Gemini not found in configuration
```

## Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test -v

# Run with coverage
go test -v -cover

# Test specific functionality
go test -v -run TestLoadConfigFromFile
```

The tests cover:
- Configuration file loading and saving
- TOML parsing and validation
- Configuration merging and overrides
- CLI option parsing
- Error handling scenarios

## Best Practices

### Configuration Security

1. **Never commit API keys** to version control
2. **Use environment variables** for sensitive data
3. **Set appropriate file permissions** (600) for config files
4. **Use separate configs** for different environments

### Production Deployment

1. **Validate configuration** before deployment
2. **Set reasonable timeouts** for your use case
3. **Enable debug mode** only for troubleshooting
4. **Monitor provider performance** and adjust accordingly

### Development Workflow

1. **Start with Ollama** for local development
2. **Test with cloud providers** before production
3. **Use version control** for configuration templates
4. **Document provider-specific settings**

## Integration Examples

### Shell Scripts

```bash
#!/bin/bash
# Simple wrapper script
RESPONSE=$(go run main.go -prompt="$1" 2>/dev/null)
echo "AI Response: $RESPONSE"
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o xollm-cli main.go

# Run with mounted config
CMD ["./xollm-cli", "-config=/config/xollm.toml"]
```

### Systemd Service

```ini
[Unit]
Description=XOStack LLM CLI Service
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/xollm-cli -config=/etc/xollm/config.toml
User=xollm
Group=xollm

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### Common Issues

1. **Config file not found**: Use `-create-config` to create one
2. **API key errors**: Check environment variables and config file
3. **Timeout errors**: Increase timeout with `-timeout` flag
4. **Provider unavailable**: Use `-list-providers` to see available options

### Debug Mode

Enable debug mode for detailed information:
```bash
go run main.go -debug -prompt="Test"
```

Output includes:
- Configuration file path
- Provider details
- Timeout settings
- Response timing

## Next Steps

After mastering config-driven CLI patterns, explore other examples:

- [`basic-usage`](../basic-usage/) - Simple single-provider usage
- [`multi-provider-comparison`](../multi-provider-comparison/) - Provider comparison
- [`conversation-bot`](../conversation-bot/) - Multi-turn conversations  
- [`batch-processing`](../batch-processing/) - Concurrent processing patterns
