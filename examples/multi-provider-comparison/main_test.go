package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
	return "mock response from " + m.providerNameVal + " for: " + prompt, nil
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
			return "Response from " + cfg.DefaultProvider + " provider: " + prompt, nil
		},
		providerNameVal: cfg.DefaultProvider,
	}, nil
}

func TestCompareProviders(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	providers := []string{"ollama", "gemini", "groq"}
	configs := map[string]config.Config{
		"ollama": config.NewConfig("ollama", 30, map[string]config.LLMConfig{
			"ollama": {BaseURL: "http://localhost:11434", Model: "gemma:2b"},
		}),
		"gemini": config.NewConfig("gemini", 30, map[string]config.LLMConfig{
			"gemini": {APIKey: "test-key", Model: "gemma-3-27b-it"},
		}),
		"groq": config.NewConfig("groq", 30, map[string]config.LLMConfig{
			"groq": {APIKey: "test-key", Model: "gemma2-9b-it"},
		}),
	}

	prompt := "Hello, world!"
	results, err := compareProviders(providers, configs, prompt)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != len(providers) {
		t.Errorf("Expected %d results, got %d", len(providers), len(results))
	}

	for _, provider := range providers {
		result, exists := results[provider]
		if !exists {
			t.Errorf("Expected result for provider %s", provider)
			continue
		}

		if result.Error != nil {
			t.Errorf("Expected no error for provider %s, got: %v", provider, result.Error)
		}

		expectedContains := "Response from " + provider + " provider"
		if !strings.Contains(result.Response, expectedContains) {
			t.Errorf("Expected response to contain %q, got %q", expectedContains, result.Response)
		}

		if result.Duration <= 0 {
			t.Errorf("Expected positive duration for provider %s, got %v", provider, result.Duration)
		}

		if result.Provider != provider {
			t.Errorf("Expected provider name %s, got %s", provider, result.Provider)
		}
	}
}

func TestCompareProvidersWithErrors(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	providers := []string{"ollama", "error", "gemini"}
	configs := map[string]config.Config{
		"ollama": config.NewConfig("ollama", 30, map[string]config.LLMConfig{
			"ollama": {BaseURL: "http://localhost:11434"},
		}),
		"error": config.NewConfig("error", 30, map[string]config.LLMConfig{
			"error": {APIKey: "test"},
		}),
		"gemini": config.NewConfig("gemini", 30, map[string]config.LLMConfig{
			"gemini": {APIKey: "test-key"},
		}),
	}

	prompt := "Test prompt"
	results, err := compareProviders(providers, configs, prompt)

	if err != nil {
		t.Fatalf("Expected no error from compareProviders, got: %v", err)
	}

	// Check that we got results for all providers
	if len(results) != len(providers) {
		t.Errorf("Expected %d results, got %d", len(providers), len(results))
	}

	// Ollama should succeed
	if ollamaResult, exists := results["ollama"]; exists {
		if ollamaResult.Error != nil {
			t.Errorf("Expected no error for ollama, got: %v", ollamaResult.Error)
		}
	} else {
		t.Error("Expected result for ollama provider")
	}

	// Error provider should fail
	if errorResult, exists := results["error"]; exists {
		if errorResult.Error == nil {
			t.Error("Expected error for error provider")
		}
	} else {
		t.Error("Expected result for error provider")
	}

	// Gemini should succeed
	if geminiResult, exists := results["gemini"]; exists {
		if geminiResult.Error != nil {
			t.Errorf("Expected no error for gemini, got: %v", geminiResult.Error)
		}
	} else {
		t.Error("Expected result for gemini provider")
	}
}

