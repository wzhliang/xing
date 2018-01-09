package worker

import (
	"fmt"
	"sync"
	"testing"
)

type worker struct {
	d Dispatcher
}

func (w *worker) Loop() {
	for {
		req := w.d.Dispatch()
		i := req.In.(int)
		fmt.Printf("i: %d\n", i)
		req.Result <- i * 2
		req.Err <- nil
	}
}

func (w *worker) SetDispatcher(d Dispatcher) {
	w.d = d
}

const (
	COUNT = 30000
)

func Test_worker_1(t *testing.T) {
	d := NewDispatcher(100)
	var wkrs [300]worker
	for i := 0; i < 300; i++ {
		d.AddWorker(&wkrs[i])
	}
	var wg sync.WaitGroup
	var mutex = &sync.Mutex{}

	total := 0
	for i := 0; i < COUNT; i++ {
		wg.Add(1)
		go func(i int) {
			res, _ := d.Send(i)
			mutex.Lock()
			total = total + res.(int)
			mutex.Unlock()
			wg.Done()
		}(i)
	}

	expected := 0
	for i := 0; i < COUNT; i++ {
		expected = expected + i*2
	}

	wg.Wait()
	if total != expected {
		t.Errorf("Sum was wrong. Expected: %d, actual: %d\n", expected, total)
	}
}
