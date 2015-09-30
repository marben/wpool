package wpool

import (
	// "log"
	"runtime"
)

// TODO: probably leaking go routine etc..
// need some polish
// TODO: would be also nice to add blocking function AddJobBlocking which would
// wait until some space in the pool can start working on the function
type WorkerPool struct {
	routinesLimit int
	jobChan       chan func()
	waiterChannel chan chan struct{}
}

// this is non-blocking
func (wp *WorkerPool) AddJob(f func()) {
	wp.jobChan <- f
}

func NewWorkerPool(routinesLimit int) *WorkerPool {
	if routinesLimit < 1 {
		panic("Creating worker pool with less than 1 routine")
	}
	pool := &WorkerPool{routinesLimit: routinesLimit}
	pool.jobChan = make(chan func())

	pool.waiterChannel = make(chan chan struct{})

	go pool.manager(pool.jobChan, pool.waiterChannel)

	return pool
}

func (wp WorkerPool) Wait() {
	ch := make(chan struct{})
	wp.waiterChannel <- ch //TODO: this seems to be wrong.
	<-ch
}

// a FIFO queue of functions
type jobQueue struct {
	queue []func()
}

func (jq *jobQueue) Push(f func()) {
	jq.queue = append(jq.queue, f)
}

func (jq *jobQueue) Pop() (f func(), found bool) {
	if len(jq.queue) == 0 {
		return nil, false
	} else {
		f = jq.queue[0]
		jq.queue = jq.queue[1:]
		return f, true
	}
}

func (wp *WorkerPool) manager(funcChannel chan func(), waiterChannel chan chan struct{}) {
	var currentlyRunning int
	var jobs jobQueue
	var waiters []chan struct{}
	doneChan := make(chan struct{})
	for {
		select {
		case f := <-funcChannel:
			if currentlyRunning < wp.routinesLimit {
				currentlyRunning++
				go func() {
					f()
					doneChan <- struct{}{}
				}()
			} else {
				jobs.Push(f)
			}
		case <-doneChan:
			currentlyRunning--
			if f, found := jobs.Pop(); found {
				currentlyRunning++
				go func() {
					f()
					doneChan <- struct{}{}
				}()
			} else {
				if currentlyRunning == 0 {
					// we inform all those waiting, that all jobs have ended for now...
					for _, funcChannel := range waiters {
						funcChannel <- struct{}{}
					}
					// empty (and mem-release) the waiting queue
					waiters = nil
				}
			}
		case w := <-waiterChannel:
			waiters = append(waiters, w)
		}
	}
}

// creates worker pool with number of goroutines equal to number of cores
func NewWorkerPoolDefault() *WorkerPool {
	numCPUs := runtime.NumCPU()
	pool := NewWorkerPool(numCPUs)
	return pool
}
