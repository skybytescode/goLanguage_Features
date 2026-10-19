# Range and iterators — Senior

How the compiler rewrites a `range` loop over a function, how `break`, `return` and `defer` in the loop body are handled, what happens when an iterator breaks the rules, and how `iter.Pull` is implemented with runtime coroutines.

## Rewriting the loop

The compiler turns the loop body into a synthesized closure, passed as `yield`:

```go,ignore
// what you write
for x := range seq {
	if x > 10 {
		break
	}
	fmt.Println(x)
}

// roughly what the compiler generates
seq(func(x int) bool {
	if x > 10 {
		return false // break
	}
	fmt.Println(x)
	return true // continue
})
```

`continue` becomes `return true`, `break` becomes `return false`. The tricky cases are statements that **leave the outer function**: `return`, labeled `break`/`continue` targeting an outer loop, `goto`. For these, the compiler uses a state variable: the closure sets it and returns `false`, and after the iterator returns, the generated code checks the state and performs the real `return` (with the saved values).

## `defer` in the loop body

A `defer` written in the body of a `range`-over-func loop belongs to the **outer function**, as in any other loop — even though technically it executes inside a closure. The runtime has special support for this (`deferrangefunc`), so the `defer` is attached to the correct frame.

## Run-time checks

Because the iterator is code anyone can write, the compiler adds checks around `yield`. Violations panic with explicit messages:

```go,panic
package main

import "fmt"

// Bad ignores yield's result.
func Bad(yield func(int) bool) {
	yield(1)
	yield(2) // called after the loop did a break
}

func main() {
	for x := range Bad {
		fmt.Println(x)
		break
	}
	// panic: runtime error: range function continued iteration after function for loop body returned false
}
```

Other detected situations: calling `yield` after the iterator returned, or an iterator that "swallows" a panic from the loop body and keeps going.

## Performance

A push iterator is a function call per element, through a closure — in theory slower than a direct loop. In practice, when the iterator and the body are small enough, the compiler **inlines** both, and the final code is equivalent to a hand-written loop. Recursive iterators (like the Middle level's tree) or those going through interfaces can't be fully inlined, but they're still far faster than the channel-and-goroutine alternative:

```text
// on an ordinary laptop, walking 1,000,000 elements
BenchmarkSliceLoop     ~ 0.3 ms
BenchmarkIterSeq       ~ 0.3 ms   // inlined: identical
BenchmarkChannel       ~ 60  ms   // one synchronization per element
```

The numbers vary by machine; the ratio is what matters. Check for your own code with `go test -bench`.

## `iter.Pull` and coroutines

Turning a push iterator into a pull one requires **suspending** the iterator between two `next()` calls. Without runtime support, the only way would be a goroutine and a channel, paying for a trip through the scheduler on every value.

`iter.Pull` uses **coroutines** instead (`runtime.coro`): the iterator runs on its own goroutine, but control transfers between it and the caller with `coroswitch`, a direct switch that **bypasses the scheduler**. Only one of the two runs at any time, so there's no parallelism and no need for synchronization. A switch costs tens of nanoseconds, several times less than a channel send.

Practical consequences:

- `next` and `stop` must not be called concurrently from several goroutines;
- a forgotten `stop()` leaves the iterator's goroutine blocked forever (a leak);
- if the iterator panics, the panic propagates to the goroutine that called `next`.

## Single-use iterators

`iter.Seq` doesn't say whether the iterator can be walked more than once. Those built over collections (`slices.Values`, `maps.Keys`) can be restarted; those built over a stream (lines read from an `io.Reader`, the results of an SQL query) usually can't. The standard library's convention is to document single-use iterators explicitly.

## Errors in iterators

An `iter.Seq[V]` has nowhere to return an error. Two common patterns:

```go,ignore
// 1. Seq2 with an error per element
func Rows(db *sql.DB) iter.Seq2[Row, error]

// 2. an iterator + an Err() method, like bufio.Scanner
it := NewReader(r)
for v := range it.All() { ... }
if err := it.Err(); err != nil { ... }
```

## Summary

- The loop body becomes a `yield` closure; `break` = `return false`.
- `return` and outward labels are handled through a state variable.
- Protocol violations panic with explicit messages.
- Small iterators are inlined and reach plain-loop speed.
- `iter.Pull` uses runtime coroutines, bypassing the scheduler; always call `stop`.
