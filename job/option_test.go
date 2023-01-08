package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	o := NewOptions(
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
