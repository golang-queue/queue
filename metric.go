package queue

import "sync/atomic"

// Metric defines the interface for tracking queue performance and worker statistics.
// All methods must be safe for concurrent access from multiple goroutines.
// Implement this interface to integrate with custom monitoring systems (Prometheus, StatsD, etc.).
type Metric interface {
	// IncBusyWorker increments the count of workers currently processing tasks.
	// Called atomically when a worker starts processing a job.
	IncBusyWorker()

	// DecBusyWorker decrements the count of workers currently processing tasks.
	// Called atomically when a worker finishes processing a job (success or failure).
	DecBusyWorker()

	// BusyWorkers returns the current number of workers actively processing tasks.
	// This value can range from 0 to the configured workerCount.
	BusyWorkers() int64

	// SuccessTasks returns the total number of tasks that completed successfully.
	// A task is considered successful if it returns no error and doesn't panic.
	SuccessTasks() uint64

	// FailureTasks returns the total number of tasks that failed.
	// A task is considered failed if it returns an error, panics, or times out.
	FailureTasks() uint64

	// SubmittedTasks returns the total number of tasks submitted to the queue.
	// This includes tasks still pending, in progress, and completed.
	SubmittedTasks() uint64

	// CompletedTasks returns the total number of tasks that have finished processing.
	// This equals SuccessTasks() + FailureTasks().
	CompletedTasks() uint64

	// IncSuccessTask increments the successful task counter.
	// Called atomically after a task completes without error.
	IncSuccessTask()

	// IncFailureTask increments the failed task counter.
	// Called atomically after a task fails, panics, or times out.
	IncFailureTask()

	// IncSubmittedTask increments the submitted task counter.
	// Called atomically when a new task is queued.
	IncSubmittedTask()
}

var _ Metric = (*metric)(nil)

// metric is the default implementation of the Metric interface.
// It uses atomic operations to ensure thread-safe updates and reads.
// All counters start at zero and are never decremented (except busyWorkers).
type metric struct {
	busyWorkers    int64  // Current number of active workers (can go up and down)
	successTasks   uint64 // Cumulative count of successful tasks (monotonically increasing)
	failureTasks   uint64 // Cumulative count of failed tasks (monotonically increasing)
	submittedTasks uint64 // Cumulative count of submitted tasks (monotonically increasing)
}

// NewMetric creates a new metric collector with all counters initialized to zero.
// The returned metric is safe for concurrent use.
func NewMetric() Metric {
	return &metric{}
}

// IncBusyWorker atomically increments the busy worker count by 1.
// Thread-safe for concurrent calls.
func (m *metric) IncBusyWorker() {
	atomic.AddInt64(&m.busyWorkers, 1)
}

// DecBusyWorker atomically decrements the busy worker count by 1.
// Uses ^int64(0) which equals -1 in two's complement representation.
// Thread-safe for concurrent calls.
func (m *metric) DecBusyWorker() {
	atomic.AddInt64(&m.busyWorkers, ^int64(0))
}

// BusyWorkers atomically reads and returns the current number of busy workers.
// Thread-safe for concurrent calls.
func (m *metric) BusyWorkers() int64 {
	return atomic.LoadInt64(&m.busyWorkers)
}

// IncSuccessTask atomically increments the successful task counter by 1.
// Thread-safe for concurrent calls.
func (m *metric) IncSuccessTask() {
	atomic.AddUint64(&m.successTasks, 1)
}

// IncFailureTask atomically increments the failed task counter by 1.
// Thread-safe for concurrent calls.
func (m *metric) IncFailureTask() {
	atomic.AddUint64(&m.failureTasks, 1)
}

// IncSubmittedTask atomically increments the submitted task counter by 1.
// Thread-safe for concurrent calls.
func (m *metric) IncSubmittedTask() {
	atomic.AddUint64(&m.submittedTasks, 1)
}

// SuccessTasks atomically reads and returns the total number of successful tasks.
// Thread-safe for concurrent calls.
func (m *metric) SuccessTasks() uint64 {
	return atomic.LoadUint64(&m.successTasks)
}

// FailureTasks atomically reads and returns the total number of failed tasks.
// Thread-safe for concurrent calls.
func (m *metric) FailureTasks() uint64 {
	return atomic.LoadUint64(&m.failureTasks)
}

// SubmittedTasks atomically reads and returns the total number of submitted tasks.
// Thread-safe for concurrent calls.
func (m *metric) SubmittedTasks() uint64 {
	return atomic.LoadUint64(&m.submittedTasks)
}

// CompletedTasks calculates and returns the total number of completed tasks.
// This is the sum of successful and failed tasks.
// Note: The two atomic reads are not performed as a single atomic operation,
// so the result represents an approximate snapshot in time.
// Thread-safe for concurrent calls.
func (m *metric) CompletedTasks() uint64 {
	return atomic.LoadUint64(&m.successTasks) + atomic.LoadUint64(&m.failureTasks)
}
