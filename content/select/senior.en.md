# Select — Senior

How the runtime implements `select` (the `selectgo` function), where the random choice comes from, how it avoids deadlocks between locks, how the compiler optimizes simple cases, and how to build a `select` with a variable number of cases.

## What the compiler does before the runtime

Not every `select` reaches `selectgo`:

| Form | What the compiler generates |
|---|---|
| `select {}` | `block()`: parks the goroutine forever |
| a single `case`, no `default` | an ordinary channel operation (`chansend`/`chanrecv`) |
| one `case` + `default` | a **non-blocking** operation (`selectnbsend`/`selectnbrecv`) |
| everything else | `runtime.selectgo` |

That's why the `select { case v := <-ch: ...; default: ... }` pattern is very cheap: a single call, often lock-free (the fast path atomically checks whether the channel is empty).

## `selectgo`, step by step

1. **Poll order (`pollorder`)**: a **random** permutation of the cases (Fisher–Yates with the runtime's fast generator). That's where fairness comes from: among cases ready at the same time, each has an equal chance.
2. **Lock order (`lockorder`)**: the cases sorted by channel **address** (heap sort). Every channel involved is locked in this global order, which rules out deadlocks between two `select`s locking the same channels in different orders. A channel appearing several times is locked once.
3. **Pass 1**: in `pollorder`, look for a ready case (a waiting receiver/sender, a buffer with room/data, a closed channel). The first one found is executed; everything is unlocked and we're done.
4. If none is ready and there's a `default`: unlock and run `default`.
5. **Pass 2**: create a `sudog` for **every** case and enqueue it on that channel's queue; then park the goroutine (`gopark`), releasing the locks.
6. When another goroutine operates on one of the channels, it finds the `sudog`, completes the transfer and wakes the goroutine. An atomic field (`selectDone`) ensures only **one** case can "win", even if two channels become ready simultaneously.
7. **Pass 3**: after waking, the goroutine locks all the channels again and removes its `sudog`s from every other queue.

So the cost is **linear in the number of cases**, three times over: a `select` with 2–4 cases is cheap, one with 50 cases does a lot of work on every execution.

## Fairness has a limit

The choice is uniform **only among cases ready at the same moment**. A case that's always ready (say a buffered channel that's always full) is chosen about `1/n` of the time; if one source produces much faster than another, it dominates. Fairness between producers is solved in the architecture (separate queues, limiting), not in `select`.

## A variable number of cases: `reflect.Select`

The number of cases in a `select` is fixed at compile time. For N channels known only at run time there's `reflect.Select`:

```go
package main

import (
	"fmt"
	"reflect"
)

func main() {
	sources := make([]chan int, 3)
	cases := make([]reflect.SelectCase, len(sources))
	for i := range sources {
		sources[i] = make(chan int, 1)
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sources[i])}
	}
	sources[1] <- 42

	chosen, value, ok := reflect.Select(cases)
	fmt.Println(chosen, value.Int(), ok) // 1 42 true
}
```

It's several times slower than a static `select` (reflection and allocations). For many sources, a fan-in into a single channel (see [Channels](../../channels/middle/)) is usually better.

## Timers in Go 1.23

The timer implementation was rewritten in Go 1.23, with direct effects on `select`:

- a timer's channel (`timer.C`, `time.After`) is now **synchronous** (unbuffered, as far as the program can tell): no value "lingers" in the channel after `Stop` or `Reset`;
- timers are tied to their channel, and an unstopped timer with no remaining references is **collected by the GC**; `time.After` in a loop no longer leaks memory.

The new behavior applies to modules with `go 1.23` or newer in `go.mod`; `GODEBUG=asynctimerchan=1` reverts to the old one.

## Interaction with the scheduler

A goroutine parked in a `select` uses no CPU and holds no P. If every goroutine is parked on channels and no timer or network event can happen anymore, the runtime detects the **global deadlock** and stops the program. Detection doesn't see partial blockage (some goroutines stuck, others still running): those show up as leaks and are found with `pprof`'s `goroutine` profile.

## Summary

- The compiler turns single-case (+`default`) `select`s into plain operations.
- `selectgo`: a random permutation for polling, address order for locking.
- The cost is linear in the number of cases; exactly one case wins, atomically.
- Fairness only applies among cases ready at the same time.
- `reflect.Select` for dynamic N; since Go 1.23, timers are synchronous and collectable.
