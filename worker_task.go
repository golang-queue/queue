package queue

import (
	"context"
	"errors"

	"github.com/golang-queue/queue/core"
)

var _ core.Worker = (*taskWorker)(nil)

// just for unit testing, don't use it.
type taskWorker struct {
	messages chan core.QueuedMessage
}

func (w *taskWorker) Run(task core.QueuedMessage) error {
	if v, ok := task.(*Job); ok {
		if v.Task != nil {
			_ = v.Task(context.Background())
		}
	}
	return nil
}

func (w *taskWorker) Shutdown() error {
	close(w.messages)
	return nil
}

func (w *taskWorker) Queue(job core.QueuedMessage) error {
	select {
	case w.messages <- job:
		return nil
	default:
		return errors.New("max capacity reached")
	}
}

func (w *taskWorker) Request() (core.QueuedMessage, error) {
	select {
	case task := <-w.messages:
		return task, nil
	default:
		return nil, ErrNoTaskInQueue
	}
}
