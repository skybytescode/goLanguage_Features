# Context — Senior

How the `context` package is implemented: the concrete types behind the interface, how cancellation propagates through the tree, what `Value` costs, how foreign contexts are integrated, and how to implement your own context type.

## The interface

```go,ignore
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}
```

Every `With...` function returns a concrete type that wraps the parent:

| Constructor | Internal type | What it adds |
|---|---|---|
| `Background()`, `TODO()` | `backgroundCtx`, `todoCtx` | nothing: `Done()` returns `nil` |
| `WithCancel`, `WithCancelCause` | `*cancelCtx` | a `done` channel, the list of children, the error |
| `WithTimeout`, `WithDeadline` | `*timerCtx` | a `cancelCtx` + a `time.Timer` |
| `WithValue` | `*valueCtx` | a key/value pair |
| `WithoutCancel` | `withoutCancelCtx` | hides the parent's cancellation |
| `AfterFunc` | `*afterFuncCtx` | a function to run on cancellation |

## `cancelCtx` and propagation

```go,ignore
type cancelCtx struct {
	Context                        // the parent
	mu       sync.Mutex
	done     atomic.Value          // chan struct{}, created lazily
	children map[canceler]struct{} // children to cancel along with this one
	err      atomic.Value
	cause    error
}
```

When a `cancelCtx` is created, `propagateCancel` links it to the parent:

1. If the parent can never be canceled (`Done() == nil`, like `Background`), do nothing.
2. If the parent is already canceled, the child is canceled immediately.
3. If the parent is (or wraps) a `*cancelCtx` from the `context` package, the child is added directly to `children` — no goroutines.
4. If the parent implements `AfterFunc` (Go 1.21+), that mechanism is used.
5. Otherwise — a **foreign** context type, written by someone else — the package starts a **goroutine** that waits on `parent.Done()` and cancels the child. So each context derived from a foreign context costs a goroutine until cancellation.

On `cancel()`: the error is set, `done` is closed, every child is canceled recursively, and the context is removed from the parent's `children` (otherwise the parent would keep references to every canceled child — a memory leak; that's why `cancel` must **always** be called).

## `Done()` is created lazily

The `done` channel is only created on the first `Done()` call. If the context is canceled before anyone asks for it, a global, already-closed channel (`closedchan`) is used, with no allocation. Contexts nobody listens to never allocate a channel.

## `timerCtx`

`WithTimeout(d)` is just `WithDeadline(time.Now().Add(d))`. If the parent's deadline is already earlier, no timer is created: a plain `cancelCtx` is returned (the effective deadline is the parent's anyway). Otherwise, a `time.AfterFunc` calls `cancel` with `DeadlineExceeded` on expiry. `cancel()` stops the timer, which is why you must call it even when the operation finishes early: otherwise the timer would keep the context alive until it fires.

## `Value` is a linked list

`WithValue` doesn't create a map: each `valueCtx` holds **a single** pair and a pointer to its parent. `ctx.Value(k)` walks up the chain until it finds the key or reaches the root:

```go
package main

import (
	"context"
	"fmt"
)

type key int

func main() {
	ctx := context.Background()
	for i := range 100 {
		ctx = context.WithValue(ctx, key(i), i)
	}
	// looking up key 0 walks 100 levels: O(depth)
	fmt.Println(ctx.Value(key(0)))
}
```

In practice, chains are a few levels deep and the cost is negligible. But it explains why context is not a good general "bag" of dependencies: lookups are linear, untyped and invisible in signatures.

Keys are compared with `==` on interfaces, so the key's type matters: `key(0)` and `int(0)` are different keys. For the same reason, the key must not be of a non-comparable type (`WithValue` panics).

## Implementing your own context type

Rarely needed, but sometimes useful (for example a context tied to an object's lifecycle). Embed an existing context and override only what you need. If your type exposes cancellation through something other than an internal `cancelCtx`, implement `AfterFunc` too, so derived contexts don't need a goroutine.

```go
package main

import (
	"context"
	"fmt"
)

// labeledCtx adds a method, keeping all of the parent's behavior.
type labeledCtx struct {
	context.Context
	label string
}

func (c labeledCtx) String() string { return "ctx[" + c.label + "]" }

func main() {
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := labeledCtx{base, "daily-import"}

	child, cancelChild := context.WithCancel(ctx) // the parent is wrapped: still no goroutine
	defer cancelChild()
	cancel()
	<-child.Done()
	fmt.Println(ctx, child.Err())
}
```

Because `labeledCtx` embeds a `*cancelCtx`, `propagateCancel` recognizes it (via `parent.Value(&cancelCtxKey)`) and links the child directly, with no goroutine.

## Context in `net/http`

The HTTP server creates one context per connection and one per request, canceled when the client closes the connection, when the `ServeHTTP` handler returns, or on `Server.Shutdown`. With HTTP/2, canceling a stream cancels only that request's context. Work started from a handler that must continue after the response needs `context.WithoutCancel(r.Context())`.

## Summary

- Each `With...` adds a layer: `cancelCtx`, `timerCtx`, `valueCtx`.
- Cancellation propagates through the `children` map, with no goroutines for standard types.
- Foreign contexts cost one goroutine per derived context, unless they implement `AfterFunc`.
- `Done()` is created lazily; `cancel()` releases the timer and the link to the parent.
- `Value` is a linear search through a linked list.
