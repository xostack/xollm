package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xostack/xollm"
	"github.com/xostack/xollm/config"
)

// basicUsageWithConfig demonstrates the simplest usage of xollm with a configuration object.
// It creates a client from the config and generates a response to the given prompt.
func basicUsageWithConfig(cfg config.Config, prompt string) (string, error) {
	// Validate the configuration before attempting to create a client
	if err := validateConfig(cfg); err != nil {
		return "", fmt.Errorf("invalid configuration: %w", err)
	}

	// Create client using the factory pattern
	client, err := xollm.GetClient(cfg, false)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Generate response using default context
	return basicUsageWithConfigAndContext(context.Background(), cfg, prompt)
}

// basicUsageWithConfigAndContext demonstrates usage with explicit context control.
// This allows for timeout and cancellation handling.
func basicUsageWithConfigAndContext(ctx context.Context, cfg config.Config, prompt string) (string, error) {
	// Check if context is already cancelled before proceeding
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Create client using the factory pattern
	client, err := xollm.GetClient(cfg, false)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Generate response with the provided context
	response, err := client.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return response, nil
}

// validateConfig validates that a configuration object has the required fields.
// It checks for the presence of a default provider and its corresponding configuration.
func validateConfig(cfg config.Config) error {
	if cfg.DefaultProvider == "" {
		return errors.New("default provider not specified")
	}

	providerConfig, exists := cfg.LLMs[cfg.DefaultProvider]
	if !exists {
		return fmt.Errorf("provider '%s' not configured", cfg.DefaultProvider)
	}

	// Validate provider-specific requirements
	switch cfg.DefaultProvider {
	case "gemini", "groq":
		if providerConfig.APIKey == "" {
			return fmt.Errorf("API key required for %s provider", cfg.DefaultProvider)
		}
	case "ollama":
		if providerConfig.BaseURL == "" {
			return fmt.Errorf("base URL required for %s provider", cfg.DefaultProvider)
		}
	}

	return nil
}

// createSampleConfigs returns a map of sample configurations for different providers.
// These can be used for testing, examples, or as templates for new configurations.
func createSampleConfigs() map[string]config.Config {
	configs := make(map[string]config.Config)

	// Ollama configuration (local, no API key required)
	configs["ollama"] = config.NewConfig("ollama", 60, map[string]config.LLMConfig{
		"ollama": {
			BaseURL: "http://localhost:11434",
			Model:   "gemma:2b",
		},
	})

	// Gemini configuration (requires API key from environment)
	configs["gemini"] = config.NewConfig("gemini", 60, map[string]config.LLMConfig{
		"gemini": {
			APIKey: getEnvOrDefault("GEMINI_API_KEY", "your-gemini-api-key"),
			Model:  "gemini-1.5-flash-latest",
		},
	})

	// Groq configuration (requires API key from environment)
	configs["groq"] = config.NewConfig("groq", 60, map[string]config.LLMConfig{
		"groq": {
			APIKey: getEnvOrDefault("GROQ_API_KEY", "your-groq-api-key"),
			Model:  "gemma:2b-8b-8192",
		},
	})

	return configs
}

// getEnvOrDefault returns the value of an environment variable or a default value if not set.
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// demonstrateBasicUsage shows the most common usage patterns for the xollm library.
func demonstrateBasicUsage() error {
	// Parse command line flags
	provider := flag.String("provider", "ollama", "LLM provider to use (ollama, gemini, groq)")
	prompt := flag.String("prompt", "Hello, world! Please introduce yourself.", "Prompt to send to the LLM")
	timeout := flag.Int("timeout", 30, "Request timeout in seconds")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	fmt.Printf("Using provider: %s\n", *provider)
	fmt.Printf("Prompt: %s\n\n", *prompt)

	// Get sample configuration for the selected provider
	configs := createSampleConfigs()
	cfg, exists := configs[*provider]
	if !exists {
		return fmt.Errorf("unsupported provider: %s", *provider)
	}

	// Update timeout if specified
	cfg.RequestTimeoutSeconds = *timeout

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	// Create client
	client, err := xollm.GetClient(cfg, *debug)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Generate response
	response, err := client.Generate(ctx, *prompt)
	if err != nil {
		return fmt.Errorf("failed to generate response: %w", err)
	}

	fmt.Printf("Response: %s\n\n", response)
	fmt.Println("Example completed successfully!")
	return nil
}

func main() {
	if err := demonstrateBasicUsage(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
