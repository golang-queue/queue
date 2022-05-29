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
		worker       core.Worker
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
func (j *Job) Bytes() []byte {
	if j.Task != nil {
		return nil
	}
	return j.Payload
}

// Encode for encoding the structure
func (j *Job) Encode() []byte {
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

// Start to enable all worker
func (q *Queue) Start() {
	if q.workerCount == 0 {
		return
	}
	q.routineGroup.Run(func() {
		q.start()
	})
}

// Shutdown stops all queues.
func (q *Queue) Shutdown() {
	if !atomic.CompareAndSwapInt32(&q.stopFlag, 0, 1) {
		return
	}

	q.stopOnce.Do(func() {
		if q.metric.BusyWorkers() > 0 {
			q.logger.Infof("shutdown all tasks: %d workers", q.metric.BusyWorkers())
		}

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

// BusyWorkers returns the numbers of success tasks.
func (q *Queue) SuccessTasks() int {
	return int(q.metric.SuccessTasks())
}

// BusyWorkers returns the numbers of failure tasks.
func (q *Queue) FailureTasks() int {
	return int(q.metric.FailureTasks())
}

// BusyWorkers returns the numbers of submitted tasks.
func (q *Queue) SubmittedTasks() int {
	return int(q.metric.SubmittedTasks())
}

// Wait all process
func (q *Queue) Wait() {
	q.routineGroup.Wait()
}

// Queue to queue all job
func (q *Queue) Queue(job core.QueuedMessage) error {
	return q.handleQueue(q.timeout, job)
}

// QueueWithTimeout to queue all job with specified timeout.
func (q *Queue) QueueWithTimeout(timeout time.Duration, job core.QueuedMessage) error {
	return q.handleQueue(timeout, job)
}

func (q *Queue) handleQueue(timeout time.Duration, job core.QueuedMessage) error {
	if atomic.LoadInt32(&q.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	if err := q.worker.Queue(&Job{
		Payload: (&Job{
			Timeout: timeout,
			Payload: job.Bytes(),
		}).Encode(),
	}); err != nil {
		return err
	}

	q.metric.IncSubmittedTask()

	return nil
}

// QueueTask to queue job task
func (q *Queue) QueueTask(task TaskFunc) error {
	return q.handleQueueTask(q.timeout, task)
}

// QueueTaskWithTimeout to queue job task with timeout
func (q *Queue) QueueTaskWithTimeout(timeout time.Duration, task TaskFunc) error {
	return q.handleQueueTask(timeout, task)
}

func (q *Queue) handleQueueTask(timeout time.Duration, task TaskFunc) error {
	if atomic.LoadInt32(&q.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	if err := q.worker.Queue(&Job{
		Timeout: timeout,
		Task:    task,
	}); err != nil {
		return err
	}

	q.metric.IncSubmittedTask()

	return nil
}

func (q *Queue) work(task core.QueuedMessage) {
	var err error
	// to handle panic cases from inside the worker
	// in such case, we start a new goroutine
	defer func() {
		q.metric.DecBusyWorker()
		e := recover()
		if e != nil {
			q.logger.Errorf("panic error: %v", e)
		}
		q.schedule()

		// increase success or failure number
		if err == nil && e == nil {
			q.metric.IncSuccessTask()
		} else {
			q.metric.IncFailureTask()
		}
	}()

	if err = q.worker.Run(task); err != nil {
		q.logger.Errorf("runtime error: %s", err.Error())
	}
}

// UpdateWorkerCount to update worker number dynamically.
func (q *Queue) UpdateWorkerCount(num int) {
	q.workerCount = num
	q.schedule()
}

func (q *Queue) schedule() {
	q.Lock()
	defer q.Unlock()
	if q.BusyWorkers() >= q.workerCount {
		return
	}

	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// start handle job
func (q *Queue) start() {
	tasks := make(chan core.QueuedMessage, 1)

	for {
		// check worker number
		q.schedule()

		select {
		// wait worker ready
		case <-q.ready:
		case <-q.quit:
			return
		}

		// request task from queue in background
		q.routineGroup.Run(func() {
			for {
				t, err := q.worker.Request()
				if t == nil || err != nil {
					if err != nil {
						select {
						case <-q.quit:
							if !errors.Is(err, ErrNoTaskInQueue) {
								close(tasks)
								return
							}
						case <-time.After(time.Second):
							// sleep 1 second to fetch new task
						}
					}
				}
				if t != nil {
					tasks <- t
					return
				}

				select {
				case <-q.quit:
					if !errors.Is(err, ErrNoTaskInQueue) {
						close(tasks)
						return
					}
				default:
				}
			}
		})

		task, ok := <-tasks
		if !ok {
			return
		}

		// start new task
		q.metric.IncBusyWorker()
		q.routineGroup.Run(func() {
			q.work(task)
		})
	}
}
