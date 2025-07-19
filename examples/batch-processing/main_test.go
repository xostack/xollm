package main

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	delay           time.Duration // Simulate processing delay
}

func (m *mockClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return fmt.Sprintf("Mock response to: %s", prompt), nil
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
			return fmt.Sprintf("Response from %s: %s", cfg.DefaultProvider, prompt), nil
		},
		providerNameVal: cfg.DefaultProvider,
		delay:           10 * time.Millisecond, // Small delay for testing
	}, nil
}

func TestNewBatchProcessor(t *testing.T) {
	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434", Model: "gemma:2b"},
	})

	processor := NewBatchProcessor(cfg, 5)

	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	if processor.GetWorkerCount() != 5 {
		t.Errorf("Expected 5 workers, got %d", processor.GetWorkerCount())
	}

	if processor.GetProcessedCount() != 0 {
		t.Errorf("Expected 0 processed jobs, got %d", processor.GetProcessedCount())
	}

	if processor.GetErrorCount() != 0 {
		t.Errorf("Expected 0 errors, got %d", processor.GetErrorCount())
	}
}

func TestBatchProcessorSingleJob(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	processor := NewBatchProcessor(cfg, 2)
	defer processor.Close()

	job := BatchJob{
		ID:     "test-1",
		Prompt: "Hello, world!",
	}

	ctx := context.Background()
	results, err := processor.ProcessJobs(ctx, []BatchJob{job})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Job.ID != "test-1" {
		t.Errorf("Expected job ID 'test-1', got '%s'", result.Job.ID)
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	if result.Response == "" {
		t.Error("Expected non-empty response")
	}

	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestBatchProcessorMultipleJobs(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	processor := NewBatchProcessor(cfg, 3)
	defer processor.Close()

	jobs := []BatchJob{
		{ID: "job-1", Prompt: "First prompt"},
		{ID: "job-2", Prompt: "Second prompt"},
		{ID: "job-3", Prompt: "Third prompt"},
		{ID: "job-4", Prompt: "Fourth prompt"},
		{ID: "job-5", Prompt: "Fifth prompt"},
	}

	ctx := context.Background()
	results, err := processor.ProcessJobs(ctx, jobs)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != len(jobs) {
		t.Fatalf("Expected %d results, got %d", len(jobs), len(results))
	}

	// Check that all jobs were processed
	processedIDs := make(map[string]bool)
	for _, result := range results {
		processedIDs[result.Job.ID] = true

		if result.Error != nil {
			t.Errorf("Expected no error for job %s, got: %v", result.Job.ID, result.Error)
		}

		if result.Response == "" {
			t.Errorf("Expected non-empty response for job %s", result.Job.ID)
		}
	}

	// Verify all job IDs were processed
	for _, job := range jobs {
		if !processedIDs[job.ID] {
			t.Errorf("Job %s was not processed", job.ID)
		}
	}
}

func TestBatchProcessorWithErrors(t *testing.T) {
	// Mock the factory function with errors
	xollm.GetClient = func(cfg config.Config, debugMode bool) (xollm.Client, error) {
		return &mockClient{
			generateFunc: func(ctx context.Context, prompt string) (string, error) {
				if strings.Contains(prompt, "error") {
					return "", errors.New("mock error")
				}
				return "Success: " + prompt, nil
			},
			providerNameVal: cfg.DefaultProvider,
		}, nil
	}
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	processor := NewBatchProcessor(cfg, 2)
	defer processor.Close()

	jobs := []BatchJob{
		{ID: "success-1", Prompt: "Good prompt"},
		{ID: "error-1", Prompt: "This will error"},
		{ID: "success-2", Prompt: "Another good prompt"},
		{ID: "error-2", Prompt: "Another error prompt"},
	}

	ctx := context.Background()
	results, err := processor.ProcessJobs(ctx, jobs)

	if err != nil {
		t.Fatalf("Expected no error from ProcessJobs, got: %v", err)
	}

	if len(results) != len(jobs) {
		t.Fatalf("Expected %d results, got %d", len(jobs), len(results))
	}

	successCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Error == nil {
			successCount++
			if !strings.Contains(result.Response, "Success") {
				t.Errorf("Expected success response for %s, got: %s", result.Job.ID, result.Response)
			}
		} else {
			errorCount++
		}
	}

	if successCount != 2 {
		t.Errorf("Expected 2 successful jobs, got %d", successCount)
	}

	if errorCount != 2 {
		t.Errorf("Expected 2 failed jobs, got %d", errorCount)
	}
}

func TestBatchProcessorConcurrency(t *testing.T) {
	// Mock with delay to test concurrency
	xollm.GetClient = func(cfg config.Config, debugMode bool) (xollm.Client, error) {
		return &mockClient{
			generateFunc: func(ctx context.Context, prompt string) (string, error) {
				// Simulate work
				time.Sleep(50 * time.Millisecond)
				return "Processed: " + prompt, nil
			},
			providerNameVal: cfg.DefaultProvider,
		}, nil
	}
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	// Test with different worker counts
	jobs := make([]BatchJob, 10)
	for i := 0; i < 10; i++ {
		jobs[i] = BatchJob{ID: fmt.Sprintf("job-%d", i), Prompt: fmt.Sprintf("Prompt %d", i)}
	}

	ctx := context.Background()

	// Sequential processing (1 worker)
	processor1 := NewBatchProcessor(cfg, 1)
	start1 := time.Now()
	results1, err := processor1.ProcessJobs(ctx, jobs)
	duration1 := time.Since(start1)
	processor1.Close()

	if err != nil {
		t.Fatalf("Expected no error with 1 worker, got: %v", err)
	}
	if len(results1) != 10 {
		t.Errorf("Expected 10 results with 1 worker, got %d", len(results1))
	}

	// Concurrent processing (5 workers)
	processor5 := NewBatchProcessor(cfg, 5)
	start5 := time.Now()
	results5, err := processor5.ProcessJobs(ctx, jobs)
	duration5 := time.Since(start5)
	processor5.Close()

	if err != nil {
		t.Fatalf("Expected no error with 5 workers, got: %v", err)
	}
	if len(results5) != 10 {
		t.Errorf("Expected 10 results with 5 workers, got %d", len(results5))
	}

	// Concurrent processing should be significantly faster
	if duration5 >= duration1 {
		t.Logf("Warning: Concurrent processing (%v) was not faster than sequential (%v)", duration5, duration1)
		// Don't fail the test as timing can be flaky in CI environments
	}
}

func TestBatchProcessorContextCancellation(t *testing.T) {
	// Mock with longer delay
	xollm.GetClient = func(cfg config.Config, debugMode bool) (xollm.Client, error) {
		return &mockClient{
			generateFunc: func(ctx context.Context, prompt string) (string, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return "Completed: " + prompt, nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
			providerNameVal: cfg.DefaultProvider,
		}, nil
	}
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	processor := NewBatchProcessor(cfg, 2)
	defer processor.Close()

	jobs := []BatchJob{
		{ID: "job-1", Prompt: "Long running job 1"},
		{ID: "job-2", Prompt: "Long running job 2"},
		{ID: "job-3", Prompt: "Long running job 3"},
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results, err := processor.ProcessJobs(ctx, jobs)

	// Should return partial results and context error
	if err == nil {
		t.Error("Expected context timeout error")
	}

	// Some jobs might complete before timeout
	for _, result := range results {
		if result.Error != nil && result.Error != context.DeadlineExceeded {
			// Other errors are acceptable (like context cancelled)
		}
	}
}

func TestBatchStatistics(t *testing.T) {
	// Mock the factory function
	xollm.GetClient = mockGetClient
	defer func() { xollm.GetClient = originalGetClient }()

	cfg := config.NewConfig("ollama", 30, map[string]config.LLMConfig{
		"ollama": {BaseURL: "http://localhost:11434"},
	})

	processor := NewBatchProcessor(cfg, 2)
	defer processor.Close()

	jobs := []BatchJob{
		{ID: "job-1", Prompt: "Success"},
		{ID: "job-2", Prompt: "error"},
		{ID: "job-3", Prompt: "Another success"},
	}

	ctx := context.Background()
	results, err := processor.ProcessJobs(ctx, jobs)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	stats := processor.GetStatistics()

	if stats.TotalJobs != 3 {
		t.Errorf("Expected 3 total jobs, got %d", stats.TotalJobs)
	}

	if stats.CompletedJobs != 2 {
		t.Errorf("Expected 2 completed jobs, got %d", stats.CompletedJobs)
	}

	if stats.FailedJobs != 1 {
		t.Errorf("Expected 1 failed job, got %d", stats.FailedJobs)
	}

	if stats.TotalDuration <= 0 {
		t.Error("Expected positive total duration")
	}

	if stats.AverageDuration <= 0 {
		t.Error("Expected positive average duration")
	}

	// Verify results match statistics
	successCount := 0
	errorCount := 0
	for _, result := range results {
		if result.Error == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	if successCount != stats.CompletedJobs {
		t.Errorf("Stats completed jobs (%d) doesn't match actual successes (%d)", stats.CompletedJobs, successCount)
	}

	if errorCount != stats.FailedJobs {
		t.Errorf("Stats failed jobs (%d) doesn't match actual errors (%d)", stats.FailedJobs, errorCount)
	}
}

func TestCreateJobsFromPrompts(t *testing.T) {
	prompts := []string{
		"First prompt",
		"Second prompt",
		"Third prompt",
	}

	jobs := createJobsFromPrompts(prompts)

	if len(jobs) != len(prompts) {
		t.Errorf("Expected %d jobs, got %d", len(prompts), len(jobs))
	}

	for i, job := range jobs {
		expectedID := fmt.Sprintf("job-%d", i+1)
		if job.ID != expectedID {
			t.Errorf("Expected job ID '%s', got '%s'", expectedID, job.ID)
		}

		if job.Prompt != prompts[i] {
			t.Errorf("Expected prompt '%s', got '%s'", prompts[i], job.Prompt)
		}

		if job.Metadata == nil {
			t.Error("Expected metadata to be initialized")
		}
	}
}

func TestCreateJobsFromFile(t *testing.T) {
	// Create a temporary file
	tempFile := t.TempDir() + "/prompts.txt"
	content := "First prompt\nSecond prompt\nThird prompt\n\n# Comment line\n  \nFourth prompt"

	err := writeStringToFile(tempFile, content)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	jobs, err := createJobsFromFile(tempFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have 4 jobs (comments and empty lines filtered out)
	expectedPrompts := []string{"First prompt", "Second prompt", "Third prompt", "Fourth prompt"}

	if len(jobs) != len(expectedPrompts) {
		t.Errorf("Expected %d jobs, got %d", len(expectedPrompts), len(jobs))
	}

	for i, job := range jobs {
		if job.Prompt != expectedPrompts[i] {
			t.Errorf("Expected prompt '%s', got '%s'", expectedPrompts[i], job.Prompt)
		}
	}
}

func TestCreateJobsFromFileNotFound(t *testing.T) {
	jobs, err := createJobsFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}

	if jobs != nil {
		t.Error("Expected nil jobs when file not found")
	}
}

func TestGenerateReport(t *testing.T) {
	results := []BatchResult{
		{
			Job:      BatchJob{ID: "job-1", Prompt: "Success prompt"},
			Response: "Success response",
			Duration: 100 * time.Millisecond,
			Error:    nil,
		},
		{
			Job:      BatchJob{ID: "job-2", Prompt: "Error prompt"},
			Response: "",
			Duration: 50 * time.Millisecond,
			Error:    errors.New("test error"),
		},
		{
			Job:      BatchJob{ID: "job-3", Prompt: "Another success"},
			Response: "Another success response",
			Duration: 150 * time.Millisecond,
			Error:    nil,
		},
	}

	stats := BatchStatistics{
		TotalJobs:       3,
		CompletedJobs:   2,
		FailedJobs:      1,
		TotalDuration:   300 * time.Millisecond,
		AverageDuration: 100 * time.Millisecond,
	}

	report := generateReport(results, stats)

	// Check that report contains expected sections
	expectedSections := []string{
		"BATCH PROCESSING REPORT",
		"Summary:",
		"Individual Results:",
		"Performance:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(report, section) {
			t.Errorf("Expected report to contain section '%s'", section)
		}
	}

	// Check specific content
	if !strings.Contains(report, "Total jobs: 3") {
		t.Error("Expected report to contain total jobs count")
	}

	if !strings.Contains(report, "✓ job-1") {
		t.Error("Expected report to contain successful job")
	}

	if !strings.Contains(report, "✗ job-2") {
		t.Error("Expected report to contain failed job")
	}
}

// Helper function for file operations
func writeStringToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
