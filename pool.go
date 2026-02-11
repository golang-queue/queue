package queue

// NewPool creates a ready-to-use in-memory queue with the Ring buffer worker.
// This is the recommended way to create a queue for most use cases.
//
// Key differences from NewQueue:
//   - Automatically creates and attaches a Ring buffer worker (no need for WithWorker)
//   - Calls Start() automatically so the queue begins processing immediately
//   - Panics on error instead of returning an error (simplifies initialization)
//
// Parameters:
//   - size: Number of worker goroutines (if <= 0, defaults to runtime.NumCPU())
//   - opts: Additional options to customize the queue (WithLogger, WithQueueSize, etc.)
//
// Example:
//
//	// Create a pool with 5 workers and custom capacity
//	q := queue.NewPool(5, queue.WithQueueSize(100))
//	defer q.Release()
//
//	// Queue tasks
//	q.QueueTask(func(ctx context.Context) error {
//	    // Process task
//	    return nil
//	})
//
// Use NewQueue instead if you need:
//   - Custom worker implementations (NSQ, NATS, Redis, etc.)
//   - Manual control over when to start the queue
//   - Error handling during queue creation
func NewPool(size int64, opts ...Option) *Queue {
	o := []Option{
		WithWorkerCount(size),
		WithWorker(NewRing(opts...)),
	}
	o = append(
		o,
		opts...,
	)

	q, err := NewQueue(o...)
	if err != nil {
		panic(err)
	}

	q.Start()

	return q
}
