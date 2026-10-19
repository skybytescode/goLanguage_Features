# Runtime: scheduler, memory, GC — Senior

The memory allocator (size classes, spans, `mcache`/`mcentral`/`mheap`), the GC algorithm (concurrent tricolor marking, write barriers, the pacer, assists), the Green Tea GC that became the default in Go 1.26, and how memory goes back to the operating system.

## The allocator: inspired by TCMalloc

Objects are split by size:

| Category | Size | How it's allocated |
|---|---|---|
| **tiny** | < 16 B, pointer-free | several small objects packed into a 16 B block |
| **small** | ≤ 32 KiB | rounded up to one of the **68 size classes**, from a span dedicated to that class |
| **large** | > 32 KiB | its own span, straight from the heap |

A **span** is a run of 8 KiB pages holding objects of a single class. The hierarchy, fast to slow:

1. **`mcache`**, per P: a current span for each class (and with/without-pointers variant). Allocation is a simple free-bit lookup, **lock-free** (the P is owned exclusively).
2. **`mcentral`**, per class: lists of partially free spans. When the `mcache` span fills up, the P asks for another one here (with a lock).
3. **`mheap`**: the free pages, organized with a radix tree; it allocates new spans and requests memory from the operating system (`mmap`) in 64 MiB chunks ("arenas") on 64-bit.

The **noscan** (pointer-free) / **scan** split matters: noscan spans aren't walked at all during marking. A 1 MiB `[]byte` costs nothing to mark; a `[]*T` of the same size means 131,072 pointers to follow.

## The GC: concurrent tricolor marking

Go's GC is **mark-sweep**, **concurrent**, **non-generational** and **non-moving** (objects aren't moved, so pointers stay stable, which simplifies cgo and `unsafe`).

The phases of a cycle:

1. **Sweep termination** (a short STW): finishes sweeping the previous cycle.
2. **Mark** (concurrent): turns on the write barrier; starts marking from the roots (stacks, globals) and propagates through object graphs. 25% of `GOMAXPROCS` is dedicated to the GC; the rest of the program keeps running.
3. **Mark termination** (a short STW): checks there's nothing left to mark.
4. **Sweep** (concurrent, lazy): spans are swept as they're reused for allocation; unmarked objects become free.

The tricolor invariant: **white** = not yet visited (a candidate for freeing), **grey** = visited but with unexamined references, **black** = fully visited. A black object must not point to a white one that nothing grey references anymore. The **hybrid write barrier** (Yuasa + Dijkstra, since Go 1.8) keeps the invariant: when a pointer is written during marking, it shades both the old and the new value grey. The upside: stacks don't need rescanning at the end, so STW pauses don't grow with the number of goroutines.

## The pacer and mark assist

The **pacer** picks when to start a cycle so that marking finishes exactly as the heap reaches its target (`GOGC`, `GOMEMLIMIT`). If the program allocates faster than the GC can mark, the allocating goroutine is forced to do **mark assist**: an amount of marking work proportional to what it allocates. That's the backpressure mechanism — and a source of unexpected latency in profiles (`gcAssistAlloc` in a flame graph means allocations are too fast for the GC).

## Green Tea (the default in Go 1.26)

Classic marking follows pointers object by object, jumping around memory: each step is likely a cache miss. **Green Tea** (an experiment in Go 1.25, the default in Go 1.26) changes the unit of work: the mark queue holds **spans** (pages), not individual objects. When a span is scanned, all the marked objects in it are processed together, with sequential memory access, and with vector instructions on modern CPUs. The announced result: a 10–40% reduction in GC CPU cost for GC-heavy programs, with no code changes.

It can be disabled temporarily at build time with `GOEXPERIMENT=nogreenteagc` if regressions show up.

## Giving memory back to the operating system

Memory freed by the GC stays in Go's heap for future allocations. A background **scavenger** gradually returns it to the operating system (`madvise`), trying to keep resident memory close to what's needed (and under `GOMEMLIMIT`). On Linux, `MADV_DONTNEED` is used by default, so RSS drops right away in `top`. `debug.FreeOSMemory()` forces a GC and an immediate return.

Fragmentation is bounded by the size classes: the maximum internal waste per class is ~12.5%.

## What Go's GC doesn't do

- **It isn't generational**: young objects aren't treated separately. Escape analysis makes up for much of this: most short-lived objects never reach the heap.
- **It doesn't compact**: objects aren't moved; size classes keep fragmentation in check.
- **Its pauses aren't proportional to the heap**: STW pauses depend on the number of Ps and active goroutines, not on heap size.

## Diagnosis

```bash
GODEBUG=gctrace=1 ./app                          # every cycle
go tool pprof http://host/debug/pprof/heap      # who holds memory (inuse_space)
go tool pprof -sample_index=alloc_space ...     # who allocates the most
go tool trace trace.out                          # GC, assists and pauses on a timeline
```

## Summary

- Lock-free allocation from the per-P `mcache`; 68 size classes; a tiny allocator under 16 B.
- Pointer-free (noscan) objects are free to mark.
- A concurrent, tricolor GC with a hybrid write barrier; sub-millisecond STW pauses.
- The pacer + mark assist keep the heap on target; assists show up as latency.
- Go 1.26: Green Tea, span-based marking, friendlier to the cache.
