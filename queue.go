package queue

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ErrQueueShutdown close the queue.
var ErrQueueShutdown = errors.New("queue has been closed")

// TaskFunc is the task function
type TaskFunc func(context.Context) error

type (
	// A Queue is a message queue.
	Queue struct {
		logger         Logger
		workerCount    int
		routineGroup   *routineGroup
		quit           chan struct{}
		worker         Worker
		stopOnce       sync.Once
		runningWorkers int32
		timeout        time.Duration
		stopFlag       int32
	}

	// Job with Timeout
	Job struct {
		Task    TaskFunc      `json:"-"`
		Timeout time.Duration `json:"timeout"`
		Body    []byte        `json:"body"`
	}
)

// Bytes get string body
func (j Job) Bytes() []byte {
	return j.Body
}

func (j Job) Encode() []byte {
	b, _ := json.Marshal(j)
	return b
}

// Option for queue system
type Option func(*Queue)

// ErrMissingWorker missing define worker
var ErrMissingWorker = errors.New("missing worker module")

// WithWorkerCount set worker count
func WithWorkerCount(num int) Option {
	return func(q *Queue) {
		q.workerCount = num
	}
}

// WithLogger set custom logger
func WithLogger(l Logger) Option {
	return func(q *Queue) {
		q.logger = l
	}
}

// WithWorker set custom worker
func WithWorker(w Worker) Option {
	return func(q *Queue) {
		q.worker = w
	}
}

// NewQueue returns a Queue.
func NewQueue(opts ...Option) (*Queue, error) {
	q := &Queue{
		workerCount:  runtime.NumCPU(),
		routineGroup: newRoutineGroup(),
		quit:         make(chan struct{}),
		logger:       NewLogger(),
		timeout:      24 * 60 * time.Minute,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(q)
	}

	if q.worker == nil {
		return nil, ErrMissingWorker
	}

	return q, nil
}

// Capacity for queue max size
func (q *Queue) Capacity() int {
	return q.worker.Capacity()
}

// Usage for count of queue usage
func (q *Queue) Usage() int {
	return q.worker.Usage()
}

// Start to enable all worker
func (q *Queue) Start() {
	q.startWorker()
}

// Shutdown stops all queues.
func (q *Queue) Shutdown() {
	if !atomic.CompareAndSwapInt32(&q.stopFlag, 0, 1) {
		return
	}

	q.stopOnce.Do(func() {
		if err := q.worker.Shutdown(); err != nil {
			q.logger.Error(err)
		}
		close(q.quit)
	})
}

// Workers returns the numbers of workers has been created.
func (q *Queue) Workers() int {
	return int(atomic.LoadInt32(&q.runningWorkers))
}

// Wait all process
func (q *Queue) Wait() {
	q.routineGroup.Wait()
}

func (q *Queue) handleQueue(timeout time.Duration, job QueuedMessage) error {
	if atomic.LoadInt32(&q.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	data := Job{
		Timeout: timeout,
		Body:    job.Bytes(),
	}

	return q.worker.Queue(Job{
		Body: data.Encode(),
	})
}

// Queue to queue all job
func (q *Queue) Queue(job QueuedMessage) error {
	return q.handleQueue(q.timeout, job)
}

// Queue to queue all job
func (q *Queue) QueueWithTimeout(timeout time.Duration, job QueuedMessage) error {
	return q.handleQueue(timeout, job)
}

func (q *Queue) handleQueueTask(timeout time.Duration, task TaskFunc) error {
	if atomic.LoadInt32(&q.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	return q.worker.Queue(Job{
		Task:    task,
		Timeout: timeout,
	})
}

// QueueTask to queue job task
func (q *Queue) QueueTask(task TaskFunc) error {
	return q.handleQueueTask(q.timeout, task)
}

// QueueTaskWithTimeout to queue job task with timeout
func (q *Queue) QueueTaskWithTimeout(timeout time.Duration, task TaskFunc) error {
	return q.handleQueueTask(timeout, task)
}

func (q *Queue) work() {
	if atomic.LoadInt32(&q.stopFlag) == 1 {
		return
	}

	num := atomic.AddInt32(&q.runningWorkers, 1)
	if err := q.worker.BeforeRun(); err != nil {
		q.logger.Fatal(err)
	}
	q.routineGroup.Run(func() {
		// to handle panic cases from inside the worker
		// in such case, we start a new goroutine
		defer func() {
			atomic.AddInt32(&q.runningWorkers, -1)
			if err := recover(); err != nil {
				q.logger.Error(err)
				q.logger.Infof("restart the new worker: %d", num)
				go q.work()
			}
		}()
		q.logger.Infof("start the worker num: %d", num)
		if err := q.worker.Run(); err != nil {
			q.logger.Errorf("runtime error: %s", err.Error())
		}
		q.logger.Infof("stop the worker num: %d", num)
	})
	if err := q.worker.AfterRun(); err != nil {
		q.logger.Fatal(err)
	}
}

func (q *Queue) startWorker() {
	for i := 0; i < q.workerCount; i++ {
		go q.work()
	}
}
