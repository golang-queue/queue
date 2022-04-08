package queue

import (
	"context"
	"runtime"
	"time"

	"github.com/golang-queue/queue/core"
)

var (
	defaultQueueSize   = 4096
	defaultWorkerCount = runtime.NumCPU()
	defaultTimeout     = 60 * time.Minute
	defaultNewLogger   = NewLogger()
	defaultFn          = func(context.Context, core.QueuedMessage) error { return nil }
	defaultMetric      = NewMetric()
)

// An Option configures a mutex.
type Option interface {
	Apply(*Options)
}

// OptionFunc is a function that configures a queue.
type OptionFunc func(*Options)

// Apply calls f(option)
func (f OptionFunc) Apply(option *Options) {
	f(option)
}

// WithWorkerCount set worker count
func WithWorkerCount(num int) Option {
	return OptionFunc(func(q *Options) {
		q.workerCount = num
	})
}

// WithQueueSize set worker count
func WithQueueSize(num int) Option {
	return OptionFunc(func(q *Options) {
		q.queueSize = num
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

// WithTimeOut set custom timeout
func WithTimeOut(t time.Duration) Option {
	return OptionFunc(func(q *Options) {
		q.timeout = t
	})
}

// Options for custom args in Queue
type Options struct {
	workerCount int
	timeout     time.Duration
	logger      Logger
	queueSize   int
	worker      core.Worker
	fn          func(context.Context, core.QueuedMessage) error
	metric      Metric
}

// NewOptions initialize the default value for the options
func NewOptions(opts ...Option) *Options {
	o := &Options{
		workerCount: defaultWorkerCount,
		queueSize:   defaultQueueSize,
		timeout:     defaultTimeout,
		logger:      defaultNewLogger,
		worker:      nil,
		fn:          defaultFn,
		metric:      defaultMetric,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt.Apply(o)
	}

	return o
}
