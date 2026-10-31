package queue

import (
	"testing"

	"github.com/golang-queue/queue/job"

	"github.com/stretchr/testify/require"
)

func TestRingShrinkHysteresisAndFIFO(t *testing.T) {
	w := NewRing()
	var expected []*job.Message
	for range 16 {
		m := new(job.Message)
		expected = append(expected, m)
		require.NoError(t, w.Queue(m))
	}
	for i := range 8 {
		m, err := w.Request()
		require.NoError(t, err)
		require.Same(t, expected[i], m)
	}
	require.Len(t, w.taskQueue, 16, "half-full buffers must not shrink")
	for i := 8; i < 12; i++ {
		m, err := w.Request()
		require.NoError(t, err)
		require.Same(t, expected[i], m)
	}
	require.Len(t, w.taskQueue, 8, "quarter-full buffers should halve")
	// Refill, wrap, grow, and drain to verify FIFO survives both resize paths.
	for range 20 {
		m := new(job.Message)
		expected = append(expected, m)
		require.NoError(t, w.Queue(m))
	}
	for _, want := range expected[12:] {
		m, err := w.Request()
		require.NoError(t, err)
		require.Same(t, want, m)
	}
	require.Len(t, w.taskQueue, 2)
}
