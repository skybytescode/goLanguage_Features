# Variables, types and constants — Senior

Here we look at the spec rules behind the behavior from the previous levels: exact constant arithmetic, type identity and assignability, initialization order, and what values look like in memory.

## Constants have arbitrary precision

Untyped constants are computed by the compiler with (practically) unlimited precision. Only the final result has to fit the type it ends up in:

```go
package main

import "fmt"

const (
	Huge  = 1 << 100   // doesn't fit any integer type
	Small = Huge >> 98 // 4: fits
	Third = 1.0 / 3
)

func main() {
	fmt.Println(Small)
	fmt.Println(Third*3 == 1) // true: computed exactly, at compile time
}
```

Each kind of untyped constant has a **default type**, used when the context doesn't ask for another (for example in `x := c`):

| Constant | Default type |
|---|---|
| `42` | `int` |
| `4.2` | `float64` |
| `'x'` | `rune` |
| `2i` | `complex128` |
| `"s"` | `string` |
| `true` | `bool` |

That's why `fmt.Println(Huge)` doesn't compile: `Huge` would become an `int`, and `1<<100` isn't representable as an `int`.

## Conversions: constants vs. run-time values

The same conversion behaves differently depending on the operand:

```go
package main

import "fmt"

func main() {
	v := 300
	fmt.Println(uint8(v)) // 44: at run time only the low bits are kept
	// fmt.Println(uint8(300)) // compile error: constant not representable

	f := 3.99
	fmt.Println(int(f))  // 3: truncation toward zero
	fmt.Println(int(-f)) // -3
}
```

Converting a `float64` that doesn't fit the target integer type (say, `int8(1e10)` on a variable) gives an **implementation-defined** result; don't rely on it.

## Identity and assignability

Every type has an **underlying type**. A **defined** type (`type T ...`) is different from every other type; type literals (`[]int`, `map[string]int`, `func(int) error`) are identical if they have the same structure.

A value of type `V` is assignable to a variable of type `T` if, among other cases:

- `V` and `T` are identical;
- they have the same underlying type **and** at least one of them is not a defined type;
- `T` is an interface and `V` implements it.

```go
package main

import "fmt"

type IDs []int
type List []int

func main() {
	raw := []int{1, 2, 3}
	var a IDs = raw // ok: []int is not a defined type
	// var b List = a // error: IDs and List are both defined types
	b := List(a) // an explicit conversion is allowed: same underlying type
	fmt.Println(a, b)
}
```

This rule is why you can pass a `func(int) int` literal where a `type Handler func(int) int` is expected, with no conversion.

## Generic aliases (Go 1.24)

Aliases can take type parameters, which helps when moving types between packages without breaking the API:

```go
package main

import "fmt"

type Set[T comparable] = map[T]struct{}

func main() {
	s := Set[string]{"a": {}, "b": {}}
	var m map[string]struct{} = s // identical: it's just an alias
	fmt.Println(len(m))
}
```

## `new` with an expression (Go 1.26)

Since Go 1.26, `new` also accepts an expression, not just a type. It's handy for optional pointer fields (for example in structs used for JSON):

```go
package main

import "fmt"

type Options struct {
	Limit *int
}

func main() {
	o := Options{Limit: new(100)} // before: v := 100; &v
	fmt.Println(*o.Limit)
}
```

## Initialization order

Package-level variables are initialized **in dependency order**, not necessarily in source order. Only then do the `init()` functions run, in the order they appear (the `go` command hands files to the compiler sorted by name):

```go
package main

import "fmt"

var (
	a = b + 1 // depends on b, so b is initialized first
	b = f()
)

func f() int {
	fmt.Println("initializing b")
	return 41
}

func init() {
	fmt.Println("init: a =", a)
}

func main() {
	fmt.Println("main")
}
```

Imported packages are fully initialized before the package that imports them. Initialization cycles are compile errors.

## What values look like in memory

Values have a fixed size and alignment. Struct fields are aligned, and the compiler inserts *padding* between them, so field order matters:

```go
package main

import (
	"fmt"
	"unsafe"
)

type Wasteful struct {
	a bool  // 1 byte + 7 padding
	b int64 // 8
	c bool  // 1 byte + 7 padding
}

type Compact struct {
	b int64 // 8
	a bool  // 1
	c bool  // 1 + 6 padding
}

func main() {
	fmt.Println(unsafe.Sizeof(Wasteful{}), unsafe.Sizeof(Compact{})) // 24 16
	fmt.Println(unsafe.Sizeof(""), unsafe.Sizeof([]int{}))           // 16 24
}
```

A `string` is a 2-word header (pointer + length); a slice is 3 words (pointer, length, capacity). The `fieldalignment` analyzer (from `golang.org/x/tools`) suggests the optimal field order.

## Floating point

`float64` follows IEEE 754, with all that implies:

```go
package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Println(0.1+0.2 == 0.3) // true: constants, computed exactly at compile time
	a, b := 0.1, 0.2
	fmt.Println(a+b == 0.3) // false: float64, at run time
	fmt.Println(a + b)      // 0.30000000000000004

	nan := math.NaN()
	fmt.Println(nan == nan) // false: NaN equals nothing

	m := map[float64]int{}
	m[nan] = 1
	m[nan] = 2
	fmt.Println(len(m)) // 2: NaN keys can never be looked up
}
```

For money, use integers (cents) or `math/big`.

## Summary

- Constants are exact; only the final value must be representable.
- Assignability depends on defined types and the underlying type.
- Package initialization follows dependencies, then `init()`.
- Field order changes the size of structs.
- Don't compare computed `float64` values with `==`.
