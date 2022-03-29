package queue

import "errors"

var (
	// ErrNoTaskInQueue there is nothing in the queue
	ErrNoTaskInQueue = errors.New("no task in queue")
	// ErrQueueHasBeenClosed the current queue is closed
	ErrQueueHasBeenClosed = errors.New("queue has been closed")
)
