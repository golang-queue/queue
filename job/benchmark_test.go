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
			WithRetryCount(100),
			WithRetryDelay(30*time.Millisecond),
			WithTimeout(30*time.Millisecond),
		)
	}
}

func BenchmarkNewOption(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewOptions(
			WithRetryCount(100),
			WithRetryDelay(30*time.Millisecond),
			WithTimeout(30*time.Millisecond),
		)
	}
}
