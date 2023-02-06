package main

import (
	"fmt"
	"runtime"
	"time"
)

const (
	MiB          = 1024 * 1024
	PoolCapacity = 1024
)

type buf struct {
	mem []byte
}

func makeBuf() buf {
	return buf{
		mem: make([]byte, MiB),
	}
}

func main() {
	bufChan := make(chan buf, 1)
	done := make(chan bool)

	allocator := func() {
		for {
			select {
			case bufChan <- makeBuf():
			case <-done:
				return
			}
		}
	}
	go allocator()

	mempool := func() {
		pool := make([]buf, PoolCapacity)

		printTicker := time.NewTicker(1 * time.Second)
		defer printTicker.Stop()

		printMemStatsHeader()
		for i := 0; ; i++ {
			select {
			case buf, ok := <-bufChan:
				if !ok {
					return
				}
				pool[i%len(pool)] = buf

				time.Sleep(200 * time.Millisecond)

			case <-printTicker.C:
				bytes := 0
				for j := 0; j < len(pool); j++ {
					if pool[j].mem != nil {
						bytes += len(pool[j].mem)
					}
				}

				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				printMemStats(bytes, &m)
			}
		}
	}
	mempool()

	close(done)
}

func printMemStatsHeader() {
	fmt.Println("HeapSys(bytes),PoolSize(MiB),HeapAlloc(MiB),HeapInuse(MiB),HeapIdle(bytes),HeapReleased(bytes)")
}

func printMemStats(poolSize int, m *runtime.MemStats) {
	fmt.Printf(
		"%9d,%9.2f,%9.2f,%9.2f,%9d,%9d\n",
		m.HeapSys, float64(poolSize)/float64(MiB), float64(m.HeapAlloc)/float64(MiB), float64(m.HeapInuse)/float64(MiB), m.HeapIdle, m.HeapReleased,
	)
}
