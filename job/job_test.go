package job

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskBinaryEncode(t *testing.T) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	out := Decode(Encode(m))

	assert.Equal(t, int64(100), out.RetryCount)
	assert.Equal(t, 30*time.Millisecond, out.RetryDelay)
}

type mockMessage struct {
	message string
}

func (m mockMessage) Bytes() []byte {
	return []byte(m.message)
}

func TestMessageBinaryEncode(t *testing.T) {
	m := NewMessage(&mockMessage{
		message: "foo",
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	m.Encode()
	out := Decode(m.Bytes())

	assert.Equal(t, int64(100), out.RetryCount)
	assert.Equal(t, 30*time.Millisecond, out.RetryDelay)
	assert.Equal(t, "foo", string(out.Payload))
}
