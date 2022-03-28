package queue

var _ Worker = (*emptyWorker)(nil)

// just for unit testing, don't use it.
type emptyWorker struct{}

func (w *emptyWorker) Run(task QueuedMessage) error    { return nil }
func (w *emptyWorker) Shutdown() error                 { return nil }
func (w *emptyWorker) Queue(task QueuedMessage) error  { return nil }
func (w *emptyWorker) Request() (QueuedMessage, error) { return nil, nil }
