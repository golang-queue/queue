package queue

import (
	"context"
	"fmt"
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
