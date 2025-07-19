package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/xostack/xollm"
	"github.com/xostack/xollm/config"
)

// CLIConfig holds command-line interface configuration options
type CLIConfig struct {
	ConfigFile     string
	Provider       string
	Prompt         string
	Timeout        int
	Debug          bool
	Interactive    bool
	CreateConfig   bool
	ListProviders  bool
	ValidateConfig bool
}

// loadConfigFromFile loads configuration from a TOML file
func loadConfigFromFile(configPath string) (*config.Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// saveConfigToFile saves configuration to a TOML file
func saveConfigToFile(cfg config.Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// findConfigFile determines the configuration file path to use
func findConfigFile(explicitPath string) string {
	if explicitPath != "" {
		return explicitPath
	}

	// Try common config locations
	configPaths := getConfigPaths()
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Return default if none found
	return "xollm.toml"
}

// getConfigPaths returns a list of potential config file locations
func getConfigPaths() []string {
	paths := []string{
		"xollm.toml",
		".xollm.toml",
	}

	// Add home directory path
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".xollm.toml"))
		paths = append(paths, filepath.Join(homeDir, ".config", "xollm.toml"))
	}

	return paths
}

// validateConfigForCLI validates configuration for CLI usage with detailed error messages
func validateConfigForCLI(cfg config.Config) error {
	if cfg.DefaultProvider == "" {
		return fmt.Errorf("no default provider specified in configuration")
	}

	if cfg.RequestTimeoutSeconds < 0 {
		return fmt.Errorf("invalid timeout: must be positive, got %d", cfg.RequestTimeoutSeconds)
	}

	if cfg.RequestTimeoutSeconds == 0 {
		cfg.RequestTimeoutSeconds = 60 // Set default
	}

	providerConfig, exists := cfg.LLMs[cfg.DefaultProvider]
	if !exists {
		return fmt.Errorf("provider '%s' not found in configuration", cfg.DefaultProvider)
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
	default:
		return fmt.Errorf("unsupported provider: %s", cfg.DefaultProvider)
	}

	return nil
}

// createDefaultConfig creates a default configuration with sample values
func createDefaultConfig() config.Config {
	return config.NewConfig("ollama", 60, map[string]config.LLMConfig{
		"ollama": {
			BaseURL: "http://localhost:11434",
			Model:   "gemma:2b",
		},
		"gemini": {
			APIKey: "your-gemini-api-key",
			Model:  "gemini-1.5-flash-latest",
		},
		"groq": {
			APIKey: "your-groq-api-key",
			Model:  "gemma:2b-8b-8192",
		},
	})
}

// listAvailableProviders returns a list of supported LLM providers
func listAvailableProviders() []string {
	return []string{"ollama", "gemini", "groq"}
}

// generateConfigTemplate generates a TOML configuration template with comments
func generateConfigTemplate() string {
	return `# XOStack xollm Configuration
# This file configures LLM providers and default settings

# Default provider to use when none is specified
default_provider = "ollama"

# Request timeout in seconds for all LLM calls
request_timeout_seconds = 60

# Ollama configuration (self-hosted)
[llms.ollama]
base_url = "http://localhost:11434"
model = "gemma:2b"

# Google Gemini configuration (cloud-based)
[llms.gemini]
api_key = "your-gemini-api-key"
model = "gemini-1.5-flash-latest"

# Groq configuration (cloud-based)
[llms.groq]
api_key = "your-groq-api-key"
model = "gemma:2b-8b-8192"

# Additional providers can be added here following the same pattern
# [llms.provider_name]
# api_key = "key"
# model = "model_name"
# base_url = "url"  # for self-hosted providers
`
}

// mergeConfigs merges two configurations, with override taking precedence
func mergeConfigs(base, override config.Config) config.Config {
	merged := config.Config{
		DefaultProvider:       base.DefaultProvider,
		RequestTimeoutSeconds: base.RequestTimeoutSeconds,
		LLMs:                  make(map[string]config.LLMConfig),
	}

	// Copy base LLM configs
	for name, cfg := range base.LLMs {
		merged.LLMs[name] = cfg
	}

	// Apply overrides
	if override.DefaultProvider != "" {
		merged.DefaultProvider = override.DefaultProvider
	}

	if override.RequestTimeoutSeconds > 0 {
		merged.RequestTimeoutSeconds = override.RequestTimeoutSeconds
	}

	// Override LLM configs
	for name, cfg := range override.LLMs {
		merged.LLMs[name] = cfg
	}

	return merged
}

