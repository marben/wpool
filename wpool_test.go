package wpool

import (
	"testing"
)

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
	// this is done in order, but jobs should be started in order, so it shouldn't block
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

	// some more projects and one more wait to see if we can reuse the wp
	for i := 0; i < chanNum; i++ {
		i := i
		wp.AddJob(func() {
			// fmt.Println("waiting for channel no:", i)
			<-funcUnblockChannels[i]
			// fmt.Println("Unlocked channel:", i)
			funcReturnChannels[i] <- struct{}{}
		})
	}
	go func() {
		for i := range funcUnblockChannels {
			t.Log("Unlocking channel:", i)
			funcUnblockChannels[i] <- struct{}{}
		}
	}()
	wp.Wait()

	// just out of curiosity
	wp.Wait()
}
