package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"unsafe"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/job"
)

// The comparison harness supplies the same retained-capacity option to each
// variant; on the baseline it is intentionally a no-op (the old policy).
func performanceRingOptions(retained bool) []Option {
	if retained {
		return []Option{WithQueueMinCapacity(1024)}
	}
	return nil
}

// One operation is a full 1024-task burst and drain. Retained bytes count only
// backing-array slots after drain, not RSS or task payloads.
func BenchmarkRingBurstDrain(b *testing.B) {
	for _, retained := range []bool{false, true} {
		b.Run(fmt.Sprintf("retained=%t", retained), func(b *testing.B) {
			w := NewRing(performanceRingOptions(retained)...)
			m := job.NewTask(func(context.Context) error { return nil })
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for range 1024 {
					if err := w.Queue(&m); err != nil {
						b.Fatal(err)
					}
				}
				for range 1024 {
					if _, err := w.Request(); err != nil {
						b.Fatal(err)
					}
				}
			}
			b.StopTimer()
			b.ReportMetric(
				float64(len(w.taskQueue))*float64(unsafe.Sizeof(core.TaskMessage(nil))),
				"retained-B",
			)
		})
	}
}

// One operation is one publicly submitted, completed task. Fixed batches bound
// the backlog; producer goroutine creation and final Release are included.
func BenchmarkQueueBurstProducers(b *testing.B) {
	for _, producers := range []int{1, 4} {
		for _, retained := range []bool{false, true} {
			b.Run(fmt.Sprintf("producers=%d/retained=%t", producers, retained), func(b *testing.B) {
				opts := append(performanceRingOptions(retained), WithLogger(NewEmptyLogger()))
				q := NewPool(4, opts...)
				var completed sync.WaitGroup
				task := func(context.Context) error { completed.Done(); return nil }
				b.ReportAllocs()
				b.ResetTimer()
				for submitted := 0; submitted < b.N; {
					n := min(1024, b.N-submitted)
					completed.Add(n)
					var senders sync.WaitGroup
					for p := range producers {
						senders.Go(func() {
							for i := p; i < n; i += producers {
								if err := q.QueueTask(task); err != nil {
									b.Error(err)
									completed.Done()
								}
							}
						})
					}
					senders.Wait()
					completed.Wait()
					submitted += n
				}
				q.Release()
				b.StopTimer()
				if got := q.SuccessTasks(); got != uint64(b.N) {
					b.Fatalf("got %d completed, want %d", got, b.N)
				}
			})
		}
	}
}
