package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-queue/queue/job"

	"github.com/stretchr/testify/require"
)

func TestResultProtocolShutdown(t *testing.T) {
	for _, kind := range []string{"success", "error", "panic"} {
		t.Run(kind, func(t *testing.T) {
			q, err := NewQueue(WithWorker(NewRing()), WithLogger(NewEmptyLogger()))
			require.NoError(t, err)
			t.Cleanup(q.Release)
			entered := make(chan struct{})
			type outcome struct {
				err error
				p   any
			}
			finished := make(chan outcome, 1)
			taskErr := errors.New("task failed")
			m := job.NewTask(func(ctx context.Context) error {
				close(entered)
				<-ctx.Done()
				if kind == "panic" {
					panic("shutdown panic")
				}
				if kind == "error" {
					return taskErr
				}
				return nil
			}, job.AllowOption{Timeout: job.Time(5 * time.Second)})
			go func() {
				var result outcome
				defer func() { result.p = recover(); finished <- result }()
				result.err = q.handle(&m)
			}()
			receiveWithin(t, entered)
			q.Shutdown()
			result := receiveWithin(t, finished)
			if kind == "panic" {
				require.Equal(t, "shutdown panic", result.p)
			} else {
				require.Nil(t, result.p)
				if kind == "error" {
					require.ErrorIs(t, result.err, taskErr)
				} else {
					require.NoError(t, result.err)
				}
			}
		})
	}
}

// TestMain's goleak check detects a handler blocked on a late result send.
func TestResultProtocolLateCompletion(t *testing.T) {
	for _, kind := range []string{"success", "error", "panic"} {
		t.Run(kind, func(t *testing.T) {
			q, err := NewQueue(WithWorker(NewRing()), WithLogger(NewEmptyLogger()))
			require.NoError(t, err)
			t.Cleanup(q.Release)
			release := make(chan struct{})
			defer close(release)
			entered := make(chan struct{})
			m := job.NewTask(func(context.Context) error {
				close(entered)
				<-release
				if kind == "panic" {
					panic("late panic")
				}
				if kind == "error" {
					return errors.New("late failure")
				}
				return nil
			}, job.AllowOption{Timeout: job.Time(5 * time.Millisecond)})
			require.ErrorIs(t, q.handle(&m), context.DeadlineExceeded)
			receiveWithin(t, entered)
		})
	}
}

func TestResultProtocolRetryCancellation(t *testing.T) {
	q, err := NewQueue(WithWorker(NewRing()), WithLogger(NewEmptyLogger()))
	require.NoError(t, err)
	t.Cleanup(q.Release)
	taskErr := errors.New("retry")
	task := func(context.Context) error { return taskErr }
	m := job.NewTask(task, job.AllowOption{
		RetryCount: job.Int64(10),
		RetryDelay: job.Time(time.Hour),
		Timeout:    job.Time(10 * time.Millisecond),
	})
	require.ErrorIs(t, q.handle(&m), context.DeadlineExceeded)
}

func TestResultProtocolRepeatedRetries(t *testing.T) {
	for _, succeed := range []bool{false, true} {
		q, err := NewQueue(WithWorker(NewRing()), WithLogger(NewEmptyLogger()))
		require.NoError(t, err)
		t.Cleanup(q.Release)
		var attempts int
		taskErr := errors.New("retry")
		m := job.NewTask(func(context.Context) error {
			attempts++
			if succeed && attempts == 11 {
				return nil
			}
			return taskErr
		}, job.AllowOption{RetryCount: job.Int64(10), RetryDelay: job.Time(time.Nanosecond)})
		err = q.handle(&m)
		if succeed {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, taskErr)
		}
		require.Equal(t, 11, attempts)
	}
}
