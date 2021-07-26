# Queue

[![Run Tests](https://github.com/appleboy/queue/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/appleboy/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/appleboy/queue/branch/master/graph/badge.svg?token=V8A1WA0P5E)](https://codecov.io/gh/appleboy/queue)
[![Build Status](https://cloud.drone.io/api/badges/appleboy/queue/status.svg)](https://cloud.drone.io/appleboy/queue)

Queue is a Golang library for spawning and managing a Goroutine pool, Alloowing you to create multiple worker according to limit CPU number of machine.

## Features

* [x] Support [buffered channel](https://gobyexample.com/channel-buffering) queue.
* [x] Support [NSQ](https://nsq.io/) (A realtime distributed messaging platform) as backend.
* [x] Support [NATS](https://nats.io/) (Connective Technology for Adaptive Edge & Distributed Systems) as backend.

## Installation

Install the stable version:

```sh
go get github.com/appleboy/queue
```

Install the latest verison:

```sh
go get github.com/appleboy/queue@master
```

## Usage

The first step to create a new job as `QueueMessage` interface:

```go
type job struct {
  Message string
}

func (j *job) Bytes() []byte {
  return []byte(j.Message)
}
```

The second step to create the new worker, use the buffered channel as an example, you can use the `stop` channel to terminate the job immediately after shutdown the queue service if need.

```go
// define the worker
w := simple.NewWorker(
  simple.WithQueueNum(taskN),
  simple.WithRunFunc(func(m queue.QueuedMessage, stop <-chan struct{}) error {
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
  nsq.WithRunFunc(func(m queue.QueuedMessage, stop <-chan struct{}) error {
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

Full example code as below or [try it in playground](https://play.golang.org/p/yaTUoYxdcaK).

```go
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

  // shutdown the service and notify all the worker
  q.Shutdown()
  // wait all jobs are complete.
  q.Wait()
}
```
