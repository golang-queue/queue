package core

// Worker interface
type Worker interface {
	// Run is called to start the worker
	Run(task QueuedMessage) error
	// Shutdown is called if stop all worker
	Shutdown() error
	// Queue to send message in Queue
	Queue(task QueuedMessage) error
	// Request to get message from Queue
	Request() (QueuedMessage, error)
}

// QueuedMessage ...
type QueuedMessage interface {
	Bytes() []byte
}
