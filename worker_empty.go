package queue

import "github.com/golang-queue/queue/core"

var _ core.Worker = (*emptyWorker)(nil)

// just for unit testing, don't use it.
type emptyWorker struct{}

func (w *emptyWorker) Run(task core.QueuedMessage) error    { return nil }
func (w *emptyWorker) Shutdown() error                      { return nil }
func (w *emptyWorker) Queue(task core.QueuedMessage) error  { return nil }
func (w *emptyWorker) Request() (core.QueuedMessage, error) { return nil, nil }
