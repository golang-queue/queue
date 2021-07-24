package queue

import (
	"errors"
	"log"
	"time"
)

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

var (
	_ Worker = (*emptyWorker)(nil)
	_ Worker = (*queueWorker)(nil)
)

type emptyWorker struct{}

func (w *emptyWorker) BeforeRun() error              { return nil }
func (w *emptyWorker) AfterRun() error               { return nil }
func (w *emptyWorker) Run(chan struct{}) error       { return nil }
func (w *emptyWorker) Shutdown() error               { return nil }
func (w *emptyWorker) Queue(job QueuedMessage) error { return nil }
func (w *emptyWorker) Capacity() int                 { return 0 }
func (w *emptyWorker) Usage() int                    { return 0 }

type queueWorker struct {
	messages chan QueuedMessage
}

func (w *queueWorker) BeforeRun() error { return nil }
func (w *queueWorker) AfterRun() error  { return nil }
func (w *queueWorker) Run(chan struct{}) error {
	for msg := range w.messages {
		log.Println("got message", msg)
		if string(msg.Bytes()) == "panic" {
			panic("show panic")
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (w *queueWorker) Shutdown() error {
	close(w.messages)
	return nil
}

func (w *queueWorker) Queue(job QueuedMessage) error {
	select {
	case w.messages <- job:
		return nil
	default:
		return errors.New("max capacity reached")
	}
}
func (w *queueWorker) Capacity() int { return cap(w.messages) }
func (w *queueWorker) Usage() int    { return len(w.messages) }
