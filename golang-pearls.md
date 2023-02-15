# Golang Pearls

2017 年左右开始接触 golang，那时国内关于 golang 的资料还很少。

现在随着云原生生态的蓬勃发展，在 kubernetes、docker 等等众多明星项目的带动下，国内有越来越多的创业公司以及大厂开始拥抱 golang，各种介绍 golang 的书籍、博客和公众号文章也变得越来越多，其中不乏质量极高的资料。

相关的资料已经足够丰富，因此这篇文章不会详述 golang 的某一个方面，而是主要从工程实践的角度出发，去介绍一些东西。因为在工作过程中，我注意到一些令人沮丧的代码，其中有些甚至来自于高级程序员。

下面我们从内存有关的话题开始。

## 1. 内存相关

### 1.1 golang 编译器内存逃逸分析

先看这样一段代码：

```go
package main

//go:noinline
func makeBuffer() []byte {
    return make([]byte, 1024)
}

func main() {
    buf := makeBuffer()
    for i := range buf {
        buf[i] = buf[i] + 1
    }
}
```

示例代码中函数 `makeBuffer` 返回的内存位于函数栈上，在 C 语言中，这是一段错误的代码，会导致未定义的行为。

在 Go 语言中，这样的写法是允许的，Go 编译器会执行 `escape analysis`：当它发现一段内存不能放置在函数栈上时，会将这段内存放置在堆内存上。例如，`makeBuffer` 向上返回栈内存，编译器自动将这段内存放在堆内存上。

通过 `-m` 选项可以查看编译器分析结果：

```bash
$ go build -gcflags="-m" escape.go
# command-line-arguments
./escape.go:8:6: can inline main
./escape.go:5:13: make([]byte, 1024) escapes to heap
```

除此之外，也存在其他一些情况会触发内存的“逃逸”：

- 全局变量，因为它们可能被多个 goroutine 并发访问

- 通过 channel 传送指针

  ```go
  type Hello struct { name string }
  ch := make(chan *Hello, 1)
  ch <- &Hello{ name: "world"}
  ```

- 通过 channel 传送的结构体中持有指针

  ```go
  type Hello struct { name *string }
  ch := make(chan *Hello, 1)
  name := "world"
  ch <- Hello{ name: &name }
  ```

- 局部变量过大，无法放在函数栈上

- 本地变量的大小在编译时未知，例如 `s := make([]int, 1024)` 也许不会被放在堆内存上，但是 `s := make([]int, n)` 会被放在堆内存上，因为其大小 `n` 是个变量

- 对 `slice` 的 `append` 操作触发了其底层数组重新分配

注意：上面列出的情况不是详尽的，并且可能随 Go 的演进发生变化。

在开发过程中，如果程序员不注意 golang 编译器的内存逃逸分析，写出的代码可能会导致“额外”的动态内存分配，而 “额外”的动态内存分配通常会和性能问题联系在一起（具体会在后面 golang gc 的章节中介绍）。

