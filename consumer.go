package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*Consumer)(nil)

var errMaxCapacity = errors.New("max capacity reached")

// Consumer for simple queue using buffer channel
type Consumer struct {
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
func (s *Consumer) Run(ctx context.Context, task core.QueuedMessage) error {
	return s.runFunc(ctx, task)
}

// Shutdown the worker
func (s *Consumer) Shutdown() error {
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
func (s *Consumer) Queue(task core.QueuedMessage) error { //nolint:stylecheck
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
func (s *Consumer) Request() (core.QueuedMessage, error) {
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

func (q *Consumer) resize(n int) {
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

// NewConsumer for create new Consumer instance
func NewConsumer(opts ...Option) *Consumer {
	o := NewOptions(opts...)
	w := &Consumer{
		taskQueue: make([]core.QueuedMessage, 2),
		capacity:  o.queueSize,
		exit:      make(chan struct{}),
		logger:    o.logger,
		runFunc:   o.fn,
	}

	return w
}
