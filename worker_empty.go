package queue

var _ Worker = (*emptyWorker)(nil)

// just for unit testing, don't use it.
type emptyWorker struct{}

func (w *emptyWorker) BeforeRun() error              { return nil }
func (w *emptyWorker) AfterRun() error               { return nil }
func (w *emptyWorker) Run() error                    { return nil }
func (w *emptyWorker) Shutdown() error               { return nil }
func (w *emptyWorker) Queue(job QueuedMessage) error { return nil }
func (w *emptyWorker) Capacity() int                 { return 0 }
func (w *emptyWorker) Usage() int                    { return 0 }
