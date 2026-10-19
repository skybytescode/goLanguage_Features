# Functions and closures — Senior

What happens on a Go function call: the register-based calling convention, inlining, how closures are represented and how goroutine stacks grow.

## Calling convention: registers (ABIInternal)

Since Go 1.17 (amd64) and 1.18 (arm64, ppc64, riscv64), arguments and results are passed in **registers**, not on the stack. The typical gain was about 5% in speed plus smaller binaries.

Code written in assembly still uses the old convention, **ABI0** (everything on the stack); the compiler automatically generates "wrapper" functions between the two. That's why you may see symbols with an `.abi0` suffix when profiling.

## Inlining

The compiler copies the body of small functions into the call site. This removes the call overhead, but more importantly it enables other optimizations (constant propagation, more precise escape analysis, bounds check elimination).

A function is a candidate if its "cost", measured in syntax-tree nodes, is under a budget (80). See the compiler's decisions with `-gcflags=-m`:

```bash
$ go build -gcflags=-m ./...
./main.go:5:6: can inline add
./main.go:12:13: inlining call to add
./main.go:9:6: cannot inline process: function too complex: cost 112 exceeds budget 80
```

- `//go:noinline` above a function prevents inlining (useful in benchmarks).
- Since Go 1.12, "mid-stack" inlining works too: a function that calls other functions can itself be inlined.
- With **PGO** (profile-guided optimization, Go 1.21+), you put a CPU profile in `default.pgo` next to `main`; the compiler raises the budget for "hot" calls and **devirtualizes** interface calls when the profile shows a dominant concrete type.

## What a closure looks like

A function value is a pointer to a `funcval` struct: the code address, followed by the captured variables.

```go,ignore
// pseudo-code: what the compiler generates for a closure
type funcval struct {
	fn uintptr // the code address
	// followed by the captured variables: values or pointers
}
```

On a call, the struct's address goes into a dedicated register (`DX` on amd64), through which the closure's code finds its variables.

A variable is captured:

- **by value**, if it's small (under 128 bytes) and never modified after capture;
- **by reference** otherwise. The variable then "escapes" to the heap so it lives as long as the closure.

```go
package main

import "fmt"

func main() {
	x := 1
	f := func() int { return x } // captured by reference: x is modified below
	x = 2
	fmt.Println(f()) // 2, not 1
}
```

An anonymous function that captures nothing is just a static pointer, with no allocation.

## Function values are not comparable

Two functions can't be compared with `==`; the only allowed comparison is with `nil`. That's why functions can't be map keys.

```go,compile_fail
package main

func a() {}

func main() {
	f, g := a, a
	println(f == g) // error: func can only be compared to nil
}
```

## Growable stacks

Every goroutine starts with a small stack (at least 2 KiB; since Go 1.19 the initial size adapts to the average observed stack usage). Each function begins with a **prologue** that compares the stack pointer to the limit; if there's no room left, it calls `runtime.morestack`.

The stack doesn't grow through "segments" (as it did up to Go 1.3), but **by copying**: the runtime allocates a stack twice as large, copies the frames and adjusts every pointer into the stack. That's possible because escape analysis guarantees nothing on the heap points into a goroutine's stack.

Go does **not** do tail call optimization. Very deep recursion stops at the maximum stack size (1 GB on 64-bit, configurable with `debug.SetMaxStack`), with a fatal error, not a recoverable panic:

```go,panic
package main

import (
	"fmt"
	"runtime/debug"
)

func deep(n int) int {
	var buf [64]byte // makes every frame bigger
	buf[0] = byte(n)
	return deep(n+1) + int(buf[0])
}

func main() {
	debug.SetMaxStack(1 << 20) // 1 MiB, so the example finishes quickly
	fmt.Println(deep(0))       // fatal error: stack overflow
}
```

## Nearly free `defer`

Up to Go 1.13, every `defer` allocated a record. Since Go 1.14, **open-coded defers**: when a function has at most 8 `defer`s and none is in a loop, the compiler inserts the calls directly at the function's exits, and the cost drops to nanoseconds. A `defer` inside a loop falls back to the slower mechanism.

## Summary

- Arguments go in registers; assembly uses ABI0, with wrappers.
- Inlining has a budget; `-gcflags=-m` shows the decisions, PGO improves them.
- A closure is a `funcval`: code + captured variables (by value or by reference).
- Stacks grow by copying; there's no tail call optimization.
- `defer` is open-coded when it's not in a loop.
