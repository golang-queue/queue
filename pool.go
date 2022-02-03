package queue

// NewPool initializes a new pool
func NewPool(size int, opts ...Option) *Queue {
	o := []Option{
		WithWorkerCount(size),
		WithWorker(NewConsumer(opts...)),
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
