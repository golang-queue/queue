package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/appleboy/com/bytesconv"
	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/job"
	"github.com/golang-queue/queue/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type mockMessage struct {
	message string
}

func (m mockMessage) Bytes() []byte {
	return bytesconv.StrToBytes(m.message)
}

func (m mockMessage) Payload() []byte {
	return bytesconv.StrToBytes(m.message)
}

func TestNewQueueWithZeroWorker(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	q, err := NewQueue()
	assert.Error(t, err)
	assert.Nil(t, q)

	w := mocks.NewMockWorker(controller)
	w.EXPECT().Shutdown().Return(nil)
	w.EXPECT().Request().Return(nil, nil).AnyTimes()
	q, err = NewQueue(
		WithWorker(w),
		WithWorkerCount(0),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int64(0), q.BusyWorkers())
	q.Release()
}

func TestNewQueueWithDefaultWorker(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	q, err := NewQueue()
	assert.Error(t, err)
	assert.Nil(t, q)

	w := mocks.NewMockWorker(controller)
	m := mocks.NewMockTaskMessage(controller)
	m.EXPECT().Bytes().Return([]byte("test")).AnyTimes()
	m.EXPECT().Payload().Return([]byte("test")).AnyTimes()
	w.EXPECT().Shutdown().Return(nil)
	w.EXPECT().Request().Return(m, nil).AnyTimes()
	w.EXPECT().Run(context.Background(), m).Return(nil).AnyTimes()
	q, err = NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	q.Start()
	q.Release()
	assert.Equal(t, int64(0), q.BusyWorkers())
}

func TestHandleTimeout(t *testing.T) {
	m := &job.Message{
		Timeout: 100 * time.Millisecond,
		Body:    []byte("foo"),
	}
	w := NewRing(
		WithFn(func(ctx context.Context, m core.TaskMessage) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}),
	)

	q, err := NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	err = q.handle(m)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	done := make(chan error)
	go func() {
		done <- q.handle(m)
	}()

	err = <-done
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestJobComplete(t *testing.T) {
	m := &job.Message{
		Timeout: 100 * time.Millisecond,
		Body:    []byte("foo"),
	}
	w := NewRing(
		WithFn(func(ctx context.Context, m core.TaskMessage) error {
			return errors.New("job completed")
		}),
	)

	q, err := NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	err = q.handle(m)
	assert.Error(t, err)
	assert.Equal(t, errors.New("job completed"), err)

	m = &job.Message{
		Timeout: 250 * time.Millisecond,
		Body:    []byte("foo"),
	}

	w = NewRing(
		WithFn(func(ctx context.Context, m core.TaskMessage) error {
			time.Sleep(200 * time.Millisecond)
			return errors.New("job completed")
		}),
	)

	q, err = NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	err = q.handle(m)
	assert.Error(t, err)
	assert.Equal(t, errors.New("job completed"), err)
}

func TestTaskJobComplete(t *testing.T) {
	m := &job.Message{
		Timeout: 100 * time.Millisecond,
		Task: func(ctx context.Context) error {
			return errors.New("job completed")
		},
	}
	w := NewRing()

	q, err := NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	err = q.handle(m)
	assert.Error(t, err)
	assert.Equal(t, errors.New("job completed"), err)

	m = &job.Message{
		Timeout: 250 * time.Millisecond,
		Task: func(ctx context.Context) error {
			return nil
		},
	}

	assert.NoError(t, q.handle(m))

	// job timeout
	m = &job.Message{
		Timeout: 50 * time.Millisecond,
		Task: func(ctx context.Context) error {
			time.Sleep(60 * time.Millisecond)
			return nil
		},
	}
	assert.Equal(t, context.DeadlineExceeded, q.handle(m))
}

func TestMockWorkerAndMessage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mocks.NewMockTaskMessage(controller)

	w := mocks.NewMockWorker(controller)
	w.EXPECT().Shutdown().Return(nil)
	w.EXPECT().Request().DoAndReturn(func() (core.TaskMessage, error) {
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
