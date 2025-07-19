package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xostack/xollm"
	"github.com/xostack/xollm/config"
)

// ProviderResult holds the result of generating text from a single provider.
type ProviderResult struct {
	Provider string        // Name of the provider (e.g., "ollama", "gemini")
	Response string        // Generated response text
	Duration time.Duration // Time taken to generate the response
	Error    error         // Error encountered during generation, if any
}

// ResultAnalysis contains summary statistics and analysis of provider comparison results.
type ResultAnalysis struct {
	TotalProviders      int           // Total number of providers tested
	SuccessfulProviders int           // Number of providers that succeeded
	FailedProviders     int           // Number of providers that failed
	FastestProvider     string        // Name of the fastest provider
	FastestDuration     time.Duration // Duration of the fastest response
	SlowestProvider     string        // Name of the slowest provider
	SlowestDuration     time.Duration // Duration of the slowest response
	AverageDuration     time.Duration // Average duration across successful providers
	ShortestResponse    int           // Length of the shortest response
	LongestResponse     int           // Length of the longest response
}

// compareProviders sends the same prompt to multiple LLM providers and compares their responses.
// It returns a map of provider names to their results, including response time and any errors.
func compareProviders(providers []string, configs map[string]config.Config, prompt string) (map[string]ProviderResult, error) {
	return compareProvidersWithContext(context.Background(), providers, configs, prompt)
}

