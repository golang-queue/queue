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
	count     int
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
		if s.count > 0 {
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
	if s.count >= s.capacity {
		return errMaxCapacity
	}

	s.Lock()
	s.taskQueue[s.count] = task
	s.count++
	s.Unlock()

	return nil
}

// Request a new task from channel
func (s *ConsumerSlice) Request() (core.QueuedMessage, error) {
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
	peak := s.taskQueue[0]
	copy(s.taskQueue, s.taskQueue[1:])
	s.taskQueue = s.taskQueue[:len(s.taskQueue)-1]
	s.count--
	s.Unlock()

	return peak, nil
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
