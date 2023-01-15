package queue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*ConsumerSlice)(nil)

// ConsumerSlice for simple queue using buffer channel
type ConsumerSlice struct {
	sync.Mutex
	taskQueue []core.QueuedMessage
	runFunc   func(context.Context, core.QueuedMessage) error
	capacity  int
	head      int
	tail      int
	exit      chan struct{}
	logger    Logger
	stopOnce  sync.Once
	stopFlag  int32
}

// Run to execute new task
func (s *ConsumerSlice) Run(ctx context.Context, task core.QueuedMessage) error {
	return s.runFunc(ctx, task)
}

// Shutdown the worker
func (s *ConsumerSlice) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
		return ErrQueueShutdown
	}

	s.stopOnce.Do(func() {
		if !s.IsEmpty() {
			<-s.exit
		}
	})
	return nil
}

// Queue send task to the buffer channel
func (s *ConsumerSlice) Queue(task core.QueuedMessage) error {
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}
	if s.IsFull() {
		return errMaxCapacity
	}

	s.Lock()
	s.taskQueue[s.tail] = task
	s.tail = (s.tail + 1) % s.capacity
	s.Unlock()

	return nil
}

// Request a new task from channel
func (s *ConsumerSlice) Request() (core.QueuedMessage, error) {
	if atomic.LoadInt32(&s.stopFlag) == 1 && s.IsEmpty() {
		select {
		case s.exit <- struct{}{}:
		default:
		}
		return nil, ErrQueueHasBeenClosed
	}

	if s.IsEmpty() {
		return nil, ErrNoTaskInQueue
	}

	s.Lock()
	data := s.taskQueue[s.head]
	s.head = (s.head + 1) % s.capacity
	s.Unlock()

	return data, nil
}

// IsEmpty returns true if queue is empty
func (s *ConsumerSlice) IsEmpty() bool {
	return s.head == s.tail
}

// IsFull returns true if queue is full
func (s *ConsumerSlice) IsFull() bool {
	return s.head == (s.tail+1)%s.capacity
}

// NewConsumerSlice for create new ConsumerSlice instance
func NewConsumerSlice(opts ...Option) *ConsumerSlice {
	o := NewOptions(opts...)
	w := &ConsumerSlice{
		taskQueue: make([]core.QueuedMessage, o.queueSize),
		capacity:  o.queueSize,
		exit:      make(chan struct{}),
		logger:    o.logger,
		runFunc:   o.fn,
	}

	return w
}
