package wpool

import (
	// "fmt"
	"runtime"
	"sync"
)

// TODO: would be nice to add a blocking function AddJobBlocking which would
// wait until some routine in the pool can start working on the added job

type WorkerPool struct {
	routinesLimit int
	jobChan       chan func()
	waiterChan    chan chan struct{}

	managerRunning bool
	managerLock    sync.Mutex
	managerEndChan chan bool
}

// this is non-blocking
func (wp *WorkerPool) AddJob(f func()) {
	wp.managerLock.Lock()
	if !wp.managerRunning {
		wp.startManager()
		wp.managerRunning = true
	}

	wp.jobChan <- f
	wp.managerLock.Unlock()
}

func (wp *WorkerPool) startManager() {

	go wp.manage()

	// we start the another go routine, that will wait for the manager to finish and make it stop afterwards
	go func() {
		for range wp.managerEndChan {
			// fmt.Println("Will make the manager stop now")
			// first let's obtain the lock so that no one adds new jobs, while we are shutting down
			wp.managerLock.Lock()
			// we got the lock. that means that no job can be added until we unlock it
			// however if someone added a job before we obtained the lock, there can now be a job running in the manager
			// we send a signal on the end channel so that manager tells us, whether it's ok to quit now or not..
			wp.managerEndChan <- true
			if ok := <-wp.managerEndChan; ok {
				// fmt.Println("End of manager was confirmed")
				wp.managerRunning = false
				wp.managerLock.Unlock()
				return
			} else {
				// fmt.Println("End of manager was cancelled")
				wp.managerLock.Unlock()
			}
		}
	}()
}

func NewPool(routinesLimit int) *WorkerPool {
	if routinesLimit < 1 {
		panic("Creating worker pool with less than 1 routine")
	}
	pool := &WorkerPool{routinesLimit: routinesLimit}
	pool.jobChan = make(chan func())
	pool.waiterChan = make(chan chan struct{})
	pool.managerEndChan = make(chan bool)
	return pool
}

// creates worker pool with number of goroutines equal to number of cores
func NewPoolDefault() *WorkerPool {
	numCPUs := runtime.NumCPU()
	pool := NewPool(numCPUs)
	return pool
}

func (wp *WorkerPool) Wait() {
	ch := make(chan struct{})
	wp.managerLock.Lock()

	if !wp.managerRunning {
		// if manager is not running, no job is active, therefore we return just like that
		wp.managerLock.Unlock()
		return
	}
	wp.waiterChan <- ch //TODO: perhaps we could just increment number of waiters and then send appropriate number of signals on another channel shared among all waiters
	wp.managerLock.Unlock()
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

func (wp *WorkerPool) manage() {
	var currentlyRunning int
	var jobs jobQueue
	var waiters []chan struct{}
	doneChan := make(chan struct{})
	for {
		select {
		case f := <-wp.jobChan:
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
					// we inform the helper go routne, that we are ready to quit
					wp.managerEndChan <- true
					// once the helper go routine obtains a lock it sends a signal on wp.managerEndChan
				}
			}
		case w := <-wp.waiterChan:
			if currentlyRunning == 0 {
				w <- struct{}{}
			} else {
				waiters = append(waiters, w)
			}
		case <-wp.managerEndChan:
			// time to quit this go routine
			// this signal is only received, when the helper function obtained a lock
			// therefore no new jobs and no new waiters will been added until we signal the helper function that it can unlock
			if currentlyRunning != 0 {
				// it seems, that before the helper function managed to get the lock, some new jobs appeared
				// therefore we inform the helper, that it's not the right time to quit
				wp.managerEndChan <- false
			} else {
				// fmt.Println("Ending wp manager...")
				// we inform all those waiting, that all jobs have ended for now...
				for _, waiterChan := range waiters {
					waiterChan <- struct{}{}
				}
				// inform the helper go routine that it can unlock and quit
				wp.managerEndChan <- true
				// end, so that we don't leak this go routine
				return
			}
		}
	}
}
