package queue

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMockWorkerAndMessage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mocks.NewMockQueuedMessage(controller)

	w := mocks.NewMockWorker(controller)
	w.EXPECT().Shutdown().Return(nil)
	w.EXPECT().Request().DoAndReturn(func() (core.QueuedMessage, error) {
		return m, errors.New("nil")
	})

	q, err := NewQueue(
		WithWorker(w),
		WithWorkerCount(1),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)
	q.Start()
	time.Sleep(50 * time.Millisecond)
	q.Release()
}
