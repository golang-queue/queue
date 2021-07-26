package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/appleboy/queue"
	"github.com/appleboy/queue/simple"
)

type job struct {
	Message string
}

func (j *job) Bytes() []byte {
	return []byte(j.Message)
}

func main() {
	taskN := 100
	rets := make(chan string, taskN)

	// define the worker
	w := simple.NewWorker(
		simple.WithQueueNum(taskN),
		simple.WithRunFunc(func(m queue.QueuedMessage, _ <-chan struct{}) error {
			v, ok := m.(*job)
			if !ok {
				if err := json.Unmarshal(m.Bytes(), &v); err != nil {
					return err
				}
			}

			rets <- v.Message
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
				Message: fmt.Sprintf("handle the job: %d", i+1),
			})
		}(i)
	}

	// wait until all tasks done
	for i := 0; i < taskN; i++ {
		fmt.Println("message:", <-rets)
		time.Sleep(50 * time.Millisecond)
	}

	q.Shutdown()
	q.Wait()
}