// compareProvidersWithContext is like compareProviders but allows specifying a context for timeout/cancellation.
func compareProvidersWithContext(ctx context.Context, providers []string, configs map[string]config.Config, prompt string) (map[string]ProviderResult, error) {
	results := make(map[string]ProviderResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process providers concurrently for better performance
	for _, provider := range providers {
		wg.Add(1)
		go func(providerName string) {
			defer wg.Done()

			result := ProviderResult{
				Provider: providerName,
			}

			// Get the configuration for this provider
			cfg, exists := configs[providerName]
			if !exists {
				result.Error = fmt.Errorf("configuration not found for provider: %s", providerName)
				mu.Lock()
				results[providerName] = result
				mu.Unlock()
				return
			}

			// Measure the time taken for the entire operation
			start := time.Now()

			// Create client for this provider
			client, err := xollm.GetClient(cfg, false)
			if err != nil {
				result.Error = fmt.Errorf("failed to create client for %s: %w", providerName, err)
				result.Duration = time.Since(start)
				mu.Lock()
				results[providerName] = result
				mu.Unlock()
				return
			}
			defer client.Close()

			// Generate response
			response, err := client.Generate(ctx, prompt)
			result.Duration = time.Since(start)

			if err != nil {
				result.Error = fmt.Errorf("generation failed for %s: %w", providerName, err)
			} else {
				result.Response = response
			}

			mu.Lock()
			results[providerName] = result
			mu.Unlock()
		}(provider)
	}

	// Wait for all providers to complete
	wg.Wait()

	return results, nil
}

// analyzeResults performs statistical analysis on the provider comparison results.
func analyzeResults(results map[string]ProviderResult) ResultAnalysis {
	analysis := ResultAnalysis{
		TotalProviders: len(results),
	}

	var successfulDurations []time.Duration
	var responseLengths []int
	fastestDuration := time.Duration(0)
	slowestDuration := time.Duration(0)

	for _, result := range results {
		if result.Error == nil {
			analysis.SuccessfulProviders++
			successfulDurations = append(successfulDurations, result.Duration)
			responseLengths = append(responseLengths, len(result.Response))

			// Track fastest provider
			if fastestDuration == 0 || result.Duration < fastestDuration {
				fastestDuration = result.Duration
				analysis.FastestProvider = result.Provider
				analysis.FastestDuration = result.Duration
			}

			// Track slowest provider
			if result.Duration > slowestDuration {
				slowestDuration = result.Duration
				analysis.SlowestProvider = result.Provider
				analysis.SlowestDuration = result.Duration
			}
		} else {
			analysis.FailedProviders++
		}
	}

	// Calculate average duration for successful providers
	if len(successfulDurations) > 0 {
		var total time.Duration
		for _, duration := range successfulDurations {
			total += duration
		}
		analysis.AverageDuration = total / time.Duration(len(successfulDurations))
	}

	// Calculate response length statistics
	if len(responseLengths) > 0 {
		sort.Ints(responseLengths)
		analysis.ShortestResponse = responseLengths[0]
		analysis.LongestResponse = responseLengths[len(responseLengths)-1]
	}

	return analysis
}

// createProviderConfigs creates sample configurations for all supported providers.
// Environment variables are used for API keys when available, otherwise placeholders are used.
func createProviderConfigs() map[string]config.Config {
	configs := make(map[string]config.Config)

	// Ollama configuration (local, no API key required)
	configs["ollama"] = config.NewConfig("ollama", 60, map[string]config.LLMConfig{
		"ollama": {
			BaseURL: getEnvOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"),
			Model:   getEnvOrDefault("OLLAMA_MODEL", "llama3"),
		},
	})

	// Gemini configuration
	configs["gemini"] = config.NewConfig("gemini", 60, map[string]config.LLMConfig{
		"gemini": {
			APIKey: getEnvOrDefault("GEMINI_API_KEY", "your-gemini-api-key"),
			Model:  getEnvOrDefault("GEMINI_MODEL", "gemini-1.5-flash-latest"),
		},
	})

	// Groq configuration
	configs["groq"] = config.NewConfig("groq", 60, map[string]config.LLMConfig{
		"groq": {
			APIKey: getEnvOrDefault("GROQ_API_KEY", "your-groq-api-key"),
			Model:  getEnvOrDefault("GROQ_MODEL", "llama3-8b-8192"),
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

// formatResults creates a formatted string representation of the comparison results and analysis.
func formatResults(results map[string]ProviderResult, analysis ResultAnalysis) string {
	var output strings.Builder

	output.WriteString("PROVIDER COMPARISON RESULTS\n")
	output.WriteString("==========================\n\n")

	// Individual Results
	output.WriteString("Individual Results:\n")
	output.WriteString("------------------\n")

	// Sort providers for consistent output
	var providers []string
	for provider := range results {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	for _, provider := range providers {
		result := results[provider]
		if result.Error == nil {
			output.WriteString(fmt.Sprintf("✓ %s: %dms\n", strings.ToUpper(result.Provider), result.Duration.Milliseconds()))
			output.WriteString(fmt.Sprintf("  Response: %s\n", truncateString(result.Response, 100)))
		} else {
			output.WriteString(fmt.Sprintf("✗ %s: FAILED\n", strings.ToUpper(result.Provider)))
			output.WriteString(fmt.Sprintf("  Error: %s\n", result.Error.Error()))
		}
		output.WriteString("\n")
	}

	// Summary Analysis
	output.WriteString("Summary Analysis:\n")
	output.WriteString("----------------\n")
	output.WriteString(fmt.Sprintf("Total Providers: %d\n", analysis.TotalProviders))
	output.WriteString(fmt.Sprintf("Successful: %d\n", analysis.SuccessfulProviders))
	output.WriteString(fmt.Sprintf("Failed: %d\n", analysis.FailedProviders))

	if analysis.SuccessfulProviders > 0 {
		output.WriteString("\nPerformance Metrics:\n")
		output.WriteString("-------------------\n")

		if analysis.SuccessfulProviders > 1 {
			output.WriteString(fmt.Sprintf("Fastest: %s (%dms)\n", analysis.FastestProvider, analysis.FastestDuration.Milliseconds()))
			output.WriteString(fmt.Sprintf("Slowest: %s (%dms)\n", analysis.SlowestProvider, analysis.SlowestDuration.Milliseconds()))
		}

		output.WriteString(fmt.Sprintf("Average Duration: %dms\n", analysis.AverageDuration.Milliseconds()))

		if analysis.ShortestResponse > 0 && analysis.LongestResponse > 0 {
			output.WriteString(fmt.Sprintf("Response Length Range: %d - %d characters\n", analysis.ShortestResponse, analysis.LongestResponse))
		}
	}

	return output.String()
}

// truncateString truncates a string to a maximum length, adding "..." if truncated.
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

// demonstrateMultiProviderComparison runs the main comparison demonstration.
func demonstrateMultiProviderComparison() error {
	// Parse command line flags
	providersFlag := flag.String("providers", "ollama,gemini,groq", "Comma-separated list of providers to compare")
	prompt := flag.String("prompt", "Explain artificial intelligence in one sentence.", "Prompt to send to all providers")
	timeout := flag.Int("timeout", 30, "Request timeout in seconds")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	// Parse providers list
	providersInput := strings.Split(*providersFlag, ",")
	var providers []string
	for _, p := range providersInput {
		provider := strings.TrimSpace(p)
		if provider != "" {
			providers = append(providers, provider)
		}
	}

	if len(providers) == 0 {
		return fmt.Errorf("no providers specified")
	}

	fmt.Printf("Multi-Provider LLM Comparison\n")
	fmt.Printf("Providers: %s\n", strings.Join(providers, ", "))
	fmt.Printf("Prompt: %s\n", *prompt)
	fmt.Printf("Timeout: %ds\n\n", *timeout)

	// Create configurations for all providers
	allConfigs := createProviderConfigs()

	// Filter to only requested providers and validate
	configs := make(map[string]config.Config)
	for _, provider := range providers {
		cfg, exists := allConfigs[provider]
		if !exists {
			fmt.Printf("Warning: Unsupported provider '%s', skipping...\n", provider)
			continue
		}

		// Update timeout
		cfg.RequestTimeoutSeconds = *timeout
		configs[provider] = cfg
	}

	if len(configs) == 0 {
		return fmt.Errorf("no valid providers configured")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout+5)*time.Second)
	defer cancel()

	fmt.Println("Running comparison...")
	start := time.Now()

	// Compare providers
	results, err := compareProvidersWithContext(ctx, providers, configs, *prompt)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	totalDuration := time.Since(start)

	// Analyze results
	analysis := analyzeResults(results)

	// Format and display results
	output := formatResults(results, analysis)
	fmt.Println(output)

	fmt.Printf("Total comparison time: %dms\n", totalDuration.Milliseconds())

	if *debug {
		fmt.Printf("\nDebug Information:\n")
		fmt.Printf("Configurations used: %d\n", len(configs))
		fmt.Printf("Concurrent execution: %t\n", true)
	}

	return nil
}

func main() {
	if err := demonstrateMultiProviderComparison(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
