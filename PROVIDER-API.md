# XOLlm Provider API Documentation

This document specifies the internal API that all LLM provider implementations must follow in the XOStack xollm library. It serves as a comprehensive guide for implementing new provider wrappers.

## Overview

The xollm library provides a unified interface for interacting with various Large Language Model providers. Each provider must implement the `Client` interface defined in `xollm.go` and follow specific patterns for initialization, configuration, and error handling.

## Core Interface Requirements

### 1. Client Interface

All providers MUST implement the `xollm.Client` interface:

```go
type Client interface {
    Generate(ctx context.Context, prompt string) (string, error)
    ProviderName() string
    Close() error
}
```

### 2. Package Structure

Each provider should be implemented as a separate package under the main xollm directory:

```
xollm/
├── providername/
│   ├── client.go        # Main implementation
│   └── client_test.go   # Comprehensive tests
```

## Implementation Requirements

### 1. Package Declaration and Documentation

```go
// Package providername provides an LLM client for [Provider Name] models.
package providername
```

**Required Documentation:**
- Clear package purpose
- Provider-specific details (API, models, etc.)
- Usage examples if provider has unique characteristics

### 2. Constants and Configuration

Each provider should define appropriate constants:

```go
const (
    defaultProviderModel = "model-name"     // Sensible default model
    providerName        = "providername"    // Lowercase provider identifier
    // Provider-specific constants (API endpoints, retry settings, etc.)
)
```

### 3. Client Structure

```go
type Client struct {
    // Provider-specific client fields
    // Examples from existing implementations:
    httpClient  *http.Client        // For HTTP-based APIs (Groq, Ollama)
    genaiClient *genai.Client       // For SDK-based APIs (Gemini)
    apiKey      string              // For authentication
    baseURL     string              // For self-hosted/custom endpoints
    modelName   string              // Current model selection
}
```

### 4. NewClient Constructor
of the
All providers now use a standardized constructor signature that includes context as the first parameter:

#### For API Key-based Providers (Cloud Services)

```go
// NewClient creates a new [Provider] client.
// Parameters:
//   - ctx: Context for timeout configuration and cancellation
//   - apiKey: Authentication token (required)
//   - modelOverride: Optional model name override (empty string uses default)
//   - requestTimeoutSeconds: HTTP request timeout
//   - debugMode: Enable verbose logging
func NewClient(ctx context.Context, apiKey string, modelOverride string, requestTimeoutSeconds int, debugMode bool) (*Client, error)
```

#### For Self-hosted Providers

```go
// NewClient creates a new [Provider] client.
// Parameters:
//   - ctx: Context for timeout configuration and cancellation
//   - baseURL: Server endpoint (e.g., "http://localhost:11434")
//   - modelOverride: Optional model name override
//   - requestTimeoutSeconds: HTTP request timeout
//   - debugMode: Enable verbose logging
func NewClient(ctx context.Context, baseURL string, modelOverride string, requestTimeoutSeconds int, debugMode bool) (*Client, error)
```

### 5. Constructor Implementation Patterns

#### Input Validation

```go
func NewClient(ctx context.Context, /* other parameters */) (*Client, error) {
    // 1. Validate required parameters
    if apiKey == "" { // or baseURL for self-hosted
        return nil, fmt.Errorf("[Provider] API key is required")
    }
    
    // 2. Validate and process optional parameters
    modelToUse := defaultProviderModel
    if modelOverride != "" {
        modelToUse = modelOverride
        if debugMode {
            log.Printf("Using overridden [Provider] model: %s", modelToUse)
        }
    } else if debugMode {
        log.Printf("Using default [Provider] model: %s", modelToUse)
    }
    
    // 3. Handle timeout configuration with context
    timeout := time.Duration(requestTimeoutSeconds) * time.Second
    if requestTimeoutSeconds <= 0 {
        // Check if context has a deadline
        if deadline, ok := ctx.Deadline(); ok {
            timeout = time.Until(deadline)
            if debugMode {
                log.Printf("Using context deadline for timeout: %v", timeout)
            }
        } else {
            timeout = 60 * time.Second // Default fallback
            if debugMode {
                log.Printf("Using default timeout: %v", timeout)
            }
        }
    }
    
    // 4. Initialize provider-specific client
    // (HTTP client, SDK client, etc.)
    
    return &Client{
        // Initialize fields
    }, nil
}
```

#### URL Validation (for self-hosted providers)

