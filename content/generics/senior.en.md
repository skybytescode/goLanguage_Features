# Generics — Senior

How Go compiles generic code (GC shape stenciling with dictionaries), what that means for performance, what type sets are, and what limitations the current design has.

## Implementation: GC shape stenciling + dictionaries

There are two classic strategies:

- **monomorphization** (C++, Rust): one copy of the code per type; fast, but big binaries and slow compilation;
- **boxing** (Java): a single copy, everything through pointers and dynamic dispatch; compact, but slow.

Go chose a middle ground. The compiler generates one copy per **GC shape** — types with the same memory representation from the allocator's and the GC's point of view. `int` and `int64` share a shape on 64-bit; a defined type `type ID int` has `int`'s shape; and, importantly, **all pointer types share a single shape**.

Each copy receives a hidden **dictionary** as its first argument, holding information specific to the concrete type: the type descriptor, the `itab`s for the constraint's methods, the equality and hash functions.

```go,ignore
// what you write
func Sum[T Number](s []T) T

// roughly what the compiler generates
func Sum[go.shape.int](dict *dictionary, s []int) int
func Sum[go.shape.float64](dict *dictionary, s []float64) float64
```

## The consequence: methods on pointer types go through the dictionary

For value types (`int`, `float64`, structs), each shape gets specialized code; arithmetic and comparisons are direct. For pointer types, they all share the **same** copy, and a call to a constraint method becomes an indirect call through the dictionary — similar to an interface call, but with one extra level of indirection:

```go,ignore
func Show[T fmt.Stringer](v T) string {
	return v.String() // for T = *Person: an indirect call through the dictionary, no inlining
}
```

The practical conclusion: generics are excellent for **data structures** and algorithms over value types. For code that just calls methods, a plain interface is as fast or faster, and the compiler can devirtualize it more easily. Benchmark before "optimizing" by switching to generics.

## Type sets

Since Go 1.18, an interface defines a **type set**: the set of types that satisfy it.

- `interface{ String() string }` — every type with a `String` method;
- `interface{ ~int | ~string }` — types whose underlying type is `int` or `string`;
- `interface{ ~int; String() string }` — the intersection of the two.

A type satisfies a constraint if it's in its type set. The operations allowed in the generic function are those valid for **every** type in the set: `+` is allowed on `~int | ~float64`, but not on `~int | ~[]byte`.

## The shared underlying type and `range`

Some operations (`range`, `make`, channel operations) require all types in the set to have **the same underlying type**. Up to Go 1.24 the spec called this shared type the **core type**; since Go 1.25 the notion was removed from the spec and the rules are described per operation, with the same practical effect. Indexing is more permissive: it's enough that all types yield the same element type:

```go
package main

import "fmt"

type Bytes interface {
	~[]byte | ~string
}

func First[T Bytes](s T) byte {
	return s[0] // indexing is allowed: both yield a byte
}

func main() {
	fmt.Println(First("abc"), First([]byte{7}))
}
```

`for range s` on a `T` with the set `~[]byte | ~string` doesn't compile (`[]byte and string have different underlying types`): `range` produces different values for the two (bytes vs. runes).

## `comparable` and interfaces

Since Go 1.20, interface types satisfy `comparable`, even though comparing them can panic at run time (see [Interfaces](../../interfete/senior/)). The spec distinguishes between **strictly comparable** types (where `==` can't panic) and comparable ones in the broad sense.

## Self-referential constraints (Go 1.26)

A generic type can appear in its own type parameter list. This allows constraints of the form "T must have a method that takes a T":

```go
package main

import "fmt"

type Adder[A Adder[A]] interface {
	Add(A) A
}

func SumAll[A Adder[A]](xs ...A) A {
	var total A
	for _, x := range xs {
		total = total.Add(x)
	}
	return total
}

type Vector struct{ X, Y int }

func (v Vector) Add(w Vector) Vector { return Vector{v.X + w.X, v.Y + w.Y} }

func main() {
	fmt.Println(SumAll(Vector{1, 2}, Vector{3, 4}, Vector{5, 6})) // {9 12}
}
```

Before Go 1.26, the same thing required an inline constraint of the form `[A interface{ Add(A) A }]`.

## The design's limitations

- **No type parameters on methods.** It would require run-time code generation or break the link between methods and interfaces.
- **No specialization.** You can't write a separate variant for `T = string`; you use a type switch on `any(v)`, which the compiler usually simplifies at instantiation.
- **No variadic type parameters** (`func F[T ...any]`) and no constant values as parameters (`Array[T, N]`).
- **Inference** doesn't use the expected result type, only the arguments (with a few exceptions for functions passed as arguments, since Go 1.21).

## Summary

- One code copy per GC shape, plus a dictionary with the type's information.
- All pointer types share the same copy: methods are called indirectly.
- Generics for data structures over value types; measure before optimizing.
- Type sets and shared underlying types decide which operations are allowed.
- Since Go 1.26, constraints can be self-referential.
