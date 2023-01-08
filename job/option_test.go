package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	o := NewOptions(
		WithRetryCount(100),
		WithRetryDelay(30*time.Millisecond),
		WithTimeout(3*time.Millisecond),
	)

	assert.Equal(t, int64(100), o.retryCount)
	assert.Equal(t, 30*time.Millisecond, o.retryDelay)
	assert.Equal(t, 3*time.Millisecond, o.timeout)
}

func TestOption2s(t *testing.T) {
	o := NewOption2s(
		WithRetryCount2(100),
		WithRetryDelay2(30*time.Millisecond),
		WithTimeout2(3*time.Millisecond),
	)

	assert.Equal(t, int64(100), o.retryCount)
	assert.Equal(t, 30*time.Millisecond, o.retryDelay)
	assert.Equal(t, 3*time.Millisecond, o.timeout)
}

func TestOption3s(t *testing.T) {
	o := NewOption3s(
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
		},
	)

	assert.Equal(t, int64(100), o.retryCount)
	assert.Equal(t, 30*time.Millisecond, o.retryDelay)
	assert.Equal(t, 3*time.Millisecond, o.timeout)
}
