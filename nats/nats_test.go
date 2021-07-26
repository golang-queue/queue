package nats

import (
	"log"
	"testing"
	"time"

	"github.com/appleboy/queue"

	"github.com/stretchr/testify/assert"
)

var host = "nats"

func TestNATSDefaultFlow(t *testing.T) {
	m := &Job{
		Body: []byte("foo"),
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("test"),
		WithQueue("test"),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, q.Queue(m))
	m.Body = []byte("new message")
	assert.NoError(t, q.Queue(m))
	q.Shutdown()
	q.Wait()
}

func TestNATSShutdown(t *testing.T) {
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("test"),
		WithQueue("test"),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(1 * time.Second)
	q.Shutdown()
	// check shutdown once
	q.Shutdown()
	q.Wait()
}

func TestNATSCustomFuncAndWait(t *testing.T) {
	m := &Job{
		Body: []byte("foo"),
	}
	w := NewWorker(
		WithAddr(host+":4222"),
		WithSubj("test"),
		WithQueue("test"),
		WithRunFunc(func(msg queue.QueuedMessage, s <-chan struct{}) error {
			log.Println("show message: " + string(msg.Bytes()))
			time.Sleep(500 * time.Millisecond)
			return nil
		}),
	)
	q, err := queue.NewQueue(
		queue.WithWorker(w),
		queue.WithWorkerCount(2),
	)
	assert.NoError(t, err)
	q.Start()
	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	assert.NoError(t, q.Queue(m))
	time.Sleep(700 * time.Millisecond)
	q.Shutdown()
	q.Wait()
	// you will see the execute time > 1000ms
}
