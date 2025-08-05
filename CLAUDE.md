# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Testing

- `go test -v` - Run all tests with verbose output
- `go test -race -v -covermode=atomic -coverprofile=coverage.out` - Run tests with race detection and coverage
- `go test -v -run=^$ -count 5 -benchmem -bench . ./...` - Run benchmarks with memory stats

### Building

- `go build` - Build the package
- `go mod download` - Download dependencies
- `go mod tidy` - Clean up go.mod and go.sum

### Code Quality

- Uses golangci-lint for linting (configured in CI)
- Bearer security scanning is enabled (config in bearer.yml)

## Architecture Overview

### Core Components

The queue library is built around these key abstractions:

1. **Queue** (`queue.go:28-42`) - Main queue implementation managing workers, job scheduling, retries, and graceful shutdown
2. **Worker Interface** (`core/worker.go:9-25`) - Abstraction for task processors with Run, Shutdown, Queue, and Request methods
3. **Ring Buffer** (`ring.go:14-26`) - Default in-memory worker implementation using circular buffer
4. **Job Messages** (`job/job.go:14-50`) - Task wrapper with retry logic, timeouts, and payload handling

### Key Design Patterns

- **Worker Pool**: Queue manages multiple goroutines processing tasks concurrently
- **Pluggable Workers**: Core Worker interface allows different backends (NSQ, NATS, Redis, etc.)
- **Graceful Shutdown**: Atomic flags and sync.Once ensure clean shutdown
- **Retry Logic**: Built-in exponential backoff with jitter for failed tasks
- **Metrics**: Built-in tracking of submitted, completed, success, and failure counts

### Main Entry Points

- `NewPool(size, opts...)` - Creates queue with Ring buffer worker, auto-starts
- `NewQueue(opts...)` - Creates queue with custom worker, requires manual Start()
- `Queue(message)` - Enqueue a QueuedMessage
- `QueueTask(func)` - Enqueue a function directly

### Worker Implementations

The library supports multiple queue backends through the Worker interface:

- **Ring** (default) - In-memory circular buffer
- **NSQ** - Distributed messaging (external package)
- **NATS** - Cloud-native messaging (external package)
- **Redis** - Pub/Sub or Streams (external package)
- **RabbitMQ** - Message broker (external package)

## Testing Notes

- Uses testify for assertions (`github.com/stretchr/testify`)
- Includes benchmark tests in `benchmark_test.go` and `job/benchmark_test.go`
- Uses go.uber.org/goleak to detect goroutine leaks
- Mock interfaces generated with go.uber.org/mock in `mocks/` directory
