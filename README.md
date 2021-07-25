# Queue

[![Run Tests](https://github.com/appleboy/queue/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/appleboy/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/appleboy/queue/branch/master/graph/badge.svg?token=V8A1WA0P5E)](https://codecov.io/gh/appleboy/queue)

Queue is a Golang library for spawning and managing a Goroutine pool, Alloowing you to create multiple worker according to limit CPU number of machine.

## Features

* [x] Support [buffered channel](https://gobyexample.com/channel-buffering) queue.
* [x] Support [NSQ](https://nsq.io/) (A realtime distributed messaging platform) as backend.

## Installation

```sh
go get github.com/appleboy/queue
```

## Usage

First to create new job as `QueueMessage` interface:

```go
type job struct {
	message string
}

func (j *job) Bytes() []byte {
	return []byte(j.message)
}
```

Second to create the new worker, use buffered channel as example:

```go
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
```

Third to create queue and initialize multiple worker, receive all job message:

```go
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
```

Full example code as below or [try it in playground](https://play.golang.org/p/DlhCQgZZ1Bb).

```go
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
```
