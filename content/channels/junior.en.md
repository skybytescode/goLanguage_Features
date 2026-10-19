# Channels — Junior

A **channel** is a "pipe" goroutines use to send each other values. Besides moving data, a channel **synchronizes**: sending and receiving can wait for one another.

## Creating, sending and receiving

```go
package main

import "fmt"

func main() {
	ch := make(chan string) // a channel of strings

	go func() {
		ch <- "hello from a goroutine" // send
	}()

	msg := <-ch // receive (waits until a value arrives)
	fmt.Println(msg)
}
```

- `ch <- v` sends the value `v`.
- `v := <-ch` receives a value.
- The channel's type (`chan string`) fixes the type of the values.

## Unbuffered channel: a rendezvous

A channel created with `make(chan T)` has no buffer. The sender **waits** until a receiver takes the value, and the receiver waits until a value arrives. It's a meeting point between two goroutines:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	done := make(chan bool)

	go func() {
		fmt.Println("working...")
		time.Sleep(50 * time.Millisecond)
		fmt.Println("finished")
		done <- true
	}()

	<-done // main waits for the signal
	fmt.Println("main goes on")
}
```

## Buffered channel

`make(chan T, n)` creates a channel with room for `n` values. Sending blocks only when the buffer is **full**; receiving, only when it's **empty**:

```go
package main

import "fmt"

func main() {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3                       // doesn't block yet: the buffer has 3 slots
	fmt.Println(len(ch), cap(ch)) // 3 3

	fmt.Println(<-ch, <-ch, <-ch) // 1 2 3: order is preserved (FIFO)
}
```

## Closing and `range`

The sender can **close** the channel with `close` to announce that no more values are coming. `range` over a channel reads until it's closed:

```go
package main

import "fmt"

func generate(n int) chan int {
	ch := make(chan int)
	go func() {
		for i := 1; i <= n; i++ {
			ch <- i
		}
		close(ch) // done: nothing more to send
	}()
	return ch
}

func main() {
	for v := range generate(5) {
		fmt.Print(v, " ") // 1 2 3 4 5
	}
	fmt.Println()
}
```

## Was it closed? The comma-ok form

After closing, receiving no longer blocks: it immediately returns the zero value. The second result tells you whether the value is real:

```go
package main

import "fmt"

func main() {
	ch := make(chan int, 1)
	ch <- 42
	close(ch)

	v, ok := <-ch
	fmt.Println(v, ok) // 42 true: a real value, from the buffer
	v, ok = <-ch
	fmt.Println(v, ok) // 0 false: closed and empty
}
```

## Channel direction

In a function signature you can say a channel is send-only (`chan<-`) or receive-only (`<-chan`). The compiler checks correct use:

```go
package main

import "fmt"

func producer(out chan<- int) { // only sends
	for i := range 3 {
		out <- i
	}
	close(out)
}

func consumer(in <-chan int) { // only receives
	for v := range in {
		fmt.Println("received", v)
	}
}

func main() {
	ch := make(chan int)
	go producer(ch)
	consumer(ch)
}
```

## Deadlock

If every goroutine is waiting and none can make progress, the runtime stops the program:

```go,panic
package main

func main() {
	ch := make(chan int)
	ch <- 1 // nobody receives: main waits forever
	// fatal error: all goroutines are asleep - deadlock!
}
```

With an unbuffered channel, sending and receiving must happen in **different** goroutines.

## Summary

- `make(chan T)` unbuffered: send and receive wait for each other.
- `make(chan T, n)` buffered: blocks only when full or empty.
- `close(ch)` signals the end; `range ch` reads until closed.
- `v, ok := <-ch` tells you whether the channel was closed.
- `chan<-` and `<-chan` restrict the direction in signatures.