示例代码给我们的启示是：注意函数签名设计，尽量避免因函数签名设计不合理而导致的不必要内存分配。向上返回一个 slice 可能会触发内存逃逸，向下传入一个 slice 则不会，这方面 [cockroach encoding function](https://github.com/cockroachdb/cockroach/blob/5fbcd8a8deac0205c7df38e340c1eb9692854383/pkg/util/encoding/encoding.go#L180) 给出了一个很好的例子。

接下来，我们看下接口相关的事情。

### 1.2 interface{} / any

any 是 golang 1.18 版本引入的，跟 interface{} 等价。

```go
type any = interface{}
```

在 golang 中，[接口实现](https://github.com/golang/go/blob/3fc8ed2543091693eca514b363fcdbbe5c7f2916/src/runtime/runtime2.go#L202) 为一个“胖”指针：一个指向实际的数据，一个指向函数指针表（类似于C++ 中的虚函数表）。

我们来看下面的代码：

```go
package interfaces

import (
	"testing"
)

var global interface{}

func BenchmarkInterface(b *testing.B) {
	var local interface{}
	for i := 0; i < b.N; i++ {
		local = calculate(i) // assign value to interface{}
	}
	global = local
}

// values is bigger than single machine word.
type values struct {
	value  int
	double int
	triple int
}

func calculate(i int) values {
	return values{
		value:  i,
		double: i * 2,
		triple: i * 3,
	}
}
```

在性能测试 `BenchmarkInterface` 中，我们将函数 `calculate` 返回的结果赋值给 `interface{}` 类型的变量。

接下来，对 `BenchmarkInterface` 执行 memory profile：

```bash
$ go test -run none -bench Interface -benchmem -memprofile mem.out

goos: darwin
goarch: arm64
pkg: github.com/cnutshell/go-pearls/memory/interfaces
BenchmarkInterface-8    101292834               11.80 ns/op           24 B/op          1 allocs/op
PASS
ok      github.com/cnutshell/go-pearls/memory/interfaces        2.759s

$ go tool pprof -alloc_space -flat mem.out
(pprof) top 
(pprof) list iface.BenchmarkInterface
Total: 2.31GB
    2.31GB     2.31GB (flat, cum) 99.89% of Total
         .          .      7:var global interface{}
         .          .      8:
         .          .      9:func BenchmarkInterface(b *testing.B) {
         .          .     10:   var local interface{}
         .          .     11:   for i := 0; i < b.N; i++ {
    2.31GB     2.31GB     12:           local = calculate(i) // assign value to interface{}
         .          .     13:   }
         .          .     14:   global = local
         .          .     15:}
         .          .     16:
         .          .     17:// values is bigger than single machine word.
(pprof)
```

从内存剖析结果看到：向接口类型的变量 `local` 赋值，会触发内存“逃逸”，导致额外的动态内存分配。

go 1.18 引入范型之前，我们都是基于接口实现多态，基于接口实现多态，主要存在下面这些问题：

1. 丢失了类型信息 ，程序行为从编译阶段转移到运行阶段；
2. 程序运行阶段不可避免地需要执行类型转换，类型断言或者反射等操作；
3. 为接口类型的变量赋值可能会导致“额外的”内存分配；
4. 基于接口的函数调用，实际的调用开销为：指针解引用（确定方法地址）+ 函数执行开销。编译器无法对其执行内联优化，也无法基于内联优化执行进一步的优化；

关于接口的使用，这里有一些提示：

- 代码中避免使用 `interface{}` 或者 `any`，至少避免在被频繁使用的数据结构或者函数中使用
- go 1.18 引入了范型，将接口类型改为范型类型，是避免额外内存分配，优化程序性能的一个手段

下面到了介绍 golang gc 的时候。

### 1.3 golang gc

前面我们了解到，golang 编译器执行 escape analysis 后，根据需要数据可能被“搬”到堆内存上。

这里简单地介绍下 golang 的 gc，从而了解写 golang 代码时为什么应该尽量避免“额外的”内存分配。

#### 1.3.1 Introduction

gc 是 go 语言非常重要的一部分，它大大简化了程序员写并发程序的复杂度。

人们发现写工作良好的并发程序似乎也不再是那少部分程序员的独有技能。

glang gc 使用一棵树来维护堆内存对象的引用，属于追踪式的 gc，它基于“标记-清除“算法工作，主要分为两个阶段：

1. 标记阶段 - 遍历所有堆内存对象，判断这些对象是否在用；
2. 清除阶段 - 遍历树，清除没有被引用的堆内存对象；

执行 gc 时，**golang 首先会执行一系列操作并停止应用程序的执行**，即 `stopping the world`，之后恢复应用程序的执行，同时 gc 其他相关的操作还会并行地执行。所以 golang 的 gc 也被称为 `concurrent mark-and-sweep`，这样做的目的是尽可能减少 `STW` 对程序运行的影响。

> 严格地说，`STW` 会发生两次，分别在标记开始和标记结束时。

golang gc 包括一个 `scavenger`，定期将不再使用的内存返还给操作系统。

> 也可以在程序中调用 `debug.FreeOSMemory()`，手动将内存返还给操作系统。

#### 1.3.2 gc 触发机制

相比于 java，golang 提供的 gc 控制方式比较简单：通过环境变量 `GOGC` 来控制。

> [runtime/debug.SetGCPercent](https://pkg.go.dev/runtime/debug#SetGCPercent) allows changing this percentage at run time.

`GOGC`  定义了触发下次 gc 时堆内存的增长率，默认值为 100，即上次 gc 后，堆内存增长一倍时，触发另一次 gc。

例如，gc 触发时当前堆内存的大小时 128MB，如果 `GOGC=100`，那么当堆内存增长为 256MB时，执行下一次 gc。

另外，如果 golang 两分钟内没有执行过 gc，会强制触发一次。

我们也可以在程序中调用 `runtime.GC()` 主动触发 gc。

```bash
# 通过设置环境变量 GODEBUG 可以显示 gc trace 信息

$ GODEBUG=gctrace=1 go test -bench=. -v

# 当 gc 运行时，相关信息会写到标准错误中
```

注意：为了减少 gc 触发次数而增加 `GOGC` 值并不一定能带来线性的收益，因为即便 gc 触发次数变少了，但是 gc 的执行可能会因为更大的堆内存而有所延长。在大多数情况下，`GOGC` 维持在默认值 100 即可。

#### 1.3.3 gc hints

如果我们的代码中存在大量“额外”的堆内存分配，尤其是在代码关键路径上，对于性能的负面影响是非常大的：

- 首先，堆内存的分配本身就是相对耗时的操作
- 其次，大量“额外”的堆内存分配意味着额外的 gc 过程，STW 会进一步影响程序执行效率；

极端情况下，短时间内大量的堆内存分配，可能会直接触发 OOM，gc 甚至都没有执行的机会。

所以，不要“天真”的以为 gc 会帮你搞定所有的事情：你留给 gc 处理的工作越少，你的性能才会越“体面”。

从性能优化的角度，消除那些“额外的”内存分配收益十分明显，通常也会是第一或者第二优先的选项。

然而，堆内存的使用并不能完全避免，当需要使用时，可以考虑采用某些技术，例如通过 `sync.Pool` 复用内存来减少 gc 压力。

#### 1.3.4 有了 gc 为什么还会有内存泄漏

即便 golang 是 gc 语言，它并不是一定没有内存泄漏，下面两种情况会导致内存泄漏的情况：

1. 引用堆内存对象的对象长期存在；
2. goroutine 需要消耗一定的内存来保存用户代码的上下文信息，goroutine 泄漏会导致内存泄漏；

#### 1.3.5 代码演示

代码见于文件 [gc.go](https://gist.github.com/cnutshell/817b17f6eb4fa5c4383c0c7d53c744c0)：

- 函数 `allocator` 通过 channel 传送 `buf` 类型的结构体，`buf` 类型的结构体持有堆内存的引用；
- 函数 `mempool` 通过 channel 接收来自 `allocator` 的 buf，循环记录在 slice 中；
- 同时，`mempool` 还会定期打印应用当前内存状态，具体含义参考 [runtime.MemStats](https://pkg.go.dev/runtime@go1.20#MemStats)

运行代码 [gc.go](https://gist.github.com/cnutshell/817b17f6eb4fa5c4383c0c7d53c744c0)：

```bash
$ go run gc.go
HeapSys(bytes),PoolSize(MiB),HeapAlloc(MiB),HeapInuse(MiB),HeapIdle(bytes),HeapReleased(bytes)
 12222464,     5.00,     7.11,     7.45,  4415488,  4300800
 16384000,    10.00,    12.11,    12.45,  3334144,  3153920
 24772608,    18.00,    20.11,    20.45,  3334144,  3121152
 28966912,    22.00,    24.11,    24.45,  3334144,  3121152
 33161216,    25.00,    27.11,    27.45,  4382720,  4169728
 37355520,    32.00,    34.11,    34.45,  1236992,   991232
 41549824,    36.00,    38.11,    38.45,  1236992,   991232
 54132736,    48.00,    50.11,    50.45,  1236992,   991232
 58327040,    51.00,    53.11,    53.45,  2285568,  2039808
```

通过程序输出结果，我们可以了解到：如果程序中存在变量持有对堆内存的引用，那么这块堆内存不会被 gc 回收。

因此使用引用了堆内存的变量赋值时，例如将其赋值给新的变量，需要注意避免出现内存泄漏。通常建议将赋值有关的操作封装在方法中，以通过合理的 API 设计避免出现“意想不到”内存泄露。并且封装还带来的好处是提高了代码的可测性。

#### 1.3.6 参考资料

[Blog: Go Data Structures: Interfaces](https://research.swtch.com/interfaces)

[GOGC on golang's document](https://pkg.go.dev/runtime@go1.20#hdr-Environment_Variables)

[GC 的认识](https://www.bookstack.cn/read/qcrao-Go-Questions/GC-GC.md)

与内存有关的介绍先告一段落，下面介绍下 golang profiling。

## 2. Golang Profiling

`profiler` 运行用户程序，同时配置操作系统定期送出 `SIGPROF` 信号：

- 收到 `SIGPRFO` 信号后，暂停用户程序执行；
- `profiler` 搜集用户程序运行状态；
- 搜集完毕恢复用户程序执行。

如此循环。

`profiler` 是基于采样的，对程序性能存在一定程度的影响。

> *"Before you profile, you must have a stable environment to get repeatable results."*

### 2.2 Golang Profiling

#### 2.2.1 Supported Profiling

By default, all the profiles are listed in [runtime/pprof.Profile](https://pkg.go.dev/runtime/pprof#Profile).

##### a. CPU Profiling

 CPU profiling 使能后，golang runtime 默认每 10ms 中断应用程序，并记录 goroutine 的堆栈信息。

##### b. Memory Profiling

Memory profiling 和 CPU profiling 一样，也是基于采样的，它会在堆内存分配时记录堆栈信息。

默认每 1000 次堆内存分配会采样一次，这个频率可以配置。

注意：Memory profiling 仅记录堆内存分配信息，忽略栈内存的使用。

##### c. Block Profiling

Block profiling 类似于 CPU profiling，不过它记录 goroutine 在共享资源上等待的时间。

它对于检查应用的并发瓶颈很有帮助，Blocking 统计对象主要包括：

- 读/写 unbuffered channel
- 写 full buffer channel，读 empty buffer channel
- 加锁操作

如果基于 `net/http/pprof`， 应用程序中需要调用 [runtime.SetBlockProfileRate](https://pkg.go.dev/runtime#SetBlockProfileRate)  配置采样频率。

##### d. Mutex Profiling

Go 1.8 引入了 mutex profile，允许你捕获一部分竞争锁的 goroutines 的堆栈。

如果基于 `net/http/pprof`， 应用程序中需要调用 [runtime.SetMutexProfileFraction](https://pkg.go.dev/runtime#SetMutexProfileFraction) 配置采样频率。

注意：通过 `net/http/pprof` 对线上服务执行 profiling 时，不建议修改 golang profiler 默认值，因为某些 profiler 参数的修改，例如增加 memory profile sample rate，可能会导致程序性能出现明显的降级，除非你明确的知道可能造成的影响。

### 2.3 profiling commands

我们可以从 `go test` 命令，或者从使用 `net/http/pprof` 的应用中获取到 profile 文件：

```bash
## 1. From unit tests
$ go test [-blockprofile | -cpuprofile | -memprofile | -mutexprofile] xxx.out

## 2. From long-running program with `net/http/pprof` enabled
## 2.1 heap profile
$ curl -o mem.out http://localhost:6060/debug/pprof/heap

## 2.2 cpu profile
$ curl -o cpu.out http://localhost:6060/debug/pprof/profile?seconds=30
```

获取到 profile 文件之后，通过 `go tool pprof` 进行分析：

```bash
# 1. View local profile
$ go tool pprof xxx.out

# 2. View profile via http endpoint
$ go tool pprof http://localhost:6060/debug/pprof/block
$ go tool pprof http://localhost:6060/debug/pprof/mutex
```

### 2.4 Golang Trace

我们可以从 `go test` 命令，或者从使用 `net/http/pprof` 的应用中获取到 trace 文件：

```bash
# 1. From unit test
$ go test -trace trace.out

# 2. From long-running program with `net/http/pprof` enabled
curl -o trace.out http://localhost:6060/debug/pprof/trace?seconds=5
```

获取到 trace 文件之后，通过 `go tool trace` 进行分析，会自动打开浏览器：

```bash
$ go tool trace trace.out
```

### 2.5 Profiling Hints

如果大量时间消耗在函数 [runtime.mallocgc](https://github.com/golang/go/blob/0b9974d3f09fe3132b4bc4aef67b839e3f84a8c8/src/runtime/malloc.go#L878)，意味着程序发生了大量堆内存分配，通过 Memory Profiling 可以确定分配堆内存的代码在哪里；

如果大量的时间消耗在同步原语（例如 channel，锁等等）上，程序可能存在并发问题，通常意味着程序工作流程需要重新设计；

如果大量的时间消耗在 `syscall.Read/Write`，那么程序有可能执行大量小 IO；

如果 GC 组件消耗了大量的时间，程序可能分配了大量的小内存，或者分配的堆内存比较大；

### 2.6 代码演示

代码见于文件 [contention_test.go](https://gist.github.com/cnutshell/80e1724c6bfcabe79485cf0b7167aca0)：

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

### 2.7 参考资料

[net/http/pprof examples](https://pkg.go.dev/net/http/pprof@go1.20#hdr-Usage_examples)

下面介绍下如何写 Benchmark。

## 3. 性能测试

性能问题不是猜测出来的，即便我们“强烈的认为”某处代码是性能瓶颈，也必须经过验证。

> *"Those who can make you believe absurdities can make you commit atrocities" - Voltaire*

对于性能测试来说，很容易写出不准确的 Benchmark，从而形成错误的印象。

### 3.1 Reset or Pause timer

```go
func BenchmarkFoo(b *testing.B) {
  heavySetup()  // 在 for 循环之前执行设置工作，如果设置工作比较耗时，那么会影响测试结果的准确性
  for i := 0; i < b.N; i++ {
    foo()
  }
}
```

- 优化方式

```go
func BenchmarkFoo(b *testing.B) {
  heavySetup()
  b.ResetTimer()  // 重置 timer，保证测试结果的准确性
  for i := 0; i < b.N; i++ {
    foo()
  }
}
```

- 如何停止 timer

```go
func BenchmarkFoo(b *testing.B) {
  for i := 0; i < b.N; i++ {
    b.StopTimer() // 停止 timer
    heavySetup()
    b.StartTimer() // 启动 timer
    foo()
  }
}
```

### 3.2 提高测试结果可信度

对于 Benchmark，有很多因素会影响结果的准确性：

- 机器负载情况
- 电源管理设置
- 热扩展(thermal scaling)
- ……

相同的性能测试代码，在不同的架构，操作系统下运行可能会产生截然不同的结果；

相同的 Benchmark 即便在同一台机器运行，前后也可能产生不一致的数据。

简单的方式是增加 Benchmark 运行次数或者多次运行测试来获取相对准确的结果：

- 通过 `-benchtime` 设置性能测试时间（默认 1秒）
- 通过 `-count` 多次运行 Benchmark

```go
package benchmark

import (
        "sync/atomic"
        "testing"
)

func BenchmarkAtomicStoreInt32(b *testing.B) {
        var v int32
        for i := 0; i < b.N; i++ {
                atomic.StoreInt32(&v, 1)
        }
}

func BenchmarkAtomicStoreInt64(b *testing.B) {
        var v int64
        for i := 0; i < b.N; i++ {
                atomic.StoreInt64(&v, 1)
        }
}
```

多次运行测试，得出置信度较高的结果：

```go
$ go test -bench Atomic -count 10 | tee stats.txt

$ benchstat stats.txt
goos: darwin
goarch: arm64
pkg: github.com/cnutshell/go-pearls/benchmark
                   │   stats.txt   │
                   │    sec/op     │
AtomicStoreInt32-8   0.3131n ± ∞ ¹
AtomicStoreInt64-8   0.3129n ± ∞ ¹
geomean              0.3130n
¹ need >= 6 samples for confidence interval at level 0.95
```

> 如果提示 `benchstat` 未找到，通过 `go install` 命令安装：`go install golang.org/x/perf/cmd/benchstat@latest`

### 3.3 注意编译器优化 

```go
package benchmark

import "testing"

const (
        m1 = 0x5555555555555555
        m2 = 0x3333333333333333
        m4 = 0x0f0f0f0f0f0f0f0f
)

func calculate(x uint64) uint64 {
        x -= (x >> 1) & m1
        x = (x & m2) + ((x >> 2) & m2)
        return (x + (x >> 4)) & m4
}

func BenchmarkCalculate(b *testing.B) {
        for i := 0; i < b.N; i++ {
                calculate(uint64(i))
        }
}

func BenchmarkCalculateEmpty(b *testing.B) {
        for i := 0; i < b.N; i++ {
                // empty body
        }
}
```

运行示例代码中的测试，两个测试的结果相同：

```bash
$ go test -bench Calculate
goos: darwin
goarch: arm64
pkg: github.com/cnutshell/go-pearls/benchmark
BenchmarkCalculate-8            1000000000               0.3196 ns/op
BenchmarkCalculateEmpty-8       1000000000               0.3154 ns/op
PASS
ok      github.com/cnutshell/go-pearls/benchmark        0.814s
```

那么如何避免这种情况呢，前面介绍 golang 接口的时候，给出了一个例子：

```go
var global interface{}

func BenchmarkInterface(b *testing.B) {
	var local interface{}
	for i := 0; i < b.N; i++ {
    local = calculate(uint64(i)) // assign value to interface{}
	}
	global = local
}
```

将被 `calculate` 的返回值赋给本地变量 `local`，循环结束后将本地变量 `local` 赋值给一个全局变量 `global`，这样可以避免函数 `calculate` 被编译器优化掉。

### 3.4 总结

错误的性能测试结果会导致我们做出错误的决定，正所谓“差之毫厘，谬以千里”，写性能测试代码并不是表面上看起来的那么简单。