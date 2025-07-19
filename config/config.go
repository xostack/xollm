// Package config handles loading and managing xollm configuration.
//
// This package provides both file-based and programmatic configuration
// management for xollm. It supports TOML configuration files following
// XDG Base Directory specifications, as well as programmatic configuration
// creation for library usage.
//
// Configuration supports multiple LLM providers with provider-specific
// settings such as API keys, base URLs, and model overrides.
//
// Example TOML configuration:
//
//	default_provider = "ollama"
//	request_timeout_seconds = 60
//
//	[llms.ollama]
//	base_url = "http://localhost:11434"
//	model = "gemma:2b"
//
//	[llms.gemini]
//	api_key = "your-gemini-api-key"
//	model = "gemini-1.5-flash-latest"
//
// Example programmatic usage:
//
//	cfg := config.NewConfig("gemini", 30, map[string]config.LLMConfig{
//		"gemini": {APIKey: "key", Model: "gemini-pro"},
//	})
package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	appName         = "xollm"
	configFileName  = "config.toml"
	DefaultDirPerm  = 0750 // rwxr-x--- // EXPORTED
	DefaultFilePerm = 0600 // rw------- (Contains potential secrets) // EXPORTED
)

// Config holds the application's configuration.
//
// This structure defines the complete configuration for xollm, including
// the default provider selection, timeout settings, and provider-specific
// configurations.
type Config struct {
	// DefaultProvider specifies which LLM provider to use by default.
	// Must match a key in the LLMs map. Common values: "gemini", "groq", "ollama".
	DefaultProvider string `toml:"default_provider"`

	// RequestTimeoutSeconds sets the timeout for LLM API requests in seconds.
	// If <= 0, a default timeout of 60 seconds will be used.
	RequestTimeoutSeconds int `toml:"request_timeout_seconds"`

	// LLMs contains provider-specific configurations keyed by provider name.
	// Each provider may have different required fields (e.g., APIKey vs BaseURL).
	LLMs map[string]LLMConfig `toml:"llms"`
}

// LLMConfig holds configuration specific to an LLM provider.
//
// Different providers require different fields:
//   - Gemini/Groq: Require APIKey
//   - Ollama: Requires BaseURL
//   - All providers: Support optional Model override
//
// Use pointers to distinguish between unset and explicitly empty values if needed,
// but simple strings are often sufficient for TOML loading.
type LLMConfig struct {
	// BaseURL is the base URL for the LLM API (used by Ollama).
	// Should include protocol (http/https) and port if non-standard.
	// Example: "http://localhost:11434"
	BaseURL string `toml:"base_url,omitempty"`

	// APIKey is the authentication key for cloud-based providers (Gemini, Groq).
	// This field contains sensitive information and should be handled securely.
	APIKey string `toml:"api_key,omitempty"`

	// Model is an optional model name override for the provider.
	// If empty, the provider's default model will be used.
	// Example: "gemini-1.5-pro", "gemma:2b", "mixtral-8x7b-32768"
	Model string `toml:"model,omitempty"`
}

// Default configuration values.
func defaultConfig() Config {
	return Config{
		DefaultProvider:       "ollama", // Default to Ollama
		RequestTimeoutSeconds: 60,       // 60-second timeout for LLM requests
		LLMs: map[string]LLMConfig{
			"ollama": {
				BaseURL: "http://localhost:11434", // Default Ollama URL
			},
			"gemini": {
				APIKey: "", // Requires user input
			},
			"groq": {
				APIKey: "", // Requires user input
			},
			// Add other providers here with their default fields
		},
	}
}

// GetConfigFilePath determines the appropriate configuration file path based on XDG specs.
//
// This function follows the XDG Base Directory Specification for determining
// where configuration files should be stored:
//   - If XDG_CONFIG_HOME is set, uses $XDG_CONFIG_HOME/xollm/config.toml
//   - Otherwise, uses $HOME/.config/xollm/config.toml
//
// Returns the full path to the configuration file, or an error if the home
// directory cannot be determined.
//
// The returned path may not exist - use os.Stat to check for existence.
func GetConfigFilePath() (string, error) { // EXPORTED and RENAMED
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine user home directory: %w", err)
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configHome, appName, configFileName), nil
}

