package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQueue(t *testing.T) {
	q, err := NewQueue()
	assert.Error(t, err)
	assert.Nil(t, q)

	w := &emptyWorker{}
	q, err = NewQueue(
		WithWorker(w),
	)
	assert.NoError(t, err)
	assert.NotNil(t, q)
}
