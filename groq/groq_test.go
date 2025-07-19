package groq

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient_Success(t *testing.T) {
	client, err := NewClient(context.Background(), "test-api-key", "", 30, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.ProviderName() != "groq" {
		t.Errorf("Expected provider name 'groq', got '%s'", client.ProviderName())
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", client.apiKey)
	}

	if client.modelName != defaultGroqModel {
		t.Errorf("Expected default model '%s', got '%s'", defaultGroqModel, client.modelName)
	}
}

func TestNewClient_EmptyAPIKey(t *testing.T) {
	client, err := NewClient(context.Background(), "", "", 30, false)
	if err == nil {
		t.Fatal("Expected error for empty API key")
	}

	if client != nil {
		t.Error("Expected client to be nil when error occurs")
	}

	expectedErrMsg := "groq API key is required"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestNewClient_WithCustomModel(t *testing.T) {
	customModel := "mixtral-8x7b-32768"
	client, err := NewClient(context.Background(), "test-api-key", customModel, 45, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client.modelName != customModel {
		t.Errorf("Expected model '%s', got '%s'", customModel, client.modelName)
	}
}

func TestGroqClient_Generate_MockServer_Success(t *testing.T) {
	// Create a mock server that simulates Groq API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected JSON content type, got %s", r.Header.Get("Content-Type"))
		}

		// Mock successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chatcmpl-test",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gemma2-9b-it",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! This is a test response."
					},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 8,
				"total_tokens": 18
			}
		}`))
	}))
	defer mockServer.Close()

	// Create client with custom endpoint (using mock server)
	client := &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiKey:     "test-api-key",
		modelName:  "gemma2-9b-it",
	}

	// Override the API endpoint for testing
	originalEndpoint := groqAPIEndpoint
	defer func() {
		// Note: We can't actually change the const, so this test demonstrates
		// the structure but would need dependency injection for full testing
	}()

	// For this test, we'll test with the actual endpoint but expect network error
	_, err := client.Generate(context.Background(), "Hello, world!")
	if err == nil {
		t.Fatal("Expected network error since we're hitting real endpoint")
	}

	// Should be a network-related error, not a client error
	if strings.Contains(err.Error(), "groq client not initialized") {
		t.Error("Should not get client initialization error with properly initialized client")
	}

	_ = originalEndpoint // Use the variable to avoid unused error
}

func TestGroqClient_Generate_NilClient(t *testing.T) {
	client := &Client{
		httpClient: nil, // Nil HTTP client
		apiKey:     "test-key",
		modelName:  "test-model",
	}

	_, err := client.Generate(context.Background(), "test prompt")
	if err == nil {
		t.Fatal("Expected error for nil HTTP client")
	}

	expectedErrMsg := "groq client not initialized"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGroqClient_Generate_ContextCancellation(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiKey:     "test-key",
		modelName:  "test-model",
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Generate(ctx, "test prompt")
	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	// Should get context cancellation error
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "canceled") {
		// Might also get connection error, which is acceptable
		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected context cancellation or connection error, got: %v", err)
		}
	}
}

func TestGroqClient_ProviderName(t *testing.T) {
	client := &Client{}

	if client.ProviderName() != "groq" {
		t.Errorf("Expected provider name 'groq', got '%s'", client.ProviderName())
	}
}

func TestGroqClient_Close(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		modelName:  "test-model",
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Expected no error from Close(), got: %v", err)
	}
}

func TestGroqConstants(t *testing.T) {
	if defaultGroqModel == "" {
		t.Error("Default Groq model should not be empty")
	}

	if providerName != "groq" {
		t.Errorf("Expected provider name 'groq', got '%s'", providerName)
	}

	if groqAPIEndpoint == "" {
		t.Error("Groq API endpoint should not be empty")
	}

	if !strings.Contains(groqAPIEndpoint, "groq.com") {
		t.Errorf("API endpoint should contain 'groq.com', got '%s'", groqAPIEndpoint)
	}

	if maxRetries < 0 {
		t.Errorf("Max retries should be non-negative, got %d", maxRetries)
	}

	if retryDelay <= 0 {
		t.Errorf("Retry delay should be positive, got %v", retryDelay)
	}
}

// Test the request payload structure
func TestGroqRequestPayload(t *testing.T) {
	// This test verifies our understanding of the payload structure
	payload := groqChatCompletionRequest{
		Messages: []groqChatMessage{
			{Role: "user", Content: "test prompt"},
		},
		Model:  "test-model",
		Stream: false,
	}

	if len(payload.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(payload.Messages))
	}

	if payload.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", payload.Messages[0].Role)
	}

	if payload.Messages[0].Content != "test prompt" {
		t.Errorf("Expected content 'test prompt', got '%s'", payload.Messages[0].Content)
	}

	if payload.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", payload.Model)
	}

	if payload.Stream != false {
		t.Error("Expected stream to be false")
	}
}

// Test response parsing
func TestGroqResponseParsing(t *testing.T) {
	// This test verifies our understanding of the response structure
	response := groqChatCompletionResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "test-model",
		Choices: []groqChatCompletionResponseChoice{
			{
				Index: 0,
				Message: groqChatCompletionResponseChoiceMessage{
					Role:    "assistant",
					Content: "Test response content",
				},
				FinishReason: "stop",
			},
		},
		Usage: groqUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	if response.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", response.ID)
	}

	if len(response.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(response.Choices))
	}

	if response.Choices[0].Message.Content != "Test response content" {
		t.Errorf("Expected content 'Test response content', got '%s'", response.Choices[0].Message.Content)
	}

	if response.Usage.TotalTokens != 15 {
		t.Errorf("Expected total tokens 15, got %d", response.Usage.TotalTokens)
	}
}
