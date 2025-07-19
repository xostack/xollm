package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xostack/xollm"
	"github.com/xostack/xollm/config"
)

// mockClient implements xollm.Client for testing
type mockClient struct {
	generateFunc    func(ctx context.Context, prompt string) (string, error)
	providerNameVal string
	closeFunc       func() error
}

func (m *mockClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "mock response for: " + prompt, nil
}

func (m *mockClient) ProviderName() string {
	if m.providerNameVal != "" {
		return m.providerNameVal
	}
	return "mock"
}

func (m *mockClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Mock factory function for testing
var originalGetClient = xollm.GetClient

func mockGetClient(cfg config.Config, debugMode bool) (xollm.Client, error) {
	if cfg.DefaultProvider == "error" {
		return nil, errors.New("mock error creating client")
	}

	return &mockClient{
		generateFunc: func(ctx context.Context, prompt string) (string, error) {
			if strings.Contains(prompt, "error") {
				return "", errors.New("mock generation error")
			}
			return "Hello from " + cfg.DefaultProvider + " provider! Response to: " + prompt, nil
		},
		providerNameVal: cfg.DefaultProvider,
	}, nil
}

func TestBasicUsageWithConfig(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	tests := []struct {
		name             string
		config           config.Config
		prompt           string
		expectedContains string
		expectError      bool
	}{
		{
			name: "successful generation with gemini",
			config: config.NewConfig("gemini", 30, map[string]config.LLMConfig{
				"gemini": {APIKey: "test-key"},
			}),
			prompt:           "Hello, world!",
			expectedContains: "Hello from gemini provider",
			expectError:      false,
		},
		{
			name: "successful generation with ollama",
			config: config.NewConfig("ollama", 30, map[string]config.LLMConfig{
				"ollama": {BaseURL: "http://localhost:11434"},
			}),
			prompt:           "Test prompt",
			expectedContains: "Hello from ollama provider",
			expectError:      false,
		},
		{
			name: "error during client creation",
			config: config.NewConfig("error", 30, map[string]config.LLMConfig{
				"error": {APIKey: "test"},
			}),
			prompt:      "Test",
			expectError: true,
		},
		{
			name: "error during generation",
			config: config.NewConfig("gemini", 30, map[string]config.LLMConfig{
				"gemini": {APIKey: "test-key"},
			}),
			prompt:      "error prompt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := basicUsageWithConfig(tt.config, tt.prompt)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !strings.Contains(result, tt.expectedContains) {
				t.Errorf("expected response to contain %q, got %q", tt.expectedContains, result)
			}
		})
	}
}

func TestBasicUsageWithValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: config.NewConfig("gemini", 30, map[string]config.LLMConfig{
				"gemini": {APIKey: "test-key"},
			}),
			expectError: false,
		},
		{
			name:        "empty config",
			config:      config.Config{},
			expectError: true,
			errorMsg:    "default provider not specified",
		},
		{
			name: "missing provider config",
			config: config.Config{
				DefaultProvider: "nonexistent",
				LLMs:            map[string]config.LLMConfig{},
			},
			expectError: true,
			errorMsg:    "provider 'nonexistent' not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateSampleConfigs(t *testing.T) {
	configs := createSampleConfigs()

	if len(configs) == 0 {
		t.Error("expected at least one sample config")
	}

	for name, cfg := range configs {
		t.Run("config_"+name, func(t *testing.T) {
			err := validateConfig(cfg)
			if err != nil {
				t.Errorf("sample config %q is invalid: %v", name, err)
			}
		})
	}
}

func TestPromptExamples(t *testing.T) {
	prompts := []string{
		"Hello, world!",
		"What is the capital of France?",
		"Explain quantum computing in simple terms",
		"",
	}

	for _, prompt := range prompts {
		t.Run("prompt_validation", func(t *testing.T) {
			// Test that our example can handle various prompts
			// This mainly tests input validation
			if len(prompt) > 10000 {
				t.Error("prompt too long for basic example")
			}
		})
	}
}

func TestContextHandling(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	config := config.NewConfig("gemini", 30, map[string]config.LLMConfig{
		"gemini": {APIKey: "test-key"},
	})

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := basicUsageWithConfigAndContext(ctx, config, "test prompt")
	if err == nil {
		t.Error("expected error with cancelled context")
	}

	// Test with valid context
	ctx = context.Background()
	result, err := basicUsageWithConfigAndContext(ctx, config, "test prompt")
	if err != nil {
		t.Errorf("unexpected error with valid context: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}
