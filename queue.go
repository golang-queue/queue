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
		sync.Mutex
		metric       *metric
		logger       Logger
		workerCount  int
		routineGroup *routineGroup
		quit         chan struct{}
		ready        chan struct{}
		worker       Worker
		stopOnce     sync.Once
		timeout      time.Duration
		stopFlag     int32
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

// Encode for encoding the structure
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
		ready:        make(chan struct{}, 1),
		workerCount:  o.workerCount,
		logger:       o.logger,
		timeout:      o.timeout,
		worker:       o.worker,
		metric:       &metric{},
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
	go q.start()
}

// Shutdown stops all queues.
func (q *Queue) Shutdown() {
	if !atomic.CompareAndSwapInt32(&q.stopFlag, 0, 1) {
		return
	}

	if q.metric.BusyWorkers() > 0 {
		q.logger.Infof("shutdown all woker numbers: %d", q.metric.BusyWorkers())
	}

	q.stopOnce.Do(func() {
		if err := q.worker.Shutdown(); err != nil {
			q.logger.Error(err)
		}
		close(q.quit)
	})
}

// Release for graceful shutdown.
func (q *Queue) Release() {
	q.Shutdown()
	q.Wait()
}

// BusyWorkers returns the numbers of workers in the running process.
func (q *Queue) BusyWorkers() int {
	return int(q.metric.BusyWorkers())
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

// QueueWithTimeout to queue all job with specified timeout.
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

func (q *Queue) work(task QueuedMessage) {
	if err := q.worker.BeforeRun(); err != nil {
		q.logger.Error(err)
	}

	// to handle panic cases from inside the worker
	// in such case, we start a new goroutine
	defer func() {
		q.metric.DecBusyWorker()
		if err := recover(); err != nil {
			q.logger.Errorf("panic error: %v", err)
		}
		q.schedule()
	}()

	if err := q.worker.Run(task); err != nil {
		q.logger.Errorf("runtime error: %s", err.Error())
	}

	if err := q.worker.AfterRun(); err != nil {
		q.logger.Error(err)
	}
}

func (q *Queue) schedule() {
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// start handle job
func (q *Queue) start() {
	var task QueuedMessage
	tasks := make(chan QueuedMessage, 1)

	for {
		if atomic.LoadInt32(&q.stopFlag) == 1 {
			return
		}

		// request task from queue in background
		q.routineGroup.Run(func() {
		loop:
			for {
				select {
				case <-q.quit:
					return
				default:
					task, err := q.worker.Request()
					if task == nil || err != nil {
						if err != nil {
							select {
							case <-q.quit:
								break loop
							case <-time.After(time.Second):
								// sleep 1 second to fetch new task
							}
						}
					}
					if task != nil {
						tasks <- task
						break loop
					}
				}
			}
		})

		// read task
		select {
		case task = <-tasks:
		case <-q.quit:
			select {
			case task = <-tasks:
				// queue task before shutdown the service
				q.worker.Queue(task)
			default:
			}
			return
		}

		// check worker number
		if q.BusyWorkers() < q.workerCount {
			q.schedule()
		}

		// get worker to execute new task
		select {
		case <-q.quit:
			q.worker.Queue(task)
			return
		case <-q.ready:
			select {
			case <-q.quit:
				q.worker.Queue(task)
				return
			default:
			}
		}

		// start new task
		q.metric.IncBusyWorker()
		q.routineGroup.Run(func() {
			q.work(task)
		})
	}
}
