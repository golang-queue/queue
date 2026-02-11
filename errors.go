package queue

import "errors"

var (
	// ErrNoTaskInQueue is returned by Worker.Request() when the queue is currently empty.
	// This is a temporary condition - new tasks may be added later.
	// The queue scheduler uses this error to determine when to retry or wait for notifications.
	ErrNoTaskInQueue = errors.New("golang-queue: no task in queue")

	// ErrQueueHasBeenClosed is returned by Worker.Request() during/after shutdown when no tasks remain.
	// This is a terminal state indicating the queue has been shut down and drained.
	// Once this error appears, no new tasks will be processed.
	// Triggered by: calling Shutdown() or Release() on the queue.
	ErrQueueHasBeenClosed = errors.New("golang-queue: queue has been closed")

	// ErrMaxCapacity is returned by Queue() or QueueTask() when the queue is at maximum capacity.
	// This only occurs when WithQueueSize() is used to set a capacity limit.
	// To handle: retry later, drop the task, or process it synchronously.
	// Triggered by: attempting to enqueue when count >= capacity.
	ErrMaxCapacity = errors.New("golang-queue: maximum size limit reached")
)
