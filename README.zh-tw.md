# Queue

[![CodeQL](https://github.com/golang-queue/queue/actions/workflows/codeql.yaml/badge.svg)](https://github.com/golang-queue/queue/actions/workflows/codeql.yaml)
[![Run Tests](https://github.com/golang-queue/queue/actions/workflows/go.yml/badge.svg)](https://github.com/golang-queue/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/golang-queue/queue/branch/master/graph/badge.svg?token=SSo3mHejOE)](https://codecov.io/gh/golang-queue/queue)

[English](./README.md) | [简体中文](./README.zh-cn.md)

Queue 是一個 Golang 函式庫，幫助你建立和管理 Goroutines（輕量級執行緒）的池。它允許你有效地並行執行多個任務，充分利用機器的 CPU 容量。

## 功能

- [x] 支援 [環形緩衝區](https://en.wikipedia.org/wiki/Circular_buffer) 隊列。
- [x] 與 [NSQ](https://nsq.io/) 整合，用於即時分佈式消息傳遞。
- [x] 與 [NATS](https://nats.io/) 整合，用於自適應邊緣和分佈式系統。
- [x] 與 [Redis Pub/Sub](https://redis.io/docs/manual/pubsub/) 整合。
- [x] 與 [Redis Streams](https://redis.io/docs/manual/data-types/streams/) 整合。
- [x] 與 [RabbitMQ](https://www.rabbitmq.com/) 整合。

## 隊列場景

使用環形緩衝區作為預設後端的簡單隊列服務。

![queue01](./images/flow-01.svg)

輕鬆切換隊列服務以使用 NSQ、NATS 或 Redis。

![queue02](./images/flow-02.svg)

支援多個生產者和消費者。

![queue03](./images/flow-03.svg)

## 要求

Go 版本 **1.22** 或以上

## 安裝

安裝穩定版本：

```sh
go get github.com/golang-queue/queue
```

安裝最新版本：

```sh
go get github.com/golang-queue/queue@master
```

## 使用方法

### 池的基本使用（使用 Task 函數）

通過調用 `QueueTask()` 方法，你可以安排任務由池中的工作者（Goroutines）執行。

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

  // 初始化隊列池
  q := queue.NewPool(5)
  // 關閉服務並通知所有工作者
  // 等待所有工作完成
  defer q.Release()

  // 將任務分配給隊列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      if err := q.QueueTask(func(ctx context.Context) error {
        rets <- fmt.Sprintf("Hi Gopher, 處理工作: %02d", +i)
        return nil
      }); err != nil {
        panic(err)
      }
    }(i)
  }

  // 等待所有任務完成
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(20 * time.Millisecond)
  }
}
```

### 池的基本使用（使用消息隊列）

定義一個新的消息結構並實現 `Bytes()` 函數來編碼消息。使用 `WithFn` 函數來處理來自隊列的消息。

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

  // 初始化隊列池
  q := queue.NewPool(5, queue.WithFn(func(ctx context.Context, m core.TaskMessage) error {
    var v job
    if err := json.Unmarshal(m.Payload(), &v); err != nil {
      return err
    }

    rets <- "Hi, " + v.Name + ", " + v.Message
    return nil
  }))
  // 關閉服務並通知所有工作者
  // 等待所有工作完成
  defer q.Release()

  // 將任務分配給隊列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      if err := q.Queue(&job{
        Name:    "Gopher",
        Message: fmt.Sprintf("處理工作: %d", i+1),
      }); err != nil {
        log.Println(err)
      }
    }(i)
  }

  // 等待所有任務完成
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }
}
```

## 使用 NSQ 作為隊列

請參閱 [NSQ 文檔](https://github.com/golang-queue/nsq) 以獲取更多詳細信息。

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

  // 定義工作者
  w := nsq.NewWorker(
    nsq.WithAddr("127.0.0.1:4150"),
    nsq.WithTopic("example"),
    nsq.WithChannel("foobar"),
    // 並發工作數量
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

  // 定義隊列
  q := queue.NewPool(
    5,
    queue.WithWorker(w),
  )

  // 將任務分配給隊列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("處理工作: %d", i+1),
      })
    }(i)
  }

  // 等待所有任務完成
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // 關閉服務並通知所有工作者
  q.Release()
}
```

## 使用 NATS 作為隊列

請參閱 [NATS 文檔](https://github.com/golang-queue/nats) 以獲取更多詳細信息。

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

  // 定義工作者
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

  // 定義隊列
  q, err := queue.NewQueue(
    queue.WithWorkerCount(10),
    queue.WithWorker(w),
  )
  if err != nil {
    log.Fatal(err)
  }

  // 啟動工作者
  q.Start()

  // 將任務分配給隊列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("處理工作: %d", i+1),
      })
    }(i)
  }

  // 等待所有任務完成
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // 關閉服務並通知所有工作者
  q.Release()
}
```

## 使用 Redis (Pub/Sub) 作為隊列

請參閱 [Redis 文檔](https://github.com/golang-queue/redisdb) 以獲取更多詳細信息。

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

  // 定義工作者
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

  // 定義隊列
  q, err := queue.NewQueue(
    queue.WithWorkerCount(10),
    queue.WithWorker(w),
  )
  if err != nil {
    log.Fatal(err)
  }

  // 啟動工作者
  q.Start()

  // 將任務分配給隊列
  for i := 0; i < taskN; i++ {
    go func(i int) {
      q.Queue(&job{
        Message: fmt.Sprintf("處理工作: %d", i+1),
      })
    }(i)
  }

  // 等待所有任務完成
  for i := 0; i < taskN; i++ {
    fmt.Println("message:", <-rets)
    time.Sleep(50 * time.Millisecond)
  }

  // 關閉服務並通知所有工作者
  q.Release()
}
```
