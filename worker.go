package queue

import (
	"context"
	"errors"
	"time"
)

// Worker interface
type Worker interface {
	// BeforeRun is called before starting the worker
	BeforeRun() error
	// Run is called to start the worker
	Run() error
	// BeforeRun is called after starting the worker
	AfterRun() error
	// Shutdown is called if stop all worker
	Shutdown() error
	// Queue to send message in Queue (single channel, NSQ or AWS SQS)
	Queue(job QueuedMessage) error
	// Capacity queue capacity = cap(channel name)
	Capacity() int
	// Usage is how many message in queue
	Usage() int
}

// QueuedMessage ...
type QueuedMessage interface {
	Bytes() []byte
}

var (
	_ Worker = (*emptyWorker)(nil)
	_ Worker = (*messageWorker)(nil)
)

type emptyWorker struct{}

func (w *emptyWorker) BeforeRun() error              { return nil }
func (w *emptyWorker) AfterRun() error               { return nil }
func (w *emptyWorker) Run() error                    { return nil }
func (w *emptyWorker) Shutdown() error               { return nil }
func (w *emptyWorker) Queue(job QueuedMessage) error { return nil }
func (w *emptyWorker) Capacity() int                 { return 0 }
func (w *emptyWorker) Usage() int                    { return 0 }

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
func (w *messageWorker) Capacity() int { return cap(w.messages) }
func (w *messageWorker) Usage() int    { return len(w.messages) }

type taskWorker struct {
	messages chan QueuedMessage
}

func (w *taskWorker) BeforeRun() error { return nil }
func (w *taskWorker) AfterRun() error  { return nil }
func (w *taskWorker) Run() error {
	for msg := range w.messages {
		if v, ok := msg.(Job); ok {
			if v.Task != nil {
				_ = v.Task(context.Background())
			}
		}
	}
	return nil
}

func (w *taskWorker) Shutdown() error {
	close(w.messages)
	return nil
}

func (w *taskWorker) Queue(job QueuedMessage) error {
	select {
	case w.messages <- job:
		return nil
	default:
		return errors.New("max capacity reached")
	}
}
func (w *taskWorker) Capacity() int { return cap(w.messages) }
func (w *taskWorker) Usage() int    { return len(w.messages) }
