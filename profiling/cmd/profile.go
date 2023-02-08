package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/cnutshell/go-pearls/profiling"
)

func init() {
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
}

func main() {
	data := profiling.NewLockedData()
	for i := 0; i < 500; i++ {
		go func() {
			for {
				data.Write()
			}
		}()

		go func() {
			for {
				data.Read()
			}
		}()
	}

	fmt.Println("pprof serve on http://localhost:6060/debug/pprof")
	http.ListenAndServe("localhost:6060", nil)
}
