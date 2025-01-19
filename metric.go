package queue

import "sync/atomic"

// Metric interface
type Metric interface {
	IncBusyWorker()
	DecBusyWorker()
	BusyWorkers() int64
	SuccessTasks() uint64
	FailureTasks() uint64
	SubmittedTasks() uint64
	CompletedTasks() uint64
	IncSuccessTask()
	IncFailureTask()
	IncSubmittedTask()
	IncCompletedTask()
}

var _ Metric = (*metric)(nil)

type metric struct {
	busyWorkers    int64
	successTasks   uint64
	failureTasks   uint64
	submittedTasks uint64
	completedTasks uint64
}

// NewMetric for default metric structure
func NewMetric() Metric {
	return &metric{}
}

func (m *metric) IncBusyWorker() {
	atomic.AddInt64(&m.busyWorkers, 1)
}

func (m *metric) DecBusyWorker() {
	atomic.AddInt64(&m.busyWorkers, ^int64(0))
}

func (m *metric) BusyWorkers() int64 {
	return atomic.LoadInt64(&m.busyWorkers)
}

func (m *metric) IncSuccessTask() {
	atomic.AddUint64(&m.successTasks, 1)
}

func (m *metric) IncFailureTask() {
	atomic.AddUint64(&m.failureTasks, 1)
}

func (m *metric) IncSubmittedTask() {
	atomic.AddUint64(&m.submittedTasks, 1)
}

func (m *metric) IncCompletedTask() {
	atomic.AddUint64(&m.completedTasks, 1)
}

func (m *metric) SuccessTasks() uint64 {
	return atomic.LoadUint64(&m.successTasks)
}

func (m *metric) FailureTasks() uint64 {
	return atomic.LoadUint64(&m.failureTasks)
}

func (m *metric) SubmittedTasks() uint64 {
	return atomic.LoadUint64(&m.submittedTasks)
}

func (m *metric) CompletedTasks() uint64 {
	return atomic.LoadUint64(&m.completedTasks)
}
