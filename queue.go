package queue

// Package queue provides a high-performance, extensible message queue implementation
// supporting multiple workers, job retries, dynamic scaling, and graceful shutdown.

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-queue/queue/core"
	"github.com/golang-queue/queue/job"

	"github.com/jpillora/backoff"
)

/*
ErrQueueShutdown is returned when an operation is attempted on a queue
that has already been closed and released.
*/
var ErrQueueShutdown = errors.New("queue has been closed and released")

type (
	// Queue represents a message queue with worker management, job scheduling,
	// retry logic, and graceful shutdown capabilities.
	Queue struct {
		sync.Mutex                  // Mutex to protect concurrent access to queue state
		metric        *metric       // Metrics collector for tracking queue and worker stats
		logger        Logger        // Logger for queue events and errors
		workerCount   int64         // Number of worker goroutines to process jobs
		routineGroup  *routineGroup // Group to manage and wait for goroutines
		quit          chan struct{} // Channel to signal shutdown to all goroutines
		ready         chan struct{} // Channel to signal worker readiness
		notify        chan struct{} // Channel to notify workers of new jobs
		worker        core.Worker   // The worker implementation that processes jobs
		stopOnce      sync.Once     // Ensures shutdown is only performed once
		stopFlag      int32         // Atomic flag indicating if shutdown has started
		afterFn       func()        // Optional callback after each job execution
		retryInterval time.Duration // Interval for retrying job requests
	}
)

/*
ErrMissingWorker is returned when a queue is created without a worker implementation.
*/
var ErrMissingWorker = errors.New("missing worker module")

// NewQueue creates and returns a new Queue instance with the provided options.
// Returns an error if no worker is specified.
func NewQueue(opts ...Option) (*Queue, error) {
	o := NewOptions(opts...)
	q := &Queue{
		routineGroup:  newRoutineGroup(),      // Manages all goroutines spawned by the queue
		quit:          make(chan struct{}),    // Signals shutdown to all goroutines
		ready:         make(chan struct{}, 1), // Signals when a worker is ready to process a job
		notify:        make(chan struct{}, 1), // Notifies workers of new jobs
		workerCount:   o.workerCount,          // Number of worker goroutines
		logger:        o.logger,               // Logger for queue events
		worker:        o.worker,               // Worker implementation
		metric:        &metric{},              // Metrics collector
		afterFn:       o.afterFn,              // Optional post-job callback
		retryInterval: o.retryInterval,        // Interval for retrying job requests
	}

	if q.worker == nil {
		return nil, ErrMissingWorker
	}

	return q, nil
}

// Start launches all worker goroutines and begins processing jobs.
// If workerCount is zero, Start is a no-op.
func (q *Queue) Start() {
	q.Lock()
	count := q.workerCount
	q.Unlock()
	if count == 0 {
		return
	}
	q.routineGroup.Run(func() {
		q.start()
	})
}

// Shutdown initiates a graceful shutdown of the queue.
// It signals all goroutines to stop, shuts down the worker, and closes the quit channel.
// Shutdown is idempotent and safe to call multiple times.
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

// Release performs a graceful shutdown and waits for all goroutines to finish.
func (q *Queue) Release() {
	q.Shutdown()
	q.Wait()
}

// BusyWorkers returns the number of workers currently processing jobs.
func (q *Queue) BusyWorkers() int64 {
	return q.metric.BusyWorkers()
}

// SuccessTasks returns the number of successfully completed tasks.
func (q *Queue) SuccessTasks() uint64 {
	return q.metric.SuccessTasks()
}

// FailureTasks returns the number of failed tasks.
func (q *Queue) FailureTasks() uint64 {
	return q.metric.FailureTasks()
}

// SubmittedTasks returns the number of tasks submitted to the queue.
func (q *Queue) SubmittedTasks() uint64 {
	return q.metric.SubmittedTasks()
}

// CompletedTasks returns the total number of completed tasks (success + failure).
func (q *Queue) CompletedTasks() uint64 {
	return q.metric.CompletedTasks()
}

// Wait blocks until all goroutines in the routine group have finished.
func (q *Queue) Wait() {
	q.routineGroup.Wait()
}

// Queue enqueues a single job (core.QueuedMessage) into the queue.
// Accepts job options for customization.
func (q *Queue) Queue(message core.QueuedMessage, opts ...job.AllowOption) error {
	data := job.NewMessage(message, opts...)

	return q.queue(&data)
}

// QueueTask enqueues a single task function into the queue.
// Accepts job options for customization.
func (q *Queue) QueueTask(task job.TaskFunc, opts ...job.AllowOption) error {
	data := job.NewTask(task, opts...)
	return q.queue(&data)
}

