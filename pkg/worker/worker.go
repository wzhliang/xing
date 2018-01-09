package worker

// Request ...
type Request struct {
	In     interface{}
	Result (chan interface{})
	Err    (chan error)
}

// Worker ...
type Worker interface {
	SetDispatcher(d Dispatcher)
	Loop()
}

// Dispatcher ...
type Dispatcher interface {
	// called by client
	Send(in interface{}) (interface{}, error)
	// called by worker to get more work
	Dispatch() Request
	// associate worker with dispatcher and start working
	AddWorker(worker Worker)
	Close()
}

type dispatcher struct {
	jobs chan Request
}

// NewDispatcher ...
func NewDispatcher(size int) Dispatcher {
	return &dispatcher{jobs: make(chan Request, size)}
}

func (d *dispatcher) Send(in interface{}) (interface{}, error) {
	results := make(chan interface{})
	errs := make(chan error)
	d.jobs <- Request{in, results, errs}
	// wait for response
	r := <-results
	e := <-errs
	return r, e
}

func (d *dispatcher) AddWorker(worker Worker) {
	worker.SetDispatcher(d)
	go worker.Loop()
}

func (d *dispatcher) Dispatch() Request {
	return <-d.jobs
}

func (d *dispatcher) Close() {
	close(d.jobs)
}
