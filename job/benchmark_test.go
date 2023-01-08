package job

import (
	"context"
	"testing"
	"time"
)

func BenchmarkNewTask(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTask(func(context.Context) error {
			return nil
		},
			AllowOption{
				RetryCount: Int64(100),
				RetryDelay: Time(30 * time.Millisecond),
				Timeout:    Time(3 * time.Millisecond),
			},
		)
	}
}

func BenchmarkNewOption(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewOptions(
			AllowOption{
				RetryCount: Int64(100),
				RetryDelay: Time(30 * time.Millisecond),
				Timeout:    Time(3 * time.Millisecond),
			},
		)
	}
}
