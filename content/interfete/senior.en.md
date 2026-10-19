# Interfaces — Senior

What an interface looks like in memory, what an `itab` is, what a dynamic call and a conversion to an interface cost, and when comparing two interfaces panics.

## Two representations: `eface` and `iface`

The runtime has two structs, two words each:

```go,ignore
// the empty interface (any)
type eface struct {
	_type *_type         // describes the dynamic type
	data  unsafe.Pointer // pointer to the value
}

// an interface with methods
type iface struct {
	tab  *itab // dynamic type + method table
	data unsafe.Pointer
}

type itab struct {
	inter *interfacetype // the interface
	_type *_type         // the concrete type
	hash  uint32         // copy of the type's hash, for type switches
	fun   [1]uintptr     // method addresses; actually variable length
}
```

An `itab` is computed **once** per (interface, concrete type) pair. Pairs known at compile time are generated statically; those discovered at run time (for example via a type assertion to an interface) are computed and kept in a global hash table.

The `data` field is always a pointer. What happens to values that aren't pointers is the subject of the next section.

## Converting to an interface can allocate

When you put a non-pointer value in an interface, the runtime needs a pointer to a copy of it. Usually that means a **heap allocation** (`runtime.convT64`, `convTstring`, etc.). Exceptions, with no allocation:

- pointer values (the pointer goes straight into `data`);
- zero-size values;
- single-byte integers and small integers (0–255), which point into a static runtime table;
- constants and values the compiler can place in read-only memory;
- cases where escape analysis proves the interface doesn't escape (the copy then lives on the stack).

```go
package main

import (
	"fmt"
	"testing"
)

var (
	sink       any
	small, big = 200, 100000 // variables, not constants
)

func main() {
	smallAllocs := testing.AllocsPerRun(1000, func() { sink = small })
	bigAllocs := testing.AllocsPerRun(1000, func() { sink = big })
	fmt.Println(smallAllocs, bigAllocs) // 0 1
}
```

In hot loops, `fmt.Println(x)` and logging with `any` arguments can produce many small allocations precisely because of this.

## The cost of an interface call

A call `r.Read(p)` through an interface means: load the `itab`, load the address from `fun[i]`, make an indirect call. On modern CPUs with good prediction, the difference from a direct call is nanoseconds. The real cost is elsewhere: an indirect call **can't be inlined**, so the compiler also loses the optimizations that would follow (including escape analysis: arguments to an interface call are generally assumed to escape).

The compiler **devirtualizes** the call when it knows the concrete type (for example right after `var r io.Reader = &buf`), and with PGO it does so based on the profile too: it emits `if type == *bytes.Buffer { direct call } else { indirect call }`.

## Type assertions and type switches

`x.(T)` with a concrete `T` only compares a type (or `itab`) pointer — very cheap. `x.(I)` with an interface `I` must find an `itab`, which involves the global hash table (with a per-call-site cache since Go 1.22). A type switch uses the `hash` field in the `itab` to jump quickly to the right case; with many concrete cases, the compiler emits a binary search on the hash.

## Comparing interfaces can panic

Two interfaces are equal if they have the same dynamic type and equal values. If the dynamic type **isn't comparable** (slice, map, function), the comparison **panics at run time**, even though everything is fine at compile time:

```go,panic
package main

import "fmt"

func main() {
	var a, b any = []int{1}, []int{1}
	fmt.Println(a == b) // panic: runtime error: comparing uncomparable type []int
}
```

The same panic happens when such a value is used as a key in a `map[any]T`. The generic `comparable` constraint accepts interfaces (since Go 1.20) with exactly this risk.

## Interfaces with type sets

Since generics (Go 1.18), an interface can also contain **type terms**:

```go,ignore
type Number interface {
	~int | ~int64 | ~float64
}
```

Such interfaces can be used **only** as constraints for type parameters, not as variable types. Details in [Generics](../../generics/middle/).

## Unexported methods: "sealed" interfaces

An interface containing an **unexported** method can't be implemented outside its package. It's Go's way to get a closed set of implementations (a limited "sum type"):

```go,ignore
package shape

type Shape interface {
	Area() float64
	sealed() // only types in package shape can implement it
}
```

A type from another package can only "implement" it by embedding a value that already does.

## Summary

- `any` is an `eface` (type + data); interfaces with methods are `iface` (itab + data).
- `itab`s are computed once per pair and cached.
- Converting non-pointer values to an interface usually allocates.
- Interface calls block inlining; PGO can devirtualize them.
- `==` on interfaces with a non-comparable dynamic type panics.
