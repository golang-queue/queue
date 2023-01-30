package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockMessage struct {
	message string
}

func (m mockMessage) Bytes() []byte {
	return []byte(m.message)
}

func TestMessageEncodeDecode(t *testing.T) {
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
	assert.Equal(t, 3*time.Millisecond, out.Timeout)
	assert.Equal(t, "foo", string(out.Payload))
}
