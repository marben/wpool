# wpool - a handy implementation of worker pool in go
A simple versatile type that lets you feed it with functions and let it execute them in parallel with limited number of goroutines.
Useful in cases when resources are scarce - for instance when parallell processing huge number of files while only having so many file descriptors.

## Basic example
we will execute a slow function 4 times and let it run in 2 goroutines

```go
package main

import (
	"fmt"
	"github.com/marben/wpool"
	"time"
)

func slowFunc() {
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Finished slow function")
}

func main() {
	// pool is created with a factory method
	// max of 2 routines will run at the same time in this case
	pool := wpool.NewPool(2)

	// four jobs we add but only two will be processed at the same time
	// adding a job doesn't block
	pool.AddJob(slowFunc)
	pool.AddJob(slowFunc)
	pool.AddJob(slowFunc)
	pool.AddJob(slowFunc)

	// now we have to wait for the jobs to finish
	pool.Wait()
	fmt.Println("All jobs are finished now")
}
```

## running on all cpu cores
a convenient factory function creates a pool with number of parallel goroutines equal to number of CPU cores:

```go
pool := wpool.NewPoolDefault()
pool.AddJob(slowFunc)
...
``` 

## concurrency
wpool is quite flexible. Jobs can be added from multiple goroutines and Wait() can be placed in multiple goroutines.
It is possible to re-use a pool once it has been waited for. Just add another job.

wpool doesn't leak goroutines. it stops it's helper goroutines once it runs out of jobs.
