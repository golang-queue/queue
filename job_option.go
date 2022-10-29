package queue

import "time"

type JobConfig struct {
	retryCount int64
	retryDelay time.Duration
	timeout    time.Duration
}

// An JobOption configures a mutex.
type JobOption interface {
	apply(*JobConfig)
}

// JobOptionFunc is a function that configures a job.
type JobOptionFunc func(*JobConfig)

// Apply calls f(option)
func (f JobOptionFunc) apply(option *JobConfig) {
	f(option)
}

func WithRetryCount(count int64) JobOption {
	return JobOptionFunc(func(q *JobConfig) {
		q.retryCount = count
	})
}

func WithRetryDelay(t time.Duration) JobOption {
	return JobOptionFunc(func(q *JobConfig) {
		q.retryDelay = t
	})
}

func WithTimeout(t time.Duration) JobOption {
	return JobOptionFunc(func(q *JobConfig) {
		q.timeout = t
	})
}
