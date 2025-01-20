package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/golang-queue/contrib/zerolog"
	"github.com/golang-queue/queue"
	"github.com/golang-queue/queue/core"
)

type job struct {
	Name    string
	Message string
}

func (j *job) Bytes() []byte {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	return b
}

func main() {
	taskN := 100
	rets := make(chan string, taskN)

	// initial queue pool
	q := queue.NewPool(5, queue.WithFn(func(ctx context.Context, m core.TaskMessage) error {
		v, ok := m.(*job)
		if !ok {
			if err := json.Unmarshal(m.Payload(), &v); err != nil {
				return err
			}
		}

		rets <- "Hi, " + v.Name + ", " + v.Message
		return nil
	}), queue.WithLogger(zerolog.New()))
	// shutdown the service and notify all the worker
	// wait all jobs done.
	defer q.Release()

	// assign tasks in queue
	for i := 0; i < taskN; i++ {
		go func(i int) {
			if err := q.Queue(&job{
				Name:    "Gopher",
				Message: fmt.Sprintf("handle the job: %d", i+1),
			}); err != nil {
				log.Println(err)
			}
		}(i)
	}

	// wait until all tasks done
	for i := 0; i < taskN; i++ {
		fmt.Println("message:", <-rets)
		time.Sleep(20 * time.Millisecond)
	}
}
