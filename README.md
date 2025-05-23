# Web Monitor

A CLI application in Go for monitoring web page availability.

## Description

This application monitors web page availability by sending HTTP GET requests at regular intervals and displays real-time statistics.

### Key Features

- ✅ **Parallel monitoring**: Each URL is monitored in parallel using separate goroutines
- ✅ **Sequential requests**: Requests to each URL are sent sequentially (one after another)  
- ✅ **5-second intervals**: New request to each URL every 5 seconds
- ✅ **10-second timeout**: Each HTTP request has a 10-second timeout
- ✅ **Real-time statistics**: Min/Avg/Max for response time and response size
- ✅ **Success tracking**: Tracks ratio of successful requests (2xx, 3xx status codes)
- ✅ **Graceful shutdown**: CTRL+C terminates the application after completing ongoing requests

## Installation

```bash
git clone https://github.com/Franta1205/web-monitor.git
cd web-monitor
go mod tidy
```

## Usage

### Basic Usage

```bash
go run . https://example.com https://seznam.cz
```

### Multiple URLs

```bash
go run . https://google.com https://github.com https://stackoverflow.com
```

### Build and Run

```bash
go build -o web-monitor .
./web-monitor https://example.com https://seznam.cz
```

## Sample Output

```
URL                            Duration Min Duration Avg Duration Max Size Min   Size Avg   Size Max   OK             
────────────────────────────   ──────────── ──────────── ──────────── ───────── ───────── ───────── ──────────────
https://example.com            45ms         67ms         89ms         1.2KB     1.4KB     1.6KB     15/16         
https://seznam.cz              123ms        145ms        167ms        45KB      47KB      52KB      14/16         
https://github.com             234ms        289ms        345ms        78KB      82KB      95KB      16/16         
```

## Project Structure

```
web-monitor/
├── go.mod          # Go module definition
├── go.sum          # Dependency checksums (auto-generated)
├── main.go         # Entry point and CLI processing
├── stats.go        # Statistics and calculations
├── monitor.go      # HTTP monitoring and worker logic
├── display.go      # Table display and formatting
├── main_test.go    # Complete test suite
└── README.md       # Documentation
```

## Implementation Details

### Architecture

- **main.go**: Entry point, argument validation, signal handling
- **stats.go**: Thread-safe statistics with min/avg/max calculations
- **monitor.go**: HTTP client, URL monitoring workers, coordination
- **display.go**: Table formatting and screen management

### Concurrency Model

- **One worker per URL**: Each URL has its own goroutine
- **Sequential requests**: Worker waits for request completion before next request
- **Parallel processing**: All workers run simultaneously
- **Thread-safe statistics**: RWMutex protects shared data
- **Graceful shutdown**: WaitGroup waits for all workers to finish

### Timing

- **5-second intervals**: `time.Ticker` for regular requests to each URL
- **10-second timeout**: HTTP client with configured timeout
- **2-second display updates**: Screen updates more frequently than requests

## Testing

### Run All Tests

```bash
go test -v
```

### Run Tests in Parallel

```bash
go test -v -parallel 4
```

### Coverage Report

```bash
go test -cover
```

### Run Specific Test

```bash
go test -v -run TestCompleteWorkflow
```

## Test Coverage

Tests cover:

- ✅ **URL Validation**: All invalid input cases
- ✅ **Statistical Calculations**: Accuracy of min/avg/max calculations
- ✅ **Thread Safety**: Concurrent access to statistics
- ✅ **HTTP Success Detection**: 2xx/3xx vs 4xx/5xx classification
- ✅ **Monitor Creation**: Proper initialization
- ✅ **Statistics Snapshots**: Thread-safe data access
- ✅ **Display Formatting**: Output formatting functions
- ✅ **Edge Cases**: Various boundary conditions

## Requirements

- **Go 1.21+**: Modern Go version
- **Standard Library**: Primarily uses Go standard library
- **httpmock**: Only for HTTP mocking in tests

## Dependencies

```go
require (
    github.com/jarcoal/httpmock v1.3.1
)
```

## Application Termination

Terminate the application using **CTRL+C**. The application will:

1. ✅ Catch SIGINT/SIGTERM signal
2. ✅ Stop creating new requests
3. ✅ Wait for all ongoing HTTP requests to complete
4. ✅ Display final statistics table
5. ✅ Exit gracefully

The final state of the table remains displayed on screen.

## Error Messages

### Invalid URL

```
Error: URL 'not-a-url' must have http or https scheme
```

### Missing Arguments

```
Usage: go run . <url1> [url2] ...
   or: ./web-monitor <url1> [url2] ...

Example: go run . https://example.com https://seznam.cz
```

### Network Errors

Network errors are silently handled and counted as unsuccessful requests in the OK ratio statistics.