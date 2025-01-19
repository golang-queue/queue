package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/golang-queue/queue/core"
)

// TaskFunc is the task function
type TaskFunc func(context.Context) error

// Message describes a task and its metadata.
type Message struct {
	Task TaskFunc `json:"-" msgpack:"-"`

	// Timeout is the duration the task can be processed by Handler.
	// zero if not specified
	// default is 60 time.Minute
	Timeout time.Duration `json:"timeout" msgpack:"timeout"`

	// Payload is the payload data of the task.
	Payload []byte `json:"body" msgpack:"body"`

	// RetryCount set count of retry
	// default is 0, no retry.
	RetryCount int64 `json:"retry_count" msgpack:"retry_count"`

	// RetryDelay set delay between retry
	// default is 100ms
	RetryDelay time.Duration `json:"retry_delay" msgpack:"retry_delay"`

	// RetryFactor is the multiplying factor for each increment step.
	//
	// Defaults to 2.
	RetryFactor float64 `json:"retry_factor" msgpack:"retry_factor"`

	// Minimum value of the counter.
	//
	// Defaults to 100 milliseconds.
	RetryMin time.Duration `json:"retry_min" msgpack:"retry_min"`

	// Maximum value of the counter.
	//
	// Defaults to 10 seconds.
	RetryMax time.Duration `json:"retry_max" msgpack:"retry_max"`

	// Jitter eases contention by randomizing backoff steps
	Jitter bool `json:"jitter" msgpack:"jitter"`

	// Data to save Unsafe cast
	Data []byte
}

// Bytes get internal data
func (m *Message) Bytes() []byte {
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

func Encode(m *Message) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return b
}

// Decode takes a byte slice and unmarshals it into a Message struct using msgpack.
// If the unmarshalling process encounters an error, the function will panic.
// It returns a pointer to the unmarshalled Message.
//
// Parameters:
//   - b: A byte slice containing the msgpack-encoded data.
//
// Returns:
//   - A pointer to the decoded Message struct.
func Decode(b []byte) *Message {
	var msg Message
	err := json.Unmarshal(b, &msg)
	if err != nil {
		panic(err)
	}

	return &msg
}
