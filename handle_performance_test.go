package queue

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-queue/queue/job"
)

// Isolate handle overhead with fresh retry state for each completed invocation.
// A 1ns retry delay emphasizes timer bookkeeping rather than real backoff waits.
func BenchmarkHandleRetry(b *testing.B) {
	for _, retries := range []int64{0, 1, 3, 10} {
		b.Run(fmt.Sprintf("retries=%d", retries), func(b *testing.B) {
			q, err := NewQueue(WithWorker(NewRing()), WithLogger(NewEmptyLogger()))
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(q.Release)
			retryErr := errors.New("retry")
			var attempts int64
			m := job.Message{
				Timeout: time.Minute, RetryDelay: time.Nanosecond,
				Task: func(context.Context) error {
					attempts++
					if attempts <= retries {
						return retryErr
					}
					return nil
				},
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				attempts = 0
				m.RetryCount = retries
				if err := q.handle(&m); err != nil {
					b.Fatal(err)
				}
				if attempts != retries+1 {
					b.Fatalf("got %d attempts, want %d", attempts, retries+1)
				}
			}
		})
	}
}
