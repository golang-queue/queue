package queue

import (
	"testing"
	"time"

	"github.com/golang-queue/queue/core"

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
	assert.Equal(t, 0, q.BusyWorkers())
	q.Shutdown()
	q.Wait()
}

func TestShtdonwOnce(t *testing.T) {
	w := &messageWorker{
		messages: make(chan core.QueuedMessage, 100),
	}
	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(2),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	assert.Equal(t, 0, q.BusyWorkers())
	q.Shutdown()
	// don't panic here
	q.Shutdown()
	q.Wait()
	assert.Equal(t, 0, q.BusyWorkers())
}

func TestCapacityReached(t *testing.T) {
	w := &messageWorker{
		messages: make(chan core.QueuedMessage, 1),
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
		messages: make(chan core.QueuedMessage, 10),
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
