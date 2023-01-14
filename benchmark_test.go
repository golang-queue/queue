package queue

import (
	"context"
	"testing"
	"time"

	"github.com/golang-queue/queue/job"
)

func BenchmarkNewCusumer(b *testing.B) {
	b.ReportAllocs()
	pool := NewConsumer(
		WithQueueSize(b.N),
		WithLogger(emptyLogger{}),
	)
	message := job.NewTask(func(context.Context) error {
		return nil
	},
		job.AllowOption{
			RetryCount: job.Int64(100),
			RetryDelay: job.Time(30 * time.Millisecond),
			Timeout:    job.Time(3 * time.Millisecond),
		},
	)

	for n := 0; n < b.N; n++ {
		_ = pool.Queue(message)
		_, _ = pool.Request()
	}
}

// func BenchmarkNewCusumerSlice(b *testing.B) {
// 	b.ReportAllocs()
// 	pool := NewConsumerSlice(
// 		WithQueueSize(b.N),
// 		WithLogger(emptyLogger{}),
// 	)
// 	message := job.NewTask(func(context.Context) error {
// 		return nil
// 	},
// 		job.AllowOption{
// 			RetryCount: job.Int64(100),
// 			RetryDelay: job.Time(30 * time.Millisecond),
// 			Timeout:    job.Time(3 * time.Millisecond),
// 		},
// 	)

// 	for n := 0; n < b.N; n++ {
// 		_ = pool.Queue(message)
// 		_, _ = pool.Request()
// 	}
// }

func BenchmarkNewCusumerList(b *testing.B) {
	b.ReportAllocs()
	pool := NewConsumerList(
		WithQueueSize(b.N),
		WithLogger(emptyLogger{}),
	)
	message := job.NewTask(func(context.Context) error {
		return nil
	},
		job.AllowOption{
			RetryCount: job.Int64(100),
			RetryDelay: job.Time(30 * time.Millisecond),
			Timeout:    job.Time(3 * time.Millisecond),
		},
	)

	for n := 0; n < b.N; n++ {
		_ = pool.Queue(message)
		_, _ = pool.Request()
	}
}

func BenchmarkNewCusumerRing(b *testing.B) {
	b.ReportAllocs()
	pool := NewConsumerRing(
		WithQueueSize(b.N),
		WithLogger(emptyLogger{}),
	)
	message := job.NewTask(func(context.Context) error {
		return nil
	},
		job.AllowOption{
			RetryCount: job.Int64(100),
			RetryDelay: job.Time(30 * time.Millisecond),
			Timeout:    job.Time(3 * time.Millisecond),
		},
	)

	for n := 0; n < b.N; n++ {
		_ = pool.Queue(message)
		_, _ = pool.Request()
	}
}

// func BenchmarkQueueTask(b *testing.B) {
// 	b.ReportAllocs()
// 	q := NewPool(
// 		0,
// 		WithLogger(emptyLogger{}),
// 	)
// 	defer q.Release()
// 	for n := 0; n < b.N; n++ {
// 		_ = q.QueueTask(func(context.Context) error {
// 			return nil
// 		})
// 	}
// }

// func BenchmarkQueueTaskList(b *testing.B) {
// 	b.ReportAllocs()
// 	w := NewConsumerList(
// 		WithLogger(emptyLogger{}),
// 	)
// 	q, err := NewQueue(
// 		WithWorker(w),
// 		WithLogger(emptyLogger{}),
// 	)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer q.Release()
// 	for n := 0; n < b.N; n++ {
// 		_ = q.QueueTask(func(context.Context) error {
// 			return nil
// 		})
// 	}
// }

// func BenchmarkQueue(b *testing.B) {
// 	b.ReportAllocs()
// 	m := &mockMessage{
// 		message: "foo",
// 	}
// 	q := NewPool(5, WithLogger(emptyLogger{}))
// 	defer q.Release()
// 	for n := 0; n < b.N; n++ {
// 		_ = q.Queue(m)
// 	}
// }

// func BenchmarkConsumerPayload(b *testing.B) {
// 	b.ReportAllocs()

// 	task := &job.Message{
// 		Timeout: 100 * time.Millisecond,
// 		Payload: []byte(`{"timeout":3600000000000}`),
// 	}
// 	w := NewConsumer(
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

// func BenchmarkConsumerTask(b *testing.B) {
// 	b.ReportAllocs()

// 	task := &job.Message{
// 		Timeout: 100 * time.Millisecond,
// 		Task: func(_ context.Context) error {
// 			return nil
// 		},
// 	}
// 	w := NewConsumer(
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

// func BenchmarkPool(b *testing.B) {
// 	b.ReportAllocs()

// 	pool := NewPool(0, WithLogger(emptyLogger{}))
// 	defer pool.Release()
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		wg := sync.WaitGroup{}
// 		wg.Add(100)
// 		for i := 0; i < 100; i++ {
// 			_ = pool.QueueTask(func(ctx context.Context) error {
// 				wg.Done()
// 				return nil
// 			})
// 		}
// 		pool.Start()
// 		wg.Wait()
// 	}
// }

// func BenchmarkNoPool(b *testing.B) {
// 	b.ReportAllocs()

// 	for i := 0; i < b.N; i++ {
// 		wg := sync.WaitGroup{}
// 		wg.Add(100)
// 		for i := 0; i < 100; i++ {
// 			go func() {
// 				wg.Done()
// 			}()
// 		}
// 		wg.Wait()
// 	}
// }
