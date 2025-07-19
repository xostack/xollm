# Batch Processing Example

This example demonstrates how to use xollm for batch processing multiple jobs concurrently with worker pools, job queuing, and comprehensive result collection.

## Features

- **Concurrent Processing**: Process multiple jobs in parallel using worker pools
- **Job Management**: Queue jobs from prompts or files
- **Result Collection**: Gather and analyze results from all processed jobs
- **Error Handling**: Robust error handling with detailed error reporting
- **Progress Tracking**: Real-time progress monitoring during batch processing
- **Context Support**: Proper context handling for cancellation and timeouts
- **Statistics**: Comprehensive batch processing statistics and reporting

## How It Works

The example implements a `BatchProcessor` that:

1. **Creates Worker Pool**: Spawns multiple workers to process jobs concurrently
2. **Job Queuing**: Distributes jobs across available workers using channels
3. **Result Collection**: Gathers all results and errors from concurrent processing
4. **Statistics**: Tracks processing time, success/failure rates, and performance metrics
5. **Reporting**: Generates detailed reports of batch processing results

## Core Components

### BatchJob
```go
type BatchJob struct {
    ID     string
    Prompt string
    Config *config.Config
}
```

### BatchResult
```go
type BatchResult struct {
    JobID     string
    Response  string
    Error     error
    Duration  time.Duration
    Provider  string
}
```

### BatchProcessor
```go
type BatchProcessor struct {
    maxWorkers int
    timeout    time.Duration
}
```

## Usage Examples

### Basic Batch Processing

```bash
# Process prompts interactively
go run main.go

# Process with specific number of workers
go run main.go -workers 5

# Process with timeout
go run main.go -timeout 30s

# Process jobs from file
go run main.go -file prompts.txt

# Combine options
go run main.go -workers 10 -timeout 60s -file batch_jobs.txt
```

### Input File Format

Create a text file with one prompt per line:

```
What is the capital of France?
Explain quantum computing in simple terms
Write a haiku about programming
Solve this math problem: 2x + 5 = 15
```

## Command Line Options

- `-workers`: Number of concurrent workers (default: 3)
- `-timeout`: Timeout for each job (default: 30s)
- `-file`: Input file with prompts (one per line)

## Example Output

```
Batch Processing with xollm
===========================

Enter prompts (one per line, empty line to finish):
> What is machine learning?
> Explain blockchain technology
> 

Starting batch processing...
Workers: 3
Jobs: 2
Timeout: 30s

Processing jobs... [████████████████████████████████████████] 100% (2/2)

Batch Processing Complete!
=========================

Total Jobs: 2
Successful: 2
Failed: 0
Total Duration: 3.45s
Average Duration: 1.73s per job
Success Rate: 100.00%

Results:
--------

Job 1 (ollama):
  Duration: 1.2s
  Response: Machine learning is a subset of artificial intelligence...

Job 2 (ollama):
  Duration: 2.25s
  Response: Blockchain technology is a distributed ledger system...
```

## Testing

The example includes comprehensive tests covering:

- Basic batch processing functionality
- Concurrent processing with multiple workers
- Error handling and recovery
- Context cancellation and timeouts
- Job creation from various sources
- Statistics calculation and reporting

Run the tests:

```bash
go test -v
go test -cover
```

## Key Features Demonstrated

1. **Worker Pool Pattern**: Efficient concurrent processing using goroutines
2. **Channel Communication**: Safe job distribution and result collection
3. **Context Management**: Proper cancellation and timeout handling
4. **Error Aggregation**: Collecting and reporting errors from concurrent operations
5. **Progress Tracking**: Real-time monitoring of batch processing progress
6. **Resource Management**: Controlled concurrency to prevent resource exhaustion
7. **Flexible Input**: Support for interactive input and file-based job loading

## Production Considerations

- **Worker Pool Size**: Adjust based on system resources and API rate limits
- **Timeout Configuration**: Set appropriate timeouts for your use case
- **Error Handling**: Implement retry logic for transient failures
- **Memory Management**: Consider memory usage for large batch sizes
- **Rate Limiting**: Respect API rate limits when processing large batches
- **Monitoring**: Add metrics and logging for production deployments

This example demonstrates enterprise-ready patterns for batch processing with proper error handling, concurrency control, and comprehensive reporting.
