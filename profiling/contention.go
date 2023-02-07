package profiling

import (
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type LockedData struct {
	mu sync.RWMutex
}

func NewLockedData() *LockedData {
	return &LockedData{}
}

func (l *LockedData) Write() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// do some job
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
}

func (l *LockedData) Read() {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// do some job
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
}
