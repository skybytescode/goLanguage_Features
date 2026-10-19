# Errors, defer, panic and recover — Senior

The exact semantics of `defer`, the strict rules for `recover`, the difference between panics and fatal runtime errors, and how error chains work under the hood.

## `defer` arguments are evaluated immediately

For the statement `defer f(x)`, the function value and the arguments are evaluated **now**; only the call is postponed. A closure, by contrast, reads variables when it runs:

```go
package main

import "fmt"

func main() {
	x := 1
	defer fmt.Println("argument evaluated immediately:", x)        // 1
	defer func() { fmt.Println("closure reads at the end:", x) }() // 2
	x = 2
}
```

The same applies to the receiver: `defer obj.Method()` evaluates `obj` immediately. A handy pattern that relies on this:

```go
package main

import (
	"fmt"
	"time"
)

func timer(name string) func() {
	start := time.Now()
	return func() { fmt.Printf("%s: %v\n", name, time.Since(start).Round(time.Millisecond)) }
}

func main() {
	defer timer("main")() // timer runs now, the returned function at the end
	time.Sleep(20 * time.Millisecond)
}
```

## The rules of `recover`

`recover` stops a panic **only** if it's called **directly** by a deferred function, in the goroutine that panicked. Anywhere else it returns `nil`:

```go,panic
package main

import "fmt"

func helper() {
	if r := recover(); r != nil { // does NOT work: not called directly by the deferred function
		fmt.Println("caught", r)
	}
}

func main() {
	defer func() {
		helper()
	}()
	panic("boom")
}
```

`defer helper()` (without the intermediate closure) would have worked: then `helper` itself is the deferred function.

Other rules:

- **Each goroutine is on its own.** A panic in a goroutine with no `recover` kills **the whole program**, however many `recover`s `main` has. Code that starts goroutines in a server must add its own `defer recover()` where appropriate.
- Since Go 1.21, `panic(nil)` produces a `*runtime.PanicNilError`, so `recover()` never returns `nil` for a real panic.
- A panic raised in a `defer` while another panic is in progress replaces it; both appear in the final message.
- After `recover`, the function containing the `defer` returns normally, with its named results as the `defer` left them. That's how you turn a panic into an error:

```go
package main

import (
	"fmt"
)

func divide(a, b int) (res int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	return a / b, nil
}

func main() {
	fmt.Println(divide(1, 0))
}
```

## Fatal errors: can't be recovered

Some situations aren't panics but **fatal runtime errors**; `recover` doesn't see them, and the program stops immediately:

| Fatal error | Cause |
|---|---|
| `concurrent map writes` | simultaneous writes to a map, unsynchronized |
| `all goroutines are asleep - deadlock!` | no goroutine can make progress |
| `stack overflow` | recursion too deep |
| `out of memory` | allocation impossible |

```go,panic
package main

func main() {
	defer func() { recover() }() // useless: the deadlock is fatal
	ch := make(chan int)
	<-ch // fatal error: all goroutines are asleep - deadlock!
}
```

## Exits that skip `defer`

- `os.Exit` ends the process **immediately**: `defer`s don't run. That's why `log.Fatal` (which calls `os.Exit(1)`) in `main` can leave files unflushed. The usual pattern is a `run() error` called from `main`, with a single `os.Exit` at the end.
- `runtime.Goexit` ends the current goroutine **after** running its `defer`s. It's what `t.FailNow()` uses in tests.

## How error chains work

Wrapping relies only on method conventions:

- `Unwrap() error` — a single "parent" (what `fmt.Errorf` with one `%w` creates);
- `Unwrap() []error` — several (what `errors.Join` or `fmt.Errorf` with several `%w` create).

`errors.Is` walks the tree **depth-first**, and at each node checks `==`, then an optional `Is(error) bool` method. That allows "semantic" equality:

```go
package main

import (
	"errors"
	"fmt"
)

type HTTPError struct{ Code int }

func (e HTTPError) Error() string { return fmt.Sprintf("http %d", e.Code) }

// Is: any 5xx matches ErrServer
func (e HTTPError) Is(target error) bool {
	return target == ErrServer && e.Code >= 500
}

var ErrServer = errors.New("server error")

func main() {
	err := fmt.Errorf("API call: %w", HTTPError{Code: 503})
	fmt.Println(errors.Is(err, ErrServer)) // true
}
```

`errors.As` can be customized the same way, with an `As(any) bool` method.

## The cost of errors

Creating an error with `errors.New` or `fmt.Errorf` allocates (and `fmt.Errorf` also formats). On hot paths where the error is frequent and "normal" (for example a parser trying several alternatives), use preallocated sentinels instead of new errors on every call. An `if err != nil` on the success path costs practically nothing: it's a comparison with zero, and branch prediction gets it right almost every time.

## Summary

- `defer f(x)` evaluates `f` and `x` immediately.
- `recover` works only when called directly from the deferred function, in the same goroutine.
- An unrecovered panic in any goroutine kills the whole program.
- Fatal errors (concurrent map writes, deadlock) can't be caught.
- `os.Exit` skips `defer`; `Unwrap`, `Is` and `As` define chains.
