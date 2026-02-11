package queue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*Ring)(nil)

// Ring is an in-memory worker implementation using a dynamic circular buffer.
// It implements the core.Worker interface and provides automatic resizing:
//   - Doubles capacity when full
//   - Halves capacity when less than 25% utilized
//
// The ring buffer uses two pointers (head and tail) to track the queue boundaries:
//   - head: points to the next task to dequeue
//   - tail: points to the next empty slot for enqueuing
//   - When head == tail, the queue is empty
//   - Both pointers wrap around using modulo arithmetic
type Ring struct {
	sync.Mutex
	taskQueue []core.TaskMessage                            // Circular buffer storing queued tasks
	runFunc   func(context.Context, core.TaskMessage) error // Function to execute when processing tasks
	capacity  int                                           // Maximum queue size (0 = unlimited, grows dynamically)
	count     int                                           // Current number of tasks in the queue
	head      int                                           // Index of the first task (dequeue position)
	tail      int                                           // Index where the next task will be added (enqueue position)
	exit      chan struct{}                                 // Signals completion of shutdown after all tasks drain
	logger    Logger                                        // Logger for debugging and error messages
	stopOnce  sync.Once                                     // Ensures shutdown logic executes only once
	stopFlag  int32                                         // Atomic flag: 0 = running, 1 = shutting down
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

// Queue adds a task to the ring buffer.
// The buffer grows dynamically (doubles in size) when full, unless capacity is set.
// Returns ErrQueueShutdown if the queue is closing, or ErrMaxCapacity if at the size limit.
//
// Thread-safety: This method is safe for concurrent calls.
func (s *Ring) Queue(task core.TaskMessage) error { //nolint:stylecheck
	// Reject new tasks if shutdown has been initiated
	if atomic.LoadInt32(&s.stopFlag) == 1 {
		return ErrQueueShutdown
	}
	// Enforce maximum capacity limit if configured
	if s.capacity > 0 && s.count >= s.capacity {
		return ErrMaxCapacity
	}

	s.Lock()
	// Grow the buffer if it's full (before adding the new task)
	if s.count == len(s.taskQueue) {
		s.resize(s.count * 2)
	}
	// Add task at tail position and advance tail pointer (with wraparound)
	s.taskQueue[s.tail] = task
	s.tail = (s.tail + 1) % len(s.taskQueue)
	s.count++
	s.Unlock()

	return nil
}

// Request dequeues and returns the next task from the ring buffer.
// The buffer shrinks automatically (halves in size) when less than 25% full.
// Returns:
//   - (task, nil) if a task is successfully dequeued
//   - (nil, ErrNoTaskInQueue) if the queue is currently empty
//   - (nil, ErrQueueHasBeenClosed) if shutdown is complete and the queue is empty
//
// During shutdown, this method signals the exit channel when the last task is dequeued,
// allowing Shutdown() to complete.
//
// Thread-safety: This method is safe for concurrent calls.
func (s *Ring) Request() (core.TaskMessage, error) {
	// If shutting down and queue is empty, signal exit and return closed error
	if atomic.LoadInt32(&s.stopFlag) == 1 && s.count == 0 {
		select {
		case s.exit <- struct{}{}: // Non-blocking send to signal shutdown completion
		default:
		}
		return nil, ErrQueueHasBeenClosed
	}

	s.Lock()
	defer s.Unlock()

	// Return early if queue is empty (but not shutting down yet)
	if s.count == 0 {
		return nil, ErrNoTaskInQueue
	}

	// Dequeue task from head position
	data := s.taskQueue[s.head]
	s.taskQueue[s.head] = nil // Clear reference to allow GC
	s.head = (s.head + 1) % len(s.taskQueue) // Advance head with wraparound
	s.count--

	// Shrink buffer if less than 25% full (to save memory)
	// Minimum buffer size is 2 to maintain circular buffer structure
	if n := len(s.taskQueue) / 2; n >= 2 && s.count <= n {
		s.resize(n)
	}

	return data, nil
}

// resize changes the ring buffer capacity to n and reorganizes the elements.
// This internal method handles the complexity of copying elements from a circular
// buffer where head and tail may have wrapped around.
//
// The ring buffer can be in two states:
//
//	Case 1: head < tail (no wraparound)
//	    [_ _ _ H X X X T _ _]  -> Tasks are contiguous from head to tail
//
//	Case 2: head >= tail (wraparound occurred)
//	    [X X T _ _ _ _ H X X]  -> Tasks wrap: [head..end] + [0..tail]
//
// After resizing, all elements are reorganized to start at index 0 with no wraparound.
//
// Parameters:
//
//	n - the new buffer capacity (must be >= count)
//
// Note: This method is NOT thread-safe and must be called while holding the lock.
func (q *Ring) resize(n int) {
	nodes := make([]core.TaskMessage, n)

	if q.head < q.tail {
		// Case 1: No wraparound - tasks are contiguous
		// Simply copy [head:tail] to the beginning of new buffer
		copy(nodes, q.taskQueue[q.head:q.tail])
	} else {
		// Case 2: Wraparound - tasks are split across the buffer
		// Copy [head:end] to start of new buffer
		copy(nodes, q.taskQueue[q.head:])
		// Append [0:tail] immediately after
		copy(nodes[len(q.taskQueue)-q.head:], q.taskQueue[:q.tail])
	}

	// Reset pointers: all tasks now start at index 0
	q.tail = q.count % n // Position for next enqueue
	q.head = 0           // Position for next dequeue
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
