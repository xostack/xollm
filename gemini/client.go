// Package gemini provides an LLM client for Google's Gemini models.
package gemini

import (
	"context"
	"fmt"
	"log" // For logging initialization errors if needed

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	defaultGeminiModel = "gemma-3-27b-it" // Default to Flash model
	providerName       = "gemini"
)

// Client implements the llm.Client interface for Gemini.
type Client struct {
	genaiClient *genai.Client
	modelName   string
}

// NewClient creates a new Gemini client.
// It requires a context for initialization (can be context.Background()),
// the API key, an optional model name (defaults to gemma-3-27b-it),
// and a debugMode flag.
func NewClient(ctx context.Context, apiKey string, modelOverride string, debugMode bool) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		// This log is more of a system/developer error, so keep it for now, or make it debug conditional too.
		// For now, let's assume it's important enough to always show if client creation fails.
		log.Printf("Error initializing Google GenAI client: %v. Make sure your API key is valid and has permissions.", err)
		return nil, fmt.Errorf("failed to create genai client: %w. [2, 5]", err)
	}

	modelToUse := defaultGeminiModel
	if modelOverride != "" {
		modelToUse = modelOverride
		if debugMode {
			log.Printf("Using overridden Gemini model: %s", modelToUse)
		}
	} else {
		if debugMode {
			log.Printf("Using default Gemini model: %s", modelToUse)
		}
	}

	return &Client{
		genaiClient: genaiClient,
		modelName:   modelToUse,
	}, nil
}

// Generate sends the prompt to the Gemini model and returns the text response.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	if c.genaiClient == nil {
		return "", fmt.Errorf("Gemini client not initialized")
	}

	model := c.genaiClient.GenerativeModel(c.modelName)
	if model == nil {
		return "", fmt.Errorf("failed to get generative model: %s", c.modelName)
	}

	// Simple text generation
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content from Gemini: %w. [2, 7]", err)
	}

	// Extract text from the response.
	// The response can have multiple candidates, we'll use the first one.
	// Each candidate can have multiple parts, we'll concatenate text parts.
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		// Check for blocked prompt/response
		if len(resp.Candidates) > 0 && resp.Candidates[0].FinishReason == genai.FinishReasonSafety {
			// You could inspect resp.Candidates[0].SafetyRatings for more details
			return "", fmt.Errorf("Gemini content generation blocked due to safety settings. [7]")
		}
		if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != genai.BlockReasonUnspecified {
			return "", fmt.Errorf("Gemini prompt blocked: %s. [2]", resp.PromptFeedback.BlockReason.String())
		}
		return "", fmt.Errorf("Gemini response was empty or malformed. [2, 6]")
	}

	var resultText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			resultText += string(txt)
		} else {
			// This library expects text output from the LLM.
			// If other parts are returned (e.g. function calls, blobs), we ignore them for now.
			// This log can be noisy, consider making it debug conditional if it becomes an issue.
			// For now, keeping it as it indicates unexpected parts.
			log.Printf("Gemini client received non-text part: %T. Ignoring.", part)
		}
	}

	if resultText == "" {
		// This might happen if the response only contained non-text parts or was genuinely empty.
		return "", fmt.Errorf("Gemini response contained no usable text content")
	}

	return resultText, nil
}

// ProviderName returns the name of this provider.
func (c *Client) ProviderName() string {
	return providerName
}

// Close cleans up the genaiClient.
// It's good practice to offer a Close method if the underlying client has one.
func (c *Client) Close() error {
	if c.genaiClient != nil {
		return c.genaiClient.Close() // [2, 4]
	}
	return nil
}
