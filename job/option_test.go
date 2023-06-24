package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMessageEncodeDecode test message encode and decode
func TestOptions(t *testing.T) {
	o := NewOptions(
		AllowOption{
			RetryCount: Int64(100),
			RetryDelay: Time(30 * time.Millisecond),
			Timeout:    Time(3 * time.Millisecond),
			Jitter:     Bool(true),
		},
	)

	assert.Equal(t, int64(100), o.retryCount)
	assert.Equal(t, 30*time.Millisecond, o.retryDelay)
	assert.Equal(t, 3*time.Millisecond, o.timeout)
	assert.Equal(t, 100*time.Millisecond, o.retryMin)
	assert.Equal(t, 10*time.Second, o.retryMax)
	assert.Equal(t, 2.0, o.retryFactor)
	assert.True(t, o.jitter)
}
