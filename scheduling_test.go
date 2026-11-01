package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/job"

	"github.com/stretchr/testify/require"
)

func receiveWithin[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for queue progress")
		var zero T
		return zero
	}
}

func TestQueueResizePauseAndDrain(t *testing.T) {
	const total = 16
	started := make(chan int, total)
	completed := make(chan struct{}, total)
	release := make(chan struct{})
	var releaseOnce sync.Once
	q := NewPool(2, WithLogger(NewEmptyLogger()), WithAfterFn(func() { completed <- struct{}{} }))
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		q.UpdateWorkerCount(2)
		q.Release()
	})
	for id := range total {
		require.NoError(t, q.QueueTask(func(context.Context) error {
			started <- id
			<-release
			return nil
		}))
	}
	seen := make(map[int]bool)
	receiveStarted := func(n int) {
		for range n {
			id := receiveWithin(t, started)
			require.False(t, seen[id], "task executed twice")
			seen[id] = true
		}
	}
	receiveStarted(2)
	q.UpdateWorkerCount(4)
	receiveStarted(2)
	require.EqualValues(t, 4, q.BusyWorkers())
	q.UpdateWorkerCount(1)
	for range 4 {
		select {
		case release <- struct{}{}:
		case <-time.After(5 * time.Second):
			t.Fatal("workers did not finish during scale-down")
		}
	}
	for range 4 {
		receiveWithin(t, completed)
	}
	receiveStarted(1)
	require.EqualValues(t, 1, q.BusyWorkers())
	q.UpdateWorkerCount(0)
	release <- struct{}{}
	receiveWithin(t, completed)
	select {
	case <-started:
		t.Fatal("task started while paused")
	case <-time.After(20 * time.Millisecond):
	}
	q.UpdateWorkerCount(2)
	receiveStarted(2)
	require.EqualValues(t, 2, q.BusyWorkers())
	releaseOnce.Do(func() { close(release) })
	q.Release()
	receiveStarted(total - len(seen))
	require.Len(t, seen, total)
	require.EqualValues(t, total, q.SuccessTasks())
	require.EqualValues(t, 0, q.BusyWorkers())
}

func TestReusableWorkerOutcomes(t *testing.T) {
	completed := make(chan struct{}, 5)
	lateRelease := make(chan struct{})
	lateDone := make(chan struct{})
	q := NewPool(1, WithLogger(NewEmptyLogger()), WithAfterFn(func() { completed <- struct{}{} }))
	t.Cleanup(func() {
		close(lateRelease)
		q.Release()
	})
	wantErr := errors.New("task failed")
	require.NoError(t, q.QueueTask(func(context.Context) error { return wantErr }))
	require.NoError(t, q.QueueTask(func(context.Context) error { panic("task panic") }))
	var attempts atomic.Int32
	require.NoError(t, q.QueueTask(func(context.Context) error {
		if attempts.Add(1) == 1 {
			return wantErr
		}
		return nil
	}, job.AllowOption{RetryCount: job.Int64(1), RetryDelay: job.Time(time.Millisecond)}))
	require.NoError(t, q.QueueTask(func(context.Context) error {
		defer close(lateDone)
		<-lateRelease // Deliberately ignore cancellation to check timeout isolation.
		return nil
	}, job.AllowOption{Timeout: job.Time(10 * time.Millisecond)}))
	require.NoError(t, q.QueueTask(func(context.Context) error { return nil }))
	for range 5 {
		receiveWithin(t, completed)
	}
	require.EqualValues(t, 2, attempts.Load())
	require.EqualValues(t, 2, q.SuccessTasks())
	require.EqualValues(t, 3, q.FailureTasks())
	require.EqualValues(t, 5, q.CompletedTasks())
	select {
	case <-lateDone:
		t.Fatal("timeout test handler finished before being released")
	default:
	}
	// Release still completes even though the timed-out handler has not returned.
	q.Release()
}

