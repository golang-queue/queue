package queue

import (
	"context"
	"testing"
	"time"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/job"
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

func BenchmarkConsumerPayload(b *testing.B) {
	b.ReportAllocs()

	task := &job.Message{
		Timeout: 100 * time.Millisecond,
		Payload: []byte(`{"timeout":3600000000000}`),
	}
	w := NewConsumer(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			return nil
		}),
	)

	q, _ := NewQueue(
		WithWorker(w),
	)

	for n := 0; n < b.N; n++ {
		_ = q.run(task)
	}
}

func BenchmarkConsumerTask(b *testing.B) {
	b.ReportAllocs()

	task := &job.Message{
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

	q, _ := NewQueue(
		WithWorker(w),
	)

	for n := 0; n < b.N; n++ {
		_ = q.run(task)
	}
}
