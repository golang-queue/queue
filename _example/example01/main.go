package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/golang-queue/queue"
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

	// define the worker
	w := queue.NewConsumer(
		queue.WithQueueSize(taskN),
		queue.WithFn(func(ctx context.Context, m queue.QueuedMessage) error {
			v, ok := m.(*job)
			if !ok {
				if err := json.Unmarshal(m.Bytes(), &v); err != nil {
					return err
				}
			}

			rets <- "Hi, " + v.Name + ", " + v.Message
			return nil
		}),
	)

	// define the queue
	q, err := queue.NewQueue(
		queue.WithWorkerCount(5),
		queue.WithWorker(w),
	)
	if err != nil {
		log.Fatal(err)
	}

	// start the five worker
	q.Start()

	// assign tasks in queue
	for i := 0; i < taskN; i++ {
		go func(i int) {
			q.Queue(&job{
				Name:    "foobar",
				Message: fmt.Sprintf("handle the job: %d", i+1),
			})
		}(i)
	}

	// wait until all tasks done
	for i := 0; i < taskN; i++ {
		fmt.Println("message:", <-rets)
		time.Sleep(50 * time.Millisecond)
	}

	// shutdown the service and notify all the worker
	q.Shutdown()
	// wait all jobs are complete.
	q.Wait()
}
