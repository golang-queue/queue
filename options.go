package queue

import (
	"context"
	"runtime"
	"time"

	"github.com/golang-queue/queue/core"
)

var (
	defaultCapacity    = 0
	defaultWorkerCount = int64(runtime.NumCPU())
	defaultNewLogger   = NewLogger()
	defaultFn          = func(context.Context, core.TaskMessage) error { return nil }
	defaultMetric      = NewMetric()
)

// Option is a functional option for configuring a Queue.
// It follows the functional options pattern for flexible and extensible configuration.
type Option interface {
	apply(*Options)
}

// OptionFunc is a function adapter that implements the Option interface.
// It allows regular functions to be used as Options.
type OptionFunc func(*Options)

// apply implements the Option interface by calling the function with the provided options.
func (f OptionFunc) apply(option *Options) {
	f(option)
}

// WithWorkerCount sets the number of concurrent worker goroutines that will process jobs.
// If num is less than or equal to 0, it defaults to runtime.NumCPU().
// More workers allow higher concurrency but consume more system resources.
//
// Example:
//
//	q := NewPool(10, WithWorkerCount(4)) // Creates a pool with 4 workers
func WithWorkerCount(num int64) Option {
	return OptionFunc(func(q *Options) {
		if num <= 0 {
			num = defaultWorkerCount
		}
		q.workerCount = num
	})
}

// WithQueueSize sets the maximum capacity of the queue.
// When set to 0 (default), the queue has unlimited capacity and will grow dynamically.
// When set to a positive value, Queue() will return ErrMaxCapacity when the limit is reached.
// Use this to prevent memory exhaustion under high load.
//
// Example:
//
//	q := NewPool(5, WithQueueSize(1000)) // Queue will hold at most 1000 pending tasks
func WithQueueSize(num int) Option {
	return OptionFunc(func(q *Options) {
		q.queueSize = num
	})
}

// WithLogger sets a custom logger for queue events and errors.
// By default, the queue uses a standard logger that writes to stderr.
// Use NewEmptyLogger() to disable logging entirely.
//
// Example:
//
//	q := NewPool(5, WithLogger(myCustomLogger))
//	// or disable logging:
//	q := NewPool(5, WithLogger(NewEmptyLogger()))
func WithLogger(l Logger) Option {
	return OptionFunc(func(q *Options) {
		q.logger = l
	})
}

// WithMetric sets a custom metrics collector for tracking queue statistics.
// The default metric tracks busy workers, success/failure counts, and submitted tasks.
// Implement the Metric interface to integrate with custom monitoring systems.
//
// Example:
//
//	q := NewPool(5, WithMetric(myPrometheusMetric))
func WithMetric(m Metric) Option {
	return OptionFunc(func(q *Options) {
		q.metric = m
	})
}

// WithWorker sets a custom worker implementation for the queue backend.
// By default, NewPool uses an in-memory Ring buffer worker.
// Use this to integrate external queue systems like NSQ, NATS, Redis, or RabbitMQ.
// This option is required when using NewQueue() instead of NewPool().
//
// Example:
//
//	q, _ := NewQueue(WithWorker(myNSQWorker), WithWorkerCount(10))
func WithWorker(w core.Worker) Option {
	return OptionFunc(func(q *Options) {
		q.worker = w
	})
}

// WithFn sets a custom handler function that will be called to process tasks.
// This function is used by the worker's Run method when processing job messages.
// The context allows cancellation and timeout control during task execution.
// If not set, defaults to a no-op function that returns nil.
//
// Example:
//
//	handler := func(ctx context.Context, msg core.TaskMessage) error {
//	    // Process the message
//	    return processTask(msg)
//	}
//	q := NewPool(5, WithFn(handler))
func WithFn(fn func(context.Context, core.TaskMessage) error) Option {
	return OptionFunc(func(q *Options) {
		q.fn = fn
	})
}

// WithAfterFn sets a callback function that will be executed after each job completes.
// This callback runs regardless of whether the job succeeded or failed.
// It executes after metrics are updated but before the worker picks up the next task.
// Useful for cleanup, logging, or triggering post-processing workflows.
//
// Example:
//
//	q := NewPool(5, WithAfterFn(func() {
//	    log.Println("Job completed")
//	}))
func WithAfterFn(afterFn func()) Option {
	return OptionFunc(func(q *Options) {
		q.afterFn = afterFn
	})
}

// WithRetryInterval sets the interval at which the queue polls for new tasks when the queue is empty.
// This determines how often Request() is retried after receiving ErrNoTaskInQueue.
// Lower values provide faster response to new tasks but increase CPU usage.
// Defaults to 1 second.
//
// Example:
//
//	q := NewPool(5, WithRetryInterval(100*time.Millisecond)) // Poll every 100ms
func WithRetryInterval(d time.Duration) Option {
	return OptionFunc(func(q *Options) {
		q.retryInterval = d
	})
}

// Options holds the configuration parameters for a Queue.
// Use the With* functions to configure these options when creating a queue.
type Options struct {
	workerCount   int64                                         // Number of concurrent worker goroutines (default: runtime.NumCPU())
	logger        Logger                                        // Logger for queue events (default: stderr logger)
	queueSize     int                                           // Maximum queue capacity, 0 means unlimited (default: 0)
	worker        core.Worker                                   // Worker implementation for queue backend (default: Ring buffer)
	fn            func(context.Context, core.TaskMessage) error // Task handler function (default: no-op)
	afterFn       func()                                        // Callback executed after each job (default: nil)
	metric        Metric                                        // Metrics collector (default: built-in metric)
	retryInterval time.Duration                                 // Polling interval when queue is empty (default: 1 second)
}

// NewOptions creates an Options struct with default values and applies any provided options.
// Default values:
//   - workerCount: runtime.NumCPU()
//   - queueSize: 0 (unlimited)
//   - logger: stderr logger with timestamps
//   - worker: nil (must be provided via WithWorker or use NewPool which sets Ring)
//   - fn: no-op function returning nil
//   - metric: built-in metric tracker
//   - retryInterval: 1 second
func NewOptions(opts ...Option) *Options {
	o := &Options{
		workerCount:   defaultWorkerCount,
		queueSize:     defaultCapacity,
		logger:        defaultNewLogger,
		worker:        nil,
		fn:            defaultFn,
		metric:        defaultMetric,
		retryInterval: time.Second,
	}

	// Apply each provided option to override defaults
	for _, opt := range opts {
		opt.apply(o)
	}

	return o
}
