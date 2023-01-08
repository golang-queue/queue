package job

import (
	"context"
	"testing"
	"time"
)

func BenchmarkNewTask01(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTask(func(context.Context) error {
			return nil
		},
			WithRetryCount(100),
			WithRetryDelay(30*time.Millisecond),
			WithTimeout(3*time.Millisecond),
		)
	}
}

func BenchmarkNewTask02(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTask2(func(context.Context) error {
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

func BenchmarkNewOption01(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewOptions(
			WithRetryCount(100),
			WithRetryDelay(30*time.Millisecond),
			WithTimeout(3*time.Millisecond),
		)
	}
}

func BenchmarkNewOption02(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewOption2s(
			WithRetryCount2(100),
			WithRetryDelay2(30*time.Millisecond),
			WithTimeout2(3*time.Millisecond),
		)
	}
}

func BenchmarkNewOption03(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewOption3s(
			AllowOption{
				RetryCount: Int64(100),
				RetryDelay: Time(30 * time.Millisecond),
				Timeout:    Time(3 * time.Millisecond),
			},
		)
	}
}
