// Package ollama provides an LLM client for Ollama models.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	// No specific Ollama SDK is typically needed, use net/http.
)

const (
	defaultOllamaModel = "gemma:2b" // A common default, user can override in config
	providerName       = "ollama"
	generateAPIPath    = "/api/generate"
)

// Client implements the llm.Client interface for Ollama.
type Client struct {
	httpClient *http.Client
	baseURL    string // e.g., "http://localhost:11434"
	modelName  string
}

// ollamaGenerateRequest is the structure for the request body to Ollama's /api/generate.
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"` // Non-streaming behavior for complete responses
	// Add other options like System, Template, Context, Options if needed later
	// System  string                 `json:"system,omitempty"`
	// Options map[string]interface{} `json:"options,omitempty"`
}

// ollamaGenerateResponse is the structure for the response from Ollama's /api/generate
// when stream is false.
type ollamaGenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"` // This is the generated text
	Done      bool      `json:"done"`
	// Context            []int                  `json:"context,omitempty"` // For subsequent requests
	// TotalDuration      time.Duration          `json:"total_duration,omitempty"`
	// LoadDuration       time.Duration          `json:"load_duration,omitempty"`
	// PromptEvalCount    int                    `json:"prompt_eval_count,omitempty"`
	// PromptEvalDuration time.Duration          `json:"prompt_eval_duration,omitempty"`
	// EvalCount          int                    `json:"eval_count,omitempty"`
	// EvalDuration       time.Duration          `json:"eval_duration,omitempty"`
	Error string `json:"error,omitempty"` // Ollama might return an error field
}

// NewClient creates a new Ollama client.
// ctx is used for timeout configuration and cancellation.
// baseURL is the address of the Ollama server (e.g., "http://localhost:11434").
// modelOverride is an optional model name to use instead of the default.
// debugMode controls verbose logging.
func NewClient(ctx context.Context, baseURL string, modelOverride string, requestTimeoutSeconds int, debugMode bool) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("Ollama base URL is required")
	}
	// Validate and clean baseURL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama base URL '%s': %w", baseURL, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("Ollama base URL scheme must be http or https, got '%s'", parsedURL.Scheme)
	}
	// Remove any trailing slash from baseURL for consistency
	cleanedBaseURL := strings.TrimSuffix(parsedURL.String(), "/")

	modelToUse := defaultOllamaModel
	if modelOverride != "" {
		modelToUse = modelOverride
		if debugMode {
			log.Printf("Using overridden Ollama model: %s", modelToUse)
		}
	} else {
		if debugMode {
			log.Printf("Using default Ollama model: %s", modelToUse)
		}
	}

	// Use context timeout if requestTimeoutSeconds is 0
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

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:   cleanedBaseURL,
		modelName: modelToUse,
	}, nil
}

// Generate sends the prompt to the Ollama model and returns the text response.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	if c.httpClient == nil {
		return "", fmt.Errorf("Ollama client not initialized")
	}

	// Construct the request payload
	payload := ollamaGenerateRequest{
		Model:  c.modelName,
		Prompt: prompt,
		Stream: false, // Non-streaming response for complete output
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Ollama request payload: %w", err)
	}

	// Construct the request
	requestURL := c.baseURL + generateAPIPath
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create Ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if the error is due to context cancellation (e.g., timeout)
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("Ollama request canceled: %w", ctx.Err())
		}
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("Ollama request timed out: %w", ctx.Err())
		}
		return "", fmt.Errorf("failed to send request to Ollama server at %s: %w", requestURL, err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Ollama response body: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		// Attempt to get more info from the body if possible
		var errResp ollamaGenerateResponse
		if json.Unmarshal(responseBody, &errResp) == nil && errResp.Error != "" {
			return "", fmt.Errorf("Ollama API error (status %d): %s. Raw: %s", resp.StatusCode, errResp.Error, string(responseBody))
		}
		return "", fmt.Errorf("Ollama API request failed with status %s. Raw: %s", resp.Status, string(responseBody))
	}

	// Parse the response
	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(responseBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Ollama response JSON: %w. Raw response: %s", err, string(responseBody))
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("Ollama returned an error in response: %s", ollamaResp.Error)
	}

	// The main generated text is in the "response" field
	if !ollamaResp.Done && ollamaResp.Response == "" {
		// This might happen if 'done' is false but no response is given yet,
		// which is unusual for stream=false.
		return "", fmt.Errorf("Ollama response indicates not done but no text was returned")
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

// ProviderName returns the name of this provider.
func (c *Client) ProviderName() string {
	return providerName
}

// Close is a placeholder as net/http.Client typically doesn't need explicit closing
// for its default transport, but can be implemented if custom transports are used.
func (c *Client) Close() error {
	// If c.httpClient.Transport needs cleanup, do it here.
	// For the default transport, this is usually a no-op.
	return nil
}
