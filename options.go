package queue

import (
	"context"
	"runtime"
	"time"

	"github.com/golang-queue/queue/core"
)

var (
	defaultQueueSize      = 4096
	defaultWorkerCount    = runtime.NumCPU()
	defaultNewLogger      = NewLogger()
	defaultFn             = func(context.Context, core.QueuedMessage) error { return nil }
	defaultMetric         = NewMetric()
	defaultRequestTimeout = 5 * time.Second
)

// An Option configures a mutex.
type Option interface {
	apply(*Options)
}

// OptionFunc is a function that configures a queue.
type OptionFunc func(*Options)

// Apply calls f(option)
func (f OptionFunc) apply(option *Options) {
	f(option)
}

// WithWorkerCount set worker count
func WithWorkerCount(num int) Option {
	return OptionFunc(func(q *Options) {
		if num <= 0 {
			num = defaultWorkerCount
		}
		q.workerCount = num
	})
}

// WithQueueSize set worker count
func WithQueueSize(num int) Option {
	return OptionFunc(func(q *Options) {
		q.queueSize = num
	})
}

// WithQueueSize set worker count
func WithRequestTimeout(timeout time.Duration) Option {
	return OptionFunc(func(q *Options) {
		q.requestTimeout = timeout
	})
}

// WithLogger set custom logger
func WithLogger(l Logger) Option {
	return OptionFunc(func(q *Options) {
		q.logger = l
	})
}

// WithMetric set custom Metric
func WithMetric(m Metric) Option {
	return OptionFunc(func(q *Options) {
		q.metric = m
	})
}

// WithWorker set custom worker
func WithWorker(w core.Worker) Option {
	return OptionFunc(func(q *Options) {
		q.worker = w
	})
}

// WithFn set custom job function
func WithFn(fn func(context.Context, core.QueuedMessage) error) Option {
	return OptionFunc(func(q *Options) {
		q.fn = fn
	})
}

// Options for custom args in Queue
type Options struct {
	workerCount int
	logger      Logger
	queueSize   int
	worker      core.Worker
	fn          func(context.Context, core.QueuedMessage) error
	metric      Metric

	// timeout for request single task
	requestTimeout time.Duration
}

// NewOptions initialize the default value for the options
func NewOptions(opts ...Option) *Options {
	o := &Options{
		workerCount: defaultWorkerCount,
		queueSize:   defaultQueueSize,
		logger:      defaultNewLogger,
		worker:      nil,
		fn:          defaultFn,
		metric:      defaultMetric,

		requestTimeout: defaultRequestTimeout,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt.apply(o)
	}

	return o
}
