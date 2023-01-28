package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-queue/queue/core"

	"github.com/stretchr/testify/assert"
)

func TestMetricData(t *testing.T) {
	w := NewRing(
		WithFn(func(ctx context.Context, m core.QueuedMessage) error {
			switch string(m.Bytes()) {
			case "foo1":
				panic("missing something")
			case "foo2":
				return errors.New("missing something")
			case "foo3":
				return nil
			}
			return nil
		}),
	)
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(4),
	)
	assert.NoError(t, err)
	assert.NoError(t, q.Queue(mockMessage{
		message: "foo1",
	}))
	assert.NoError(t, q.Queue(mockMessage{
		message: "foo2",
	}))
	assert.NoError(t, q.Queue(mockMessage{
		message: "foo3",
	}))
	assert.NoError(t, q.Queue(mockMessage{
		message: "foo4",
	}))
	q.Start()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 4, q.SubmittedTasks())
	assert.Equal(t, 2, q.SuccessTasks())
	assert.Equal(t, 2, q.FailureTasks())
	q.Release()
}
