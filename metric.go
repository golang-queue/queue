package queue

import "sync/atomic"

// Metric interface
type Metric interface {
	IncBusyWorker()
	DecBusyWorker()
	BusyWorkers() uint64
}

type metric struct {
	busyWorkers uint64
}

func newMetric() Metric {
	return &metric{}
}

func (m *metric) IncBusyWorker() {
	atomic.AddUint64(&m.busyWorkers, 1)
}

func (m *metric) DecBusyWorker() {
	atomic.AddUint64(&m.busyWorkers, ^uint64(0))
}

func (m *metric) BusyWorkers() uint64 {
	return atomic.LoadUint64(&m.busyWorkers)
}
