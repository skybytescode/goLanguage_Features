# Channels — Senior

The runtime's `hchan` struct, exactly what happens on a send and a receive, the guarantees in Go's memory model, and how to choose between channels and mutexes.

## The internal structure: `hchan`

`make(chan T, n)` allocates a `runtime.hchan` struct on the heap; a channel variable is just a pointer to it (which is why channels can be copied and compared).

```go,ignore
type hchan struct {
	qcount   uint           // how many elements are in the buffer now
	dataqsiz uint           // buffer capacity (n)
	buf      unsafe.Pointer // circular buffer of dataqsiz elements
	elemsize uint16
	closed   uint32
	elemtype *_type
	sendx    uint  // where the next send writes
	recvx    uint  // where the next receive reads
	recvq    waitq // goroutines blocked receiving (a list of sudogs)
	sendq    waitq // goroutines blocked sending
	lock     mutex // protects all the fields above
}
```

A `sudog` represents a goroutine waiting on a channel: a pointer to the G, a pointer to the value to send or to where the received value must be written, plus list links. A goroutine blocked in a `select` over 5 channels has 5 `sudog`s, one in each queue.

## A send, step by step

`ch <- v` calls `runtime.chansend`:

1. `nil` channel → the goroutine parks forever.
2. Take `lock`. If closed → unlock and **panic**.
3. If there's a receiver in `recvq`: dequeue it and **copy the value directly onto the receiver's stack** (bypassing the buffer), then mark it runnable (`goready`). This is the only place in the runtime where one goroutine writes to another's stack.
4. Otherwise, if the buffer has room: copy the value into `buf[sendx]`, advance `sendx`, increment `qcount`.
5. Otherwise: create a `sudog`, put it in `sendq` and **park** the goroutine (`gopark`), releasing the lock. A receiver will wake it, taking the value straight from the `sudog`.

Receiving (`chanrecv`) is symmetric. One detail: when the buffer is full and senders are blocked, the receiver takes the element at the head of the buffer and moves the first blocked sender's value to the tail of the buffer, so FIFO order is preserved.

There's also a lock-free "fast path": a non-blocking receive from an empty, open channel (for example a `select` with `default`) can decide by atomically reading just `qcount` and `closed`.

## `close`

`closechan` takes the lock, sets `closed`, then wakes **every** goroutine in `recvq` (they get the zero value and `ok == false`) and in `sendq` (those will panic). That's why closing is a cheap broadcast, and senders blocked on a channel someone else closed are a serious bug.

## What it costs

An operation that doesn't wait costs a lock, a copy and an unlock: tens of nanoseconds. One that blocks adds parking and waking a goroutine through the scheduler: hundreds of nanoseconds, plus a possible context switch to another P. Under heavy contention (many senders on one channel), the lock becomes the hot spot.

For large elements, the value is copied twice when it goes through the buffer (sender → buffer → receiver). Send pointers or small structs.

## The memory model: what a channel guarantees

Go's memory model defines the **happens-before** relation. For channels:

1. A send **happens before** the corresponding receive completes.
2. Closing a channel happens before a receive that returns the zero value because of the close.
3. For an **unbuffered** channel, the receive happens before the send completes.
4. For a channel with capacity `C`, the `k`-th receive happens before the `k+C`-th send completes.

The consequence of rule 1: everything the sender wrote **before** sending is visible to the receiver **after** receiving, with no other synchronization:

```go
package main

import "fmt"

var config map[string]string

func main() {
	ready := make(chan struct{})
	go func() {
		config = map[string]string{"env": "prod"} // written before close
		close(ready)
	}()
	<-ready
	fmt.Println(config["env"]) // guaranteed "prod": no data race
}
```

Rule 4 is the basis of the buffered-channel semaphore: at most `C` goroutines can be between "send" and "receive" at once.

## `len` and `cap` are only hints

`len(ch)` reads `qcount` without the lock. The value can be stale right after you read it. Don't make synchronization decisions on it (`if len(ch) > 0 { <-ch }` can block); use a `select` with `default`.

## Channels or a mutex?

| Channels, when… | A mutex, when… |
|---|---|
| you transfer **ownership** of data between goroutines | you protect a struct's internal **state** |
| you coordinate stages (pipeline, worker pool) | the operation is short: read/update a field |
| you signal events (ready, stop, timeout) | you need performance under heavy contention |
| you combine with `select` for timeouts and cancellation | the logic is clearer as a critical section |

A cache or counter guarded by a channel and an "owner" goroutine is usually slower and more complicated than the same thing with a `sync.Mutex`. The proverb "share memory by communicating" is about architecture, not a ban on mutexes.

## Summary

- `hchan`: a circular buffer, two `sudog` queues, a lock.
- Sending to a waiting receiver writes directly onto its stack.
- `close` wakes every waiter; blocked senders will panic.
- A send happens before the matching receive: data written earlier becomes visible.
- Channels for transfer and coordination, a mutex for shared state.
