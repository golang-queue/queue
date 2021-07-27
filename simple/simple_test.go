package simple

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/appleboy/queue"

	"github.com/stretchr/testify/assert"
)

type mockMessage struct {
	msg string
}

func (m mockMessage) Bytes() []byte {
	return []byte(m.msg)
}

func TestQueueUsage(t *testing.T) {
	w := NewWorker()
	assert.Equal(t, defaultQueueSize, w.Capacity())
	assert.Equal(t, 0, w.Usage())

	assert.NoError(t, w.Queue(&mockMessage{}))
	assert.Equal(t, 1, w.Usage())
}

func TestMaxCapacity(t *testing.T) {
	w := NewWorker(WithQueueNum(2))
	assert.Equal(t, 2, w.Capacity())
	assert.Equal(t, 0, w.Usage())

	assert.NoError(t, w.Queue(&mockMessage{}))
	assert.Equal(t, 1, w.Usage())
	assert.NoError(t, w.Queue(&mockMessage{}))
	assert.Equal(t, 2, w.Usage())
	assert.Error(t, w.Queue(&mockMessage{}))
	assert.Equal(t, 2, w.Usage())

	err := w.Queue(&mockMessage{})
	assert.Equal(t, errMaxCapacity, err)
}

func TestCustomFuncAndWait(t *testing.T) {
	m := mockMessage{
		msg: "foo",
	}
	w := NewWorker(
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		}),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	time.Sleep(600 * time.Millisecond)
	q.Shutdown()
	q.Wait()
	// you will see the execute time > 1000ms
}

func TestEnqueueJobAfterShutdown(t *testing.T) {
	m := mockMessage{
		msg: "foo",
	}
	w := NewWorker()
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	q.Shutdown()
	// can't queue task after shutdown
	err = q.Queue(m)
	assert.Error(t, err)
	assert.Equal(t, queue.ErrQueueShutdown, err)
	q.Wait()
}

func TestWorkerNumAfterShutdown(t *testing.T) {
	w := NewWorker()
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	q.Start()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 4, q.Workers())
	q.Shutdown()
	q.Wait()
	assert.Equal(t, 0, q.Workers())
	// show queue has been shutdown meesgae
	q.Start()
	q.Start()
	assert.Equal(t, 0, q.Workers())
}

func TestJobReachTimeout(t *testing.T) {
	m := mockMessage{
		msg: "foo",
	}
	w := NewWorker(
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			for {
				select {
				case <-ctx.Done():
					log.Println("get data:", string(m.Bytes()))
					if errors.Is(ctx.Err(), context.Canceled) {
						log.Println("queue has been shutdown and cancel the job")
					} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						log.Println("job deadline exceeded")
					}
					return nil
				default:
				}
				time.Sleep(50 * time.Millisecond)
			}
		}),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	assert.NoError(t, q.QueueWithTimeout(30*time.Millisecond, m))
	time.Sleep(50 * time.Millisecond)
	q.Shutdown()
	q.Wait()
}

func TestCancelJobAfterShutdown(t *testing.T) {
	m := mockMessage{
		msg: "foo",
	}
	w := NewWorker(
		WithLogger(queue.NewEmptyLogger()),
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			for {
				select {
				case <-ctx.Done():
					log.Println("get data:", string(m.Bytes()))
					if errors.Is(ctx.Err(), context.Canceled) {
						log.Println("queue has been shutdown and cancel the job")
					} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						log.Println("job deadline exceeded")
					}
					return nil
				default:
				}
				time.Sleep(50 * time.Millisecond)
			}
		}),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	assert.NoError(t, q.QueueWithTimeout(100*time.Millisecond, m))
	q.Shutdown()
	q.Wait()
}

func TestGoroutineLeak(t *testing.T) {
	m := mockMessage{
		msg: "foo",
	}
	w := NewWorker(
		WithLogger(queue.NewEmptyLogger()),
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			for {
				select {
				case <-ctx.Done():
					log.Println("get data:", string(m.Bytes()))
					if errors.Is(ctx.Err(), context.Canceled) {
						log.Println("queue has been shutdown and cancel the job")
					} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						log.Println("job deadline exceeded")
					}
					return nil
				default:
					log.Println("get data:", string(m.Bytes()))
					time.Sleep(50 * time.Millisecond)
					return nil
				}
			}
		}),
	)
	q, err := queue.NewQueue(
		queue.WithLogger(queue.NewEmptyLogger()),
		queue.WithWorker(w),
		queue.WithWorkerCount(10),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	for i := 0; i < 500; i++ {
		m.msg = fmt.Sprintf("foobar: %d", i+1)
		assert.NoError(t, q.Queue(m))
	}
	time.Sleep(2 * time.Second)
	q.Shutdown()
	q.Wait()
	fmt.Println("number of goroutines:", runtime.NumGoroutine())
}

func TestGoroutinePanic(t *testing.T) {
	m := mockMessage{
		msg: "foo",
	}
	w := NewWorker(
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			panic("missing something")
		}),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	assert.NoError(t, q.Queue(m))
	time.Sleep(50 * time.Millisecond)
	q.Shutdown()
	q.Wait()
}
