package queue

import (
	"context"
	"testing"
)

func BenchmarkQueueTask(b *testing.B) {
	b.ReportAllocs()
	q := NewPool(5)
	defer q.Release()
	for n := 0; n < b.N; n++ {
		_ = q.QueueTask(func(context.Context) error {
			return nil
		})
	}
}

func BenchmarkQueue(b *testing.B) {
	b.ReportAllocs()
	m := &mockMessage{
		message: "foo",
	}
	q := NewPool(5)
	defer q.Release()
	for n := 0; n < b.N; n++ {
		_ = q.Queue(m)
	}
}
