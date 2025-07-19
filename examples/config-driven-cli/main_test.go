package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xostack/xollm/config"
)

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.toml")

	configContent := `
default_provider = "ollama"
request_timeout_seconds = 45

[llms.ollama]
base_url = "http://localhost:11434"
model = "gemma:2b"

[llms.gemini]
api_key = "test-gemini-key"
model = "gemini-1.5-pro"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.DefaultProvider != "ollama" {
		t.Errorf("Expected default provider 'ollama', got '%s'", cfg.DefaultProvider)
	}

	if cfg.RequestTimeoutSeconds != 45 {
		t.Errorf("Expected timeout 45, got %d", cfg.RequestTimeoutSeconds)
	}

	// Check ollama config
	ollamaConfig, exists := cfg.LLMs["ollama"]
	if !exists {
		t.Fatal("Expected ollama configuration")
	}

	if ollamaConfig.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected ollama base URL 'http://localhost:11434', got '%s'", ollamaConfig.BaseURL)
	}

	if ollamaConfig.Model != "gemma:2b" {
		t.Errorf("Expected ollama model 'gemma:2b', got '%s'", ollamaConfig.Model)
	}

	// Check gemini config
	geminiConfig, exists := cfg.LLMs["gemini"]
	if !exists {
		t.Fatal("Expected gemini configuration")
	}

	if geminiConfig.APIKey != "test-gemini-key" {
		t.Errorf("Expected gemini API key 'test-gemini-key', got '%s'", geminiConfig.APIKey)
	}

	if geminiConfig.Model != "gemini-1.5-pro" {
		t.Errorf("Expected gemini model 'gemini-1.5-pro', got '%s'", geminiConfig.Model)
	}
}

func TestLoadConfigFromFileNotFound(t *testing.T) {
	cfg, err := loadConfigFromFile("/nonexistent/config.toml")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}

	if cfg != nil {
		t.Error("Expected nil config when file not found")
	}
}

func TestLoadConfigFromFileInvalidTOML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.toml")

	invalidContent := `
default_provider = "ollama"
[llms.ollama
base_url = "invalid toml
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := loadConfigFromFile(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid TOML")
	}

	if cfg != nil {
		t.Error("Expected nil config when TOML is invalid")
	}
}

func TestSaveConfigToFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "saved-config.toml")

	cfg := config.NewConfig("gemini", 30, map[string]config.LLMConfig{
		"gemini": {
			APIKey: "test-key",
			Model:  "gemma-3-27b-it",
		},
		"ollama": {
			BaseURL: "http://localhost:11434",
			Model:   "gemma:2b",
		},
	})

	err := saveConfigToFile(cfg, configPath)
	if err != nil {
		t.Fatalf("Expected no error saving config, got: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Expected config file to be created")
	}

	// Reload and verify
	reloadedCfg, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Expected no error reloading config, got: %v", err)
	}

	if reloadedCfg.DefaultProvider != cfg.DefaultProvider {
		t.Errorf("Expected default provider '%s', got '%s'", cfg.DefaultProvider, reloadedCfg.DefaultProvider)
	}

	if reloadedCfg.RequestTimeoutSeconds != cfg.RequestTimeoutSeconds {
		t.Errorf("Expected timeout %d, got %d", cfg.RequestTimeoutSeconds, reloadedCfg.RequestTimeoutSeconds)
	}

	if len(reloadedCfg.LLMs) != len(cfg.LLMs) {
		t.Errorf("Expected %d LLM configs, got %d", len(cfg.LLMs), len(reloadedCfg.LLMs))
	}
}

func TestFindConfigFile(t *testing.T) {
	// Test with explicit path
	explicitPath := "/path/to/config.toml"
	foundPath := findConfigFile(explicitPath)
	if foundPath != explicitPath {
		t.Errorf("Expected explicit path '%s', got '%s'", explicitPath, foundPath)
	}

	// Test with empty path (should return default)
	defaultPath := findConfigFile("")
	expectedDefault := "xollm.toml"
	if !strings.HasSuffix(defaultPath, expectedDefault) {
		t.Errorf("Expected default path to end with '%s', got '%s'", expectedDefault, defaultPath)
	}
}

