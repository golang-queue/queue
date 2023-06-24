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

func newDefaultOptions() Options {
	return Options{
		retryCount:  0,
		retryDelay:  0,
		retryFactor: 2,
		retryMin:    100 * time.Millisecond,
		retryMax:    10 * time.Second,
		timeout:     60 * time.Minute,
	}
}

type AllowOption struct {
	RetryCount  *int64
	RetryDelay  *time.Duration
	RetryFactor *float64
	RetryMin    *time.Duration
	RetryMax    *time.Duration
	Timeout     *time.Duration
}

// NewOptions with custom parameter
func NewOptions(opts ...AllowOption) Options {
	o := newDefaultOptions()

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

		if opts[0].RetryFactor != nil && *opts[0].RetryFactor != o.retryFactor {
			o.retryFactor = *opts[0].RetryFactor
		}

		if opts[0].RetryMin != nil && *opts[0].RetryMin != o.retryMin {
			o.retryMin = *opts[0].RetryMin
		}

		if opts[0].RetryMax != nil && *opts[0].RetryMax != o.retryMax {
			o.retryMax = *opts[0].RetryMax
		}
	}

	return o
}

func Int64(val int64) *int64 {
	return &val
}

func Time(v time.Duration) *time.Duration {
	return &v
}
