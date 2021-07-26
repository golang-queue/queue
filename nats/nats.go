package nats

import (
	"sync"

	"github.com/appleboy/queue"

	"github.com/nats-io/nats.go"
)

var _ queue.Worker = (*Worker)(nil)

// Option for queue system
type Option func(*Worker)

// Job with NSQ message
type Job struct {
	Body []byte
}

// Bytes get bytes format
func (j *Job) Bytes() []byte {
	return j.Body
}

// Worker for NSQ
type Worker struct {
	addr     string
	subj     string
	queue    string
	client   *nats.Conn
	stop     chan struct{}
	stopOnce sync.Once
	runFunc  func(queue.QueuedMessage, <-chan struct{}) error
}

// WithAddr setup the addr of NATS
func WithAddr(addr string) Option {
	return func(w *Worker) {
		w.addr = "nats://" + addr
	}
}

// WithSubj setup the subject of NATS
func WithSubj(subj string) Option {
	return func(w *Worker) {
		w.subj = subj
	}
}

// WithQueue setup the queue of NATS
func WithQueue(queue string) Option {
	return func(w *Worker) {
		w.queue = queue
	}
}

// WithRunFunc setup the run func of queue
func WithRunFunc(fn func(queue.QueuedMessage, <-chan struct{}) error) Option {
	return func(w *Worker) {
		w.runFunc = fn
	}
}

// NewWorker for struc
func NewWorker(opts ...Option) *Worker {
	var err error
	w := &Worker{
		addr:  "nats://127.0.0.1:4222",
		subj:  "foobar",
		queue: "foobar",
		stop:  make(chan struct{}),
		runFunc: func(queue.QueuedMessage, <-chan struct{}) error {
			return nil
		},
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(w)
	}

	w.client, err = nats.Connect(w.addr)
	if err != nil {
		panic(err)
	}

	return w
}

// BeforeRun run script before start worker
func (s *Worker) BeforeRun() error {
	return nil
}

// AfterRun run script after start worker
func (s *Worker) AfterRun() error {
	return nil
}

// Run start the worker
func (s *Worker) Run() error {
	wg := &sync.WaitGroup{}

	_, err := s.client.QueueSubscribe(s.subj, s.queue, func(m *nats.Msg) {
		wg.Add(1)
		defer wg.Done()
		job := &Job{
			Body: m.Data,
		}

		// run custom process function
		_ = s.runFunc(job, s.stop)
	})
	if err != nil {
		return err
	}

	// wait close signal
	<-s.stop

	// wait job completed
	wg.Wait()

	return nil
}

// Shutdown worker
func (s *Worker) Shutdown() error {
	s.stopOnce.Do(func() {
		close(s.stop)
		s.client.Close()
	})
	return nil
}

// Capacity for channel
func (s *Worker) Capacity() int {
	return 0
}

// Usage for count of channel usage
func (s *Worker) Usage() int {
	return 0
}

// Queue send notification to queue
func (s *Worker) Queue(job queue.QueuedMessage) error {
	err := s.client.Publish(s.subj, job.Bytes())
	if err != nil {
		return err
	}

	return nil
}
