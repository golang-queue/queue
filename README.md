# Queue

[![Run Tests](https://github.com/appleboy/queue/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/appleboy/queue/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/appleboy/queue/branch/master/graph/badge.svg?token=V8A1WA0P5E)](https://codecov.io/gh/appleboy/queue)

Queue is a Golang library for spawning and managing a Goroutine pool, Alloowing you to create multiple worker according to limit CPU number of machine.

## Features

* [x] Support [buffered channel](https://gobyexample.com/channel-buffering) queue.
* [x] Support [NSQ](https://nsq.io/) (A realtime distributed messaging platform)

## Installation

```sh
go get github.com/appleboy/queue
```

## Usage
