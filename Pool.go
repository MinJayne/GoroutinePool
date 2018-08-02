package GoroutinePool

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type sig struct{}

type f func() error

// When Pool gets a task, run task by available worker,
// if there is no available worker, check if running workers number is greater than pool size,
// if so, block and wait available worker, if not, create a new worker to run the task.
type Pool struct {
	// total worker number of the pool.
	size int32

	// number of running goroutines.
	runningNum int32

	// expired time (second) of every worker.
	expiryTime time.Duration

	// signal is used to inform pool there are available workers.
	signal chan sig

	// availableWorkers records the available workers.
	availableWorkers []*Worker

	// inform the pool to be terminated.
	isTerminated chan sig

	// lock for synchronous operation
	lock sync.Mutex

	once sync.Once
}


func CreatePool(size int) (*Pool, error) {
	return CreateExpiryPool(size, DefaultExpiryTime)
}


func CreateExpiryPool(size, expiry int) (*Pool, error) {
	if size <= 0 {
		return nil, ErrInvalidPoolSize
	}
	if expiry <= 0 {
		return nil, ErrInvalidPoolExpiryTime
	}
	p := &Pool{
		size:       	int32(size),
		signal:     	make(chan sig, math.MaxInt32),
		isTerminated:   make(chan sig, 1),
		expiryTime: 	time.Duration(expiry) * time.Second,
	}
	
	// clear expiry worker at regular intervals
	p.monitorAndClear()
	return p, nil
}

// Submit a task 
func (p *Pool) Submit(task f) error {
	if len(p.isTerminated) > 0 {
		return ErrPoolTerminated
	}
	w := p.getAvailableWorker()
	w.task <- task
	return nil
}

// Returns number of the running goroutines
func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.runningNum))
}

// Returns size of this pool
func (p *Pool) Size() int {
	return int(atomic.LoadInt32(&p.size))
}

// Get a available worker 
func (p *Pool) getAvailableWorker() *Worker {
	var w *Worker

	// flag to record if number of running workers is greater than pool size
	waiting := false

	// verify if there are any available workers
	p.lock.Lock()
	availableWorkers := p.availableWorkers
	n := len(availableWorkers) - 1
	
	if n < 0 {
		// there is no available worker
		if p.Running() >= p.Size() {
			waiting = true
		} else {
			atomic.AddInt32(&p.runningNum, 1)
		}
	} else {
		// get a available worker from the end of the queue
		<-p.signal
		w = availableWorkers[n]
		availableWorkers[n] = nil
		p.availableWorkers = availableWorkers[:n]
	}
	// verification finished
	p.lock.Unlock()

	if waiting {
		// Pool is full, new task have to wait
		// block and wait for available worker
		<-p.signal
		p.lock.Lock()
		availableWorkers = p.availableWorkers
		l := len(availableWorkers) - 1
		w = availableWorkers[l]
		availableWorkers[l] = nil
		p.availableWorkers = availableWorkers[:l]
		p.lock.Unlock()
	} else if w == nil {
		// there is no available worker,
		// however, pool is not full
		// just create a new worker
		w = &Worker{
			pool: p,
			task: make(chan f),
		}
		w.run()
	}
	return w
}

// Recycle a worker back into pool.
func (p *Pool) RecycleWorker(worker *Worker) {
	worker.freeTime = time.Now()
	p.lock.Lock()
	p.availableWorkers = append(p.availableWorkers, worker)
	p.lock.Unlock()
	p.signal <- sig{}
}