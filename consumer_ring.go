package queue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*ConsumerRing)(nil)

// ConsumerRing for simple queue using buffer channel
type ConsumerRing struct {
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
func (s *ConsumerRing) Run(ctx context.Context, task core.QueuedMessage) error {
	return s.runFunc(ctx, task)
}

// Shutdown the worker
func (s *ConsumerRing) Shutdown() error {
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
func (s *ConsumerRing) Queue(task core.QueuedMessage) error { //nolint:stylecheck
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}
	if s.count >= s.capacity {
		return errMaxCapacity
	}

	if s.count == len(s.taskQueue) {
		s.resize(s.count * 2)
	}
	s.taskQueue[s.tail] = task
	s.tail = (s.tail + 1) % len(s.taskQueue)
	s.count++

	return nil
}

// Request a new task from channel
func (s *ConsumerRing) Request() (core.QueuedMessage, error) {
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
	data := s.taskQueue[s.head]
	s.head = (s.head + 1) % len(s.taskQueue)
	s.count--

	if n := len(s.taskQueue) / 2; n > 2 && s.count <= n {
		s.resize(n)
	}

	return data, nil
}

func (q *ConsumerRing) resize(n int) {
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

// NewConsumerRing for create new ConsumerRing instance
func NewConsumerRing(opts ...Option) *ConsumerRing {
	o := NewOptions(opts...)
	w := &ConsumerRing{
		taskQueue: make([]core.QueuedMessage, 2),
		capacity:  o.queueSize,
		exit:      make(chan struct{}),
		logger:    o.logger,
		runFunc:   o.fn,
	}

	return w
}
