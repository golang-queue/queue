package job

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"
	"unsafe"

	"github.com/goccy/go-json"
	"github.com/golang-queue/queue/core"
	"github.com/stretchr/testify/assert"
)

func TestJSONEncodeAndDecode(t *testing.T) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	out := &Message{}
	if err := json.Unmarshal(m.Encode(), out); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, out.RetryCount, m.RetryCount)
	assert.Equal(t, out.RetryDelay, m.RetryDelay)
}

func BenchmarkJSONEncode(b *testing.B) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	for i := 0; i < b.N; i++ {
		m.Encode()
	}
}

func BenchmarkJSONDecode(b *testing.B) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	data := m.Encode()
	var out Message
	for i := 0; i < b.N; i++ {
		json.Unmarshal(data, &out)
	}
}

func BenchmarkBinaryEncode(b *testing.B) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	data := make([]byte, 0, movementSize)
	buffer := bytes.NewBuffer(data)

	for i := 0; i < b.N; i++ {
		buffer.Reset()
		binary.Write(buffer, binary.BigEndian, m.RetryCount) //nolint
		binary.Write(buffer, binary.BigEndian, m.RetryDelay) //nolint
		binary.Write(buffer, binary.BigEndian, m.Timeout)    //nolint
	}
}

func BenchmarkBinaryWholeStructEncode(b *testing.B) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	data := make([]byte, 0, movementSize)
	buffer := bytes.NewBuffer(data)

	for i := 0; i < b.N; i++ {
		buffer.Reset()
		binary.Write(buffer, binary.BigEndian, m) //nolint
	}
}

func BenchmarkUnsafetEncode(b *testing.B) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	for i := 0; i < b.N; i++ {
		_ = (*[movementSize]byte)(unsafe.Pointer(m))[:]
	}
}

func BenchmarkUnsafetDecode(b *testing.B) {
	m := NewTask(func(context.Context) error {
		return nil
	},
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	data := (*[movementSize]byte)(unsafe.Pointer(m))[:]

	for i := 0; i < b.N; i++ {
		_ = *(*Message)(unsafe.Pointer(&data[0]))
	}
}

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

	// data := (*[movementSize]byte)(unsafe.Pointer(m))[:]
	// out := (*Message)(unsafe.Pointer(&data[0]))
	// data := m.Encode()
	// out := Decode(data)
	// m.UnsafeEncode()
	// out := m.UnsafeDecode()

	// m.UnsafeEncode()
	// m.RetryCount = 0
	// m.RetryDelay = 0
	// out := Decode(m.Bytes())

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

func test(t *testing.T, task core.QueuedMessage) {
	out := Decode(task.Bytes())

	assert.Equal(t, int64(100), out.RetryCount)
	assert.Equal(t, 30*time.Millisecond, out.RetryDelay)
	assert.Equal(t, "foo", string(out.Payload))
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

	// m.data = Encode(m)
	m.UnsafeEncode()

	test(t, m)

	// out := Decode(m.Bytes())
	// assert.Equal(t, int64(100), out.RetryCount)
	// assert.Equal(t, 30*time.Millisecond, out.RetryDelay)
	// assert.Equal(t, "foo", string(out.Payload))
}
