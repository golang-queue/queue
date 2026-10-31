package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/golang-queue/queue/job"
)

// BenchmarkRingBoundary holds the backlog near a power-of-two capacity, where
// growing at full and shrinking at half can otherwise copy the entire backlog
// twice per operation. One operation is one successful enqueue/dequeue pair.
func BenchmarkRingBoundary(b *testing.B) {
	for _, backlog := range []int{64, 1024} {
		b.Run(fmt.Sprintf("backlog=%d", backlog), func(b *testing.B) {
			w := NewRing()
			m := job.NewTask(func(context.Context) error { return nil })
			for range backlog {
				if err := w.Queue(&m); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := w.Queue(&m); err != nil {
					b.Fatal(err)
				}
				if _, err := w.Request(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkQueueEndToEnd measures public submission through handler completion
// and final Release. Batches bound the backlog independently of b.N.
func BenchmarkQueueEndToEnd(b *testing.B) {
	for _, workers := range []int64{1, 4, 12} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			benchmarkQueueCompletion(b, workers, false)
		})
	}
}

// BenchmarkQueueAfterFnEndToEnd also waits for every configured callback.
func BenchmarkQueueAfterFnEndToEnd(b *testing.B) {
	benchmarkQueueCompletion(b, 4, true)
}

func benchmarkQueueCompletion(b *testing.B, workers int64, withCallback bool) {
	var completed sync.WaitGroup
	opts := []Option{WithLogger(NewEmptyLogger())}
	task := func(context.Context) error {
		completed.Done()
		return nil
	}
	if withCallback {
		opts = append(opts, WithAfterFn(completed.Done))
		task = func(context.Context) error { return nil }
	}
	q := NewPool(workers, opts...)
	b.ReportAllocs()
	b.ResetTimer()
	for submitted := 0; submitted < b.N; {
		n := min(1024, b.N-submitted)
		completed.Add(n)
		for range n {
			if err := q.QueueTask(task); err != nil {
				b.Fatal(err)
			}
		}
		completed.Wait()
		submitted += n
	}
	q.Release()
	b.StopTimer()
	if got := q.SuccessTasks(); got != uint64(b.N) {
		b.Fatalf("completed %d tasks, want %d", got, b.N)
	}
}
