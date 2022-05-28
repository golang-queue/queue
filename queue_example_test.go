package queue_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-queue/queue"
)

func ExampleNewPool_queueTask() {
	taskN := 7
	rets := make(chan int, taskN)
	// allocate a pool with 5 goroutines to deal with those tasks
	p := queue.NewPool(5)
	// don't forget to release the pool in the end
	defer p.Release()

	// assign tasks to asynchronous goroutine pool
	for i := 0; i < taskN; i++ {
		idx := i
		if err := p.QueueTask(func(context.Context) error {
			// sleep and return the index
			time.Sleep(20 * time.Millisecond)
			rets <- idx
			return nil
		}); err != nil {
			log.Println(err)
		}
	}

	// wait until all tasks done
	for i := 0; i < taskN; i++ {
		fmt.Println("index:", <-rets)
	}

	// Unordered output:
	// index: 3
	// index: 0
	// index: 2
	// index: 4
	// index: 5
	// index: 6
	// index: 1
}

func ExampleNewPool_queueTaskTimeout() {
	taskN := 7
	rets := make(chan int, taskN)
	resps := make(chan error, 1)
	// allocate a pool with 5 goroutines to deal with those tasks
	q := queue.NewPool(5)
	// don't forget to release the pool in the end
	defer q.Release()

	// assign tasks to asynchronous goroutine pool
	for i := 0; i < taskN; i++ {
		idx := i
		if err := q.QueueTaskWithTimeout(100*time.Millisecond, func(ctx context.Context) error {
			// panic job
			if idx == 5 {
				panic("system error")
			}
			// timeout job
			if idx == 6 {
				time.Sleep(105 * time.Millisecond)
			}
			select {
			case <-ctx.Done():
				resps <- ctx.Err()
			default:
			}

			rets <- idx
			return nil
		}); err != nil {
			log.Println(err)
		}
	}

	// wait until all tasks done
	for i := 0; i < taskN-1; i++ {
		fmt.Println("index:", <-rets)
	}
	close(resps)
	for e := range resps {
		fmt.Println(e.Error())
	}

	fmt.Println("success task count:", q.SuccessTasks())
	fmt.Println("failure task count:", q.FailureTasks())

	// Unordered output:
	// index: 3
	// index: 0
	// index: 2
	// index: 4
	// index: 6
	// index: 1
	// context deadline exceeded
	// success task count: 5
	// failure task count: 2
}
