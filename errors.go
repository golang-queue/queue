package queue

import "errors"

var (
	// ErrNoTaskInQueue there is nothing in the queue
	ErrNoTaskInQueue = errors.New("golang-queue: no task in queue")
	// ErrQueueHasBeenClosed the current queue is closed
	ErrQueueHasBeenClosed = errors.New("golang-queue: queue has been closed")
)
