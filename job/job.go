package job

import (
	"context"
	"time"
	"unsafe"

	"github.com/golang-queue/queue/core"

	"github.com/goccy/go-json"
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

	Data []byte
}

const (
	movementSize = int(unsafe.Sizeof(Message{}))
)

// Bytes get string body
func (m *Message) Bytes() []byte {
	// if m.Task != nil {
	// 	return nil
	// }
	return m.Data
}

// Encode for encoding the structure
func (m *Message) Encode() []byte {
	b, _ := json.Marshal(m)

	return b
}

// Encode for encoding the structure
func (m *Message) UnsafeEncode() {
	m.Data = (*[movementSize]byte)(unsafe.Pointer(m))[:]
}

// Encode for encoding the structure
func (m *Message) UnsafeDecode() *Message {
	return (*Message)(unsafe.Pointer(&m.Payload[0]))
}

// Rest for reset default value
func (m *Message) Rest() {
	m.Task = nil
	m.Payload = nil
	m.RetryCount = 0
	// m.Timeout = 60 * time.Minute
	// m.RetryDelay = 100 * time.Millisecond
	m.Timeout = 0
	m.RetryDelay = 0
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

func Encode(m *Message) []byte {
	return (*[movementSize]byte)(unsafe.Pointer(m))[:]
}

func Decode(m []byte) *Message {
	return (*Message)(unsafe.Pointer(&m[0]))
}
