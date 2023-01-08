package job

import "time"

type Option2s struct {
	retryCount int64
	retryDelay time.Duration
	timeout    time.Duration
}

// An Option2 configures a mutex.
type Option2 interface {
	apply(Option2s) Option2s
}

// Option2Func is a function that configures a job.
type Option2Func func(Option2s) Option2s

// apply calls f(option)
func (f Option2Func) apply(option Option2s) Option2s {
	return f(option)
}

func newDefaultOption2s() Option2s {
	return Option2s{
		retryCount: 0,
		retryDelay: 100 * time.Millisecond,
		timeout:    60 * time.Minute,
	}
}

type AllowOption struct {
	RetryCount *int64
	RetryDelay *time.Duration
	Timeout    *time.Duration
}

// NewOption2s with custom parameter
func NewOption2s(opts ...Option2) Option2s {
	o := newDefaultOption2s()

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		o = opt.apply(o)
	}

	return o
}

func NewOption3s(opts ...AllowOption) Option2s {
	o := newDefaultOption2s()

	if len(opts) != 0 {
		if opts[0].RetryCount != nil && *opts[0].RetryCount != o.retryCount {
			o.retryCount = *opts[0].RetryCount
		}

		if opts[0].RetryDelay != nil && *opts[0].RetryDelay != o.retryDelay {
			o.retryDelay = *opts[0].RetryDelay
		}

		if opts[0].Timeout != nil && *opts[0].Timeout != o.timeout {
			o.timeout = *opts[0].Timeout
		}
	}

	return o
}

func WithRetryCount2(count int64) Option2 {
	return Option2Func(func(o Option2s) Option2s {
		o.retryCount = count
		return o
	})
}

func WithRetryDelay2(t time.Duration) Option2 {
	return Option2Func(func(o Option2s) Option2s {
		o.retryDelay = t
		return o
	})
}

func WithTimeout2(t time.Duration) Option2 {
	return Option2Func(func(o Option2s) Option2s {
		o.timeout = t
		return o
	})
}

func Int64(val int64) *int64 {
	return &val
}

func Time(v time.Duration) *time.Duration {
	return &v
}
