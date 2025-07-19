package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	// Test default provider
	if cfg.DefaultProvider != "ollama" {
		t.Errorf("Expected default provider 'ollama', got '%s'", cfg.DefaultProvider)
	}

	// Test default timeout
	if cfg.RequestTimeoutSeconds != 60 {
		t.Errorf("Expected default timeout 60, got %d", cfg.RequestTimeoutSeconds)
	}

	// Test that default providers are configured
	expectedProviders := []string{"ollama", "gemini", "groq"}
	for _, provider := range expectedProviders {
		if _, exists := cfg.LLMs[provider]; !exists {
			t.Errorf("Expected provider '%s' to be configured by default", provider)
		}
	}

	// Test ollama default URL
	if cfg.LLMs["ollama"].BaseURL != "http://localhost:11434" {
		t.Errorf("Expected ollama default URL 'http://localhost:11434', got '%s'", cfg.LLMs["ollama"].BaseURL)
	}
}

func TestConfig_GetLLMConfig(t *testing.T) {
	cfg := Config{
		LLMs: map[string]LLMConfig{
			"test-provider": {
				APIKey: "test-key",
				Model:  "test-model",
			},
		},
	}

	// Test existing provider
	llmCfg, exists := cfg.GetLLMConfig("test-provider")
	if !exists {
		t.Error("Expected provider to exist")
	}
	if llmCfg.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", llmCfg.APIKey)
	}
	if llmCfg.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", llmCfg.Model)
	}

	// Test non-existing provider
	_, exists = cfg.GetLLMConfig("non-existent")
	if exists {
		t.Error("Expected provider to not exist")
	}
}

func TestGetConfigFilePath_NoXDGConfigHome(t *testing.T) {
	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Remove XDG_CONFIG_HOME
	os.Unsetenv("XDG_CONFIG_HOME")

	path, err := GetConfigFilePath()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should use ~/.config/xollm/config.toml
	if !strings.Contains(path, ".config/xollm/config.toml") {
		t.Errorf("Expected path to contain '.config/xollm/config.toml', got '%s'", path)
	}
}

func TestGetConfigFilePath_WithXDGConfigHome(t *testing.T) {
	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Set custom XDG_CONFIG_HOME
	testDir := "/tmp/test-config"
	os.Setenv("XDG_CONFIG_HOME", testDir)

	path, err := GetConfigFilePath()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := filepath.Join(testDir, "xollm", "config.toml")
	if path != expected {
		t.Errorf("Expected path '%s', got '%s'", expected, path)
	}
}

func TestNewConfigFromMap(t *testing.T) {
	// Test programmatic configuration creation (library-friendly)
	providerConfigs := map[string]LLMConfig{
		"gemini": {
			APIKey: "test-gemini-key",
			Model:  "gemma-3-27b-it",
		},
		"ollama": {
			BaseURL: "http://localhost:11434",
			Model:   "gemma:2b",
		},
	}

	cfg := NewConfig("gemini", 30, providerConfigs)

	if cfg.DefaultProvider != "gemini" {
		t.Errorf("Expected default provider 'gemini', got '%s'", cfg.DefaultProvider)
	}

	if cfg.RequestTimeoutSeconds != 30 {
		t.Errorf("Expected timeout 30, got %d", cfg.RequestTimeoutSeconds)
	}

	if len(cfg.LLMs) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(cfg.LLMs))
	}

	geminiCfg, exists := cfg.GetLLMConfig("gemini")
	if !exists {
		t.Error("Expected gemini provider to exist")
	}
	if geminiCfg.APIKey != "test-gemini-key" {
		t.Errorf("Expected gemini API key 'test-gemini-key', got '%s'", geminiCfg.APIKey)
	}
}

func TestValidateOllamaURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		shouldErr bool
	}{
		{
			name:      "valid http URL",
			url:       "http://localhost:11434",
			shouldErr: false, // Note: this will fail in tests due to no actual server
		},
		{
			name:      "valid https URL",
			url:       "https://example.com:11434",
			shouldErr: false, // Note: this will fail in tests due to no actual server
		},
		{
			name:      "empty URL",
			url:       "",
			shouldErr: true,
		},
		{
			name:      "invalid scheme",
			url:       "ftp://localhost:11434",
			shouldErr: true,
		},
		{
			name:      "malformed URL",
			url:       "not-a-url",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOllamaURL(tt.url, false)
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				// For valid URLs, we expect network errors since no server is running
				// but not validation errors about scheme or format
				if strings.Contains(err.Error(), "scheme must be") ||
					strings.Contains(err.Error(), "invalid URL format") ||
					strings.Contains(err.Error(), "URL cannot be empty") {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestLoad_NonInteractive_MockMode(t *testing.T) {
	// Test loading configuration without interactive prompts (library mode)
	// This test directly uses LoadFromFile to avoid mocking global functions

	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	configContent := `default_provider = "ollama"
request_timeout_seconds = 45

[llms.ollama]
base_url = "http://localhost:11434"
model = "gemma:2b"

[llms.gemini]
api_key = "test-key"
model = "gemma-3-27b-it"
`

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.DefaultProvider != "ollama" {
		t.Errorf("Expected default provider 'ollama', got '%s'", cfg.DefaultProvider)
	}

	if cfg.RequestTimeoutSeconds != 45 {
		t.Errorf("Expected timeout 45, got %d", cfg.RequestTimeoutSeconds)
	}

	ollamaCfg, exists := cfg.GetLLMConfig("ollama")
	if !exists {
		t.Error("Expected ollama config to exist")
	}
	if ollamaCfg.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected ollama URL 'http://localhost:11434', got '%s'", ollamaCfg.BaseURL)
	}
}
