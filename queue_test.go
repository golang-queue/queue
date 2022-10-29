package queue

import (
	"testing"
	"time"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/mocks"

	"github.com/golang/mock/gomock"
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

func TestNewQueueWithZeroWorker(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	q, err := NewQueue()
	assert.Error(t, err)
	assert.Nil(t, q)

	w := mocks.NewMockWorker(controller)
	w.EXPECT().Shutdown().Return(nil)
	q, err = NewQueue(
		WithWorker(w),
		WithWorkerCount(0),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, q.BusyWorkers())
	q.Release()
}

func TestNewQueueWithDefaultWorker(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	q, err := NewQueue()
	assert.Error(t, err)
	assert.Nil(t, q)

	w := mocks.NewMockWorker(controller)
	m := mocks.NewMockQueuedMessage(controller)
	m.EXPECT().Bytes().Return([]byte("test")).AnyTimes()
	w.EXPECT().Shutdown().Return(nil)
	w.EXPECT().Request().Return(m, nil).AnyTimes()
	w.EXPECT().Run(m).Return(nil).AnyTimes()
	q, err = NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	q.Release()
	assert.Equal(t, 0, q.BusyWorkers())
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
	err = q.Queue(mockMessage{
		message: "foobar",
	}, WithTimeout(10*time.Millisecond))
	assert.Error(t, err)
	assert.Equal(t, ErrQueueShutdown, err)
}
