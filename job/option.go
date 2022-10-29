package job

import "time"

type Options struct {
	retryCount int64
	retryDelay time.Duration
	timeout    time.Duration
}

// An Option configures a mutex.
type Option interface {
	Apply(*Options)
}

// OptionFunc is a function that configures a job.
type OptionFunc func(*Options)

// Apply calls f(option)
func (f OptionFunc) Apply(option *Options) {
	f(option)
}

func NewOptions(opts ...Option) *Options {
	o := &Options{
		retryCount: 0,
		retryDelay: 100 * time.Millisecond,
		timeout:    60 * time.Minute,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt.Apply(o)
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

func WithTimeout(t time.Duration) Option {
	return OptionFunc(func(o *Options) {
		o.timeout = t
	})
}