// Load reads the configuration file, creates it interactively if missing,
// merges with defaults, and returns the final Config.
func Load(debugMode bool) (Config, error) { // MODIFIED: Added debugMode
	cfgPath, err := GetConfigFilePath()
	if err != nil {
		return Config{}, fmt.Errorf("failed to determine config path: %w", err)
	}

	// Start with default config
	cfg := defaultConfig()

	_, err = os.Stat(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Config file doesn't exist, ask to create
			if debugMode {
				fmt.Printf("Configuration file not found at %s\n", cfgPath)
			}
			if askToCreateConfigFile() {
				err = createConfigFileInteractive(cfgPath, &cfg, debugMode) // MODIFIED: Pass debugMode
				if err != nil {
					return Config{}, fmt.Errorf("failed to create configuration file: %w", err)
				}
				// File created, proceed to load (or just use the interactively filled cfg)
				if debugMode {
					fmt.Printf("Configuration file created successfully at %s\n", cfgPath)
				}
				// No need to reload here, createConfigFileInteractive populates cfg
			} else {
				return Config{}, fmt.Errorf("configuration file creation declined by user.\n\nTo create a configuration file later, use the xollm config API or manually create the config file")
			}
		} else {
			// Other error accessing the file (e.g., permissions)
			return Config{}, fmt.Errorf("failed to access config file %s: %w", cfgPath, err)
		}
	} else {
		// File exists, load it and merge over defaults
		if debugMode {
			fmt.Printf("Loading configuration from %s\n", cfgPath) // MODIFIED: Conditional print
		}
		meta, err := toml.DecodeFile(cfgPath, &cfg)
		if err != nil {
			return Config{}, fmt.Errorf("failed to decode TOML config file %s: %w", cfgPath, err)
		}
		// Optional: Check for undecoded keys if strictness is desired
		if len(meta.Undecoded()) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: Unknown configuration keys found in %s: %v\n", cfgPath, meta.Undecoded())
		}
	}

	// Final validation (e.g., ensure default provider is configured)
	if _, exists := cfg.LLMs[cfg.DefaultProvider]; !exists {
		return Config{}, fmt.Errorf("default provider '%s' is specified but has no configuration section in [llms]", cfg.DefaultProvider)
	}
	// Add more validation as needed

	return cfg, nil
}

