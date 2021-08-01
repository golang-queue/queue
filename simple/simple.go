package simple

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/appleboy/queue"
)

const defaultQueueSize = 4096

var _ queue.Worker = (*Worker)(nil)

// Option for queue system
type Option func(*Worker)

var errMaxCapacity = errors.New("max capacity reached")

// Worker for simple queue using channel
type Worker struct {
	taskQueue chan queue.QueuedMessage
	runFunc   func(context.Context, queue.QueuedMessage) error
	stop      chan struct{}
	logger    queue.Logger
	stopOnce  sync.Once
	stopFlag  int32
}

// BeforeRun run script before start worker
func (s *Worker) BeforeRun() error {
	return nil
}

// AfterRun run script after start worker
func (s *Worker) AfterRun() error {
	return nil
}

func (s *Worker) handle(job queue.Job) error {
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
func (s *Worker) Run() error {
	// check queue status
	select {
	case <-s.stop:
		return queue.ErrQueueShutdown
	default:
	}

	for task := range s.taskQueue {
		var data queue.Job
		_ = json.Unmarshal(task.Bytes(), &data)
		if v, ok := task.(queue.Job); ok {
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
func (s *Worker) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
		return queue.ErrQueueShutdown
	}

	s.stopOnce.Do(func() {
		close(s.stop)
		close(s.taskQueue)
	})
	return nil
}

// Capacity for channel
func (s *Worker) Capacity() int {
	return cap(s.taskQueue)
}

// Usage for count of channel usage
func (s *Worker) Usage() int {
	return len(s.taskQueue)
}

// Queue send notification to queue
func (s *Worker) Queue(job queue.QueuedMessage) error {
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return queue.ErrQueueShutdown
	}

	select {
	case s.taskQueue <- job:
		return nil
	default:
		return errMaxCapacity
	}
}

// WithQueueNum setup the capcity of queue
func WithQueueNum(num int) Option {
	return func(w *Worker) {
		w.taskQueue = make(chan queue.QueuedMessage, num)
	}
}

// WithRunFunc setup the run func of queue
func WithRunFunc(fn func(context.Context, queue.QueuedMessage) error) Option {
	return func(w *Worker) {
		w.runFunc = fn
	}
}

// WithLogger set custom logger
func WithLogger(l queue.Logger) Option {
	return func(w *Worker) {
		w.logger = l
	}
}

// NewWorker for struc
func NewWorker(opts ...Option) *Worker {
	w := &Worker{
		taskQueue: make(chan queue.QueuedMessage, defaultQueueSize),
		stop:      make(chan struct{}),
		logger:    queue.NewLogger(),
		runFunc: func(context.Context, queue.QueuedMessage) error {
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
