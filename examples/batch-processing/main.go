package main

import (
	"bufio"
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

// BatchJob represents a single job to be processed
type BatchJob struct {
	ID       string                 // Unique identifier for the job
	Prompt   string                 // The prompt to send to the LLM
	Metadata map[string]interface{} // Additional metadata for the job
}

// BatchResult represents the result of processing a single job
type BatchResult struct {
	Job      BatchJob      // The original job
	Response string        // The LLM response
	Duration time.Duration // Time taken to process the job
	Error    error         // Any error that occurred during processing
	Worker   int           // Which worker processed this job
}

// BatchStatistics holds statistics about batch processing
type BatchStatistics struct {
	TotalJobs       int           // Total number of jobs processed
	CompletedJobs   int           // Number of successfully completed jobs
	FailedJobs      int           // Number of failed jobs
	TotalDuration   time.Duration // Total time for all jobs
	AverageDuration time.Duration // Average time per job
	WorkerCount     int           // Number of workers used
	StartTime       time.Time     // When batch processing started
	EndTime         time.Time     // When batch processing ended
}

// BatchProcessor manages concurrent processing of multiple LLM jobs
type BatchProcessor struct {
	config      config.Config    // LLM configuration
	workerCount int              // Number of concurrent workers
	stats       BatchStatistics  // Processing statistics
	mutex       sync.RWMutex     // For thread-safe access to statistics
}

// NewBatchProcessor creates a new batch processor with the specified number of workers
func NewBatchProcessor(cfg config.Config, workerCount int) *BatchProcessor {
	if workerCount <= 0 {
		workerCount = 1
	}

	return &BatchProcessor{
		config:      cfg,
		workerCount: workerCount,
		stats: BatchStatistics{
			WorkerCount: workerCount,
		},
	}
}

// GetWorkerCount returns the number of workers configured for this processor
func (bp *BatchProcessor) GetWorkerCount() int {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.workerCount
}

// GetProcessedCount returns the number of jobs processed so far
func (bp *BatchProcessor) GetProcessedCount() int {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.stats.CompletedJobs
}

// GetErrorCount returns the number of jobs that failed
func (bp *BatchProcessor) GetErrorCount() int {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.stats.FailedJobs
}

// GetStatistics returns a copy of the current processing statistics
func (bp *BatchProcessor) GetStatistics() BatchStatistics {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.stats
}

// ProcessJobs processes a batch of jobs concurrently using the configured number of workers
func (bp *BatchProcessor) ProcessJobs(ctx context.Context, jobs []BatchJob) ([]BatchResult, error) {
	if len(jobs) == 0 {
		return []BatchResult{}, nil
	}

	// Initialize statistics
	bp.mutex.Lock()
	bp.stats = BatchStatistics{
		TotalJobs:   len(jobs),
		WorkerCount: bp.workerCount,
		StartTime:   time.Now(),
	}
	bp.mutex.Unlock()

	// Create channels for job distribution and result collection
	jobChan := make(chan BatchJob, len(jobs))
	resultChan := make(chan BatchResult, len(jobs))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < bp.workerCount; i++ {
		wg.Add(1)
		go bp.worker(ctx, i+1, jobChan, resultChan, &wg)
	}

	// Send jobs to workers
	go func() {
		defer close(jobChan)
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	var results []BatchResult
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
		
		// Update statistics
		bp.mutex.Lock()
		if result.Error == nil {
			bp.stats.CompletedJobs++
		} else {
			bp.stats.FailedJobs++
		}
		bp.stats.TotalDuration += result.Duration
		bp.mutex.Unlock()
	}

	// Finalize statistics
	bp.mutex.Lock()
	bp.stats.EndTime = time.Now()
	if bp.stats.TotalJobs > 0 {
		bp.stats.AverageDuration = bp.stats.TotalDuration / time.Duration(bp.stats.TotalJobs)
	}
	bp.mutex.Unlock()

	return results, ctx.Err()
}

// worker processes jobs from the job channel and sends results to the result channel
func (bp *BatchProcessor) worker(ctx context.Context, workerID int, jobChan <-chan BatchJob, resultChan chan<- BatchResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create LLM client for this worker
	client, err := xollm.GetClient(bp.config, false)
	if err != nil {
		// Send error result for any jobs this worker would have processed
		for job := range jobChan {
			resultChan <- BatchResult{
				Job:    job,
				Error:  fmt.Errorf("failed to create LLM client: %w", err),
				Worker: workerID,
			}
		}
		return
	}
	defer client.Close()

	// Process jobs
	for {
		select {
		case job, ok := <-jobChan:
			if !ok {
				return // Channel closed, no more jobs
			}

			start := time.Now()
			response, genErr := client.Generate(ctx, job.Prompt)
			duration := time.Since(start)

			result := BatchResult{
				Job:      job,
				Response: response,
				Duration: duration,
				Error:    genErr,
				Worker:   workerID,
			}

			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// Close cleans up resources used by the batch processor
func (bp *BatchProcessor) Close() error {
	// Nothing to clean up for the processor itself
	return nil
}

// createJobsFromPrompts creates a slice of BatchJob from a slice of prompt strings
func createJobsFromPrompts(prompts []string) []BatchJob {
	jobs := make([]BatchJob, len(prompts))
	for i, prompt := range prompts {
		jobs[i] = BatchJob{
			ID:       fmt.Sprintf("job-%d", i+1),
			Prompt:   prompt,
			Metadata: make(map[string]interface{}),
		}
	}
	return jobs
}

// createJobsFromFile reads prompts from a file and creates BatchJob objects
func createJobsFromFile(filename string) ([]BatchJob, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var prompts []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			prompts = append(prompts, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return createJobsFromPrompts(prompts), nil
}

// generateReport creates a formatted report of batch processing results
func generateReport(results []BatchResult, stats BatchStatistics) string {
	var report strings.Builder

	report.WriteString("BATCH PROCESSING REPORT\n")
	report.WriteString("======================\n\n")

	// Summary section
	report.WriteString("Summary:\n")
	report.WriteString("--------\n")
	report.WriteString(fmt.Sprintf("Total jobs: %d\n", stats.TotalJobs))
	report.WriteString(fmt.Sprintf("Completed: %d\n", stats.CompletedJobs))
	report.WriteString(fmt.Sprintf("Failed: %d\n", stats.FailedJobs))
	report.WriteString(fmt.Sprintf("Success rate: %.1f%%\n", float64(stats.CompletedJobs)/float64(stats.TotalJobs)*100))
	report.WriteString(fmt.Sprintf("Workers: %d\n", stats.WorkerCount))
	report.WriteString("\n")

	// Performance section
	report.WriteString("Performance:\n")
	report.WriteString("-----------\n")
	report.WriteString(fmt.Sprintf("Total duration: %v\n", stats.TotalDuration.Round(time.Millisecond)))
	report.WriteString(fmt.Sprintf("Average per job: %v\n", stats.AverageDuration.Round(time.Millisecond)))
	
	if !stats.StartTime.IsZero() && !stats.EndTime.IsZero() {
		wallTime := stats.EndTime.Sub(stats.StartTime)
		report.WriteString(fmt.Sprintf("Wall clock time: %v\n", wallTime.Round(time.Millisecond)))
		
		if wallTime > 0 {
			throughput := float64(stats.TotalJobs) / wallTime.Seconds()
			report.WriteString(fmt.Sprintf("Throughput: %.2f jobs/second\n", throughput))
		}
	}
	report.WriteString("\n")

	// Individual results section
	report.WriteString("Individual Results:\n")
	report.WriteString("------------------\n")

	// Sort results by job ID for consistent output
	sortedResults := make([]BatchResult, len(results))
	copy(sortedResults, results)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].Job.ID < sortedResults[j].Job.ID
	})

	for _, result := range sortedResults {
		if result.Error == nil {
			report.WriteString(fmt.Sprintf("✓ %s: %dms (worker %d)\n", 
				result.Job.ID, result.Duration.Milliseconds(), result.Worker))
			
			// Truncate long responses for readability
			response := result.Response
			if len(response) > 100 {
				response = response[:97] + "..."
			}
			response = strings.ReplaceAll(response, "\n", " ")
			report.WriteString(fmt.Sprintf("  Response: %s\n", response))
		} else {
			report.WriteString(fmt.Sprintf("✗ %s: FAILED (worker %d)\n", 
				result.Job.ID, result.Worker))
			report.WriteString(fmt.Sprintf("  Error: %s\n", result.Error.Error()))
		}
	}

	return report.String()
}

// saveResultsToFile saves batch results to a JSON file
func saveResultsToFile(results []BatchResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer file.Close()

	// Simple JSON-like format for results
	file.WriteString("[\n")
	for i, result := range results {
		file.WriteString("  {\n")
		file.WriteString(fmt.Sprintf("    \"id\": \"%s\",\n", result.Job.ID))
		file.WriteString(fmt.Sprintf("    \"prompt\": \"%s\",\n", strings.ReplaceAll(result.Job.Prompt, "\"", "\\\"")))
		
		if result.Error == nil {
			file.WriteString(fmt.Sprintf("    \"response\": \"%s\",\n", strings.ReplaceAll(result.Response, "\"", "\\\"")))
			file.WriteString("    \"success\": true,\n")
		} else {
			file.WriteString(fmt.Sprintf("    \"error\": \"%s\",\n", strings.ReplaceAll(result.Error.Error(), "\"", "\\\"")))
			file.WriteString("    \"success\": false,\n")
		}
		
		file.WriteString(fmt.Sprintf("    \"duration_ms\": %d,\n", result.Duration.Milliseconds()))
		file.WriteString(fmt.Sprintf("    \"worker\": %d\n", result.Worker))
		
		if i < len(results)-1 {
			file.WriteString("  },\n")
		} else {
			file.WriteString("  }\n")
		}
	}
	file.WriteString("]\n")

	return nil
}

// demonstrateBatchProcessing runs the main batch processing demonstration
func demonstrateBatchProcessing() error {
	// Parse command line flags
	provider := flag.String("provider", "ollama", "LLM provider to use (ollama, gemini, groq)")
	workers := flag.Int("workers", 3, "Number of concurrent workers")
	timeout := flag.Int("timeout", 60, "Request timeout in seconds")
	inputFile := flag.String("input", "", "File containing prompts (one per line)")
	outputFile := flag.String("output", "", "File to save results (JSON format)")
	reportFile := flag.String("report", "", "File to save human-readable report")
	debug := flag.Bool("debug", false, "Enable debug mode")
	showProgress := flag.Bool("progress", true, "Show progress during processing")
	flag.Parse()

	// Create configuration
	var cfg config.Config
	switch *provider {
	case "ollama":
		cfg = config.NewConfig("ollama", *timeout, map[string]config.LLMConfig{
			"ollama": {
				BaseURL: getEnvOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"),
				Model:   getEnvOrDefault("OLLAMA_MODEL", "llama3"),
			},
		})
	case "gemini":
		apiKey := getEnvOrDefault("GEMINI_API_KEY", "")
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY environment variable is required for Gemini provider")
		}
		cfg = config.NewConfig("gemini", *timeout, map[string]config.LLMConfig{
			"gemini": {
				APIKey: apiKey,
				Model:  getEnvOrDefault("GEMINI_MODEL", "gemini-1.5-flash-latest"),
			},
		})
	case "groq":
		apiKey := getEnvOrDefault("GROQ_API_KEY", "")
		if apiKey == "" {
			return fmt.Errorf("GROQ_API_KEY environment variable is required for Groq provider")
		}
		cfg = config.NewConfig("groq", *timeout, map[string]config.LLMConfig{
			"groq": {
				APIKey: apiKey,
				Model:  getEnvOrDefault("GROQ_MODEL", "llama3-8b-8192"),
			},
		})
	default:
		return fmt.Errorf("unsupported provider: %s", *provider)
	}

	// Get jobs
	var jobs []BatchJob
	var err error

	if *inputFile != "" {
		jobs, err = createJobsFromFile(*inputFile)
		if err != nil {
			return fmt.Errorf("failed to load jobs from file: %w", err)
		}
	} else if len(flag.Args()) > 0 {
		// Use command line arguments as prompts
		jobs = createJobsFromPrompts(flag.Args())
	} else {
		// Use default sample prompts
		samplePrompts := []string{
			"What is artificial intelligence?",
			"Explain quantum computing in simple terms.",
			"What are the benefits of renewable energy?",
			"How does machine learning work?",
			"What is the future of space exploration?",
		}
		jobs = createJobsFromPrompts(samplePrompts)
	}

	if len(jobs) == 0 {
		return fmt.Errorf("no jobs to process")
	}

	if *debug {
		fmt.Printf("Configuration:\n")
		fmt.Printf("  Provider: %s\n", cfg.DefaultProvider)
		fmt.Printf("  Workers: %d\n", *workers)
		fmt.Printf("  Timeout: %ds\n", *timeout)
		fmt.Printf("  Jobs: %d\n", len(jobs))
		if *inputFile != "" {
			fmt.Printf("  Input file: %s\n", *inputFile)
		}
		if *outputFile != "" {
			fmt.Printf("  Output file: %s\n", *outputFile)
		}
		fmt.Println()
	}

	// Create batch processor
	processor := NewBatchProcessor(cfg, *workers)
	defer processor.Close()

	fmt.Printf("Processing %d jobs with %d workers using %s provider...\n", 
		len(jobs), *workers, cfg.DefaultProvider)

	// Process jobs
	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeout+10)*time.Second)
		defer cancel()
	}

	// Show progress if requested
	var progressTicker *time.Ticker
	if *showProgress {
		progressTicker = time.NewTicker(1 * time.Second)
		defer progressTicker.Stop()
		
		go func() {
			for range progressTicker.C {
				stats := processor.GetStatistics()
				completed := stats.CompletedJobs + stats.FailedJobs
				fmt.Printf("\rProgress: %d/%d jobs completed (%.1f%%)", 
					completed, len(jobs), float64(completed)/float64(len(jobs))*100)
			}
		}()
	}

	start := time.Now()
	results, err := processor.ProcessJobs(ctx, jobs)
	totalTime := time.Since(start)

	if *showProgress {
		progressTicker.Stop()
		fmt.Printf("\rProgress: %d/%d jobs completed (100.0%%)\n", len(jobs), len(jobs))
	}

	if err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("batch processing failed: %w", err)
	}

	// Generate statistics and report
	stats := processor.GetStatistics()
	stats.EndTime = start.Add(totalTime) // Ensure end time is set

	fmt.Printf("\nBatch processing completed in %v\n", totalTime.Round(time.Millisecond))
	fmt.Printf("Completed: %d/%d jobs (%.1f%% success rate)\n", 
		stats.CompletedJobs, stats.TotalJobs, 
		float64(stats.CompletedJobs)/float64(stats.TotalJobs)*100)

	if stats.FailedJobs > 0 {
		fmt.Printf("Failed: %d jobs\n", stats.FailedJobs)
	}

	// Save results to file if requested
	if *outputFile != "" {
		if err := saveResultsToFile(results, *outputFile); err != nil {
			fmt.Printf("Warning: Failed to save results to %s: %v\n", *outputFile, err)
		} else {
			fmt.Printf("Results saved to: %s\n", *outputFile)
		}
	}

	// Generate and save report if requested
	report := generateReport(results, stats)
	if *reportFile != "" {
		if err := os.WriteFile(*reportFile, []byte(report), 0644); err != nil {
			fmt.Printf("Warning: Failed to save report to %s: %v\n", *reportFile, err)
		} else {
			fmt.Printf("Report saved to: %s\n", *reportFile)
		}
	} else {
		// Print report to console
		fmt.Println("\n" + report)
	}

	return nil
}

// getEnvOrDefault returns the value of an environment variable or a default value if not set
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	if err := demonstrateBatchProcessing(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
