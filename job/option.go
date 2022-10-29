package job

import "time"

type Config struct {
	RetryCount int64
	RetryDelay time.Duration
	Timeout    time.Duration
}

// An Option configures a mutex.
type Option interface {
	Apply(*Config)
}

// OptionFunc is a function that configures a job.
type OptionFunc func(*Config)

// Apply calls f(option)
func (f OptionFunc) Apply(option *Config) {
	f(option)
}

func DefaultOptions(opts ...Option) *Config {
	o := &Config{
		RetryCount: 0,
		RetryDelay: 100 * time.Millisecond,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt.Apply(o)
	}

	return o
}

func WithRetryCount(count int64) Option {
	return OptionFunc(func(q *Config) {
		q.RetryCount = count
	})
}

func WithRetryDelay(t time.Duration) Option {
	return OptionFunc(func(q *Config) {
		q.RetryDelay = t
	})
}

func WithTimeout(t time.Duration) Option {
	return OptionFunc(func(q *Config) {
		q.Timeout = t
	})
}
