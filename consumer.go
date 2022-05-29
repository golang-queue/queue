package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*Consumer)(nil)

var errMaxCapacity = errors.New("max capacity reached")

// Consumer for simple queue using buffer channel
type Consumer struct {
	taskQueue chan core.QueuedMessage
	runFunc   func(context.Context, core.QueuedMessage) error
	stop      chan struct{}
	exit      chan struct{}
	logger    Logger
	stopOnce  sync.Once
	stopFlag  int32
}

func (s *Consumer) handle(job *Job) error {
	// create channel with buffer size 1 to avoid goroutine leak
	done := make(chan error, 1)
	panicChan := make(chan interface{}, 1)
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
	defer func() {
		cancel()
	}()

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

// Run to execute new task
func (s *Consumer) Run(task core.QueuedMessage) error {
	data := task.(*Job)
	if data.Task == nil {
		_ = json.Unmarshal(task.Bytes(), data)
	}

	if err := s.handle(data); err != nil {
		return err
	}

	return nil
}

// Shutdown the worker
func (s *Consumer) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
		return ErrQueueShutdown
	}

	s.stopOnce.Do(func() {
		close(s.stop)
		close(s.taskQueue)
		if len(s.taskQueue) > 0 {
			<-s.exit
		}
	})
	return nil
}

// Queue send task to the buffer channel
func (s *Consumer) Queue(task core.QueuedMessage) error {
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	select {
	case s.taskQueue <- task:
		return nil
	default:
		return errMaxCapacity
	}
}

// Request a new task from channel
func (s *Consumer) Request() (core.QueuedMessage, error) {
	clock := 0
loop:
	for {
		select {
		case task, ok := <-s.taskQueue:
			if !ok {
				select {
				case s.exit <- struct{}{}:
				default:
				}
				return nil, ErrQueueHasBeenClosed
			}
			return task, nil
		case <-time.After(1 * time.Second):
			if clock == 5 {
				break loop
			}
			clock += 1
		}
	}

	return nil, ErrNoTaskInQueue
}

// NewConsumer for create new consumer instance
func NewConsumer(opts ...Option) *Consumer {
	o := NewOptions(opts...)
	w := &Consumer{
		taskQueue: make(chan core.QueuedMessage, o.queueSize),
		stop:      make(chan struct{}),
		exit:      make(chan struct{}),
		logger:    o.logger,
		runFunc:   o.fn,
	}

	return w
}
