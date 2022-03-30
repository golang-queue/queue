package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
	q.Shutdown()
	assert.Equal(t, ErrQueueShutdown, q.QueueTask(func(ctx context.Context) error {
		return nil
	}))
	q.Wait()
}
