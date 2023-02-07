package profiling

import (
	"sync"
	"testing"
)

// TestContention creates goroutines that will content.
func TestContention(t *testing.T) {
	t.Log("Starting Test")

	var wg sync.WaitGroup
	lock := NewLockedData()

	wg.Add(1000)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()
			lock.Write()
		}()

		go func() {
			defer wg.Done()
			lock.Read()
		}()
	}
	wg.Wait()

	t.Log("Test Complete")
}
