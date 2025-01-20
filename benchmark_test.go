package queue

import (
	"context"
	"testing"
	"time"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/job"
)

var count = 1

type testqueue interface {
	Queue(task core.QueuedMessage) error
	Request() (core.QueuedMessage, error)
}

func testQueue(b *testing.B, pool testqueue) {
	message := job.NewTask(func(context.Context) error {
		return nil
	},
		job.AllowOption{
			RetryCount: job.Int64(100),
			RetryDelay: job.Time(30 * time.Millisecond),
			Timeout:    job.Time(3 * time.Millisecond),
		},
	)

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < count; i++ {
			_ = pool.Queue(&message)
			_, _ = pool.Request()
		}
	}
}

func BenchmarkNewRing(b *testing.B) {
	pool := NewRing(
		WithQueueSize(b.N*count),
		WithLogger(emptyLogger{}),
	)

	testQueue(b, pool)
}

func BenchmarkQueueTask(b *testing.B) {
	w := NewRing()
	q, _ := NewQueue(
		WithWorker(w),
		WithLogger(emptyLogger{}),
	)
	b.ReportAllocs()
	b.ResetTimer()

	m := job.NewTask(func(context.Context) error {
		return nil
	})

	for n := 0; n < b.N; n++ {
		if err := q.queue(&m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueue(b *testing.B) {
	w := NewRing()
	q, _ := NewQueue(
		WithWorker(w),
		WithLogger(emptyLogger{}),
	)
	b.ReportAllocs()
	b.ResetTimer()

	m := job.NewMessage(&mockMessage{
		message: "foo",
	})

	for n := 0; n < b.N; n++ {
		if err := q.queue(&m); err != nil {
			b.Fatal(err)
		}
	}
}

// func BenchmarkRingPayload(b *testing.B) {
// 	b.ReportAllocs()

// 	task := &job.Message{
// 		Timeout: 100 * time.Millisecond,
// 		Payload: []byte(`{"timeout":3600000000000}`),
// 	}
// 	w := NewRing(
// 		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
// 			return nil
// 		}),
// 	)

// 	q, _ := NewQueue(
// 		WithWorker(w),
// 		WithLogger(emptyLogger{}),
// 	)

// 	for n := 0; n < b.N; n++ {
// 		_ = q.run(task)
// 	}
// }

func BenchmarkRingWithTask(b *testing.B) {
	b.ReportAllocs()

	task := job.Message{
		Timeout: 100 * time.Millisecond,
		Task: func(_ context.Context) error {
			return nil
		},
	}
	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			return nil
		}),
	)

	q, _ := NewQueue(
		WithWorker(w),
		WithLogger(emptyLogger{}),
	)

	for n := 0; n < b.N; n++ {
		_ = q.run(&task)
	}
}
