package queue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const defaultQueueSize = 4096

var _ Worker = (*Consumer)(nil)

// ConsumerOption for queue system
type ConsumerOption func(*Consumer)

var errMaxCapacity = errors.New("max capacity reached")

// Worker for simple queue using channel
type Consumer struct {
	taskQueue chan QueuedMessage
	runFunc   func(context.Context, QueuedMessage) error
	stop      chan struct{}
	logger    Logger
	stopOnce  sync.Once
	stopFlag  int32
}

// BeforeRun run script before start worker
func (s *Consumer) BeforeRun() error {
	return nil
}

// AfterRun run script after start worker
func (s *Consumer) AfterRun() error {
	return nil
}

func (s *Consumer) handle(job Job) error {
	// create channel with buffer size 1 to avoid goroutine leak
	done := make(chan error, 1)
	panicChan := make(chan interface{}, 1)
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
	defer cancel()

	// run the job
	go func() {
		// handle panic issue
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()

		// run custom process function
		if job.Task != nil {
			done <- job.Task(ctx)
		} else {
			done <- s.runFunc(ctx, job)
		}
	}()

	select {
	case p := <-panicChan:
		panic(p)
	case <-ctx.Done(): // timeout reached
		return ctx.Err()
	case <-s.stop: // shutdown service
		// cancel job
		cancel()

		leftTime := job.Timeout - time.Since(startTime)
		// wait job
		select {
		case <-time.After(leftTime):
			return context.DeadlineExceeded
		case err := <-done: // job finish
			return err
		case p := <-panicChan:
			panic(p)
		}
	case err := <-done: // job finish
		return err
	}
}

// Run start the worker
func (s *Consumer) Run() error {
	// check queue status
	select {
	case <-s.stop:
		return ErrQueueShutdown
	default:
	}

	for task := range s.taskQueue {
		var data Job
		_ = json.Unmarshal(task.Bytes(), &data)
		if v, ok := task.(Job); ok {
			if v.Task != nil {
				data.Task = v.Task
			}
		}
		if err := s.handle(data); err != nil {
			s.logger.Error(err.Error())
		}
	}
	return nil
}

// Shutdown worker
func (s *Consumer) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
		return ErrQueueShutdown
	}

	s.stopOnce.Do(func() {
		close(s.stop)
		close(s.taskQueue)
	})
	return nil
}

// Capacity for channel
func (s *Consumer) Capacity() int {
	return cap(s.taskQueue)
}

// Usage for count of channel usage
func (s *Consumer) Usage() int {
	return len(s.taskQueue)
}

// Queue send notification to queue
func (s *Consumer) Queue(job QueuedMessage) error {
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	select {
	case s.taskQueue <- job:
		return nil
	default:
		return errMaxCapacity
	}
}

// WithQueueNum setup the capcity of queue
func WithConsumerQueueNum(num int) ConsumerOption {
	return func(w *Consumer) {
		w.taskQueue = make(chan QueuedMessage, num)
	}
}

// WithRunFunc setup the run func of queue
func WithConsumerRunFunc(fn func(context.Context, QueuedMessage) error) ConsumerOption {
	return func(w *Consumer) {
		w.runFunc = fn
	}
}

// WithConsumerLogger set custom logger
func WithConsumerLogger(l Logger) ConsumerOption {
	return func(w *Consumer) {
		w.logger = l
	}
}

// NewConsumer for struc
func NewConsumer(opts ...ConsumerOption) *Consumer {
	w := &Consumer{
		taskQueue: make(chan QueuedMessage, defaultQueueSize),
		stop:      make(chan struct{}),
		logger:    NewLogger(),
		runFunc: func(context.Context, QueuedMessage) error {
			return nil
		},
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(w)
	}

	return w
}
