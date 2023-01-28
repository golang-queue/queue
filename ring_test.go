package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/job"
	"github.com/golang-queue/queue/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMaxCapacity(t *testing.T) {
	w := NewRing(WithQueueSize(2))

	assert.NoError(t, w.Queue(&mockMessage{}))
	assert.NoError(t, w.Queue(&mockMessage{}))
	assert.Error(t, w.Queue(&mockMessage{}))

	err := w.Queue(&mockMessage{})
	assert.Equal(t, ErrMaxCapacity, err)
}

func TestCustomFuncAndWait(t *testing.T) {
	m := mockMessage{
		message: "foo",
	}
	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		}),
	)
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
		WithLogger(NewLogger()),
	)
	assert.NoError(t, err)
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, int(q.metric.BusyWorkers()))
	time.Sleep(600 * time.Millisecond)
	q.Shutdown()
	q.Wait()
	// you will see the execute time > 1000ms
}

func TestEnqueueJobAfterShutdown(t *testing.T) {
	m := mockMessage{
		message: "foo",
	}
	w := NewRing()
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	q.Shutdown()
	// can't queue task after shutdown
	err = q.Queue(m)
	assert.Error(t, err)
	assert.Equal(t, ErrQueueShutdown, err)
	q.Wait()
}

func TestJobReachTimeout(t *testing.T) {
	m := mockMessage{
		message: "foo",
	}
	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
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
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	assert.NoError(t, q.Queue(m, job.AllowOption{Timeout: job.Time(30 * time.Millisecond)}))
	q.Start()
	time.Sleep(50 * time.Millisecond)
	q.Release()
}

func TestCancelJobAfterShutdown(t *testing.T) {
	m := mockMessage{
		message: "foo",
	}
	w := NewRing(
		WithLogger(NewEmptyLogger()),
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
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
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	assert.NoError(t, q.Queue(m, job.AllowOption{Timeout: job.Time(100 * time.Millisecond)}))
	assert.NoError(t, q.Queue(m, job.AllowOption{Timeout: job.Time(100 * time.Millisecond)}))
	q.Start()
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 2, int(q.metric.busyWorkers))
	q.Release()
}

func TestGoroutineLeak(t *testing.T) {
	w := NewRing(
		WithLogger(NewLogger()),
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			for {
				select {
				case <-ctx.Done():
					if errors.Is(ctx.Err(), context.Canceled) {
						log.Println("queue has been shutdown and cancel the job: " + string(m.Bytes()))
					} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						log.Println("job deadline exceeded: " + string(m.Bytes()))
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
	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(10),
	)
	assert.NoError(t, err)
	for i := 0; i < 400; i++ {
		m := mockMessage{
			message: fmt.Sprintf("new message: %d", i+1),
		}

		assert.NoError(t, q.Queue(m))
	}

	q.Start()
	time.Sleep(1 * time.Second)
	q.Release()
	fmt.Println("number of goroutines:", runtime.NumGoroutine())
}

func TestGoroutinePanic(t *testing.T) {
	m := mockMessage{
		message: "foo",
	}
	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			panic("missing something")
		}),
	)
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	assert.NoError(t, q.Queue(m))
	q.Start()
	time.Sleep(10 * time.Millisecond)
	q.Release()
}

func TestIncreaseWorkerCount(t *testing.T) {
	w := NewRing(
		WithLogger(NewEmptyLogger()),
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		}),
	)
	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(5),
	)
	assert.NoError(t, err)

	for i := 1; i <= 10; i++ {
		m := mockMessage{
			message: fmt.Sprintf("new message: %d", i),
		}
		assert.NoError(t, q.Queue(m))
	}

	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 5, q.BusyWorkers())
	q.UpdateWorkerCount(10)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 10, q.BusyWorkers())
	q.Release()
}

func TestDecreaseWorkerCount(t *testing.T) {
	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		}),
	)
	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(5),
	)
	assert.NoError(t, err)

	for i := 1; i <= 10; i++ {
		m := mockMessage{
			message: fmt.Sprintf("test message: %d", i),
		}
		assert.NoError(t, q.Queue(m))
	}

	q.Start()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 5, q.BusyWorkers())
	q.UpdateWorkerCount(3)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 3, q.BusyWorkers())
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, q.BusyWorkers())
	q.Release()
}

