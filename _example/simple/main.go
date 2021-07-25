package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/appleboy/queue"
	"github.com/appleboy/queue/simple"
)

type job struct {
	message string
}

func (j *job) Bytes() []byte {
	return []byte(j.message)
}

func main() {
	taskN := 100
	rets := make(chan string, taskN)

	// define the worker
	w := simple.NewWorker(
		simple.WithQueueNum(taskN),
		simple.WithRunFunc(func(m queue.QueuedMessage) error {
			j, ok := m.(*job)
			if !ok {
				return errors.New("message is not job type")
			}

			rets <- j.message
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
				message: fmt.Sprintf("handle the job: %d", i+1),
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
