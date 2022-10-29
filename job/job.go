package job

import (
	"context"
	"time"

	"github.com/goccy/go-json"
)

// TaskFunc is the task function
type TaskFunc func(context.Context) error

// Message describes a task and its metadata.
type Message struct {
	Task TaskFunc `json:"-"`

	// Timeout is the duration the task can be processed by Handler.
	// zero if not specified
	Timeout time.Duration `json:"timeout"`

	// Payload is the payload data of the task.
	Payload []byte `json:"body"`

	// RetryCount retry count if failure
	RetryCount int64 `json:"retry_count"`

	// RetryCount retry count if failure
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
