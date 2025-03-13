package queue

import (
	"testing"
	"time"
)

func TestWithRetryInterval(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     time.Duration
	}{
		{
			name:     "Set 2 seconds retry interval",
			duration: 2 * time.Second,
			want:     2 * time.Second,
		},
		{
			name:     "Set 500ms retry interval",
			duration: 500 * time.Millisecond,
			want:     500 * time.Millisecond,
		},
		{
			name:     "Set zero retry interval",
			duration: 0,
			want:     0,
		},
		{
			name:     "Set negative retry interval",
			duration: -1 * time.Second,
			want:     -1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(WithRetryInterval(tt.duration))
			if opts.retryInterval != tt.want {
				t.Errorf("WithRetryInterval() = %v, want %v", opts.retryInterval, tt.want)
			}
		})
	}
}
