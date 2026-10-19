# Sync and atomic — Senior

How `sync.Mutex` is implemented (normal mode and starvation mode), what the memory model guarantees for atomic operations, how the per-P `sync.Pool` with its "victim cache" works, and what false sharing is.

## `sync.Mutex` under the hood

A mutex has just two fields: `state int32` and `sema uint32`. The bits of `state` encode: locked, woken (a goroutine has already been woken), starving, plus the number of waiting goroutines.

**The fast path**: `Lock` tries a single `CompareAndSwap(0, locked)`. With no contention, that's it — a few nanoseconds, inlined.

**The slow path**:

1. **Spinning**: if the mutex is held and conditions are right (several cores, other active Ps, an empty local run queue), the goroutine spins for a few iterations, hoping the holder releases soon. It avoids the cost of parking.
2. **Parking**: otherwise, the goroutine registers as a waiter and sleeps on the runtime semaphore (`runtime_SemacquireMutex`), implemented as a balanced tree (treap) of queues keyed by address.

## Normal mode vs. starvation mode

In **normal mode**, a woken goroutine competes for the lock with newly arrived goroutines that are already on a CPU. The newcomers usually win, which is good for throughput (no context switch), but can leave one goroutine waiting a very long time.

If a waiter fails to get the lock for **more than 1 ms**, the mutex switches to **starvation mode**: on `Unlock`, ownership is handed **directly** to the first in line; newcomers don't try to grab it and queue up instead. The mutex returns to normal mode when the last waiter gets it, or when one waited less than 1 ms.

The trade-off: normal mode gives throughput, starvation mode gives predictable tail latency.

## `RWMutex`: writer preference

When a writer calls `Lock`, it announces its intent (it subtracts a large constant from `readerCount`); **new readers** block immediately, even while other readers are still reading. The writer only waits for existing readers. This prevents writer starvation, but has a consequence: **recursive `RLock` can deadlock**:

```go,ignore
func (c *Cache) A() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.B() // if a writer requested Lock in between, the RLock in B blocks forever
}

func (c *Cache) B() {
	c.mu.RLock()
	defer c.mu.RUnlock()
}
```

## The memory model and `atomic`

Since 2022, Go's memory model states explicitly: operations in `sync/atomic` are **sequentially consistent** — they behave as if executed in a single global order consistent with each goroutine's order. If the effect of atomic operation A is observed by atomic operation B, then A happens before B.

This means an atomic flag can "publish" data written normally:

```go
package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

var (
	data  []int       // written normally
	ready atomic.Bool // publication
)

func main() {
	go func() {
		data = []int{1, 2, 3}
		ready.Store(true) // the write to data happens before
	}()
	for !ready.Load() {
		time.Sleep(time.Millisecond)
	}
	fmt.Println(data) // guaranteed [1 2 3]
}
```

Go doesn't expose weaker orderings (acquire/release, relaxed) like C++ or Rust; the simplicity costs little on x86 (where loads are ordered anyway) and somewhat more on ARM.

The `atomic.Int64` & co. types also guarantee **64-bit alignment** on 32-bit platforms — the reason they're preferred over the old `atomic.AddInt64(&x, 1)` functions.

## `sync.Pool`: per-P caches and the victim cache

Each P has its own `poolLocal`: a `private` slot (no synchronization at all) and a lock-free `shared` queue (a dequeue). `Get` tries, in order: `private`, the head of its own queue, **stealing** from other Ps' queues, the victim cache, `New`.

On every GC, the pool's contents move to the **victim cache**, and the previous victim cache is dropped. An object therefore survives at least one GC cycle, which removed the performance cliff from before Go 1.13, when the pool was emptied completely on every GC.

Consequence: a `sync.Pool` isn't suitable for resources that must be closed (connections, files): objects can vanish without notice.

## `sync.Map` (Go 1.24)

The original implementation used two maps (a read-only one read atomically, a "dirty" one guarded by a mutex), promoting the second after enough misses. Since Go 1.24, `sync.Map` is built on a **concurrent hash-trie** (`HashTrieMap`, the same one the `unique` package uses), with far less contention on writes to disjoint keys and no costly promotions.

## False sharing

CPUs synchronize memory at **cache line** granularity (usually 64 bytes). Two independent atomic variables in the same line, modified from different cores, "fight" over the line even though they're logically unrelated:

```go,ignore
type Counters struct {
	a atomic.Int64 // the same 64 bytes...
	b atomic.Int64 // ...: every Add invalidates the line for the other core
}

type SeparateCounters struct {
	a atomic.Int64
	_ [56]byte // padding: b lands on a different cache line
	b atomic.Int64
}
```

The runtime uses exactly this padding (`cpu.CacheLinePad`) for per-P structures. In application code, it's only worth it once a profile shows contention on such fields.

## Sharding

When a single mutex becomes the hot spot, split the data into N shards, each with its own mutex, chosen by the key's hash. Contention drops roughly N-fold, at the cost of more complicated global operations (counting, iterating).

## Summary

- `Mutex`: CAS on the fast path, spinning, then parking on a semaphore.
- Starvation mode after 1 ms of waiting: direct handoff, predictable latency.
- `RWMutex` prefers writers; recursive `RLock` can deadlock.
- Atomics are sequentially consistent and can publish data.
- `sync.Pool` is per-P, with a victim cache; beware of false sharing.
