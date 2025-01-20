package core

import (
	"context"
)

// Worker represents a worker that processes queued messages.
// It provides methods to run the worker, shut it down, queue messages, and request messages from the queue.
type Worker interface {
	// Run starts the worker and processes the given task in the provided context.
	// It returns an error if the task cannot be processed.
	Run(ctx context.Context, task QueuedMessage) error

	// Shutdown stops the worker and performs any necessary cleanup.
	// It returns an error if the shutdown process fails.
	Shutdown() error

	// Queue adds a task to the worker's queue.
	// It returns an error if the task cannot be added to the queue.
	Queue(task QueuedMessage) error

	// Request retrieves a task from the worker's queue.
	// It returns the queued message and an error if the retrieval fails.
	Request() (QueuedMessage, error)
}

// QueuedMessage represents an interface for a message that can be queued.
// It requires the implementation of a Bytes method, which returns the message
// content as a slice of bytes.
type QueuedMessage interface {
	Bytes() []byte
}

type TaskMessage interface {
	QueuedMessage
	Payload() []byte
}
