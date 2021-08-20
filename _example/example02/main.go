package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-queue/queue"
)

func main() {
	taskN := 100
	rets := make(chan string, taskN)

	// define the worker
	w := queue.NewConsumer(
		queue.WithQueueSize(taskN),
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
			q.QueueTask(func(ctx context.Context) error {
				rets <- fmt.Sprintf("Hi appleboy, handle the job: %02d", +i)
				return nil
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
