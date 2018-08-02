package GoroutinePool

import (
	"time"
)

// Worker starts a goroutine and runs the task.
type Worker struct {
	// Where the worker belongs to 
	pool *Pool

	task chan f

	// Update freeTime when putting a worker back into queue.
	freeTime time.Time
}

// Starts a goroutine to run the task.
func (w *Worker) run() {
	go func() {
		// monitor the tasks, run the task once task is ready
		for f := range w.task {
			if f == nil {
				atomic.AddInt32(&w.pool.runningNum, -1)
				return
			}
			f()
			
			w.pool.RecycleWorker(w)
		}
	}()
}