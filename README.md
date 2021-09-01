# Queue

[![Run Tests](https://github.com/golang-queue/queue/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/golang-queue/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/golang-queue/nsq/branch/main/graph/badge.svg?token=D3CUES8M62)](https://codecov.io/gh/golang-queue/nsq)

Queue is a Golang library for spawning and managing a Goroutine pool, Alloowing you to create multiple worker according to limit CPU number of machine.

## Features

* [x] Support [buffered channel](https://gobyexample.com/channel-buffering) queue.
* [x] Support [NSQ](https://nsq.io/) (A realtime distributed messaging platform) as backend.
* [x] Support [NATS](https://nats.io/) (Connective Technology for Adaptive Edge & Distributed Systems) as backend.
* [x] Support [Redis Pub/Sub](https://pkg.go.dev/github.com/go-redis/redis/v8#PubSub) as backend.

## Installation

Install the stable version:

```sh
go get github.com/golang-queue/queue
```

Install the latest verison:

```sh
go get github.com/golang-queue/queue@master
```

## Usage

### Basic usage of Pool (use Task function)

By calling `QueueTask()` method, it schedules the task executed by worker (goroutines) in the Pool.

```go
package main

import (
  "context"
  "fmt"
  "time"

  "github.com/golang-queue/queue"
)

func main() {
  taskN := 100
  rets := make(chan string, taskN)

  // initial queue pool
  q := queue.NewPool(5)
  // shutdown the service and notify all the worker
  // wait all jobs are complete.
  defer q.Release()

  // assign tasks in queue
  for i := 0; i < taskN; i++ {
    go func(i int) {
      if err := q.QueueTask(func(ctx context.Context) error {
        rets <- fmt.Sprintf("Hi Gopher, handle the job: %02d", +i)
        return nil
      }); err != nil {
        panic(err)
      }
    }(i)
  }

  // wait until all tasks done
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(20 * time.Millisecond)
  }
}
```

### Basic usage of Pool (use message queue)

Define the new message struct and implement the `Bytes()` func to encode message. Give the `WithFn` func
to handle the message from Queue.

```go
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

  // initial queue pool
  q := queue.NewPool(5, queue.WithFn(func(ctx context.Context, m queue.QueuedMessage) error {
    v, ok := m.(*job)
    if !ok {
      if err := json.Unmarshal(m.Bytes(), &v); err != nil {
        return err
      }
    }

    rets <- "Hi, " + v.Name + ", " + v.Message
    return nil
  }))
  // shutdown the service and notify all the worker
  // wait all jobs are complete.
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
    time.Sleep(50 * time.Millisecond)
  }
}
```
