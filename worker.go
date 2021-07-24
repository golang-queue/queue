package queue

// Worker interface
type Worker interface {
	BeforeRun() error
	Run(chan struct{}) error
	AfterRun() error

	Shutdown() error
	Queue(job QueuedMessage) error
	Capacity() int
	Usage() int
}

// QueuedMessage ...
type QueuedMessage interface {
	Bytes() []byte
}

var _ Worker = (*emptyWorker)(nil)

type emptyWorker struct{}

func (w *emptyWorker) BeforeRun() error              { return nil }
func (w *emptyWorker) AfterRun() error               { return nil }
func (w *emptyWorker) Run(chan struct{}) error       { return nil }
func (w *emptyWorker) Shutdown() error               { return nil }
func (w *emptyWorker) Queue(job QueuedMessage) error { return nil }
func (w *emptyWorker) Capacity() int                 { return 0 }
func (w *emptyWorker) Usage() int                    { return 0 }
