package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

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