func TestAfterFnCanWaitForNextTask(t *testing.T) {
	var q *Queue
	var callbacks atomic.Int32
	callbackDone := make(chan error, 1)
	q = NewPool(1, WithLogger(NewEmptyLogger()), WithAfterFn(func() {
		if callbacks.Add(1) != 1 {
			return
		}
		nextTask := make(chan struct{})
		if err := q.QueueTask(func(context.Context) error {
			close(nextTask)
			return nil
		}); err != nil {
			callbackDone <- err
			return
		}
		select {
		case <-nextTask:
			callbackDone <- nil
		case <-time.After(time.Second):
			callbackDone <- errors.New("afterFn blocked the next task")
		}
	}))
	t.Cleanup(q.Release)
	require.NoError(t, q.QueueTask(func(context.Context) error { return nil }))
	require.NoError(t, receiveWithin(t, callbackDone))
	q.Release()
	require.EqualValues(t, 2, q.SuccessTasks())
	require.EqualValues(t, 2, callbacks.Load())
}

func TestReleaseWaitsForAfterFn(t *testing.T) {
	entered := make(chan struct{})
	unblock := make(chan struct{})
	var once sync.Once
	q := NewPool(1, WithLogger(NewEmptyLogger()), WithAfterFn(func() {
		close(entered)
		<-unblock
	}))
	t.Cleanup(func() {
		once.Do(func() { close(unblock) })
		q.Release()
	})
	require.NoError(t, q.QueueTask(func(context.Context) error { return nil }))
	receiveWithin(t, entered)
	released := make(chan struct{})
	go func() {
		q.Release()
		close(released)
	}()
	select {
	case <-released:
		t.Fatal("Release returned before afterFn finished")
	case <-time.After(20 * time.Millisecond):
	}
	once.Do(func() { close(unblock) })
	receiveWithin(t, released)
}

// blockingWorker models backends whose Request blocks until work or shutdown.
// Counting active calls detects accidental parallel Request calls.
type blockingWorker struct {
	requests chan core.TaskMessage
	closed   chan struct{}
	entered  chan struct{}
	once     sync.Once
	active   atomic.Int32
	parallel atomic.Bool
}

func (w *blockingWorker) Queue(m core.TaskMessage) error {
	w.requests <- m
	return nil
}

func (w *blockingWorker) Request() (core.TaskMessage, error) {
	if w.active.Add(1) != 1 {
		w.parallel.Store(true)
	}
	defer w.active.Add(-1)
	select {
	case w.entered <- struct{}{}:
	default:
	}
	select {
	case m := <-w.requests:
		return m, nil
	case <-w.closed:
		return nil, ErrQueueHasBeenClosed
	}
}

func (w *blockingWorker) Run(context.Context, core.TaskMessage) error { return nil }

func (w *blockingWorker) Shutdown() error {
	w.once.Do(func() { close(w.closed) })
	return nil
}

func TestBlockingBackendAndRepeatedStart(t *testing.T) {
	w := &blockingWorker{
		requests: make(chan core.TaskMessage, 64),
		closed:   make(chan struct{}),
		entered:  make(chan struct{}, 128),
	}
	completed := make(chan struct{}, 64)
	q, err := NewQueue(WithWorker(w), WithWorkerCount(4), WithLogger(NewEmptyLogger()),
		WithAfterFn(func() { completed <- struct{}{} }))
	require.NoError(t, err)
	t.Cleanup(q.Release)
	q.Start()
	receiveWithin(t, w.entered)
	q.Start()
	for range 64 {
		require.NoError(t, q.QueueTask(func(context.Context) error { return nil }))
	}
	for range 64 {
		receiveWithin(t, completed)
	}
	released := make(chan struct{})
	go func() {
		q.Release()
		close(released)
	}()
	receiveWithin(t, released)
	require.False(t, w.parallel.Load(), "backend Request must remain serialized")
	require.EqualValues(t, 64, q.SuccessTasks())
}
