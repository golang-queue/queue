# 队列

[![CodeQL](https://github.com/golang-queue/queue/actions/workflows/codeql.yaml/badge.svg)](https://github.com/golang-queue/queue/actions/workflows/codeql.yaml)
[![Run Tests](https://github.com/golang-queue/queue/actions/workflows/go.yml/badge.svg)](https://github.com/golang-queue/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/golang-queue/queue/branch/master/graph/badge.svg?token=SSo3mHejOE)](https://codecov.io/gh/golang-queue/queue)

[English](./README.md) | [繁體中文](./README.zh-tw.md)

Queue 是一个 Golang 库，帮助您创建和管理 Goroutines（轻量级线程）池。它允许您高效地并行运行多个任务，利用机器的 CPU 容量。

## 特性

- [x] 支持 [循环缓冲区](https://en.wikipedia.org/wiki/Circular_buffer) 队列。
- [x] 集成 [NSQ](https://nsq.io/) 进行实时分布式消息传递。
- [x] 集成 [NATS](https://nats.io/) 以适应边缘和分布式系统。
- [x] 集成 [Redis Pub/Sub](https://redis.io/docs/manual/pubsub/)。
- [x] 集成 [Redis Streams](https://redis.io/docs/manual/data-types/streams/)。
- [x] 集成 [RabbitMQ](https://www.rabbitmq.com/)。

## 队列场景

使用环形缓冲区作为默认后端的简单队列服务。

![queue01](./images/flow-01.svg)

轻松切换队列服务以使用 NSQ、NATS 或 Redis。

![queue02](./images/flow-02.svg)

支持多个生产者和消费者。

![queue03](./images/flow-03.svg)

## 要求

Go 版本 **1.22** 或以上

## 安装

安装稳定版本：

```sh
go get github.com/golang-queue/queue
```

安装最新版本：

```sh
go get github.com/golang-queue/queue@master
```

## 使用

### 池的基本用法（使用 Task 函数）

通过调用 `QueueTask()` 方法，您可以安排任务由池中的工作者（Goroutines）执行。

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

  // 初始化队列池
  q := queue.NewPool(5)
  // 关闭服务并通知所有工作者
  // 等待所有作业完成
  defer q.Release()

  // 将任务分配到队列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      if err := q.QueueTask(func(ctx context.Context) error {
        rets <- fmt.Sprintf("Hi Gopher, 处理作业: %02d", +i)
        return nil
      }); err != nil {
        panic(err)
      }
    }(i)
  }

  // 等待所有任务完成
  for i := 0; i < taskN; i++ {
    fmt.Println("消息:", <-rets)
    time.Sleep(20 * time.Millisecond)
  }
}
```

### 池的基本用法（使用消息队列）

定义一个新的消息结构并实现 `Bytes()` 函数来编码消息。使用 `WithFn` 函数处理来自队列的消息。

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

  // 初始化队列池
  q := queue.NewPool(5, queue.WithFn(func(ctx context.Context, m core.TaskMessage) error {
    var v job
    if err := json.Unmarshal(m.Payload(), &v); err != nil {
      return err
    }

    rets <- "Hi, " + v.Name + ", " + v.Message
    return nil
  }))
  // 关闭服务并通知所有工作者
  // 等待所有作业完成
  defer q.Release()

  // 将任务分配到队列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      if err := q.Queue(&job{
        Name:    "Gopher",
        Message: fmt.Sprintf("处理作业: %d", i+1),
      }); err != nil {
        log.Println(err)
      }
    }(i)
  }

  // 等待所有任务完成
  for i := 0; i < taskN; i++ {
    fmt.Println("消息:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }
}
```

## 使用 NSQ 作为队列

请参阅 [NSQ 文档](https://github.com/golang-queue/nsq) 了解更多详情。

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

  // 定义工作者
  w := nsq.NewWorker(
    nsq.WithAddr("127.0.0.1:4150"),
    nsq.WithTopic("example"),
    nsq.WithChannel("foobar"),
    // 并发作业数量
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

  // 定义队列
  q := queue.NewPool(
    5,
    queue.WithWorker(w),
  )

  // 将任务分配到队列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("处理作业: %d", i+1),
      })
    }(i)
  }

  // 等待所有任务完成
  for i := 0; i < taskN; i++ {
    fmt.Println("消息:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // 关闭服务并通知所有工作者
  q.Release()
}
```

## 使用 NATS 作为队列

请参阅 [NATS 文档](https://github.com/golang-queue/nats) 了解更多详情。

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

  // 定义工作者
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

  // 定义队列
  q, err := queue.NewQueue(
    queue.WithWorkerCount(10),
    queue.WithWorker(w),
  )
  if err != nil {
    log.Fatal(err)
  }

  // 启动工作者
  q.Start()

  // 将任务分配到队列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("处理作业: %d", i+1),
      })
    }(i)
  }

  // 等待所有任务完成
  for i := 0; i < taskN; i++ {
    fmt.Println("消息:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // 关闭服务并通知所有工作者
  q.Release()
}
```

## 使用 Redis (Pub/Sub) 作为队列

请参阅 [Redis 文档](https://github.com/golang-queue/redisdb) 了解更多详情。

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

  // 定义工作者
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

  // 定义队列
  q, err := queue.NewQueue(
    queue.WithWorkerCount(10),
    queue.WithWorker(w),
  )
  if err != nil {
    log.Fatal(err)
  }

  // 启动工作者
  q.Start()

  // 将任务分配到队列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("处理作业: %d", i+1),
      })
    }(i)
  }

  // 等待所有任务完成
  for i := 0; i < taskN; i++ {
    fmt.Println("消息:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // 关闭服务并通知所有工作者
  q.Release()
}
```
