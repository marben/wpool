package wpool

import (
	// "log"
	"testing"
	"time"
)

func sleepFunc() {
	time.Sleep(100 * time.Millisecond)
}

func TestRun(t *testing.T) {
	wp := NewWorkerPool(2)
	if wp.routinesLimit != 2 {
		t.Fatalf("Failed to create worker pool with correct number of threads")
	}
	wp.AddJob(sleepFunc)
	wp.Wait()
}

func TestRun2(t *testing.T) {
	ch := make(chan int, 10)
	wp := NewWorkerPool(2)

	for i := 0; i < 10; i++ {
		i := i
		wp.AddJob(func() { ch <- i })
		// go func() {
		// 	ch <- i
		// }()
	}
	wp.Wait()
	close(ch)
	var sum int
	for x := range ch {
		sum += x
	}
	if sum != 45 {
		t.Fatalf("The sum is:", sum)
	}
}
