package queue

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRoutineGroupRun(t *testing.T) {
	t.Run("execute single function", func(t *testing.T) {
		g := newRoutineGroup()
		var counter int32

		g.Run(func() {
			atomic.AddInt32(&counter, 1)
		})

		g.Wait()

		if atomic.LoadInt32(&counter) != 1 {
			t.Errorf("expected counter to be 1, got %d", counter)
		}
	})

	t.Run("execute multiple functions", func(t *testing.T) {
		g := newRoutineGroup()
		var counter int32
		numRoutines := 10

		for i := 0; i < numRoutines; i++ {
			g.Run(func() {
				atomic.AddInt32(&counter, 1)
				time.Sleep(10 * time.Millisecond)
			})
		}

		g.Wait()

		if atomic.LoadInt32(&counter) != int32(numRoutines) {
			t.Errorf("expected counter to be %d, got %d", numRoutines, counter)
		}
	})
}
