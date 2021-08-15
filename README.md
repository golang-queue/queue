# Queue

[![Run Tests](https://github.com/golang-queue/queue/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/golang-queue/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/golang-queue/nsq/branch/main/graph/badge.svg?token=D3CUES8M62)](https://codecov.io/gh/golang-queue/nsq)

Queue is a Golang library for spawning and managing a Goroutine pool, Alloowing you to create multiple worker according to limit CPU number of machine.

## Features

* [x] Support [buffered channel](https://gobyexample.com/channel-buffering) queue.
* [x] Support [NSQ](https://nsq.io/) (A realtime distributed messaging platform) as backend.
* [x] Support [NATS](https://nats.io/) (Connective Technology for Adaptive Edge & Distributed Systems) as backend.

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

The first step to create a new job as `QueueMessage` interface:

```go
type job struct {
  Message string
}

func (j *job) Bytes() []byte {
  b, err := json.Marshal(j)
  if err != nil {
    panic(err)
  }
  return b
}
```

The second step to create the new worker, use the buffered channel as an example.

```go
// define the worker
w := simple.NewWorker(
  simple.WithQueueNum(taskN),
  simple.WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
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
```

or use the [NSQ](https://nsq.io/) as backend, see the worker example:

```go
// define the worker
w := nsq.NewWorker(
  nsq.WithAddr("127.0.0.1:4150"),
  nsq.WithTopic("example"),
  nsq.WithChannel("foobar"),
  // concurrent job number
  nsq.WithMaxInFlight(10),
  nsq.WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
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
```

or use the [NATS](https://nats.io/) as backend, see the worker example:

```go
w := nats.NewWorker(
  nats.WithAddr("127.0.0.1:4222"),
  nats.WithSubj("example"),
  nats.WithQueue("foobar"),
  nats.WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
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
```

The third step to create a queue and initialize multiple workers, receive all job messages:

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
```

Full example code as below or [try it in playground](https://play.golang.org/p/77PtkZRaPE-).

```go
package main

import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "time"

  "github.com/golang-queue/queue"
  "github.com/golang-queue/queue/simple"
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
  w := simple.NewWorker(
    simple.WithQueueNum(taskN),
    simple.WithRunFunc(func(ctx context.Context, m queue.QueuedMessage) error {
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
```
