package wpool

import (
	// "log"
	// "fmt"
	"testing"
	"time"
)

func sleepFunc() {
	time.Sleep(100 * time.Millisecond)
}

func TestRun(t *testing.T) {
	wp := NewPool(2)
	if wp.routinesLimit != 2 {
		t.Fatalf("Failed to create worker pool with correct number of threads")
	}
	wp.AddJob(sleepFunc)
	wp.Wait()
}

func TestRun2(t *testing.T) {
	ch := make(chan int, 10)
	wp := NewPool(2)

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

func TestMultiWait(t *testing.T) {
	// fmt.Println("Test output")
	wp := NewPool(2)

	chanNum := 10

	funcReturnChannels := make([]chan struct{}, chanNum)
	for i := range funcReturnChannels {
		funcReturnChannels[i] = make(chan struct{}, 1)
	}
	funcUnblockChannels := make([]chan struct{}, chanNum)
	for i := range funcUnblockChannels {
		funcUnblockChannels[i] = make(chan struct{})
	}
	// fmt.Println("About to start 10 go routines")
	for i := 0; i < chanNum; i++ {
		i := i
		wp.AddJob(func() {
			// fmt.Println("waiting for channel no:", i)
			<-funcUnblockChannels[i]
			// fmt.Println("Unlocked channel:", i)
			funcReturnChannels[i] <- struct{}{}
		})
	}

	// fmt.Println("Spawned all the go routines")
	waiterReturn := make(chan struct{})
	// go routines are now blocked
	// let's wait for them in another go routine
	go func() {
		wp.Wait()
		waiterReturn <- struct{}{}
	}()

	// ok now let's unblock all the jobs from another go routine
	go func() {
		for i := range funcUnblockChannels {
			t.Log("Unlocking channel:", i)
			funcUnblockChannels[i] <- struct{}{}
		}
	}()

	// pick up to go-routins signals
	for i := range funcReturnChannels {
		<-funcReturnChannels[i]
		// fmt.Println("Got signal from channel: ", i)
	}

	// wait for the waiter that runs in another go routine
	<-waiterReturn
	wp.Wait()
	// fmt.Println("TestMultiWait test finished")
}
