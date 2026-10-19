# Pointers — Senior

Pointers determine where a value lives (stack or heap) and how much work the garbage collector has. At this level we look at escape analysis, interior pointers, weak pointers (`weak`) and cleanup functions.

## Escape analysis: stack or heap?

The Go spec doesn't talk about stack and heap. The compiler decides: a variable lives on the function's stack if it can prove it's **not used after the function returns**. Otherwise it "escapes" to the heap, where the GC will free it.

Stack allocations are practically free; heap allocations cost when allocating and, above all, later, at collection time. The decision is visible with `-gcflags=-m`:

```go,ignore
package main

type Point struct{ X, Y int }

func onStack() int {
	p := Point{1, 2} // doesn't escape: stays on the stack
	return p.X + p.Y
}

func onHeap() *Point {
	p := Point{1, 2} // moved to heap: p
	return &p
}

func viaInterface(v any) {}

func main() {
	x := 42
	viaInterface(x) // x escapes to heap (usually: the value is copied into an interface)
}
```

```bash
$ go build -gcflags='-m' .
./main.go:10:2: moved to heap: p
./main.go:18:15: x escapes to heap
```

Common causes of escape:

- returning a pointer to a local variable;
- storing a pointer in a global variable, a map, a channel or a field of a heap object;
- converting to an interface (e.g. arguments to `fmt.Println`), if the compiler can't prove the value isn't retained;
- closures that capture by reference and outlive the function;
- sizes unknown at compile time (`make([]byte, n)` with a variable `n`) or very large values.

The typical optimization in a hot loop: reuse a buffer instead of allocating a new one each iteration, or take the buffer from the caller (as `io.Reader.Read(p []byte)` does).

## Interior pointers keep the whole object alive

Go's GC accepts pointers **into** an object (to a field or an array element). As long as such a pointer exists, the **whole** object stays allocated:

```go
package main

import "fmt"

type Big struct {
	data [1 << 20]byte // 1 MiB
	id   int
}

func idOf() *int {
	b := &Big{id: 7}
	return &b.id // 8 bytes used, 1 MiB kept alive
}

func main() {
	fmt.Println(*idOf())
}
```

Sub-slices have the same effect: `big[:10]` keeps the entire array behind `big` alive. The fix is a copy: `slices.Clone(big[:10])`.

## Weak pointers: the `weak` package (Go 1.24)

A `weak.Pointer[T]` points to an object without keeping it alive. Once the object becomes unreachable and is collected, `Value()` returns `nil`. It's the foundation for caches that don't prevent memory from being freed:

```go
package main

import (
	"fmt"
	"runtime"
	"weak"
)

type Image struct{ pixels []byte }

func main() {
	img := &Image{pixels: make([]byte, 1<<20)}
	wp := weak.Make(img)

	fmt.Println(wp.Value() != nil) // true: img is still in use
	runtime.KeepAlive(img)

	img = nil
	runtime.GC()
	fmt.Println(wp.Value() == nil) // true: the object was collected
}
```

Two weak pointers made from the same object are equal (`==`), so they can be map keys; the `unique` package (Go 1.23) uses exactly this mechanism to intern values.

## Cleanup: `runtime.AddCleanup` (Go 1.24)

`runtime.AddCleanup(ptr, f, arg)` schedules `f(arg)` after the object at `ptr` becomes unreachable. It replaces `runtime.SetFinalizer`, which had serious pitfalls (it resurrected the object, didn't work on cycles, one finalizer per object):

```go
package main

import (
	"fmt"
	"runtime"
	"time"
)

type Resource struct{ id int }

func main() {
	r := &Resource{id: 1}
	runtime.AddCleanup(r, func(id int) {
		fmt.Println("cleaning up resource", id)
	}, r.id) // the argument must NOT be r: it would keep it alive

	r = nil
	runtime.GC()
	time.Sleep(10 * time.Millisecond) // cleanups run on a separate goroutine
}
```

Cleanups aren't guaranteed to run (for example, at program exit). Don't use them for resources that **must** be released; that's what `Close()` and `defer` are for.

## `runtime.KeepAlive`

An object can be collected **before** the last line where its variable appears, if the compiler sees it's no longer used. When an object owns an external resource (a file descriptor, C memory) and has a cleanup attached, `runtime.KeepAlive(x)` marks the point until which `x` must stay alive.

## `uintptr` is not a pointer

`uintptr` is an integer. The GC doesn't track it: if you store an object's address in a `uintptr`, the object may be collected or moved (stacks move when they grow). Conversions between `unsafe.Pointer` and `uintptr` are safe only in the precise patterns documented in the `unsafe` package; details in the [Reflection and unsafe](../../reflect-unsafe/senior/) topic.

## Write barriers

Go's GC runs concurrently with the program. So it doesn't miss pointers moved while it's marking, the compiler inserts a **write barrier** whenever a pointer is written to heap memory while the GC is active. It's one reason why pointer-free structures (e.g. `[]int` vs `[]*int`) are cheaper for the GC: they don't need scanning and don't trigger barriers.

## Summary

- Escape analysis picks stack or heap; check with `-gcflags=-m`.
- An interior pointer keeps the whole object alive; copy what you need.
- `weak.Pointer` for caches, `runtime.AddCleanup` instead of `SetFinalizer`.
- `uintptr` doesn't keep objects alive.
- Fewer pointers means less work for the GC.
