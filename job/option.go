package job

import "time"

// Options configures retry behavior and timeouts for individual jobs.
// These settings control how failed tasks are retried and when they time out.
type Options struct {
	retryCount  int64         // Maximum number of retry attempts (0 means no retries)
	retryDelay  time.Duration // Fixed delay between retries (0 uses exponential backoff)
	retryFactor float64       // Exponential backoff multiplier (default: 2)
	retryMin    time.Duration // Minimum backoff delay (default: 100ms)
	retryMax    time.Duration // Maximum backoff delay (default: 10s)
	jitter      bool          // Add randomization to backoff to prevent thundering herd (default: false)

	timeout time.Duration // Maximum time allowed for job execution (default: 60 minutes)
}

// newDefaultOptions creates job options with sensible defaults:
//   - retryCount: 0 (no automatic retries)
//   - retryDelay: 0 (uses exponential backoff when retries are enabled)
//   - retryFactor: 2 (doubles the delay between each retry)
//   - retryMin: 100ms (minimum backoff delay)
//   - retryMax: 10s (maximum backoff delay)
//   - timeout: 60 minutes (job execution timeout)
//   - jitter: false (no randomization in backoff)
func newDefaultOptions() Options {
	return Options{
		retryCount:  0,
		retryDelay:  0,
		retryFactor: 2,
		retryMin:    100 * time.Millisecond,
		retryMax:    10 * time.Second,
		timeout:     60 * time.Minute,
		jitter:      false,
	}
}

// AllowOption provides optional configuration for individual job execution.
// All fields are pointers to distinguish between "not set" and "set to zero value".
// This allows partial configuration while keeping unspecified fields at their defaults.
//
// Example usage:
//
//	opts := AllowOption{
//	    RetryCount: job.Int64(3),        // Retry failed jobs up to 3 times
//	    Timeout:    job.Time(5 * time.Minute), // 5 minute timeout
//	}
//	q.QueueTask(myTask, opts)
type AllowOption struct {
	RetryCount  *int64         // Maximum retry attempts (nil uses default: 0)
	RetryDelay  *time.Duration // Fixed delay between retries (nil uses exponential backoff)
	RetryFactor *float64       // Backoff multiplier (nil uses default: 2)
	RetryMin    *time.Duration // Minimum backoff delay (nil uses default: 100ms)
	RetryMax    *time.Duration // Maximum backoff delay (nil uses default: 10s)
	Jitter      *bool          // Enable backoff jitter (nil uses default: false)
	Timeout     *time.Duration // Job execution timeout (nil uses default: 60 minutes)
}

// NewOptions creates a job Options struct by merging defaults with provided AllowOption values.
// Only non-nil fields in AllowOption will override the defaults.
// This allows partial configuration where unspecified options retain their default values.
//
// Example:
//
//	// Only override retry count and timeout, keep other defaults
//	opts := job.NewOptions(job.AllowOption{
//	    RetryCount: job.Int64(5),
//	    Timeout:    job.Time(10 * time.Minute),
//	})
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

		if opts[0].Jitter != nil && *opts[0].Jitter != o.jitter {
			o.jitter = *opts[0].Jitter
		}
	}

	return o
}

// Int64 is a helper function that creates a pointer to an int64 value.
// Useful for setting AllowOption fields that require *int64.
//
// Example:
//
//	opts := AllowOption{RetryCount: job.Int64(3)}
func Int64(val int64) *int64 {
	return &val
}

// Float64 is a helper function that creates a pointer to a float64 value.
// Useful for setting AllowOption fields that require *float64.
//
// Example:
//
//	opts := AllowOption{RetryFactor: job.Float64(1.5)}
func Float64(val float64) *float64 {
	return &val
}

// Time is a helper function that creates a pointer to a time.Duration value.
// Useful for setting AllowOption fields that require *time.Duration.
//
// Example:
//
//	opts := AllowOption{
//	    Timeout:    job.Time(5 * time.Minute),
//	    RetryDelay: job.Time(2 * time.Second),
//	}
func Time(v time.Duration) *time.Duration {
	return &v
}

// Bool is a helper function that creates a pointer to a bool value.
// Useful for setting AllowOption fields that require *bool.
//
// Example:
//
//	opts := AllowOption{Jitter: job.Bool(true)}
func Bool(val bool) *bool {
	return &val
}