// askToCreateConfigFile prompts the user if they want to create the config file.
func askToCreateConfigFile() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Configuration file not found. xollm requires at least one LLM provider to be configured.\n")
	fmt.Printf("Available providers: Ollama (local), Gemini (cloud), Groq (cloud)\n")
	fmt.Print("Do you want to create it now? (y/N): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// createConfigFileInteractive guides the user through setting up the initial config.
func createConfigFileInteractive(cfgPath string, cfg *Config, debugMode bool) error { // MODIFIED: Added debugMode
	reader := bufio.NewReader(os.Stdin)
	configuredProvider := false

	fmt.Println("\n--- Initial Configuration ---")
	fmt.Println("Please provide details for at least one LLM provider.")

	// --- Ollama ---
	fmt.Printf("Enter Ollama Base URL (leave empty to skip, default: %s): ", cfg.LLMs["ollama"].BaseURL)
	ollamaURLInput, _ := reader.ReadString('\n')
	ollamaURLInput = strings.TrimSpace(ollamaURLInput)
	if ollamaURLInput != "" {
		if err := validateOllamaURL(ollamaURLInput, debugMode); err != nil {
			fmt.Printf("⚠️  Warning: Could not connect to Ollama at %s: %v\n", ollamaURLInput, err)
			fmt.Printf("   The configuration will be saved anyway. Make sure Ollama is running.\n")
		} else {
			fmt.Printf("✅ Successfully connected to Ollama at %s\n", ollamaURLInput)
		}
		cfg.LLMs["ollama"] = LLMConfig{BaseURL: ollamaURLInput} // Update map entry
		configuredProvider = true
	} else {
		// User skipped, keep default or remove if default is empty
		if cfg.LLMs["ollama"].BaseURL == "" {
			delete(cfg.LLMs, "ollama")
		} else {
			// Keep default URL if user just hits enter
			fmt.Printf("Using default Ollama URL: %s\n", cfg.LLMs["ollama"].BaseURL)
			configuredProvider = true // Default counts
		}
	}

	// --- Gemini ---
	fmt.Print("Enter Gemini API Key (leave empty to skip): ")
	geminiKeyInput, _ := reader.ReadString('\n')
	geminiKeyInput = strings.TrimSpace(geminiKeyInput)
	if geminiKeyInput != "" {
		// Basic validation: non-empty
		cfg.LLMs["gemini"] = LLMConfig{APIKey: geminiKeyInput}
		fmt.Printf("✅ Gemini API key configured\n")
		configuredProvider = true
	} else {
		delete(cfg.LLMs, "gemini") // Remove if skipped
	}

	// --- Groq ---
	fmt.Print("Enter Groq API Key (leave empty to skip): ")
	groqKeyInput, _ := reader.ReadString('\n')
	groqKeyInput = strings.TrimSpace(groqKeyInput)
	if groqKeyInput != "" {
		cfg.LLMs["groq"] = LLMConfig{APIKey: groqKeyInput}
		fmt.Printf("✅ Groq API key configured\n")
		configuredProvider = true
	} else {
		delete(cfg.LLMs, "groq") // Remove if skipped
	}

	// --- Check if at least one provider is configured ---
	if !configuredProvider {
		fmt.Printf("\n❌ No LLM providers configured.\n")
		fmt.Printf("You need at least one provider to use xollm.\n")
		fmt.Printf("Please configure at least one provider in your xollm config.\n")
		return errors.New("at least one LLM provider must be configured")
	}

	// --- Default Provider ---
	availableProviders := make([]string, 0, len(cfg.LLMs))
	for provider := range cfg.LLMs {
		availableProviders = append(availableProviders, provider)
	}

	fmt.Printf("Enter default LLM provider (available: %s; default: %s): ", strings.Join(availableProviders, ", "), cfg.DefaultProvider)
	defaultProviderInput, _ := reader.ReadString('\n')
	defaultProviderInput = strings.TrimSpace(defaultProviderInput)
	if defaultProviderInput != "" {
		if _, exists := cfg.LLMs[defaultProviderInput]; !exists {
			return fmt.Errorf("invalid default provider '%s': no configuration found for this provider", defaultProviderInput)
		}
		cfg.DefaultProvider = defaultProviderInput
		fmt.Printf("✅ Default provider set to: %s\n", defaultProviderInput)
	} else if _, exists := cfg.LLMs[cfg.DefaultProvider]; !exists {
		// If user skipped and the original default isn't configured anymore, pick the first available one
		for provider := range cfg.LLMs {
			cfg.DefaultProvider = provider
			fmt.Printf("Default provider '%s' not configured, setting default to '%s'.\n", defaultConfig().DefaultProvider, cfg.DefaultProvider)
			break
		}
	} else {
		fmt.Printf("✅ Using default provider: %s\n", cfg.DefaultProvider)
	}

	// --- Create Directory ---
	configDir := filepath.Dir(cfgPath)
	err := os.MkdirAll(configDir, DefaultDirPerm)
	if err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// --- Write File ---
	file, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultFilePerm)
	if err != nil {
		return fmt.Errorf("failed to create config file %s: %w", cfgPath, err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	// Optional: Indent nested tables for better readability
	// encoder.Indent = "  " // Uncomment if desired

	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode configuration to TOML: %w", err)
	}

	fmt.Printf("\n✅ Configuration file created successfully at %s\n", cfgPath)
	fmt.Printf("Configuration saved successfully!\n")
	fmt.Printf("You can now use xollm library in your applications.\n")
	fmt.Printf("\nTo modify configuration later, use the xollm config API.\n")

	return nil // Success
}

// validateOllamaURL attempts to connect to the Ollama base URL.
func validateOllamaURL(rawURL string, debugMode bool) error { // MODIFIED: Added debugMode
	if rawURL == "" {
		return errors.New("URL cannot be empty")
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL scheme must be http or https")
	}

	// Simple check: try to make a request to the base path.
	// Ollama usually responds at the root, even if it's just "Ollama is running".
	// A more robust check might target a specific health endpoint if available (e.g., /api/tags or /api/health)
	client := &http.Client{
		Timeout: 5 * time.Second, // Short timeout for validation
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err) // Should be rare
	}

	resp, err := client.Do(req)
	if err != nil {
		// Only print warning if in debug mode, otherwise fail more silently for interactive setup
		if debugMode {
			fmt.Fprintf(os.Stderr, "Warning: Ollama URL validation failed for %s: %v.\n", rawURL, err)
		}
		return fmt.Errorf("failed to connect to Ollama server at %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	// Allow various success codes, maybe even 404 if the base path doesn't serve anything specific
	// but the connection worked. The main goal is reachability.
	// if resp.StatusCode < 200 || resp.StatusCode >= 400 {
	// return fmt.Errorf("server responded with status %s", resp.Status)
	// }
	// For now, just succeeding the connection is good enough validation.
	if debugMode {
		fmt.Printf("Successfully connected to Ollama at %s (Status: %s)\n", rawURL, resp.Status) // MODIFIED: Conditional print
	}
	return nil
}

// GetLLMConfig retrieves the specific configuration for a given provider.
func (c *Config) GetLLMConfig(provider string) (LLMConfig, bool) {
	llmCfg, exists := c.LLMs[provider]
	return llmCfg, exists
}

// NewConfig creates a new configuration programmatically (library-friendly approach).
//
// This function allows applications to create configurations without file I/O
// or user interaction, making it suitable for library usage and testing.
//
// Parameters:
//   - defaultProvider: The provider to use by default (must exist in providers map)
//   - timeoutSeconds: Request timeout in seconds (must be > 0)
//   - providers: Map of provider name to provider configuration
//
// Returns a complete Config struct ready for use with GetClient.
//
// Example:
//
//	cfg := NewConfig("gemini", 30, map[string]LLMConfig{
//		"gemini": {APIKey: "your-key", Model: "gemini-pro"},
//		"ollama": {BaseURL: "http://localhost:11434"},
//	})
func NewConfig(defaultProvider string, timeoutSeconds int, providers map[string]LLMConfig) Config {
	return Config{
		DefaultProvider:       defaultProvider,
		RequestTimeoutSeconds: timeoutSeconds,
		LLMs:                  providers,
	}
}

// LoadFromFile loads configuration from a specific file path.
//
// This is a library-friendly function that doesn't involve interactive prompts
// or automatic file creation. It loads the TOML configuration from the specified
// path and merges it with default values.
//
// The function performs validation to ensure:
//   - The file exists and is readable
//   - The TOML format is valid
//   - The default provider is configured
//
// Returns an error if the file doesn't exist, contains invalid TOML,
// or fails validation.
//
// For interactive configuration loading with automatic setup, use Load() instead.
func LoadFromFile(filePath string) (Config, error) {
	// Start with default config
	cfg := defaultConfig()

	// Check if file exists
	_, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("configuration file not found at %s", filePath)
		}
		return Config{}, fmt.Errorf("failed to access config file %s: %w", filePath, err)
	}

	// Load and merge with defaults
	meta, err := toml.DecodeFile(filePath, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("failed to decode TOML config file %s: %w", filePath, err)
	}

	// Optional: Check for undecoded keys if strictness is desired
	if len(meta.Undecoded()) > 0 {
		// Note: In library mode, we might want to be stricter about unknown keys
		// For now, we'll just ignore them but could return an error in strict mode
	}

	// Final validation (e.g., ensure default provider is configured)
	if _, exists := cfg.LLMs[cfg.DefaultProvider]; !exists {
		return Config{}, fmt.Errorf("default provider '%s' is specified but has no configuration section in [llms]", cfg.DefaultProvider)
	}

	return cfg, nil
}
