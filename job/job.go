package job

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang-queue/queue/core"
)

// TaskFunc is the task function
type TaskFunc func(context.Context) error

// Message describes a task and its metadata.
type Message struct {
	Task TaskFunc `json:"-"`

	// Timeout is the duration the task can be processed by Handler.
	// zero if not specified
	// default is 60 time.Minute
	Timeout time.Duration `json:"timeout"`

	// Payload is the payload data of the task.
	Payload []byte `json:"body"`

	// RetryCount set count of retry
	// default is 0, no retry.
	RetryCount int64 `json:"retry_count"`

	// RetryDelay set delay between retry
	// default is 100ms
	RetryDelay time.Duration `json:"retry_delay"`
}

// Bytes get string body
func (m *Message) Bytes() []byte {
	if m.Task != nil {
		return nil
	}
	return m.Payload
}

// Encode for encoding the structure
func (m *Message) Encode() []byte {
	b, _ := json.Marshal(m)

	return b
}

func NewMessage(m core.QueuedMessage, opts ...AllowOption) *Message {
	o := NewOptions(opts...)

	return &Message{
		RetryCount: o.retryCount,
		RetryDelay: o.retryDelay,
		Timeout:    o.timeout,
		Payload:    m.Bytes(),
	}
}

func NewTask(task TaskFunc, opts ...AllowOption) *Message {
	o := NewOptions(opts...)

	return &Message{
		Timeout:    o.timeout,
		RetryCount: o.retryCount,
		RetryDelay: o.retryDelay,
		Task:       task,
	}
}
