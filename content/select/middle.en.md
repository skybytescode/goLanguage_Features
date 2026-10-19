# Select — Middle

Real-world patterns: event loops with `select`, timers and tickers used correctly, `nil` channels to disable cases, prioritization and rate limiting.

## The event loop

A goroutine reacting to several sources (data, a timer, a stop signal) is almost always a `for` + `select`:

```go
package main

import (
	"context"
	"fmt"
	"time"
)

func aggregator(ctx context.Context, in <-chan int) {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	sum := 0
	for {
		select {
		case v := <-in:
			sum += v
		case <-ticker.C:
			fmt.Println("periodic report, sum =", sum)
		case <-ctx.Done():
			fmt.Println("stopping:", ctx.Err())
			return
		}
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()

	in := make(chan int)
	go func() {
		for i := 1; ; i++ {
			select {
			case in <- i:
				time.Sleep(5 * time.Millisecond)
			case <-ctx.Done():
				return
			}
		}
	}()
	aggregator(ctx, in)
}
```

Notice that the **sender** listens to `ctx.Done()` too. Without it, once the aggregator stops, the sender would stay blocked forever on `in <- i`.

## Timers: `time.After` vs. `time.NewTimer`

`time.After` creates a new timer on every call. In a loop, each iteration makes a new one. Since Go 1.23, unstopped and unreferenced timers are collected by the GC, so it's no longer a memory leak, but it's still an allocation per iteration. For an **inactivity** timeout, a single reset timer is clearer:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	events := make(chan string)
	go func() {
		for _, e := range []string{"a", "b", "c"} {
			events <- e
			time.Sleep(10 * time.Millisecond)
		}
	}()

	const idle = 50 * time.Millisecond
	timer := time.NewTimer(idle)
	defer timer.Stop()

	for {
		select {
		case e := <-events:
			fmt.Println("event", e)
			timer.Reset(idle) // Go 1.23+: Reset is safe without draining the channel
		case <-timer.C:
			fmt.Println("no activity for", idle)
			return
		}
	}
}
```

Before Go 1.23, `Reset` on a timer that had already fired required manually draining `timer.C`, otherwise the loop could receive a stale value. Since 1.23, the timer's channel is synchronous and `Reset`/`Stop` guarantee no stale value arrives (the behavior depends on the `go` version in `go.mod`).

## Disabling a case with a `nil` channel

Operations on a `nil` channel block forever, so a `case` with a `nil` channel **is never chosen**. You can "turn it off" by setting the variable to `nil`. The classic pattern: merging two channels until both are closed:

```go
package main

import "fmt"

func source(vals ...int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for _, v := range vals {
			ch <- v
		}
	}()
	return ch
}

func main() {
	a, b := source(1, 2, 3), source(10, 20)
	sum := 0
	for a != nil || b != nil {
		select {
		case v, ok := <-a:
			if !ok {
				a = nil // closed: stop choosing this case
				continue
			}
			sum += v
		case v, ok := <-b:
			if !ok {
				b = nil
				continue
			}
			sum += v
		}
	}
	fmt.Println(sum) // 36
}
```

Without `a = nil`, a closed channel would be "always ready" (it returns zero immediately), and the loop would spin uselessly.

## Priority

`select` picks randomly among ready cases; there's no priority. If stopping must take precedence, check it separately first:

```go,ignore
for {
	select {
	case <-ctx.Done(): // priority check, non-blocking
		return
	default:
	}

	select {
	case <-ctx.Done():
		return
	case job := <-jobs:
		process(job)
	}
}
```

## Rate limiting with a ticker

A `time.Ticker` delivers a "token" at a fixed interval. Waiting for the token before each operation keeps you under a given rate:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	requests := []string{"a", "b", "c", "d"}
	limiter := time.NewTicker(10 * time.Millisecond) // at most 100 requests/second
	defer limiter.Stop()

	start := time.Now()
	for _, r := range requests {
		<-limiter.C
		fmt.Printf("%s at %v\n", r, time.Since(start).Round(10*time.Millisecond))
	}
}
```

For limiting with "bursts", use `golang.org/x/time/rate`.

## Sending with a timeout

The same pattern works for sending too, so a producer doesn't block forever on a slow consumer:

```go
package main

import (
	"errors"
	"fmt"
	"time"
)

var ErrFull = errors.New("queue full")

func send(ch chan<- int, v int, wait time.Duration) error {
	select {
	case ch <- v:
		return nil
	case <-time.After(wait):
		return ErrFull
	}
}

func main() {
	queue := make(chan int, 1)
	fmt.Println(send(queue, 1, 10*time.Millisecond)) // <nil>
	fmt.Println(send(queue, 2, 10*time.Millisecond)) // queue full
}
```

## Summary

- `for` + `select` with `ctx.Done()` is the standard event loop.
- Both ends (sender and receiver) must listen for the stop signal.
- A reset timer for inactivity timeouts; `Reset` is safe since Go 1.23.
- A `nil` channel disables a case.
- `select` has no priority; check urgent things separately.
