package xollm

import (
	"testing"

	"github.com/xostack/xollm/config"
)

func TestGetClient_Gemini(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "gemini",
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"gemini": {
				APIKey: "test-api-key",
				Model:  "gemma-3-27b-it",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.ProviderName() != "gemini" {
		t.Errorf("Expected provider name 'gemini', got '%s'", client.ProviderName())
	}
}

func TestGetClient_Ollama(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "ollama",
		RequestTimeoutSeconds: 45,
		LLMs: map[string]config.LLMConfig{
			"ollama": {
				BaseURL: "http://localhost:11434",
				Model:   "gemma:2b",
			},
		},
	}

	client, err := GetClient(cfg, true) // Test with debug mode
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.ProviderName() != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", client.ProviderName())
	}
}

func TestGetClient_Groq(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "groq",
		RequestTimeoutSeconds: 60,
		LLMs: map[string]config.LLMConfig{
			"groq": {
				APIKey: "test-groq-key",
				Model:  "gemma2-9b-it",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.ProviderName() != "groq" {
		t.Errorf("Expected provider name 'groq', got '%s'", client.ProviderName())
	}
}

func TestGetClient_MissingDefaultProvider(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "", // Empty default provider
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"gemini": {
				APIKey: "test-api-key",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err == nil {
		t.Fatal("Expected error for missing default provider")
	}

	if client != nil {
		t.Error("Expected client to be nil when error occurs")
	}

	expectedErrMsg := "no default LLM provider specified in configuration"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGetClient_ProviderNotConfigured(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "gemini",
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"ollama": { // Different provider configured
				BaseURL: "http://localhost:11434",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err == nil {
		t.Fatal("Expected error for unconfigured provider")
	}

	if client != nil {
		t.Error("Expected client to be nil when error occurs")
	}

	expectedErrMsg := "configuration for provider 'gemini' not found"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGetClient_MissingAPIKey(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "gemini",
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"gemini": {
				APIKey: "", // Empty API key
				Model:  "gemma-3-27b-it",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err == nil {
		t.Fatal("Expected error for missing API key")
	}

	if client != nil {
		t.Error("Expected client to be nil when error occurs")
	}

	expectedErrMsg := "API key for Gemini not found in configuration"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGetClient_MissingBaseURL(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "ollama",
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"ollama": {
				BaseURL: "", // Empty base URL
				Model:   "gemma:2b",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err == nil {
		t.Fatal("Expected error for missing base URL")
	}

	if client != nil {
		t.Error("Expected client to be nil when error occurs")
	}

	expectedErrMsg := "base URL for Ollama not found in configuration"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGetClient_UnsupportedProvider(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "unsupported-provider",
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"unsupported-provider": {
				APIKey: "test-key",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err == nil {
		t.Fatal("Expected error for unsupported provider")
	}

	if client != nil {
		t.Error("Expected client to be nil when error occurs")
	}

	expectedErrMsg := "unsupported LLM provider: unsupported-provider"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGetClient_DefaultTimeout(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "ollama",
		RequestTimeoutSeconds: 0, // Invalid timeout, should use default
		LLMs: map[string]config.LLMConfig{
			"ollama": {
				BaseURL: "http://localhost:11434",
				Model:   "gemma:2b",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// We can't directly test the timeout value since it's internal to the client,
	// but we can verify the client was created successfully
	if client.ProviderName() != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", client.ProviderName())
	}
}

func TestGetClient_WithCustomModels(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		config   config.LLMConfig
	}{
		{
			name:     "gemini with custom model",
			provider: "gemini",
			config: config.LLMConfig{
				APIKey: "test-key",
				Model:  "gemini-1.5-pro",
			},
		},
		{
			name:     "ollama with custom model",
			provider: "ollama",
			config: config.LLMConfig{
				BaseURL: "http://localhost:11434",
				Model:   "codellama",
			},
		},
		{
			name:     "groq with custom model",
			provider: "groq",
			config: config.LLMConfig{
				APIKey: "test-key",
				Model:  "mixtral-8x7b-32768",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				DefaultProvider:       tt.provider,
				RequestTimeoutSeconds: 30,
				LLMs: map[string]config.LLMConfig{
					tt.provider: tt.config,
				},
			}

			client, err := GetClient(cfg, false)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if client == nil {
				t.Fatal("Expected client to be non-nil")
			}

			if client.ProviderName() != tt.provider {
				t.Errorf("Expected provider name '%s', got '%s'", tt.provider, client.ProviderName())
			}
		})
	}
}

func TestClient_Close_Interface(t *testing.T) {
	// Test that all provider clients can be closed through the interface
	tests := []struct {
		name     string
		provider string
		config   config.LLMConfig
	}{
		{
			name:     "gemini client close",
			provider: "gemini",
			config: config.LLMConfig{
				APIKey: "test-api-key",
				Model:  "gemma-3-27b-it",
			},
		},
		{
			name:     "ollama client close",
			provider: "ollama",
			config: config.LLMConfig{
				BaseURL: "http://localhost:11434",
				Model:   "gemma:2b",
			},
		},
		{
			name:     "groq client close",
			provider: "groq",
			config: config.LLMConfig{
				APIKey: "test-groq-key",
				Model:  "gemma2-9b-it",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				DefaultProvider:       tt.provider,
				RequestTimeoutSeconds: 30,
				LLMs: map[string]config.LLMConfig{
					tt.provider: tt.config,
				},
			}

			client, err := GetClient(cfg, false)
			if err != nil {
				t.Fatalf("Expected no error creating client, got: %v", err)
			}

			if client == nil {
				t.Fatal("Expected client to be non-nil")
			}

			// Test that Close() method is available through the interface
			err = client.Close()
			if err != nil {
				t.Errorf("Expected Close() to succeed for %s provider, got error: %v", tt.provider, err)
			}
		})
	}
}

func TestClient_CloseIsIdempotent(t *testing.T) {
	// Test that calling Close() multiple times is safe
	cfg := config.Config{
		DefaultProvider:       "ollama",
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"ollama": {
				BaseURL: "http://localhost:11434",
				Model:   "gemma:2b",
			},
		},
	}

	client, err := GetClient(cfg, false)
	if err != nil {
		t.Fatalf("Expected no error creating client, got: %v", err)
	}

	// First close
	err = client.Close()
	if err != nil {
		t.Errorf("Expected first Close() to succeed, got error: %v", err)
	}

	// Second close should also be safe
	err = client.Close()
	if err != nil {
		t.Errorf("Expected second Close() to succeed (idempotent), got error: %v", err)
	}
}