```go
// Validate and clean baseURL
parsedURL, err := url.Parse(baseURL)
if err != nil {
    return nil, fmt.Errorf("invalid [Provider] base URL '%s': %w", baseURL, err)
}
if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
    return nil, fmt.Errorf("[Provider] base URL scheme must be http or https, got '%s'", parsedURL.Scheme)
}
// Remove trailing slash for consistency
cleanedBaseURL := strings.TrimSuffix(parsedURL.String(), "/")
```

### 6. Generate Method Implementation

The `Generate` method is the core functionality and must handle various scenarios:

```go
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
    // 1. Validate client state
    if c.httpClient == nil { // or appropriate client field
        return "", fmt.Errorf("[provider] client not initialized")
    }
    
    // 2. Prepare request (provider-specific)
    // Examples:
    // - Create HTTP request payload
    // - Convert prompt to provider's expected format
    // - Set up API call parameters
    
    // 3. Make the API call with context support
    // - Use ctx for cancellation/timeout
    // - Handle retries if appropriate
    // - Parse provider-specific response format
    
    // 4. Extract and return text response
    // - Handle empty responses
    // - Trim whitespace
    // - Validate response format
    
    return strings.TrimSpace(responseText), nil
}
```

#### Error Handling Patterns

**Context Errors (High Priority):**
```go
if ctx.Err() == context.Canceled {
    return "", fmt.Errorf("[Provider] request canceled: %w", ctx.Err())
}
if ctx.Err() == context.DeadlineExceeded {
    return "", fmt.Errorf("[Provider] request timed out: %w", ctx.Err())
}
```

**API Errors with Codes:**
```go
// For providers that return structured error responses
if resp.Error != nil {
    return "", fmt.Errorf("[provider] API error: %s (Type: %s, Code: %s). HTTP Status: %s", 
        resp.Error.Message, resp.Error.Type, resp.Error.Code, httpResp.Status)
}
```

**HTTP Status Errors:**
```go
if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("[provider] API request failed with status %s. Body: %s", 
        resp.Status, string(responseBody))
}
```

**Empty Response Handling:**
```go
if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
    // Log additional context if available
    log.Printf("[Provider] response details: ID=%s, Model=%s, FinishReason=%s", 
        response.ID, response.Model, response.Choices[0].FinishReason)
    return "", fmt.Errorf("[provider] response contained no choices or empty message content")
}
```

### 7. ProviderName Method

Simple implementation returning the provider identifier:

```go
func (c *Client) ProviderName() string {
    return providerName
}
```

### 8. Close Method

Resource cleanup implementation:

```go
func (c *Client) Close() error {
    if c.genaiClient != nil {  // For SDK-based clients
        return c.genaiClient.Close()
    }
    // For HTTP-only clients, usually a no-op
    return nil
}
```

## HTTP-Based Provider Implementation Details

### Request Structure

For OpenAI-compatible APIs:

```go
type providerChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type providerChatCompletionRequest struct {
    Messages    []providerChatMessage `json:"messages"`
    Model       string               `json:"model"`
    Temperature *float64             `json:"temperature,omitempty"`
    MaxTokens   *int                 `json:"max_tokens,omitempty"`
    Stream      bool                 `json:"stream"`
}
```

### Response Structure

```go
type providerChatCompletionResponse struct {
    ID      string `json:"id"`
    Object  string `json:"object"`
    Created int64  `json:"created"`
    Model   string `json:"model"`
    Choices []struct {
        Index   int `json:"index"`
        Message struct {
            Role    string `json:"role"`
            Content string `json:"content"`
        } `json:"message"`
        FinishReason string `json:"finish_reason"`
    } `json:"choices"`
    Usage struct {
        PromptTokens     int `json:"prompt_tokens"`
        CompletionTokens int `json:"completion_tokens"`
        TotalTokens      int `json:"total_tokens"`
    } `json:"usage"`
    Error *struct {
        Message string `json:"message"`
        Type    string `json:"type"`
        Code    string `json:"code,omitempty"`
    } `json:"error,omitempty"`
}
```

### Retry Logic

For transient network issues:

```go
const (
    maxRetries = 1
    retryDelay = 1 * time.Second
)

var lastErr error
for i := 0; i <= maxRetries; i++ {
    resp, err = c.httpClient.Do(req)
    if err != nil {
        lastErr = fmt.Errorf("failed to send request: %w", err)
        // Don't retry on context errors
        if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
            return "", lastErr
        }
        log.Printf("[Provider] request attempt %d failed: %v. Retrying in %v...", i+1, err, retryDelay)
        time.Sleep(retryDelay)
        continue
    }
    break
}
```

## Factory Integration

### Configuration Structure

Add your provider to the `LLMConfig` struct in `config/config.go`:

