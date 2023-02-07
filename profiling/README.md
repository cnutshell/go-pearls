# Golang Profiling

## 1. Golang Profiler

`profiler` 运行用户程序，同时配置操作系统定期送出 `SIGPROF` 信号：

- 收到 `SIGPRFO` 信号后，暂停用户程序执行；
- `profiler` 搜集用户程序运行状态；
- 搜集完毕恢复用户程序执行。

如此循环。

`profiler` 是基于采样的，对程序性能存在一定影响。

> 注意：通过 `net/http/pprof` 对线上服务执行 profiling 时，不建议修改 golang profiler 默认值，因为某些 profiler 参数的修改，例如增加 memory profile sample rate，可能会导致程序性能出现明显的降级，除非你明确的知道可能造成的影响。

> NOTE: Before you profile, you must have a stable environment to get repeatable results.

## 2. Golang Profiling

### 2.1 Supported Profiling

By default, all the profiles are listed in [runtime/pprof.Profile](https://pkg.go.dev/runtime/pprof#Profile).

#### a. CPU Profiling

 CPU profiling 使能后，golang runtime 默认每 10ms 中断应用程序，并记录 goroutine 的堆栈信息。

#### b. Memory Profiling

Memory profiling 和 CPU profiling 一样，也是基于采样的，它会在堆内存分配时记录堆栈信息。

默认每 1000 次堆内存分配会采样一次，这个频率可以配置。

注意：Memory profiling 仅记录堆内存分配信息，忽略栈内存的使用。

#### c. Block Profiling

Block profiling 类似于 CPU profiling，不过它记录 goroutine 在共享资源上等待的时间。

它对于检查应用的并发瓶颈很有帮助，Blocking 统计对象主要包括：

- 读/写 unbuffered channel
- 写 full buffer channel，读 empty buffer channel
- 加锁操作

如果基于 `net/http/pprof`， 应用程序中需要调用 [runtime.SetBlockProfileRate](https://pkg.go.dev/runtime#SetBlockProfileRate)  配置采样频率。

#### d. Mutex Profiling

Go 1.8 引入了 mutex profile，允许你捕获一部分竞争锁的 goroutines 的堆栈。

如果基于 `net/http/pprof`， 应用程序中需要调用 [runtime.SetMutexProfileFraction](https://pkg.go.dev/runtime#SetMutexProfileFraction) 配置采样频率。

### 2.2 profiling commands

Generate a profile.

```bash
## 1. From unit tests
$ go test [-blockprofile | -cpuprofile | -memprofile | -mutexprofile] xxx.out

## 2. From long-running program with `net/http/pprof` enabled
## 2.1 heap profile
$ curl -o mem.out http://localhost:6060/debug/pprof/heap

## 2.2 cpu profile
$ curl -o cpu.out http://localhost:6060/debug/pprof/profile?seconds=30
```

Run the pprof tool to view the profile.

```bash
# 1. View local profile
$ go tool pprof xxx.out

# 2. View profile via http endpoint
$ go tool pprof http://localhost:6060/debug/pprof/block
$ go tool pprof http://localhost:6060/debug/pprof/mutex
```

## 3. Golang Trace

Generate a trace

```bash
# 1. From unit test
$ go test -trace trace.out

# 2. From long-running program with `net/http/pprof` enabled
curl -o trace.out http://localhost:6060/debug/pprof/trace?seconds=5
```

Run the trace tool to review the trace.

```bash
$ go tool trace trace.out
```

## 4. Profiling Hints

如果大量时间消耗在函数 `runtime.mallocgc`，那么程序有可能产生了大量堆内存分配，通过 Memory Profiling 可以确定分配堆内存的代码有哪些；

如果大量的时间消耗在同步原语（例如 channel，锁等等）上，程序可能存在并发问题，通常意味着执行流程需要重新设计；

如果大量的时间消耗在 `syscall.Read/Write`，那么程序有可能执行大量小 IO；

如果 GC 组件消耗了大量的时间，程序可能分配了大量的小内存，或者分配的堆内存比较大；

## 5. 相关命令

- Block Profiling with Unit Test

```bash
$ go test -run ^TestContention$ -blockprofile block.out
$ go tool pprof block.out
(pprof) top
(pprof) web
```

- Mutex Profiling with Unit Test

```bash
$ go test -run ^TestContention$ -mutexprofile mutex.out
$ go tool pprof mutex.out
(pprof) top
(pprof) web
```

- Trace with Unit Test

```bash
$ go test -run ^TestContention$ -trace trace.out
$ go tool trace trace.out
```

- Block Profiling with `net/http/pprof`

```bash
$ go run cmd/profile.go
pprof serve on http://localhost:6060/debug/pprof

# On another terminal
$ go tool pprof http://localhost:6060/debug/pprof/block
(pprof) top
(pprof) web
```

- Trace with `net/http/pprof`

```bash
$ curl -o trace.out  http://localhost:6060/debug/pprof/tracef
$ go tool trace trace.out
```

## 参考资料

[net/http/pprof examples](https://pkg.go.dev/net/http/pprof@go1.20#hdr-Usage_examples)

[Private Docs：Go profile 获取与查看](https://github.com/matrixorigin/docs/blob/main/guide/debug/profiling-guide.md)

