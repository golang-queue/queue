package queue

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRoutineGroupRun(t *testing.T) {
	t.Run("execute single function", func(t *testing.T) {
		g := newRoutineGroup()
		var counter atomic.Int32

		g.Run(func() {
			counter.Add(1)
		})

		g.Wait()

		if counter.Load() != 1 {
			t.Errorf("expected counter to be 1, got %d", counter.Load())
		}
	})

	t.Run("execute multiple functions", func(t *testing.T) {
		g := newRoutineGroup()
		var counter atomic.Int32
		numRoutines := 10

		for range numRoutines {
			g.Run(func() {
				counter.Add(1)
				time.Sleep(10 * time.Millisecond)
			})
		}

		g.Wait()

		if counter.Load() != int32(numRoutines) {
			t.Errorf("expected counter to be %d, got %d", numRoutines, counter.Load())
		}
	})
}