func TestValidateConfigForCLI(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: config.NewConfig("ollama", 30, map[string]config.LLMConfig{
				"ollama": {BaseURL: "http://localhost:11434", Model: "gemma:2b"},
			}),
			expectError: false,
		},
		{
			name:        "empty config",
			config:      config.Config{},
			expectError: true,
			errorMsg:    "no default provider specified",
		},
		{
			name: "missing provider config",
			config: config.Config{
				DefaultProvider: "nonexistent",
				LLMs:            map[string]config.LLMConfig{},
			},
			expectError: true,
			errorMsg:    "provider 'nonexistent' not found in configuration",
		},
		{
			name: "invalid timeout",
			config: config.Config{
				DefaultProvider:       "ollama",
				RequestTimeoutSeconds: -1,
				LLMs: map[string]config.LLMConfig{
					"ollama": {BaseURL: "http://localhost:11434"},
				},
			},
			expectError: true,
			errorMsg:    "invalid timeout",
		},
		{
			name: "missing API key for cloud provider",
			config: config.Config{
				DefaultProvider:       "gemini",
				RequestTimeoutSeconds: 30,
				LLMs: map[string]config.LLMConfig{
					"gemini": {Model: "gemma-3-27b-it"}, // Missing API key
				},
			},
			expectError: true,
			errorMsg:    "API key required",
		},
		{
			name: "missing base URL for Ollama",
			config: config.Config{
				DefaultProvider:       "ollama",
				RequestTimeoutSeconds: 30,
				LLMs: map[string]config.LLMConfig{
					"ollama": {Model: "gemma:2b"}, // Missing base URL
				},
			},
			expectError: true,
			errorMsg:    "base URL required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigForCLI(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()

	if cfg.DefaultProvider == "" {
		t.Error("Expected default provider to be set")
	}

	if cfg.RequestTimeoutSeconds <= 0 {
		t.Error("Expected positive timeout")
	}

	if len(cfg.LLMs) == 0 {
		t.Error("Expected at least one LLM configuration")
	}

	// Validate the created config
	err := validateConfigForCLI(cfg)
	if err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestListAvailableProviders(t *testing.T) {
	providers := listAvailableProviders()

	expectedProviders := []string{"ollama", "gemini", "groq"}
	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	for _, expected := range expectedProviders {
		found := false
		for _, provider := range providers {
			if provider == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find provider '%s' in list", expected)
		}
	}
}

func TestGenerateConfigTemplate(t *testing.T) {
	template := generateConfigTemplate()

	// Should contain TOML structure
	if !strings.Contains(template, "default_provider") {
		t.Error("Expected template to contain 'default_provider'")
	}

	if !strings.Contains(template, "request_timeout_seconds") {
		t.Error("Expected template to contain 'request_timeout_seconds'")
	}

	if !strings.Contains(template, "[llms.") {
		t.Error("Expected template to contain LLM section headers")
	}

	// Should contain all supported providers
	expectedProviders := []string{"ollama", "gemini", "groq"}
	for _, provider := range expectedProviders {
		sectionHeader := "[llms." + provider + "]"
		if !strings.Contains(template, sectionHeader) {
			t.Errorf("Expected template to contain section '%s'", sectionHeader)
		}
	}
}

func TestMergeConfigs(t *testing.T) {
	base := config.Config{
		DefaultProvider:       "ollama",
		RequestTimeoutSeconds: 30,
		LLMs: map[string]config.LLMConfig{
			"ollama": {BaseURL: "http://localhost:11434", Model: "gemma:2b"},
		},
	}

	override := config.Config{
		DefaultProvider:       "gemini",
		RequestTimeoutSeconds: 60,
		LLMs: map[string]config.LLMConfig{
			"gemini": {APIKey: "test-key", Model: "gemma-3-27b-it"},
		},
	}

	merged := mergeConfigs(base, override)

	// Override should take precedence
	if merged.DefaultProvider != "gemini" {
		t.Errorf("Expected merged default provider 'gemini', got '%s'", merged.DefaultProvider)
	}

	if merged.RequestTimeoutSeconds != 60 {
		t.Errorf("Expected merged timeout 60, got %d", merged.RequestTimeoutSeconds)
	}

	// Should have both LLM configs
	if len(merged.LLMs) != 2 {
		t.Errorf("Expected 2 LLM configs in merged result, got %d", len(merged.LLMs))
	}

	if _, exists := merged.LLMs["ollama"]; !exists {
		t.Error("Expected ollama config to be preserved in merge")
	}

	if _, exists := merged.LLMs["gemini"]; !exists {
		t.Error("Expected gemini config to be added in merge")
	}
}

func TestGetConfigPaths(t *testing.T) {
	paths := getConfigPaths()

	// Should return at least one path
	if len(paths) == 0 {
		t.Error("Expected at least one config path")
	}

	// All paths should be absolute or relative valid paths
	for _, path := range paths {
		if path == "" {
			t.Error("Config paths should not be empty")
		}
	}
}

func TestInitializeConfigInteractive(t *testing.T) {
	// Test with non-interactive mode (empty inputs)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "new-config.toml")

	cfg := createDefaultConfig()
	err := saveConfigToFile(cfg, configPath)
	if err != nil {
		t.Fatalf("Failed to save default config: %v", err)
	}

	// Verify the file was created and is valid
	savedCfg, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	err = validateConfigForCLI(*savedCfg)
	if err != nil {
		t.Errorf("Saved config should be valid: %v", err)
	}
}
