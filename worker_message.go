package queue

import (
	"errors"
	"time"
)

var _ Worker = (*messageWorker)(nil)

// just for unit testing, don't use it.
type messageWorker struct {
	messages chan QueuedMessage
}

func (w *messageWorker) Run(task QueuedMessage) error {
	if string(task.Bytes()) == "panic" {
		panic("show panic")
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (w *messageWorker) Shutdown() error {
	close(w.messages)
	return nil
}

func (w *messageWorker) Queue(task QueuedMessage) error {
	select {
	case w.messages <- task:
		return nil
	default:
		return errors.New("max capacity reached")
	}
}

func (w *messageWorker) Request() (QueuedMessage, error) {
	select {
	case task := <-w.messages:
		return task, nil
	default:
		return nil, ErrNoTaskInQueue
	}
}
