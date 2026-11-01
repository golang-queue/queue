package queue

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPoolWithQueueTask(t *testing.T) {
	totalN := int64(5)
	taskN := 100
	rets := make(chan struct{}, taskN)

	p := NewPool(totalN)
	for range taskN {
		require.NoError(t, p.QueueTask(func(context.Context) error {
			rets <- struct{}{}
			return nil
		}))
	}

	for range taskN {
		<-rets
	}

	// shutdown all, and now running worker is 0
	p.Release()
	assert.Equal(t, int64(0), p.BusyWorkers())
}

func TestPoolNumber(t *testing.T) {
	p := NewPool(0)
	p.Start()
	// shutdown all, and now running worker is 0
	p.Release()
	assert.Equal(t, int64(0), p.BusyWorkers())
}
