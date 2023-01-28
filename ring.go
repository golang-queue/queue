package queue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*Ring)(nil)

// Ring for simple queue using buffer channel
type Ring struct {
	sync.Mutex
	taskQueue []core.QueuedMessage
	runFunc   func(context.Context, core.QueuedMessage) error
	capacity  int
	count     int
	head      int
	tail      int
	exit      chan struct{}
	logger    Logger
	stopOnce  sync.Once
	stopFlag  int32
}

// Run to execute new task
func (s *Ring) Run(ctx context.Context, task core.QueuedMessage) error {
	return s.runFunc(ctx, task)
}

// Shutdown the worker
func (s *Ring) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
		return ErrQueueShutdown
	}

	s.stopOnce.Do(func() {
		if s.count > 0 {
			<-s.exit
		}
	})
	return nil
}

// Queue send task to the buffer channel
func (s *Ring) Queue(task core.QueuedMessage) error { //nolint:stylecheck
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}
	if s.capacity > 0 && s.count >= s.capacity {
		return ErrMaxCapacity
	}

	s.Lock()
	if s.count == len(s.taskQueue) {
		s.resize(s.count * 2)
	}
	s.taskQueue[s.tail] = task
	s.tail = (s.tail + 1) % len(s.taskQueue)
	s.count++
	s.Unlock()

	return nil
}

// Request a new task from channel
func (s *Ring) Request() (core.QueuedMessage, error) {
	if atomic.LoadInt32(&s.stopFlag) == 1 && s.count == 0 {
		select {
		case s.exit <- struct{}{}:
		default:
		}
		return nil, ErrQueueHasBeenClosed
	}

	if s.count == 0 {
		return nil, ErrNoTaskInQueue
	}
	s.Lock()
	data := s.taskQueue[s.head]
	s.taskQueue[s.head] = nil
	s.head = (s.head + 1) % len(s.taskQueue)
	s.count--

	if n := len(s.taskQueue) / 2; n > 2 && s.count <= n {
		s.resize(n)
	}
	s.Unlock()

	return data, nil
}

func (q *Ring) resize(n int) {
	nodes := make([]core.QueuedMessage, n)
	if q.head < q.tail {
		copy(nodes, q.taskQueue[q.head:q.tail])
	} else {
		copy(nodes, q.taskQueue[q.head:])
		copy(nodes[len(q.taskQueue)-q.head:], q.taskQueue[:q.tail])
	}

	q.tail = q.count % n
	q.head = 0
	q.taskQueue = nodes
}

// NewRing for create new Ring instance
func NewRing(opts ...Option) *Ring {
	o := NewOptions(opts...)
	w := &Ring{
		taskQueue: make([]core.QueuedMessage, 2),
		capacity:  o.queueSize,
		exit:      make(chan struct{}),
		logger:    o.logger,
		runFunc:   o.fn,
	}

	return w
}
