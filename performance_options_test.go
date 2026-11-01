package queue

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-queue/queue/job"

	"github.com/stretchr/testify/require"
)

type legacyRetryLogger struct {
	Logger
	messages []string
}

func (l *legacyRetryLogger) Infof(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}

type filteredRetryLogger struct {
	*legacyRetryLogger
	enabled bool
}

func (l *filteredRetryLogger) InfoEnabled() bool { return l.enabled }

func TestRetryInfoEnabled(t *testing.T) {
	for _, mode := range []string{"legacy", "enabled", "disabled", "dynamic"} {
		t.Run(mode, func(t *testing.T) {
			capture := &legacyRetryLogger{Logger: NewEmptyLogger()}
			filtered := &filteredRetryLogger{legacyRetryLogger: capture, enabled: mode == "enabled"}
			var logger Logger = filtered
			if mode == "legacy" {
				logger = capture
			}
			q, err := NewQueue(WithWorker(NewRing()), WithLogger(logger))
			require.NoError(t, err)
			defer q.Release()
			attempts := 0
			m := job.NewTask(func(context.Context) error {
				attempts++
				if mode == "dynamic" {
					filtered.enabled = attempts == 2
				}
				if attempts <= 3 {
					return errors.New("retry")
				}
				return nil
			}, job.AllowOption{RetryCount: job.Int64(3), RetryDelay: job.Time(time.Nanosecond)})
			require.NoError(t, q.handle(&m))
			require.Equal(t, 4, attempts)
			want := 3
			if mode == "disabled" {
				want = 0
			}
			if mode == "dynamic" {
				want = 1
			}
			require.Len(t, capture.messages, want)
			if mode == "legacy" || mode == "enabled" {
				require.Equal(t, "retry remaining times: 2, delay time: 1ns", capture.messages[0])
			}
		})
	}
	require.False(t, infoEnabled(NewEmptyLogger()))
}

func TestRingMinCapacity(t *testing.T) {
	for _, tc := range []struct {
		name               string
		floor, limit, want int
	}{
		{"default", 0, 0, 2},
		{"negative", -1, 0, 2},
		{"retained", 64, 0, 64},
		{"non-power-of-two", 7, 0, 7},
		{"capped", 64, 10, 10},
		{"single", 64, 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := NewRing(WithQueueMinCapacity(tc.floor), WithQueueSize(tc.limit))
			require.Len(t, w.taskQueue, tc.want)
			n := 257
			if tc.limit > 0 {
				n = tc.limit
			}
			for range 3 {
				expected := make([]*job.Message, n)
				for i := range n {
					expected[i] = &job.Message{Body: []byte{byte(i)}}
					require.NoError(t, w.Queue(expected[i]))
				}
				if tc.limit > 0 {
					require.ErrorIs(t, w.Queue(new(job.Message)), ErrMaxCapacity)
				}
				// Drain partly and refill to force wraparound before the final drain.
				for i := range n / 2 {
					got, err := w.Request()
					require.NoError(t, err)
					require.Same(t, expected[i], got)
				}
				for i := range n / 2 {
					require.NoError(t, w.Queue(expected[i]))
				}
				order := append(append([]*job.Message{}, expected[n/2:]...), expected[:n/2]...)
				for _, want := range order {
					got, err := w.Request()
					require.NoError(t, err)
					require.Same(t, want, got)
				}
				require.Len(t, w.taskQueue, tc.want)
				for _, slot := range w.taskQueue {
					require.Nil(t, slot)
				}
				_, err := w.Request()
				require.ErrorIs(t, err, ErrNoTaskInQueue)
			}
			require.NoError(t, w.Shutdown())
			require.ErrorIs(t, w.Queue(new(job.Message)), ErrQueueShutdown)
		})
	}
	// Option ordering must not affect the cap.
	require.Len(t, NewRing(WithQueueSize(10), WithQueueMinCapacity(64)).taskQueue, 10)
	q := NewPool(1, WithQueueMinCapacity(64), WithLogger(NewEmptyLogger()))
	require.Equal(t, 64, q.worker.(*Ring).minCapacity)
	require.NoError(t, q.QueueTask(func(context.Context) error { return nil }))
	q.Release()
	require.EqualValues(t, 1, q.SuccessTasks())
}
