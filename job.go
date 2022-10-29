package queue

import (
	"time"

	"github.com/goccy/go-json"
)

// Job describes a task and its metadata.
type Job struct {
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
func (j *Job) Bytes() []byte {
	if j.Task != nil {
		return nil
	}
	return j.Payload
}

// Encode for encoding the structure
func (j *Job) encode() []byte {
	b, _ := json.Marshal(j)

	return b
}
