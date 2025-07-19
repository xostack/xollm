// Package xollm provides unified interfaces and abstractions for interacting with Large Language Models.
//
// This package offers a consistent API for working with various LLM providers including:
//   - Google Gemini (cloud-based)
//   - Groq (cloud-based)
//   - Ollama (self-hosted)
//
// The core interface is designed to be simple yet flexible, allowing applications to switch
// between providers with minimal code changes.
//
// Example usage:
//
//	// Using the factory pattern with configuration
//	cfg := config.NewConfig("gemini", 60, map[string]config.LLMConfig{
//		"gemini": {APIKey: "your-api-key"},
//	})
//	client, err := xollm.GetClient(cfg, false)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	response, err := client.Generate(context.Background(), "Hello, world!")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(response)
//
// For more control, you can also create providers directly:
//
//	// Direct provider instantiation
//	client, err := gemini.NewClient(context.Background(), "api-key", "", false)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// All providers implement the Client interface, ensuring consistent behavior across
// different LLM backends.
package xollm

import (
	"context"
)

// Client is the interface that all LLM provider clients must implement.
//
// This interface provides a unified way to interact with different LLM providers,
// abstracting away provider-specific implementation details while maintaining
// a consistent API surface.
//
// All methods should be safe for concurrent use unless otherwise specified.
type Client interface {
	// Generate takes a context and a prompt string and returns the LLM's response string.
	//
	// The context can be used for cancellation and timeout handling. Implementations
	// should respect context cancellation and return appropriate errors.
	//
	// The prompt should be a complete, well-formed prompt for the LLM. Different
	// providers may handle prompts differently (e.g., some expect chat format,
	// others expect raw text), but implementations should handle the conversion
	// internally.
	//
	// Returns the generated text response from the LLM, or an error if the
	// generation fails. Network errors, authentication failures, content
	// filtering, and other provider-specific issues should be wrapped in
	// descriptive error messages.
	Generate(ctx context.Context, prompt string) (string, error)

	// ProviderName returns the name of the LLM provider (e.g., "gemini", "ollama", "groq").
	//
	// This is useful for logging, debugging, and conditional behavior based on
	// the underlying provider. The returned name should be a lowercase, stable
	// identifier that matches the provider's configuration key.
	ProviderName() string
}