// queue is an internal helper to enqueue a job.Message into the worker.
// It increments the submitted task metric and notifies workers if possible.
func (q *Queue) queue(m *job.Message) error {
	if atomic.LoadInt32(&q.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	if err := q.worker.Queue(m); err != nil {
		return err
	}

	q.metric.IncSubmittedTask()
	// Notify a worker that a new job is available.
	// If the notify channel is full, the worker is busy and we avoid blocking.
	select {
	case q.notify <- struct{}{}:
	default:
	}

	return nil
}

// work executes a single task, handling panics and updating metrics accordingly.
// After execution, it schedules the next worker if needed.
func (q *Queue) work(task core.TaskMessage) {
	var err error
	// Defer block to handle panics, update metrics, and run afterFn callback.
	defer func() {
		q.metric.DecBusyWorker()
		e := recover()
		if e != nil {
			q.logger.Fatalf("panic error: %v", e)
		}
		q.schedule()

		// Update success or failure metrics based on execution result.
		if err == nil && e == nil {
			q.metric.IncSuccessTask()
		} else {
			q.metric.IncFailureTask()
		}
		if q.afterFn != nil {
			q.afterFn()
		}
	}()

	if err = q.run(task); err != nil {
		q.logger.Errorf("runtime error: %s", err.Error())
	}
}

// run dispatches the task to the appropriate handler based on its type.
// Returns an error if the task type is invalid.
func (q *Queue) run(task core.TaskMessage) error {
	switch t := task.(type) {
	case *job.Message:
		return q.handle(t)
	default:
		return errors.New("invalid task type")
	}
}

// handle executes a job.Message, supporting retries, timeouts, and panic recovery.
// Returns an error if the job fails or times out.
func (q *Queue) handle(m *job.Message) error {
	// done: receives the result of the job execution
	// panicChan: receives any panic that occurs in the job goroutine
	done := make(chan error, 1)
	panicChan := make(chan any, 1)
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer func() {
		cancel()
	}()

	// Run the job in a separate goroutine to support timeout and panic recovery.
	go func() {
		// Defer block to catch panics and send to panicChan
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()

		var err error

		// Set up backoff for retry logic
		b := &backoff.Backoff{
			Min:    m.RetryMin,
			Max:    m.RetryMax,
			Factor: m.RetryFactor,
			Jitter: m.Jitter,
		}
		delay := m.RetryDelay
	loop:
		for {
			// If a custom Task function is provided, use it; otherwise, use the worker's Run method.
			if m.Task != nil {
				err = m.Task(ctx)
			} else {
				err = q.worker.Run(ctx, m)
			}

			// If no error or no retries left, exit loop.
			if err == nil || m.RetryCount == 0 {
				break
			}
			m.RetryCount--

			// If no fixed retry delay, use backoff.
			if m.RetryDelay == 0 {
				delay = b.Duration()
			}

			select {
			case <-time.After(delay): // Wait before retrying
				q.logger.Infof("retry remaining times: %d, delay time: %s", m.RetryCount, delay)
			case <-ctx.Done(): // Timeout reached
				err = ctx.Err()
				break loop
			}
		}

		done <- err
	}()

	select {
	case p := <-panicChan:
		panic(p)
	case <-ctx.Done(): // Timeout reached
		return ctx.Err()
	case <-q.quit: // Queue is shutting down
		// Cancel job and wait for remaining time or job completion
		cancel()
		leftTime := m.Timeout - time.Since(startTime)
		select {
		case <-time.After(leftTime):
			return context.DeadlineExceeded
		case err := <-done: // Job finished
			return err
		case p := <-panicChan:
			panic(p)
		}
	case err := <-done: // Job finished
		return err
	}
}

// UpdateWorkerCount dynamically updates the number of worker goroutines.
// Triggers scheduling to adjust to the new worker count.
func (q *Queue) UpdateWorkerCount(num int64) {
	q.Lock()
	q.workerCount = num
	q.Unlock()
	q.schedule()
}

// schedule checks if more workers can be started based on the current busy count.
// If so, it signals readiness to start a new worker.
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

/*
start launches the main worker loop, which manages job scheduling and execution.

- It uses a ticker to periodically retry job requests if the queue is empty.
- For each available worker slot, it requests a new task from the worker.
- If a task is available, it is sent to the tasks channel and processed by a new goroutine.
- The loop exits when the quit channel is closed.
*/
func (q *Queue) start() {
	tasks := make(chan core.TaskMessage, 1)
	ticker := time.NewTicker(q.retryInterval)
	defer ticker.Stop()

	for {
		// Ensure the number of busy workers does not exceed the configured worker count.
		q.schedule()

		select {
		case <-q.ready: // Wait for a worker slot to become available
		case <-q.quit: // Shutdown signal received
			return
		}

		// Request a task from the worker in a background goroutine.
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
						case <-ticker.C:
						case <-q.notify:
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

		// Start processing the new task in a separate goroutine.
		q.metric.IncBusyWorker()
		q.routineGroup.Run(func() {
			q.work(task)
		})
	}
}
