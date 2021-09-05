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

func (w *messageWorker) BeforeRun() error { return nil }
func (w *messageWorker) AfterRun() error  { return nil }
func (w *messageWorker) Run() error {
	for msg := range w.messages {
		if string(msg.Bytes()) == "panic" {
			panic("show panic")
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

func (w *messageWorker) Shutdown() error {
	close(w.messages)
	return nil
}

func (w *messageWorker) Queue(job QueuedMessage) error {
	select {
	case w.messages <- job:
		return nil
	default:
		return errors.New("max capacity reached")
	}
}
func (w *messageWorker) Capacity() int       { return cap(w.messages) }
func (w *messageWorker) Usage() int          { return len(w.messages) }
func (w *messageWorker) BusyWorkers() uint64 { return uint64(0) }
