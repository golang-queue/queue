package queue

import (
	"context"
	"runtime"
	"time"
)

var (
	defaultQueueSize   = 4096
	defaultWorkerCount = runtime.NumCPU()
	defaultTimeout     = 60 * time.Minute
	defaultNewLogger   = NewLogger()
	defaultFn          = func(context.Context, QueuedMessage) error { return nil }
)

// Option for queue system
type Option func(*Options)

// WithWorkerCount set worker count
func WithWorkerCount(num int) Option {
	return func(q *Options) {
		q.workerCount = num
	}
}

// WithQueueSize set worker count
func WithQueueSize(num int) Option {
	return func(q *Options) {
		q.queueSize = num
	}
}

// WithLogger set custom logger
func WithLogger(l Logger) Option {
	return func(q *Options) {
		q.logger = l
	}
}

// WithWorker set custom worker
func WithWorker(w Worker) Option {
	return func(q *Options) {
		q.worker = w
	}
}

// WithFn set custom job function
func WithFn(fn func(context.Context, QueuedMessage) error) Option {
	return func(q *Options) {
		q.fn = fn
	}
}

type Options struct {
	workerCount int
	timeout     time.Duration
	logger      Logger
	queueSize   int
	worker      Worker
	fn          func(context.Context, QueuedMessage) error
}

func NewOptions(opts ...Option) *Options {
	o := &Options{
		workerCount: defaultWorkerCount,
		queueSize:   defaultQueueSize,
		timeout:     defaultTimeout,
		logger:      defaultNewLogger,
		worker:      nil,
		fn:          defaultFn,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(o)
	}

	return o
}
