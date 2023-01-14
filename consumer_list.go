package queue

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*ConsumerList)(nil)

// ConsumerList for simple queue using buffer channel
type ConsumerList struct {
	taskQueue *list.List
	runFunc   func(context.Context, core.QueuedMessage) error
	capacity  int
	exit      chan struct{}
	logger    Logger
	stopOnce  sync.Once
	stopFlag  int32
}

// Run to execute new task
func (s *ConsumerList) Run(ctx context.Context, task core.QueuedMessage) error {
	return s.runFunc(ctx, task)
}

// Shutdown the worker
func (s *ConsumerList) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
		return ErrQueueShutdown
	}

	s.stopOnce.Do(func() {
		if s.taskQueue.Len() > 0 {
			<-s.exit
		}
	})
	return nil
}

// Queue send task to the buffer channel
func (s *ConsumerList) Queue(task core.QueuedMessage) error {
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	if s.taskQueue.Len() >= s.capacity {
		return errMaxCapacity
	}

	s.taskQueue.PushBack(task)

	return nil
}

// Request a new task from channel
func (s *ConsumerList) Request() (core.QueuedMessage, error) {
	if atomic.LoadInt32(&s.stopFlag) == 1 && s.taskQueue.Len() == 0 {
		select {
		case s.exit <- struct{}{}:
		default:
		}
		return nil, ErrQueueHasBeenClosed
	}

	if s.taskQueue.Len() == 0 {
		return nil, ErrNoTaskInQueue
	}

	peak := s.taskQueue.Back()
	s.taskQueue.Remove(s.taskQueue.Back())

	return peak.Value.(core.QueuedMessage), nil
}

// NewConsumerList for create new ConsumerList instance
func NewConsumerList(opts ...Option) *ConsumerList {
	o := NewOptions(opts...)
	w := &ConsumerList{
		taskQueue: list.New(),
		capacity:  o.queueSize,
		exit:      make(chan struct{}),
		logger:    o.logger,
		runFunc:   o.fn,
	}

	return w
}
