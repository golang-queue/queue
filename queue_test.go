package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type mockMessage struct {
	message string
}

func (m mockMessage) Bytes() []byte {
	return []byte(m.message)
}

func TestNewQueue(t *testing.T) {
	q, err := NewQueue()
	assert.Error(t, err)
	assert.Nil(t, q)

	w := &emptyWorker{}
	q, err = NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	assert.Equal(t, uint64(0), w.BusyWorkers())
	q.Shutdown()
	q.Wait()
}

func TestWorkerNum(t *testing.T) {
	w := &messageWorker{
		messages: make(chan QueuedMessage, 100),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	q.Start()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 4, q.Workers())
	assert.Equal(t, uint64(0), w.BusyWorkers())
	q.Shutdown()
	q.Wait()
}

func TestShtdonwOnce(t *testing.T) {
	w := &messageWorker{
		messages: make(chan QueuedMessage, 100),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, q.Workers())
	q.Shutdown()
	// don't panic here
	q.Shutdown()
	q.Wait()
	assert.Equal(t, 0, q.Workers())
}

func TestWorkerStatus(t *testing.T) {
	m := mockMessage{
		message: "foobar",
	}
	w := &messageWorker{
		messages: make(chan QueuedMessage, 100),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.QueueWithTimeout(10*time.Millisecond, m))
	assert.NoError(t, q.QueueWithTimeout(10*time.Millisecond, m))
	assert.Equal(t, 100, q.Capacity())
	assert.Equal(t, 4, q.Usage())
	q.Start()
	time.Sleep(20 * time.Millisecond)
	q.Shutdown()
	q.Wait()
}

func TestWorkerPanic(t *testing.T) {
	w := &messageWorker{
		messages: make(chan QueuedMessage, 10),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(5),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	assert.NoError(t, q.Queue(mockMessage{
		message: "foobar",
	}))
	assert.NoError(t, q.Queue(mockMessage{
		message: "foobar",
	}))
	assert.NoError(t, q.Queue(mockMessage{
		message: "panic",
	}))
	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 5, q.Workers())
	q.Shutdown()
	q.Wait()
	assert.Equal(t, 0, q.Workers())
}

func TestCapacityReached(t *testing.T) {
	w := &messageWorker{
		messages: make(chan QueuedMessage, 1),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(5),
		WithLogger(NewEmptyLogger()),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	assert.NoError(t, q.Queue(mockMessage{
		message: "foobar",
	}))
	// max capacity reached
	assert.Error(t, q.Queue(mockMessage{
		message: "foobar",
	}))
}

func TestCloseQueueAfterShutdown(t *testing.T) {
	w := &messageWorker{
		messages: make(chan QueuedMessage, 10),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(5),
		WithLogger(NewEmptyLogger()),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	assert.NoError(t, q.Queue(mockMessage{
		message: "foobar",
	}))
	q.Shutdown()
	err = q.Queue(mockMessage{
		message: "foobar",
	})
	assert.Error(t, err)
	assert.Equal(t, ErrQueueShutdown, err)
	err = q.QueueWithTimeout(10*time.Millisecond, mockMessage{
		message: "foobar",
	})
	assert.Error(t, err)
	assert.Equal(t, ErrQueueShutdown, err)
}

func TestQueueTaskJob(t *testing.T) {
	w := &taskWorker{
		messages: make(chan QueuedMessage, 10),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(5),
		WithLogger(NewLogger()),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)
	q.Start()
	assert.NoError(t, q.QueueTask(func(ctx context.Context) error {
		time.Sleep(120 * time.Millisecond)
		q.logger.Info("Add new task 1")
		return nil
	}))
	assert.NoError(t, q.QueueTask(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		q.logger.Info("Add new task 2")
		return nil
	}))
	assert.NoError(t, q.QueueTaskWithTimeout(50*time.Millisecond, func(ctx context.Context) error {
		time.Sleep(80 * time.Millisecond)
		return nil
	}))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, uint64(0), w.BusyWorkers())
	q.Shutdown()
	assert.Equal(t, ErrQueueShutdown, q.QueueTask(func(ctx context.Context) error {
		return nil
	}))
	q.Wait()
}
