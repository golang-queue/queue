package queue

import (
	"context"
	"runtime"

	"github.com/golang-queue/queue/core"
)

var (
	defaultCapacity    = 0
	defaultWorkerCount = int64(runtime.NumCPU())
	defaultNewLogger   = NewLogger()
	defaultFn          = func(context.Context, core.TaskMessage) error { return nil }
	defaultMetric      = NewMetric()
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
func WithWorkerCount(num int64) Option {
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
func WithFn(fn func(context.Context, core.TaskMessage) error) Option {
	return OptionFunc(func(q *Options) {
		q.fn = fn
	})
}

// WithAfterFn set callback function after job done
func WithAfterFn(afterFn func()) Option {
	return OptionFunc(func(q *Options) {
		q.afterFn = afterFn
	})
}

// WithContext set context
func WithContext(ctx context.Context) Option {
	return OptionFunc(func(q *Options) {
		q.ctx = ctx
	})
}

// Options for custom args in Queue
type Options struct {
	ctx         context.Context
	workerCount int64
	logger      Logger
	queueSize   int
	worker      core.Worker
	fn          func(context.Context, core.TaskMessage) error
	afterFn     func()
	metric      Metric
}

// NewOptions initialize the default value for the options
func NewOptions(opts ...Option) *Options {
	o := &Options{
		ctx:         context.Background(),
		workerCount: defaultWorkerCount,
		queueSize:   defaultCapacity,
		logger:      defaultNewLogger,
		worker:      nil,
		fn:          defaultFn,
		metric:      defaultMetric,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt.apply(o)
	}

	return o
}
