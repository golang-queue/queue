package queue

// Worker interface
type Worker interface {
	// BeforeRun is called before starting the worker
	BeforeRun() error
	// Run is called to start the worker
	Run(task QueuedMessage) error
	// BeforeRun is called after starting the worker
	AfterRun() error
	// Shutdown is called if stop all worker
	Shutdown() error
	// Queue to send message in Queue
	Queue(task QueuedMessage) error
	// Request to get message from Queue
	Request() (QueuedMessage, error)
	// Capacity queue capacity = cap(channel name)
	Capacity() int
	// Usage is how many message in queue
	Usage() int
	// BusyWorkers return count of busy worker currently
	BusyWorkers() uint64
}

// QueuedMessage ...
type QueuedMessage interface {
	Bytes() []byte
}
