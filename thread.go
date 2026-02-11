package queue

import "sync"

// routineGroup is a thin wrapper around sync.WaitGroup for managing goroutine lifecycles.
// It simplifies the common pattern of spawning goroutines and waiting for their completion.
//
// Design rationale:
//   - Encapsulates WaitGroup.Add(1) + go func() + defer Done() into a single Run() call
//   - Reduces boilerplate and prevents common mistakes (forgetting to call Done, wrong Add count)
//   - Provides a cleaner API for goroutine management in the queue implementation
type routineGroup struct {
	waitGroup sync.WaitGroup
}

// newRoutineGroup creates a new routineGroup for managing goroutines.
func newRoutineGroup() *routineGroup {
	return new(routineGroup)
}

// Run launches a goroutine to execute the provided function.
// The function is automatically registered with the WaitGroup and will be
// tracked until it completes. This method is safe to call concurrently.
//
// Example:
//
//	rg := newRoutineGroup()
//	rg.Run(func() {
//	    // Do work in background
//	})
//	rg.Wait() // Wait for all goroutines to complete
func (g *routineGroup) Run(fn func()) {
	g.waitGroup.Add(1)

	go func() {
		defer g.waitGroup.Done()
		fn()
	}()
}

// Wait blocks until all goroutines launched via Run() have completed.
// This method is safe to call multiple times and from multiple goroutines.
func (g *routineGroup) Wait() {
	g.waitGroup.Wait()
}
