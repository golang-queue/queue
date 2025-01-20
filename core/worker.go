package core

import (
	"context"
)

// Worker represents an interface for a worker that processes tasks.
// It provides methods to run tasks, shut down the worker, queue tasks, and request tasks from the queue.
type Worker interface {
	// Run starts the worker and processes the given task in the provided context.
	// It returns an error if the task cannot be processed.
	Run(ctx context.Context, task TaskMessage) error

	// Shutdown stops the worker and performs any necessary cleanup.
	// It returns an error if the shutdown process fails.
	Shutdown() error

	// Queue adds a task to the worker's queue.
	// It returns an error if the task cannot be added to the queue.
	Queue(task TaskMessage) error

	// Request retrieves a task from the worker's queue.
	// It returns the queued message and an error if the retrieval fails.
	Request() (TaskMessage, error)
}

// QueuedMessage represents an interface for a message that can be queued.
// It requires the implementation of a Bytes method, which returns the message
// content as a slice of bytes.
type QueuedMessage interface {
	Bytes() []byte
}

// TaskMessage represents an interface for a task message that can be queued.
// It embeds the QueuedMessage interface and adds a method to retrieve the payload of the message.
type TaskMessage interface {
	QueuedMessage
	Payload() []byte
}