func TestCompareProvidersWithTimeout(t *testing.T) {
	// Mock the factory function with slow response
	xollm.GetClient = func(cfg config.Config, debugMode bool) (xollm.Client, error) {
		return &mockClient{
			generateFunc: func(ctx context.Context, prompt string) (string, error) {
				// Simulate slow response
				select {
				case <-time.After(100 * time.Millisecond):
					return "slow response from " + cfg.DefaultProvider, nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
			providerNameVal: cfg.DefaultProvider,
		}, nil
	}
	defer func() { xollm.GetClient = originalGetClient }()

	providers := []string{"ollama"}
	configs := map[string]config.Config{
		"ollama": config.NewConfig("ollama", 30, map[string]config.LLMConfig{
			"ollama": {BaseURL: "http://localhost:11434"},
		}),
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	results, err := compareProvidersWithContext(ctx, providers, configs, "test")
	if err != nil {
		t.Fatalf("Expected no error from compareProvidersWithContext, got: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results["ollama"]
	if result.Error == nil {
		t.Error("Expected timeout error for ollama provider")
	}
}

func TestAnalyzeResults(t *testing.T) {
	results := map[string]ProviderResult{
		"ollama": {
			Provider: "ollama",
			Response: "Hello from Ollama! This is a longer response.",
			Duration: 150 * time.Millisecond,
			Error:    nil,
		},
		"gemini": {
			Provider: "gemini",
			Response: "Hi there! Gemini here.",
			Duration: 80 * time.Millisecond,
			Error:    nil,
		},
		"groq": {
			Provider: "groq",
			Response: "",
			Duration: 0,
			Error:    errors.New("API error"),
		},
	}

	analysis := analyzeResults(results)

	// Check basic counts
	if analysis.TotalProviders != 3 {
		t.Errorf("Expected 3 total providers, got %d", analysis.TotalProviders)
	}

	if analysis.SuccessfulProviders != 2 {
		t.Errorf("Expected 2 successful providers, got %d", analysis.SuccessfulProviders)
	}

	if analysis.FailedProviders != 1 {
		t.Errorf("Expected 1 failed provider, got %d", analysis.FailedProviders)
	}

	// Check fastest provider
	if analysis.FastestProvider != "gemini" {
		t.Errorf("Expected fastest provider to be gemini, got %s", analysis.FastestProvider)
	}

	if analysis.FastestDuration != 80*time.Millisecond {
		t.Errorf("Expected fastest duration to be 80ms, got %v", analysis.FastestDuration)
	}

	// Check slowest provider
	if analysis.SlowestProvider != "ollama" {
		t.Errorf("Expected slowest provider to be ollama, got %s", analysis.SlowestProvider)
	}

	if analysis.SlowestDuration != 150*time.Millisecond {
		t.Errorf("Expected slowest duration to be 150ms, got %v", analysis.SlowestDuration)
	}

	// Check average duration (only successful providers)
	expectedAvg := (150 + 80) / 2 * time.Millisecond
	if analysis.AverageDuration != expectedAvg {
		t.Errorf("Expected average duration to be %v, got %v", expectedAvg, analysis.AverageDuration)
	}

	// Check response lengths
	if analysis.ShortestResponse != 22 { // "Hi there! Gemini here." = 22 chars
		t.Errorf("Expected shortest response length to be 22, got %d", analysis.ShortestResponse)
	}

	if analysis.LongestResponse != 45 { // "Hello from Ollama! This is a longer response." = 45 chars
		t.Errorf("Expected longest response length to be 45, got %d", analysis.LongestResponse)
	}
}

func TestCreateProviderConfigs(t *testing.T) {
	configs := createProviderConfigs()

	expectedProviders := []string{"ollama", "gemini", "groq"}
	if len(configs) != len(expectedProviders) {
		t.Errorf("Expected %d provider configs, got %d", len(expectedProviders), len(configs))
	}

	for _, provider := range expectedProviders {
		cfg, exists := configs[provider]
		if !exists {
			t.Errorf("Expected config for provider %s", provider)
			continue
		}

		if cfg.DefaultProvider != provider {
			t.Errorf("Expected default provider to be %s, got %s", provider, cfg.DefaultProvider)
		}

		providerConfig, exists := cfg.LLMs[provider]
		if !exists {
			t.Errorf("Expected LLM config for provider %s", provider)
			continue
		}

		// Validate provider-specific requirements
		switch provider {
		case "ollama":
			if providerConfig.BaseURL == "" {
				t.Errorf("Expected base URL for ollama provider")
			}
		case "gemini", "groq":
			if providerConfig.APIKey == "" {
				t.Errorf("Expected API key placeholder for %s provider", provider)
			}
		}
	}
}

func TestFormatResults(t *testing.T) {
	results := map[string]ProviderResult{
		"ollama": {
			Provider: "ollama",
			Response: "Hello from Ollama!",
			Duration: 150 * time.Millisecond,
			Error:    nil,
		},
		"groq": {
			Provider: "groq",
			Response: "",
			Duration: 0,
			Error:    errors.New("API error"),
		},
	}

	analysis := ResultAnalysis{
		TotalProviders:      2,
		SuccessfulProviders: 1,
		FailedProviders:     1,
		FastestProvider:     "ollama",
		FastestDuration:     150 * time.Millisecond,
		SlowestProvider:     "ollama",
		SlowestDuration:     150 * time.Millisecond,
		AverageDuration:     150 * time.Millisecond,
		ShortestResponse:    18,
		LongestResponse:     18,
	}

	output := formatResults(results, analysis)

	// Check that the output contains expected sections
	expectedSections := []string{
		"PROVIDER COMPARISON RESULTS",
		"Individual Results:",
		"Summary Analysis:",
		"Performance Metrics:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Expected output to contain section %q", section)
		}
	}

	// Check that it contains provider names and status
	if !strings.Contains(output, "OLLAMA") {
		t.Error("Expected output to contain OLLAMA provider")
	}

	if !strings.Contains(output, "GROQ") {
		t.Error("Expected output to contain GROQ provider")
	}

	if !strings.Contains(output, "✓") {
		t.Error("Expected output to contain success symbol")
	}

	if !strings.Contains(output, "✗") {
		t.Error("Expected output to contain failure symbol")
	}
}
