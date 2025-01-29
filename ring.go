package queue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*Ring)(nil)

// Ring represents a simple queue using a buffer channel.
type Ring struct {
	sync.Mutex
	taskQueue []core.TaskMessage                            // taskQueue holds the tasks in the ring buffer.
	runFunc   func(context.Context, core.TaskMessage) error // runFunc is the function responsible for processing tasks.
	capacity  int                                           // capacity is the maximum number of tasks the queue can hold.
	count     int                                           // count is the current number of tasks in the queue.
	head      int                                           // head is the index of the first task in the queue.
	tail      int                                           // tail is the index where the next task will be added.
	exit      chan struct{}                                 // exit is used to signal when the queue is shutting down.
	logger    Logger                                        // logger is used for logging messages.
	stopOnce  sync.Once                                     // stopOnce ensures the shutdown process only runs once.
	stopFlag  int32                                         // stopFlag indicates whether the queue is shutting down.
}

// Run executes a new task using the provided context and task message.
// It calls the runFunc function, which is responsible for processing the task.
// The context allows for cancellation and timeout control of the task execution.
func (s *Ring) Run(ctx context.Context, task core.TaskMessage) error {
	return s.runFunc(ctx, task)
}

// Shutdown gracefully shuts down the worker.
// It sets the stopFlag to indicate that the queue is shutting down and prevents new tasks from being added.
// If the queue is already shut down, it returns ErrQueueShutdown.
// It waits for all tasks to be processed before completing the shutdown.
func (s *Ring) Shutdown() error {
	// Attempt to set the stopFlag from 0 to 1. If it fails, the queue is already shut down.
	if !atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
		return ErrQueueShutdown
	}

	// Ensure the shutdown process only runs once.
	s.stopOnce.Do(func() {
		s.Lock()
		count := s.count
		s.Unlock()
		// If there are tasks in the queue, wait for them to be processed.
		if count > 0 {
			<-s.exit
		}
	})
	return nil
}

// Queue adds a task to the ring buffer queue.
// It returns an error if the queue is shut down or has reached its maximum capacity.
func (s *Ring) Queue(task core.TaskMessage) error { //nolint:stylecheck
	// Check if the queue is shut down
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}
	// Check if the queue has reached its maximum capacity
	if s.capacity > 0 && s.count >= s.capacity {
		return ErrMaxCapacity
	}

	s.Lock()
	// Resize the queue if necessary
	if s.count == len(s.taskQueue) {
		s.resize(s.count * 2)
	}
	// Add the task to the queue
	s.taskQueue[s.tail] = task
	s.tail = (s.tail + 1) % len(s.taskQueue)
	s.count++
	s.Unlock()

	return nil
}

// Request retrieves the next task message from the ring queue.
// If the queue has been stopped and is empty, it signals the exit channel
// and returns an error indicating the queue has been closed.
// If the queue is empty but not stopped, it returns an error indicating
// there are no tasks in the queue.
// If a task is successfully retrieved, it is removed from the queue,
// and the queue may be resized if it is less than half full.
// Returns the task message and nil on success, or an error if the queue
// is empty or has been closed.
func (s *Ring) Request() (core.TaskMessage, error) {
	if atomic.LoadInt32(&s.stopFlag) == 1 && s.count == 0 {
		select {
		case s.exit <- struct{}{}:
		default:
		}
		return nil, ErrQueueHasBeenClosed
	}

	s.Lock()
	defer s.Unlock()
	if s.count == 0 {
		return nil, ErrNoTaskInQueue
	}
	data := s.taskQueue[s.head]
	s.taskQueue[s.head] = nil
	s.head = (s.head + 1) % len(s.taskQueue)
	s.count--

	if n := len(s.taskQueue) / 2; n > 2 && s.count <= n {
		s.resize(n)
	}

	return data, nil
}

// resize adjusts the size of the ring buffer to the specified capacity n.
// It reallocates the underlying slice to the new size and copies the existing
// elements to the new slice in the correct order. The head and tail pointers
// are updated accordingly to maintain the correct order of elements in the
// resized buffer.
//
// Parameters:
//
//	n - the new capacity of the ring buffer.
func (q *Ring) resize(n int) {
	nodes := make([]core.TaskMessage, n)
	if q.head < q.tail {
		copy(nodes, q.taskQueue[q.head:q.tail])
	} else {
		copy(nodes, q.taskQueue[q.head:])
		copy(nodes[len(q.taskQueue)-q.head:], q.taskQueue[:q.tail])
	}

	q.tail = q.count % n
	q.head = 0
	q.taskQueue = nodes
}

// NewRing creates a new Ring instance with the provided options.
// It initializes the task queue with a default size of 2, sets the capacity
// based on the provided options, and configures the logger and run function.
// The function returns a pointer to the newly created Ring instance.
//
// Parameters:
//
//	opts - A variadic list of Option functions to configure the Ring instance.
//
// Returns:
//
//	*Ring - A pointer to the newly created Ring instance.
func NewRing(opts ...Option) *Ring {
	o := NewOptions(opts...)
	w := &Ring{
		taskQueue: make([]core.TaskMessage, 2),
		capacity:  o.queueSize,
		exit:      make(chan struct{}),
		logger:    o.logger,
		runFunc:   o.fn,
	}

	return w
}
