package queue

import (
	"context"
	"testing"
	"time"

	"github.com/golang-queue/queue/core"
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

func BenchmarkConsumerRunUnmarshal(b *testing.B) {
	b.ReportAllocs()

	job := &Job{
		Timeout: 100 * time.Millisecond,
		Payload: []byte(`{"timeout":3600000000000}`),
	}
	w := NewConsumer(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			return nil
		}),
	)

	for n := 0; n < b.N; n++ {
		_ = w.Run(job)
	}
}

func BenchmarkConsumerRunTask(b *testing.B) {
	b.ReportAllocs()

	job := &Job{
		Timeout: 100 * time.Millisecond,
		Task: func(_ context.Context) error {
			return nil
		},
	}
	w := NewConsumer(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			return nil
		}),
	)

	for n := 0; n < b.N; n++ {
		_ = w.Run(job)
	}
}
