package job

import "time"

type Options struct {
	retryCount  int64
	retryDelay  time.Duration
	retryFactor float64
	retryMin    time.Duration
	retryMax    time.Duration

	timeout time.Duration
}

// An Option configures a mutex.
type Option interface {
	apply(*Options)
}

// OptionFunc is a function that configures a job.
type OptionFunc func(*Options)

// apply calls f(option)
func (f OptionFunc) apply(option *Options) {
	f(option)
}

func newDefaultOptions() *Options {
	return &Options{
		retryCount:  0,
		retryDelay:  0,
		retryFactor: 2,
		retryMin:    100 * time.Millisecond,
		retryMax:    10 * time.Second,
		timeout:     60 * time.Minute,
	}
}

// NewOptions with custom parameter
func NewOptions(opts ...Option) *Options {
	o := newDefaultOptions()

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt.apply(o)
	}

	return o
}

func WithRetryCount(count int64) Option {
	return OptionFunc(func(o *Options) {
		o.retryCount = count
	})
}

func WithRetryDelay(t time.Duration) Option {
	return OptionFunc(func(o *Options) {
		o.retryDelay = t
	})
}

func WithRetryFactor(t float64) Option {
	return OptionFunc(func(o *Options) {
		o.retryFactor = t
	})
}

func WithRetryMin(t time.Duration) Option {
	return OptionFunc(func(o *Options) {
		o.retryMin = t
	})
}

func WithRetryMax(t time.Duration) Option {
	return OptionFunc(func(o *Options) {
		o.retryMax = t
	})
}

func WithTimeout(t time.Duration) Option {
	return OptionFunc(func(o *Options) {
		o.timeout = t
	})
}
