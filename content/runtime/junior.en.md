# Runtime: scheduler, memory, GC — Junior

Every Go program contains a **runtime**: code automatically included in the binary that schedules goroutines onto the CPU, allocates memory and frees it through the **garbage collector** (GC). You don't have to start or configure it for it to work, but it helps to know what it does.

## The binary contains everything

```bash
go build -o app .
ls -lh app      # a few MB, even for "hello world"
./app           # runs with no Go installed and no other libraries
```

The size comes mostly from the runtime and the standard library. The upside: a single file, easy to distribute and to put in a minimal container.

## You don't free memory by hand

Go has no `free` or `delete` for memory. You allocate with `make`, `new`, `&T{}` or literals, and the GC automatically frees what's no longer used:

```go
package main

import (
	"fmt"
	"runtime"
)

func main() {
	var stats runtime.MemStats

	data := make([][]byte, 0, 100)
	for range 100 {
		data = append(data, make([]byte, 1<<20)) // 100 × 1 MiB
	}
	runtime.ReadMemStats(&stats)
	fmt.Printf("in use: ~%d MiB\n", stats.HeapAlloc>>20)

	data = nil   // we no longer reference anything
	runtime.GC() // force a collection (for the demo only)
	runtime.ReadMemStats(&stats)
	fmt.Printf("after GC: ~%d MiB\n", stats.HeapAlloc>>20)
	fmt.Println("collections so far:", stats.NumGC)
}
```

An object is freed when there's **no longer any path** to reach it from the program (variables, other reachable objects). In ordinary code you never call `runtime.GC()`.

## Stack and heap

- Local variables usually live on the goroutine's **stack**: allocating and freeing is practically free.
- Values that must outlive the function (for example a returned pointer) go on the **heap**, where the GC frees them.

The compiler decides by itself where each variable lives (see [Pointers](../../pointeri/senior/)).

## The scheduler

The runtime spreads goroutines over a small number of operating system threads. How many actually run in parallel is set by `GOMAXPROCS`, by default the number of cores (or the container's CPU limit, since Go 1.25):

```go
package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {
	fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0))

	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() { _ = make([]byte, 1024) })
	}
	fmt.Println("goroutines active now:", runtime.NumGoroutine())
	wg.Wait()
	fmt.Println("after Wait:", runtime.NumGoroutine())
}
```

Details on how it works in [Goroutines](../../goroutines/senior/).

## Information about the runtime

```go
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("Go version:", runtime.Version())
	fmt.Println("system:", runtime.GOOS, runtime.GOARCH)
	fmt.Println("cores:", runtime.NumCPU())
}
```

## Important environment variables

| Variable | What it controls |
|---|---|
| `GOMAXPROCS` | how many threads run Go code at once |
| `GOGC` | how often the GC runs (100 by default) |
| `GOMEMLIMIT` | a "soft" memory limit for the runtime |
| `GODEBUG` | debugging and compatibility options |

The Middle level explains `GOGC` and `GOMEMLIMIT`.

## Summary

- The runtime is included in every binary: scheduler, allocator, GC.
- Memory is freed automatically once it's unreachable.
- The stack is cheap; the heap is managed by the GC.
- `GOMAXPROCS` = how many goroutines actually run in parallel.
- Don't call `runtime.GC()` in normal code.
