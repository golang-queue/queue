package queue

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ErrQueueShutdown close the queue.
var ErrQueueShutdown = errors.New("queue has been closed")

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
	}

	// Job with Timeout
	Job struct {
		Timeout time.Duration
		Body    []byte
	}
)

// Bytes get string body
func (j Job) Bytes() []byte {
	return j.Body
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

// Queue to queue all job
func (q *Queue) Queue(job QueuedMessage) error {
	return q.worker.Queue(Job{
		Timeout: q.timeout,
		Body:    job.Bytes(),
	})
}

// Queue to queue all job
func (q *Queue) QueueWithTimeout(timeout time.Duration, job QueuedMessage) error {
	return q.worker.Queue(Job{
		Timeout: timeout,
		Body:    job.Bytes(),
	})
}

func (q *Queue) work() {
	select {
	case <-q.quit:
		return
	default:
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