// initializeConfigInteractive guides the user through creating a configuration file
func initializeConfigInteractive(configPath string) error {
	fmt.Printf("Creating new xollm configuration at: %s\n\n", configPath)

	scanner := bufio.NewScanner(os.Stdin)

	// Get default provider
	fmt.Print("Select default LLM provider (ollama/gemini/groq) [ollama]: ")
	scanner.Scan()
	defaultProvider := strings.TrimSpace(scanner.Text())
	if defaultProvider == "" {
		defaultProvider = "ollama"
	}

	// Validate provider choice
	validProviders := map[string]bool{"ollama": true, "gemini": true, "groq": true}
	if !validProviders[defaultProvider] {
		return fmt.Errorf("invalid provider: %s", defaultProvider)
	}

	// Get timeout
	fmt.Print("Request timeout in seconds [60]: ")
	scanner.Scan()
	timeoutStr := strings.TrimSpace(scanner.Text())
	timeout := 60
	if timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
			timeout = t
		}
	}

	// Create configuration
	cfg := config.Config{
		DefaultProvider:       defaultProvider,
		RequestTimeoutSeconds: timeout,
		LLMs:                  make(map[string]config.LLMConfig),
	}

	// Configure selected provider
	switch defaultProvider {
	case "ollama":
		fmt.Print("Ollama base URL [http://localhost:11434]: ")
		scanner.Scan()
		baseURL := strings.TrimSpace(scanner.Text())
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}

		fmt.Print("Ollama model [gemma:2b]: ")
		scanner.Scan()
		model := strings.TrimSpace(scanner.Text())
		if model == "" {
			model = "gemma:2b"
		}

		cfg.LLMs["ollama"] = config.LLMConfig{
			BaseURL: baseURL,
			Model:   model,
		}

	case "gemini":
		fmt.Print("Gemini API key: ")
		scanner.Scan()
		apiKey := strings.TrimSpace(scanner.Text())

		fmt.Print("Gemini model [gemini-1.5-flash-latest]: ")
		scanner.Scan()
		model := strings.TrimSpace(scanner.Text())
		if model == "" {
			model = "gemini-1.5-flash-latest"
		}

		cfg.LLMs["gemini"] = config.LLMConfig{
			APIKey: apiKey,
			Model:  model,
		}

	case "groq":
		fmt.Print("Groq API key: ")
		scanner.Scan()
		apiKey := strings.TrimSpace(scanner.Text())

		fmt.Print("Groq model [gemma:2b-8b-8192]: ")
		scanner.Scan()
		model := strings.TrimSpace(scanner.Text())
		if model == "" {
			model = "gemma:2b-8b-8192"
		}

		cfg.LLMs["groq"] = config.LLMConfig{
			APIKey: apiKey,
			Model:  model,
		}
	}

	// Save configuration
	if err := saveConfigToFile(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\nConfiguration saved successfully!\n")
	fmt.Printf("You can edit %s to add more providers or modify settings.\n", configPath)

	return nil
}

// runCLICommand executes the main CLI functionality based on parsed options
func runCLICommand(opts CLIConfig) error {
	// Handle special commands first
	if opts.ListProviders {
		providers := listAvailableProviders()
		fmt.Println("Available LLM providers:")
		for _, provider := range providers {
			fmt.Printf("  - %s\n", provider)
		}
		return nil
	}

	if opts.CreateConfig {
		configPath := findConfigFile(opts.ConfigFile)
		if opts.Interactive {
			return initializeConfigInteractive(configPath)
		} else {
			// Create default config
			cfg := createDefaultConfig()
			if err := saveConfigToFile(cfg, configPath); err != nil {
				return fmt.Errorf("failed to create config: %w", err)
			}
			fmt.Printf("Default configuration created at: %s\n", configPath)
			fmt.Println("Edit the file to customize your settings.")
			return nil
		}
	}

	// Load configuration
	configPath := findConfigFile(opts.ConfigFile)
	cfg, err := loadConfigFromFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Config file not found: %s\n", configPath)
			fmt.Println("Run with -create-config to create a new configuration file.")
			return err
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override provider if specified
	if opts.Provider != "" {
		cfg.DefaultProvider = opts.Provider
	}

	// Override timeout if specified
	if opts.Timeout > 0 {
		cfg.RequestTimeoutSeconds = opts.Timeout
	}

	if opts.ValidateConfig {
		if err := validateConfigForCLI(*cfg); err != nil {
			fmt.Printf("Configuration validation failed: %v\n", err)
			return err
		}
		fmt.Println("Configuration is valid!")
		return nil
	}

	// Validate configuration
	if err := validateConfigForCLI(*cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create client
	client, err := xollm.GetClient(*cfg, opts.Debug)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.RequestTimeoutSeconds)*time.Second)
	defer cancel()

	// Generate response
	fmt.Printf("Using provider: %s\n", cfg.DefaultProvider)
	fmt.Printf("Prompt: %s\n\n", opts.Prompt)

	start := time.Now()
	response, err := client.Generate(ctx, opts.Prompt)
	duration := time.Since(start)

	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	fmt.Printf("Response (%dms):\n%s\n", duration.Milliseconds(), response)

	if opts.Debug {
		fmt.Printf("\nDebug Information:\n")
		fmt.Printf("Config file: %s\n", configPath)
		fmt.Printf("Provider: %s\n", client.ProviderName())
		fmt.Printf("Timeout: %ds\n", cfg.RequestTimeoutSeconds)
		fmt.Printf("Response time: %dms\n", duration.Milliseconds())
	}

	return nil
}

// parseFlags parses command line flags and returns CLI configuration
func parseFlags() CLIConfig {
	var opts CLIConfig

	flag.StringVar(&opts.ConfigFile, "config", "", "Path to configuration file")
	flag.StringVar(&opts.Provider, "provider", "", "Override default LLM provider")
	flag.StringVar(&opts.Prompt, "prompt", "Hello, world! Please introduce yourself.", "Prompt to send to the LLM")
	flag.IntVar(&opts.Timeout, "timeout", 0, "Override request timeout in seconds")
	flag.BoolVar(&opts.Debug, "debug", false, "Enable debug output")
	flag.BoolVar(&opts.Interactive, "interactive", false, "Use interactive configuration setup")
	flag.BoolVar(&opts.CreateConfig, "create-config", false, "Create a new configuration file")
	flag.BoolVar(&opts.ListProviders, "list-providers", false, "List available LLM providers")
	flag.BoolVar(&opts.ValidateConfig, "validate-config", false, "Validate configuration file")

	flag.Parse()

	return opts
}

func main() {
	opts := parseFlags()

	if err := runCLICommand(opts); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
