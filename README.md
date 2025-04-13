# Queue

[![CodeQL](https://github.com/golang-queue/queue/actions/workflows/codeql.yaml/badge.svg)](https://github.com/golang-queue/queue/actions/workflows/codeql.yaml)
[![Run Tests](https://github.com/golang-queue/queue/actions/workflows/go.yml/badge.svg)](https://github.com/golang-queue/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/golang-queue/queue/branch/master/graph/badge.svg?token=SSo3mHejOE)](https://codecov.io/gh/golang-queue/queue)

[繁體中文](./README.zh-tw.md) | [简体中文](./README.zh-cn.md)

Queue is a Golang library designed to help you create and manage a pool of Goroutines (lightweight threads). It allows you to efficiently run multiple tasks in parallel, utilizing the full CPU capacity of your machine.

## Features

- [x] Supports [Circular buffer](https://en.wikipedia.org/wiki/Circular_buffer) queues.
- [x] Integrates with [NSQ](https://nsq.io/) for real-time distributed messaging.
- [x] Integrates with [NATS](https://nats.io/) for adaptive edge and distributed systems.
- [x] Integrates with [Redis Pub/Sub](https://redis.io/docs/manual/pubsub/).
- [x] Integrates with [Redis Streams](https://redis.io/docs/manual/data-types/streams/).
- [x] Integrates with [RabbitMQ](https://www.rabbitmq.com/).

## Queue Scenario

A simple queue service using a ring buffer as the default backend.

![queue01](./images/flow-01.svg)

Easily switch the queue service to use NSQ, NATS, or Redis.

![queue02](./images/flow-02.svg)

Supports multiple producers and consumers.

![queue03](./images/flow-03.svg)

## Requirements

Go version **1.22** or above

## Installation

To install the stable version:

```sh
go get github.com/golang-queue/queue
```

To install the latest version:

```sh
go get github.com/golang-queue/queue@master
```

## Usage

### Basic Usage of Pool (using the Task function)

By calling the `QueueTask()` method, you can schedule tasks to be executed by workers (Goroutines) in the pool.

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

  // initialize the queue pool
  q := queue.NewPool(5)
  // shut down the service and notify all workers
  // wait until all jobs are complete
  defer q.Release()

  // assign tasks to the queue
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

  // wait until all tasks are done
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(20 * time.Millisecond)
  }
}
```

### Basic Usage of Pool (using a message queue)

Define a new message struct and implement the `Bytes()` function to encode the message. Use the `WithFn` function to handle messages from the queue.

```go
package main

import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "time"

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

  // initialize the queue pool
  q := queue.NewPool(5, queue.WithFn(func(ctx context.Context, m core.TaskMessage) error {
    var v job
    if err := json.Unmarshal(m.Payload(), &v); err != nil {
      return err
    }

    rets <- "Hi, " + v.Name + ", " + v.Message
    return nil
  }))
  // shut down the service and notify all workers
  // wait until all jobs are complete
  defer q.Release()

  // assign tasks to the queue
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

  // wait until all tasks are done
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }
}
```

## Using NSQ as a Queue

Refer to the [NSQ documentation](https://github.com/golang-queue/nsq) for more details.

```go
package main

import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "time"

  "github.com/golang-queue/nsq"
  "github.com/golang-queue/queue"
  "github.com/golang-queue/queue/core"
)

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

func main() {
  taskN := 100
  rets := make(chan string, taskN)

  // define the worker
  w := nsq.NewWorker(
    nsq.WithAddr("127.0.0.1:4150"),
    nsq.WithTopic("example"),
    nsq.WithChannel("foobar"),
    // concurrent job number
    nsq.WithMaxInFlight(10),
    nsq.WithRunFunc(func(ctx context.Context, m core.TaskMessage) error {
      var v job
      if err := json.Unmarshal(m.Payload(), &v); err != nil {
        return err
      }

      rets <- v.Message
      return nil
    }),
  )

  // define the queue
  q := queue.NewPool(
    5,
    queue.WithWorker(w),
  )

  // assign tasks to the queue
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("handle the job: %d", i+1),
      })
    }(i)
  }

  // wait until all tasks are done
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // shut down the service and notify all workers
  q.Release()
}
```

## Using NATS as a Queue

Refer to the [NATS documentation](https://github.com/golang-queue/nats) for more details.

```go
package main

import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "time"

  "github.com/golang-queue/nats"
  "github.com/golang-queue/queue"
  "github.com/golang-queue/queue/core"
)

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

func main() {
  taskN := 100
  rets := make(chan string, taskN)

  // define the worker
  w := nats.NewWorker(
    nats.WithAddr("127.0.0.1:4222"),
    nats.WithSubj("example"),
    nats.WithQueue("foobar"),
    nats.WithRunFunc(func(ctx context.Context, m core.TaskMessage) error {
      var v job
      if err := json.Unmarshal(m.Payload(), &v); err != nil {
        return err
      }

      rets <- v.Message
      return nil
    }),
  )

  // define the queue
  q, err := queue.NewQueue(
    queue.WithWorkerCount(10),
    queue.WithWorker(w),
  )
  if err != nil {
    log.Fatal(err)
  }

  // start the workers
  q.Start()

  // assign tasks to the queue
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("handle the job: %d", i+1),
      })
    }(i)
  }

  // wait until all tasks are done
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // shut down the service and notify all workers
  q.Release()
}
```

## Using Redis (Pub/Sub) as a Queue

Refer to the [Redis documentation](https://github.com/golang-queue/redisdb) for more details.

```go
package main

import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "time"

  "github.com/golang-queue/queue"
  "github.com/golang-queue/queue/core"
  "github.com/golang-queue/redisdb"
)

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

func main() {
  taskN := 100
  rets := make(chan string, taskN)

  // define the worker
  w := redisdb.NewWorker(
    redisdb.WithAddr("127.0.0.1:6379"),
    redisdb.WithChannel("foobar"),
    redisdb.WithRunFunc(func(ctx context.Context, m core.TaskMessage) error {
      var v job
      if err := json.Unmarshal(m.Payload(), &v); err != nil {
        return err
      }

      rets <- v.Message
      return nil
    }),
  )

  // define the queue
  q, err := queue.NewQueue(
    queue.WithWorkerCount(10),
    queue.WithWorker(w),
  )
  if err != nil {
    log.Fatal(err)
  }

  // start the workers
  q.Start()

  // assign tasks to the queue
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("handle the job: %d", i+1),
      })
    }(i)
  }

  // wait until all tasks are done
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // shut down the service and notify all workers
  q.Release()
}
```
