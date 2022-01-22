package queue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrQueueShutdown the queue is released and closed.
var ErrQueueShutdown = errors.New("queue has been closed and released")

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

	// Job describes a task and its metadata.
	Job struct {
		Task TaskFunc `json:"-"`

		// Timeout is the duration the task can be processed by Handler.
		// zero if not specified
		Timeout time.Duration `json:"timeout"`

		// Payload is the payload data of the task.
		Payload []byte `json:"body"`
	}
)

// Bytes get string body
func (j Job) Bytes() []byte {
	return j.Payload
}

func (j Job) Encode() []byte {
	b, _ := json.Marshal(j)
	return b
}

// ErrMissingWorker missing define worker
var ErrMissingWorker = errors.New("missing worker module")

// NewQueue returns a Queue.
func NewQueue(opts ...Option) (*Queue, error) {
	o := NewOptions(opts...)
	q := &Queue{
		routineGroup: newRoutineGroup(),
		quit:         make(chan struct{}),
		workerCount:  o.workerCount,
		logger:       o.logger,
		timeout:      o.timeout,
		worker:       o.worker,
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

	if q.runningWorkers > 0 {
		q.logger.Infof("shutdown all woker numbers: %d", q.runningWorkers)
	}

	q.stopOnce.Do(func() {
		if err := q.worker.Shutdown(); err != nil {
			q.logger.Error(err)
		}
		close(q.quit)
	})
}

// Workers returns the numbers of workers has been created.
func (q *Queue) Release() {
	q.Shutdown()
	q.Wait()
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
		Payload: job.Bytes(),
	}

	return q.worker.Queue(Job{
		Payload: data.Encode(),
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

	data := Job{
		Timeout: timeout,
	}

	return q.worker.Queue(Job{
		Task:    task,
		Payload: data.Encode(),
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
