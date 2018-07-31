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

	// number of current running goroutines.
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