```go
type LLMConfig struct {
    BaseURL string `toml:"base_url,omitempty"`  // For self-hosted
    APIKey  string `toml:"api_key,omitempty"`   // For cloud services
    Model   string `toml:"model,omitempty"`     // Optional override
    // Add provider-specific fields if needed
}
```

### Factory Registration

Add your provider to the `GetClient` function in `factory.go`:

```go
switch providerName {
case "yournewprovider":
    if llmCfg.APIKey == "" {  // or appropriate validation
        return nil, fmt.Errorf("API key for YourNewProvider not found in configuration")
    }
    return yournewprovider.NewClient(context.Background(), llmCfg.APIKey, llmCfg.Model, requestTimeout, debugMode)
// ... other cases
}
```

### Configuration Defaults

Add default configuration in `config/config.go`:

```go
func defaultConfig() Config {
    return Config{
        // ...
        LLMs: map[string]LLMConfig{
            // ...
            "yournewprovider": {
                APIKey: "", // Requires user input
                // or BaseURL: "http://localhost:port" for self-hosted
            },
        },
    }
}
```

## Testing Requirements

### Comprehensive Test Coverage

Each provider must include thorough tests:

```go
func TestNewClient(t *testing.T) {
    // Test successful creation
    // Test validation failures (empty API key, invalid URL, etc.)
    // Test debug mode logging
}

func TestGenerate(t *testing.T) {
    // Test successful generation with mocked HTTP responses
    // Test various error scenarios (network, API errors, empty responses)
    // Test context cancellation and timeout
    // Test different response formats
}

func TestProviderName(t *testing.T) {
    // Verify correct provider name returned
}

func TestClose(t *testing.T) {
    // Test resource cleanup
}
```

### Mock Testing Patterns

Use HTTP test servers for testing:

```go
func TestGenerate_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        response := `{"choices":[{"message":{"content":"test response"}}]}`
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(response))
    }))
    defer server.Close()

    client, err := NewClient(context.Background(), server.URL, "test-model", 30, false)
    require.NoError(t, err)
    
    result, err := client.Generate(context.Background(), "test prompt")
    require.NoError(t, err)
    assert.Equal(t, "test response", result)
}
```

## XOStack-Specific Guidelines

### 1. Error Message Format

Use consistent error message formatting:
- Include provider name in error messages
- Use descriptive error context
- Include HTTP status codes when relevant
- Reference error codes if provider supplies them

### 2. Logging Standards

- Use conditional debug logging with `debugMode` parameter
- Log important state changes (model selection, connection status)
- Avoid logging sensitive information (API keys, full prompts)
- Use structured logging when possible

### 3. Idiomatic Go Practices

Follow the Go style guidelines specified in the repository:
- Use proper error wrapping with `fmt.Errorf` and `%w` verb
- Implement proper resource cleanup in `Close()`
- Use context appropriately for cancellation and timeouts
- Follow naming conventions (mixedCaps, clear names)
- Keep interfaces small and focused

### 4. Documentation Standards

- Document all exported functions with clear descriptions
- Include parameter descriptions and requirements
- Provide usage examples for complex configurations
- Document any provider-specific limitations or behaviors

### 5. Configuration Integration

- Support configuration through both factory and direct instantiation
- Validate configuration parameters thoroughly
- Provide sensible defaults for optional parameters
- Support debug mode for troubleshooting

## Common Implementation Pitfalls

1. **Not handling context cancellation** - Always check `ctx.Err()` in network operations
2. **Missing input validation** - Validate all required parameters in constructors
3. **Poor error messages** - Include provider name and context in error messages
4. **Resource leaks** - Implement proper cleanup in `Close()` method
5. **Inconsistent timeouts** - Use the provided `requestTimeoutSeconds` parameter
6. **Ignoring debug mode** - Provide helpful debug output when enabled
7. **Not trimming response text** - Always trim whitespace from LLM responses

## Example Provider Implementation

See the existing implementations in `gemini/`, `groq/`, and `ollama/` packages for concrete examples of these patterns in action. Each demonstrates different approaches based on provider characteristics:

- **Gemini**: SDK-based implementation with context requirements
- **Groq**: HTTP-based OpenAI-compatible API
- **Ollama**: Self-hosted HTTP API with custom request/response format

## Testing Your Implementation

Before submitting a new provider:

1. Implement comprehensive unit tests with mocked responses
2. Test error scenarios (network failures, API errors, malformed responses)
3. Verify integration with the factory pattern
4. Test configuration loading and validation
5. Ensure proper resource cleanup
6. Validate debug logging output
7. Check compliance with Go style guidelines

Use the `make test` command to run the full test suite and ensure your implementation doesn't break existing functionality.
