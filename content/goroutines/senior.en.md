# Goroutines — Senior

How the runtime runs millions of goroutines on a handful of operating system threads: the G-M-P model, run queues and work stealing, system calls, the netpoller and preemption.

## The G-M-P model

| Element | What it is |
|---|---|
| **G** (goroutine) | the stack, saved registers, state; a few KB |
| **M** (machine) | an operating system thread |
| **P** (processor) | a "right to run Go code": the local G queue, the allocation cache (`mcache`), etc. |

There are exactly `GOMAXPROCS` Ps. An M must hold a P to execute Go code; there can be more Ms (those blocked in system calls don't hold a P). So at most `GOMAXPROCS` goroutines run Go code in parallel, however many threads exist.

```text
          ┌──── global queue ─────┐
          │  G  G  G              │
          └───────────────────────┘
   P0 [G G G G]   P1 [G G]   P2 [ ]  ← local queues (max 256)
    │              │          │
    M0             M1         M2 (looking for work / stealing)
```

## Queues and work stealing

A `go f()` puts the new G in the current P's **local** queue (more precisely in the `runnext` slot, so the new goroutine runs soon while the cache is warm). The local queue has 256 slots; on overflow, half of it moves to the **global** queue.

An idle M looks, in order, at: `runnext` and the local queue, the global queue (it also checks it periodically anyway, every 61 schedules, so Gs there don't starve), the netpoller, then **steals half** of a randomly chosen P's local queue. If it finds nothing, the M "spins" briefly, then goes to sleep.

## System calls: handoff

When a G makes a blocking system call (for example reading a file), its M blocks in the kernel. So the P doesn't sit idle, the runtime **hands it off** to another M, existing or new. When the call returns, the G tries to reacquire a P; if none is free, it goes on the global queue and the M parks.

That's why a program with many blocking file operations can have far more threads than `GOMAXPROCS` (the default limit is 10,000, adjustable with `debug.SetMaxThreads`).

## The netpoller: networking without blocked threads

Socket operations don't block Ms. Sockets are in non-blocking mode; when a read would block, the G is **parked** and the descriptor is registered with the netpoller (epoll on Linux, kqueue on BSD/macOS, IOCP on Windows). When data arrives, the netpoller marks the G runnable. That's how a Go server can hold 100,000 open connections on a few threads.

Your code sees an ordinary blocking call (`conn.Read`); the complexity is hidden in the runtime.

## Preemption

Up to Go 1.13, preemption was **cooperative**: a G could only be interrupted at function calls (in the stack-checking prologue). A tight loop with no calls could hog a P forever, including blocking the GC waiting to stop everyone.

Since Go 1.14, preemption is also **asynchronous**: the monitoring thread `sysmon` spots Gs running for more than 10 ms and sends them a signal (`SIGURG` on Unix); the signal handler stops the G at a safe point.

```go
package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(1)
	go func() {
		for { // a tight loop, no function calls
		}
	}()
	time.Sleep(50 * time.Millisecond)
	fmt.Println("main got the processor back") // printed since Go 1.14
}
```

## `sysmon`

A special thread, with no P, that runs periodically (between 20 µs and 10 ms): it preempts Gs running too long, retakes Ps stuck in system calls, polls the network if nobody has for a while, and forces a GC every 2 minutes.

## `GOMAXPROCS` in containers (Go 1.25)

The default used to be the machine's core count. In a container limited to 2 CPUs on a 64-core machine, that meant 64 Ps and severe throttling by the cgroup. Since Go 1.25, the runtime reads the CPU limit from the cgroup and adjusts `GOMAXPROCS` automatically, including when the limit changes at run time. Setting the `GOMAXPROCS` environment variable explicitly disables the adjustment.

## Pinned threads: `runtime.LockOSThread`

Some C libraries or system APIs (OpenGL, some Linux namespace calls) require calls to come from the **same** thread. `runtime.LockOSThread` binds the current G to its M until `UnlockOSThread`. If the goroutine exits without unlocking, the thread is destroyed, not reused.

## Observing the scheduler

```bash
GODEBUG=schedtrace=1000 ./app          # one line per second: Ps, Ms, queues
GODEBUG=schedtrace=1000,scheddetail=1 ./app
go test -trace=trace.out && go tool trace trace.out
```

`go tool trace` shows, on a timeline, which G ran on which P, blocking, the GC and system calls — the best tool for latency problems.

## Summary

- Gs scheduled onto Ms, through `GOMAXPROCS` Ps.
- 256-slot local queues, a global queue, stealing half of another P's queue.
- Blocking system calls hand off the P; networking uses the netpoller.
- Asynchronous, signal-based preemption since Go 1.14.
- Since Go 1.25, `GOMAXPROCS` respects container CPU limits.
