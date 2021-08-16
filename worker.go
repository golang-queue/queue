package queue

// Worker interface
type Worker interface {
	// BeforeRun is called before starting the worker
	BeforeRun() error
	// Run is called to start the worker
	Run() error
	// BeforeRun is called after starting the worker
	AfterRun() error
	// Shutdown is called if stop all worker
	Shutdown() error
	// Queue to send message in Queue (single channel, NSQ or AWS SQS)
	Queue(job QueuedMessage) error
	// Capacity queue capacity = cap(channel name)
	Capacity() int
	// Usage is how many message in queue
	Usage() int
}

// QueuedMessage ...
type QueuedMessage interface {
	Bytes() []byte
}
