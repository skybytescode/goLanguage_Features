# Goroutines — Middle

Starting a goroutine is easy. The hard part is knowing **when it stops**, how many run at once, and what happens to its errors and panics. At this level: data races, goroutine leaks, limiting concurrency and collecting errors.

## Data races and `-race`

A **data race** happens when two goroutines access the same variable, at least one writes, and there's no synchronization. The result is undefined:

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	counter := 0
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			counter++ // RACE: unsynchronized read + write
		})
	}
	wg.Wait()
	fmt.Println(counter) // often less than 1000
}
```

The race detector finds them at run time:

```bash
go run -race .
go test -race ./...
```

Run your tests with `-race` in CI. The fixes are a `sync.Mutex`, an `atomic.Int64` (see [Sync and atomic](../../sync/middle/)) or channels.

## Goroutine leaks

A goroutine blocked forever (on a channel nobody writes to anymore, for example) isn't collected by the GC. It, and everything it references, stays in memory:

```go
package main

import (
	"fmt"
	"runtime"
	"time"
)

func findFirst(sources []string) string {
	res := make(chan string) // UNBUFFERED: that's the problem
	for _, s := range sources {
		go func() {
			res <- "answer from " + s // only the first is read; the rest stay blocked
		}()
	}
	return <-res
}

func main() {
	fmt.Println(findFirst([]string{"a", "b", "c"}))
	time.Sleep(10 * time.Millisecond)
	fmt.Println("active goroutines:", runtime.NumGoroutine()) // 3: main + 2 blocked
}
```

The fix here: `make(chan string, len(sources))`, so every send succeeds. In general: **every goroutine you start needs a clear way to finish**, usually via `context` (see [Context](../../context/middle/)).

## Limiting concurrency

Starting 10,000 goroutines that open connections at once can exhaust file descriptors or overload a server. A buffered channel works as a **semaphore**:

```go
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	const maxConcurrent = 3
	sem := make(chan struct{}, maxConcurrent)
	var active, peak atomic.Int32
	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			sem <- struct{}{}        // take a slot (blocks if there are 3)
			defer func() { <-sem }() // release the slot

			n := active.Add(1)
			for {
				m := peak.Load()
				if n <= m || peak.CompareAndSwap(m, n) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond) // "the work"
			active.Add(-1)
		})
	}
	wg.Wait()
	fmt.Println("peak concurrency:", peak.Load()) // 3
}
```

The classic alternative is a **worker pool**: a fixed number of goroutines reading tasks from a channel. A full example in [Channels](../../channels/middle/).

## Collecting errors

Goroutines don't return errors. A simple pattern: each one writes its result (or error) at its own position in a preallocated slice — different positions, so no race:

```go
package main

import (
	"errors"
	"fmt"
	"sync"
)

func process(id int) error {
	if id == 2 {
		return fmt.Errorf("task %d: failed", id)
	}
	return nil
}

func main() {
	tasks := []int{1, 2, 3, 4}
	errs := make([]error, len(tasks))
	var wg sync.WaitGroup
	for i, id := range tasks {
		wg.Go(func() {
			errs[i] = process(id) // each goroutine has its own index
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		fmt.Println("errors:", err)
	}
}
```

For "stop everything at the first error", the `golang.org/x/sync/errgroup` package combines a `WaitGroup`, the first error and cancellation through a `context`, plus concurrency limiting with `SetLimit`:

```go,ignore
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(8)
for _, url := range urls {
	g.Go(func() error {
		return download(ctx, url)
	})
}
if err := g.Wait(); err != nil { ... } // the first error; ctx is canceled for the others
```

## Panics in a goroutine

An unrecovered panic in a goroutine **kills the whole program**. Goroutines started by a server or a library need their own `recover` if you don't want that:

```go
package main

import (
	"fmt"
	"sync"
)

func safely(wg *sync.WaitGroup, f func()) {
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("goroutine recovered:", r)
			}
		}()
		f()
	})
}

func main() {
	var wg sync.WaitGroup
	safely(&wg, func() { panic("oops") })
	wg.Wait()
	fmt.Println("the program goes on")
}
```

## Summary

- Always run tests with `-race`.
- Every goroutine needs a way to finish; otherwise it's a leak.
- A buffered channel is a simple semaphore for limiting concurrency.
- Collect errors at separate positions or with `errgroup`.
- A panic in a goroutine kills the whole process.
