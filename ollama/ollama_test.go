package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient_Success(t *testing.T) {
	client, err := NewClient(context.Background(), "http://localhost:11434", "", 30, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.ProviderName() != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", client.ProviderName())
	}

	if client.baseURL != "http://localhost:11434" {
		t.Errorf("Expected base URL 'http://localhost:11434', got '%s'", client.baseURL)
	}

	if client.modelName != defaultOllamaModel {
		t.Errorf("Expected default model '%s', got '%s'", defaultOllamaModel, client.modelName)
	}
}

func TestNewClient_EmptyBaseURL(t *testing.T) {
	client, err := NewClient(context.Background(), "", "", 30, false)
	if err == nil {
		t.Fatal("Expected error for empty base URL")
	}

	if client != nil {
		t.Error("Expected client to be nil when error occurs")
	}

	expectedErrMsg := "Ollama base URL is required"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestNewClient_InvalidBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{
			name:    "invalid scheme",
			baseURL: "ftp://localhost:11434",
		},
		{
			name:    "malformed URL",
			baseURL: "not-a-url",
		},
		{
			name:    "missing scheme",
			baseURL: "localhost:11434",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(context.Background(), tt.baseURL, "", 30, false)
			if err == nil {
				t.Fatal("Expected error for invalid base URL")
			}

			if client != nil {
				t.Error("Expected client to be nil when error occurs")
			}
		})
	}
}

func TestNewClient_WithCustomModel(t *testing.T) {
	customModel := "codellama"
	client, err := NewClient(context.Background(), "http://localhost:11434", customModel, 45, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client.modelName != customModel {
		t.Errorf("Expected model '%s', got '%s'", customModel, client.modelName)
	}
}

func TestNewClient_URLCleaning(t *testing.T) {
	// Test that trailing slash is removed
	client, err := NewClient(context.Background(), "http://localhost:11434/", "", 30, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client.baseURL != "http://localhost:11434" {
		t.Errorf("Expected cleaned URL 'http://localhost:11434', got '%s'", client.baseURL)
	}
}

func TestOllamaClient_Generate_MockServer_Success(t *testing.T) {
	// Create a mock server that simulates Ollama API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != generateAPIPath {
			t.Errorf("Expected path '%s', got '%s'", generateAPIPath, r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected JSON content type, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected JSON accept header, got %s", r.Header.Get("Accept"))
		}

		// Mock successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"model": "gemma:2b",
			"created_at": "2024-01-01T12:00:00Z",
			"response": "Hello! This is a test response from Ollama.",
			"done": true
		}`))
	}))
	defer mockServer.Close()

	// Create client with mock server URL
	client, err := NewClient(context.Background(), mockServer.URL, "", 10, false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	response, err := client.Generate(context.Background(), "Hello, world!")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedResponse := "Hello! This is a test response from Ollama."
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}
}

func TestOllamaClient_Generate_MockServer_Error(t *testing.T) {
	// Create a mock server that simulates Ollama API error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"error": "Model not found"
		}`))
	}))
	defer mockServer.Close()

	client, err := NewClient(context.Background(), mockServer.URL, "", 10, false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.Generate(context.Background(), "Hello, world!")
	if err == nil {
		t.Fatal("Expected error from API")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected error to mention status code 400, got: %v", err)
	}
}

func TestOllamaClient_Generate_MockServer_WithErrorField(t *testing.T) {
	// Create a mock server that returns error in JSON response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"model": "gemma:2b",
			"created_at": "2024-01-01T12:00:00Z",
			"response": "",
			"done": true,
			"error": "Something went wrong"
		}`))
	}))
	defer mockServer.Close()

	client, err := NewClient(context.Background(), mockServer.URL, "", 10, false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.Generate(context.Background(), "Hello, world!")
	if err == nil {
		t.Fatal("Expected error from API")
	}

	if !strings.Contains(err.Error(), "Something went wrong") {
		t.Errorf("Expected error to contain 'Something went wrong', got: %v", err)
	}
}

func TestOllamaClient_Generate_NilClient(t *testing.T) {
	client := &Client{
		httpClient: nil, // Nil HTTP client
		baseURL:    "http://localhost:11434",
		modelName:  "test-model",
	}

	_, err := client.Generate(context.Background(), "test prompt")
	if err == nil {
		t.Fatal("Expected error for nil HTTP client")
	}

	expectedErrMsg := "Ollama client not initialized"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestOllamaClient_Generate_ContextCancellation(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    "http://localhost:11434",
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
	if !strings.Contains(err.Error(), "canceled") && !strings.Contains(err.Error(), "context") {
		// Might also get connection error, which is acceptable
		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected context cancellation or connection error, got: %v", err)
		}
	}
}

func TestOllamaClient_ProviderName(t *testing.T) {
	client := &Client{}

	if client.ProviderName() != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", client.ProviderName())
	}
}

func TestOllamaClient_Close(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{},
		baseURL:    "http://localhost:11434",
		modelName:  "test-model",
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Expected no error from Close(), got: %v", err)
	}
}

func TestOllamaConstants(t *testing.T) {
	if defaultOllamaModel == "" {
		t.Error("Default Ollama model should not be empty")
	}

	if providerName != "ollama" {
		t.Errorf("Expected provider name 'ollama', got '%s'", providerName)
	}

	if generateAPIPath == "" {
		t.Error("Generate API path should not be empty")
	}

	if !strings.HasPrefix(generateAPIPath, "/") {
		t.Errorf("API path should start with '/', got '%s'", generateAPIPath)
	}
}

// Test the request payload structure
func TestOllamaRequestPayload(t *testing.T) {
	payload := ollamaGenerateRequest{
		Model:  "test-model",
		Prompt: "test prompt",
		Stream: false,
	}

	if payload.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", payload.Model)
	}

	if payload.Prompt != "test prompt" {
		t.Errorf("Expected prompt 'test prompt', got '%s'", payload.Prompt)
	}

	if payload.Stream != false {
		t.Error("Expected stream to be false")
	}
}

// Test response parsing
func TestOllamaResponseParsing(t *testing.T) {
	response := ollamaGenerateResponse{
		Model:    "test-model",
		Response: "Test response content",
		Done:     true,
		Error:    "",
	}

	if response.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", response.Model)
	}

	if response.Response != "Test response content" {
		t.Errorf("Expected response 'Test response content', got '%s'", response.Response)
	}

	if !response.Done {
		t.Error("Expected done to be true")
	}

	if response.Error != "" {
		t.Errorf("Expected empty error, got '%s'", response.Error)
	}
}
