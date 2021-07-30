package nats

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

var host = "nats"

type mockMessage struct {
	Message string
}

func (m mockMessage) Bytes() []byte {
	return []byte(m.Message)
}

func TestNATSDefaultFlow(t *testing.T) {
	m := &mockMessage{
		Message: "foo",
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("test"),
		WithQueue("test"),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, q.Queue(m))
	m.Message = "new message"
	assert.NoError(t, q.Queue(m))
	q.Shutdown()
	q.Wait()
}

func TestNATSShutdown(t *testing.T) {
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("test"),
		WithQueue("test"),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(1 * time.Second)
	q.Shutdown()
	// check shutdown once
	q.Shutdown()
	q.Wait()
}

func TestNATSCustomFuncAndWait(t *testing.T) {
	m := &mockMessage{
		Message: "foo",
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("test"),
		WithQueue("test"),
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			log.Println("show message: " + string(m.Bytes()))
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
		Message: "foo",
	}
	w := NewWorker(
		WithAddr(host + ":4222"),
	)
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
	w := NewWorker(
		WithAddr(host + ":4222"),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 4, q.Workers())
	q.Shutdown()
	q.Wait()
	assert.Equal(t, 0, q.Workers())
	q.Start()
	q.Start()
	assert.Equal(t, 0, q.Workers())
}

func TestJobReachTimeout(t *testing.T) {
	m := mockMessage{
		Message: "foo",
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("JobReachTimeout"),
		WithQueue("test"),
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
	assert.NoError(t, q.QueueWithTimeout(20*time.Millisecond, m))
	time.Sleep(100 * time.Millisecond)
	q.Shutdown()
	q.Wait()
}

func TestCancelJobAfterShutdown(t *testing.T) {
	m := mockMessage{
		Message: "test",
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("CancelJob"),
		WithQueue("test"),
		WithLogger(queue.NewLogger()),
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
	assert.NoError(t, q.QueueWithTimeout(150*time.Millisecond, m))
	time.Sleep(100 * time.Millisecond)
	q.Shutdown()
	q.Wait()
}

func TestGoroutineLeak(t *testing.T) {
	m := mockMessage{
		Message: "foo",
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("GoroutineLeak"),
		WithQueue("test"),
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
		m.Message = fmt.Sprintf("foobar: %d", i+1)
		assert.NoError(t, q.Queue(m))
	}
	time.Sleep(400 * time.Millisecond)
	q.Shutdown()
	q.Wait()
	fmt.Println("number of goroutines:", runtime.NumGoroutine())
}

func TestGoroutinePanic(t *testing.T) {
	m := mockMessage{
		Message: "foo",
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("GoroutinePanic"),
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
	assert.NoError(t, q.Queue(m))
	time.Sleep(2 * time.Second)
	q.Shutdown()
	assert.Error(t, q.Queue(m))
	q.Wait()
}

func TestHandleTimeout(t *testing.T) {
	job := queue.Job{
		Timeout: 100 * time.Millisecond,
		Body:    []byte("foo"),
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}),
	)

	err := w.handle(job)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	job = queue.Job{
		Timeout: 150 * time.Millisecond,
		Body:    []byte("foo"),
	}

	w = NewWorker(
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}),
	)

	done := make(chan error)
	go func() {
		done <- w.handle(job)
	}()

	assert.NoError(t, w.Shutdown())

	err = <-done
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestJobComplete(t *testing.T) {
	job := queue.Job{
		Timeout: 100 * time.Millisecond,
		Body:    []byte("foo"),
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			return errors.New("job completed")
		}),
	)

	err := w.handle(job)
	assert.Error(t, err)
	assert.Equal(t, errors.New("job completed"), err)

	job = queue.Job{
		Timeout: 250 * time.Millisecond,
		Body:    []byte("foo"),
	}

	w = NewWorker(
		WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
			time.Sleep(200 * time.Millisecond)
			return errors.New("job completed")
		}),
	)

	done := make(chan error)
	go func() {
		done <- w.handle(job)
	}()

	assert.NoError(t, w.Shutdown())

	err = <-done
	assert.Error(t, err)
	assert.Equal(t, errors.New("job completed"), err)
}
