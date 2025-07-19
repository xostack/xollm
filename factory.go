// Package xollm provides the factory for creating LLM clients.
package xollm

import (
	"context" // Required for Gemini client initialization
	"fmt"

	"github.com/xostack/xollm/config"
	"github.com/xostack/xollm/gemini"
	"github.com/xostack/xollm/groq"
	"github.com/xostack/xollm/ollama"
)

// GetClient is a factory function that returns an LLM client based on the
// DefaultProvider specified in the configuration.
//
// This function provides a convenient way to create LLM clients without needing
// to know the specific provider implementation details. It handles provider-specific
// initialization logic and configuration validation.
//
// Parameters:
//   - cfg: Configuration containing provider settings and credentials
//   - debugMode: Whether to enable verbose logging for debugging
//
// Returns:
//   - Client: A provider-specific client implementing the Client interface
//   - error: Any error that occurred during client creation
//
// Supported providers:
//   - "gemini": Google Gemini (requires APIKey)
//   - "groq": Groq (requires APIKey)
//   - "ollama": Ollama (requires BaseURL)
//
// Example:
//
//	cfg := config.Config{
//		DefaultProvider: "gemini",
//		RequestTimeoutSeconds: 60,
//		LLMs: map[string]config.LLMConfig{
//			"gemini": {APIKey: "your-api-key"},
//		},
//	}
//	client, err := GetClient(cfg, false)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
// The function validates that:
//   - A default provider is specified
//   - The provider has configuration
//   - Required credentials/settings are present
//   - The provider is supported
//
// Making it a variable to allow for easy mocking in tests.
var GetClient func(cfg config.Config, debugMode bool) (Client, error) = func(cfg config.Config, debugMode bool) (Client, error) {
	providerName := cfg.DefaultProvider
	if providerName == "" {
		return nil, fmt.Errorf("no default LLM provider specified in configuration")
	}

	llmCfg, exists := cfg.LLMs[providerName]
	if !exists {
		return nil, fmt.Errorf("configuration for provider '%s' not found", providerName)
	}

	requestTimeout := cfg.RequestTimeoutSeconds
	if requestTimeout <= 0 {
		requestTimeout = 60 // Default to 60 seconds if not set or invalid
	}

	switch providerName {
	case "gemini":
		if llmCfg.APIKey == "" {
			return nil, fmt.Errorf("API key for Gemini not found in configuration")
		}
		return gemini.NewClient(context.Background(), llmCfg.APIKey, llmCfg.Model, debugMode)
	case "ollama":
		if llmCfg.BaseURL == "" {
			return nil, fmt.Errorf("base URL for Ollama not found in configuration")
		}
		return ollama.NewClient(llmCfg.BaseURL, llmCfg.Model, requestTimeout, debugMode)
	case "groq":
		if llmCfg.APIKey == "" {
			return nil, fmt.Errorf("API key for Groq not found in configuration")
		}
		return groq.NewClient(llmCfg.APIKey, llmCfg.Model, requestTimeout, debugMode)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerName)
	}
}