func TestHandleAllJobBeforeShutdownRing(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mocks.NewMockQueuedMessage(controller)

	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}),
	)

	done := make(chan struct{})
	assert.NoError(t, w.Queue(m))
	assert.NoError(t, w.Queue(m))
	go func() {
		assert.NoError(t, w.Shutdown())
		done <- struct{}{}
	}()
	time.Sleep(50 * time.Millisecond)
	task, err := w.Request()
	assert.NotNil(t, task)
	assert.NoError(t, err)
	task, err = w.Request()
	assert.NotNil(t, task)
	assert.NoError(t, err)
	task, err = w.Request()
	assert.Nil(t, task)
	assert.True(t, errors.Is(err, ErrQueueHasBeenClosed))
	<-done
}

func TestHandleAllJobBeforeShutdownRingInQueue(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mocks.NewMockQueuedMessage(controller)
	m.EXPECT().Bytes().Return([]byte("test")).AnyTimes()

	messages := make(chan string, 10)

	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			time.Sleep(10 * time.Millisecond)
			messages <- string(m.Bytes())
			return nil
		}),
	)

	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(1),
	)
	assert.NoError(t, err)

	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.Len(t, messages, 0)
	q.Start()
	q.Release()
	assert.Len(t, messages, 2)
}

func TestRetryCountWithNewMessage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mocks.NewMockQueuedMessage(controller)
	m.EXPECT().Bytes().Return([]byte("test")).AnyTimes()

	messages := make(chan string, 10)
	keep := make(chan struct{})
	count := 1

	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			if count%3 != 0 {
				count++
				return errors.New("count not correct")
			}
			close(keep)
			messages <- string(m.Bytes())
			return nil
		}),
	)

	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(1),
	)
	assert.NoError(t, err)

	assert.NoError(t, q.Queue(
		m,
		job.AllowOption{
			RetryCount: job.Int64(3),
			RetryDelay: job.Time(50 * time.Millisecond),
		},
	))
	assert.Len(t, messages, 0)
	q.Start()
	// wait retry twice.
	<-keep
	q.Release()
	assert.Len(t, messages, 1)
}

func TestRetryCountWithNewTask(t *testing.T) {
	messages := make(chan string, 10)
	count := 1

	w := NewRing()

	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(1),
	)
	assert.NoError(t, err)

	keep := make(chan struct{})

	assert.NoError(t, q.QueueTask(
		func(ctx context.Context) error {
			if count%3 != 0 {
				count++
				return errors.New("count not correct")
			}
			close(keep)
			messages <- "foobar"
			return nil
		},
		job.AllowOption{
			RetryCount: job.Int64(3),
			RetryDelay: job.Time(50 * time.Millisecond),
		},
	))
	assert.Len(t, messages, 0)
	q.Start()
	// wait retry twice.
	<-keep
	q.Release()
	assert.Len(t, messages, 1)
}

func TestCancelRetryCountWithNewTask(t *testing.T) {
	messages := make(chan string, 10)
	count := 1

	w := NewRing()

	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(1),
	)
	assert.NoError(t, err)

	assert.NoError(t, q.QueueTask(
		func(ctx context.Context) error {
			if count%3 != 0 {
				count++
				q.logger.Info("add count")
				return errors.New("count not correct")
			}
			messages <- "foobar"
			return nil
		},
		job.AllowOption{
			RetryCount: job.Int64(3),
			RetryDelay: job.Time(100 * time.Millisecond),
		},
	))
	assert.Len(t, messages, 0)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	q.Release()
	assert.Len(t, messages, 0)
	assert.Equal(t, 2, count)
}

func TestCancelRetryCountWithNewMessage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mocks.NewMockQueuedMessage(controller)
	m.EXPECT().Bytes().Return([]byte("test")).AnyTimes()

	messages := make(chan string, 10)
	count := 1

	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			if count%3 != 0 {
				count++
				return errors.New("count not correct")
			}
			messages <- string(m.Bytes())
			return nil
		}),
	)

	q, err := NewQueue(
		WithLogger(NewLogger()),
		WithWorker(w),
		WithWorkerCount(1),
	)
	assert.NoError(t, err)

	assert.NoError(t, q.Queue(
		m,
		job.AllowOption{
			RetryCount: job.Int64(3),
			RetryDelay: job.Time(100 * time.Millisecond),
		},
	))
	assert.Len(t, messages, 0)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	q.Release()
	assert.Len(t, messages, 0)
	assert.Equal(t, 2, count)
}

func TestErrNoTaskInQueue(t *testing.T) {
	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			return nil
		}),
	)
	task, err := w.Request()
	assert.Nil(t, task)
	assert.Error(t, err)
	assert.Equal(t, ErrNoTaskInQueue, err)
}
