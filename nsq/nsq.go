package nsq

import (
	"runtime"
	"sync"
	"time"

	"github.com/appleboy/queue"

	"github.com/nsqio/go-nsq"
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
	q           *nsq.Consumer
	p           *nsq.Producer
	startOnce   sync.Once
	stopOnce    sync.Once
	stop        chan struct{}
	maxInFlight int
	addr        string
	topic       string
	channel     string
	runFunc     func(queue.QueuedMessage, <-chan struct{}) error
}

// WithAddr setup the addr of NSQ
func WithAddr(addr string) Option {
	return func(w *Worker) {
		w.addr = addr
	}
}

// WithTopic setup the topic of NSQ
func WithTopic(topic string) Option {
	return func(w *Worker) {
		w.topic = topic
	}
}

// WithChannel setup the channel of NSQ
func WithChannel(channel string) Option {
	return func(w *Worker) {
		w.channel = channel
	}
}

// WithRunFunc setup the run func of queue
func WithRunFunc(fn func(queue.QueuedMessage, <-chan struct{}) error) Option {
	return func(w *Worker) {
		w.runFunc = fn
	}
}

// WithMaxInFlight Maximum number of messages to allow in flight (concurrency knob)
func WithMaxInFlight(num int) Option {
	return func(w *Worker) {
		w.maxInFlight = num
	}
}

// NewWorker for struc
func NewWorker(opts ...Option) *Worker {
	var err error
	w := &Worker{
		addr:        "127.0.0.1:4150",
		topic:       "gorush",
		channel:     "ch",
		maxInFlight: runtime.NumCPU(),
		stop:        make(chan struct{}),
		runFunc: func(queue.QueuedMessage, <-chan struct{}) error {
			return nil
		},
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(w)
	}

	cfg := nsq.NewConfig()
	cfg.MaxInFlight = w.maxInFlight
	w.q, err = nsq.NewConsumer(w.topic, w.channel, cfg)
	if err != nil {
		panic(err)
	}

	w.p, err = nsq.NewProducer(w.addr, cfg)
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
	s.startOnce.Do(func() {
		time.Sleep(100 * time.Millisecond)
		err := s.q.ConnectToNSQD(s.addr)
		if err != nil {
			panic("Could not connect nsq server: " + err.Error())
		}
	})

	return nil
}

// Run start the worker
func (s *Worker) Run() error {
	wg := &sync.WaitGroup{}
	s.q.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		wg.Add(1)
		defer wg.Done()
		if len(msg.Body) == 0 {
			// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
			// In this case, a message with an empty body is simply ignored/discarded.
			return nil
		}

		job := &Job{
			Body: msg.Body,
		}

		// run custom process function
		return s.runFunc(job, s.stop)
	}))

	// wait close signal
	<-s.stop

	// wait job completed
	wg.Wait()

	return nil
}

// Shutdown worker
func (s *Worker) Shutdown() error {
	s.stopOnce.Do(func() {
		s.q.Stop()
		s.p.Stop()
		close(s.stop)
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
	err := s.p.Publish(s.topic, job.Bytes())
	if err != nil {
		return err
	}

	return nil
}
