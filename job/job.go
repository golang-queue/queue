package job

import (
	"context"
	"time"
	"unsafe"

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

	// RetryFactor is the multiplying factor for each increment step.
	//
	// Defaults to 2.
	RetryFactor float64 `json:"retry_factor"`

	// Minimum value of the counter.
	//
	// Defaults to 100 milliseconds.
	RetryMin time.Duration `json:"retry_min"`

	// Maximum value of the counter.
	//
	// Defaults to 10 seconds.
	RetryMax time.Duration `json:"retry_max"`

	// Jitter eases contention by randomizing backoff steps
	Jitter bool `json:"jitter"`

	// Data to save Unsafe cast
	Data []byte
}

const (
	movementSize = int(unsafe.Sizeof(Message{}))
)

// Bytes get internal data
func (m Message) Bytes() []byte {
	return m.Data
}

// Encode for encoding the structure
func (m *Message) Encode() {
	m.Data = Encode(m)
}

// NewMessage create new message
func NewMessage(m core.QueuedMessage, opts ...AllowOption) Message {
	o := NewOptions(opts...)

	return Message{
		RetryCount:  o.retryCount,
		RetryDelay:  o.retryDelay,
		RetryFactor: o.retryFactor,
		RetryMin:    o.retryMin,
		RetryMax:    o.retryMax,
		Timeout:     o.timeout,
		Payload:     m.Bytes(),
	}
}

// NewTask create new task
func NewTask(task TaskFunc, opts ...AllowOption) Message {
	o := NewOptions(opts...)

	return Message{
		Timeout:     o.timeout,
		RetryCount:  o.retryCount,
		RetryDelay:  o.retryDelay,
		RetryFactor: o.retryFactor,
		RetryMin:    o.retryMin,
		RetryMax:    o.retryMax,
		Task:        task,
	}
}

// Encode for encoding the structure
func Encode(m *Message) []byte {
	return (*[movementSize]byte)(unsafe.Pointer(m))[:]
}

// Decode for decoding the structure
func Decode(m []byte) *Message {
	return (*Message)(unsafe.Pointer(&m[0]))
}
