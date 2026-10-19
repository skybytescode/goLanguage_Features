# Variables, types and constants — Junior

Go is **statically typed**: every variable has a type known at compile time, and the compiler won't let you mix types by accident. Most of the time, though, you don't have to write the type: the compiler infers it.

## Your first program

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, Go!")
}
```

Every executable program has a `main` package and a `main` function. `fmt` is the package for printing and formatting.

## Declaring variables

There are three common forms:

```go
package main

import "fmt"

func main() {
	var a int = 10 // type and value
	var b = 20     // type inferred from the value: int
	c := 30        // short declaration, only inside functions

	var name string // no value: gets the "zero value"
	fmt.Println(a, b, c, name == "")
}
```

`:=` is the form you'll use most. It **declares** a new variable; to change an existing variable, use `=`.

## The zero value

Go has no uninitialized variables. Every type has a **zero value**:

| Type | Zero value |
|---|---|
| `int`, `float64` | `0` |
| `bool` | `false` |
| `string` | `""` |
| pointers, slices, maps, channels, functions, interfaces | `nil` |

```go
package main

import "fmt"

func main() {
	var n int
	var ok bool
	var s string
	var p *int
	fmt.Printf("%d %t %q %v\n", n, ok, s, p) // 0 false "" <nil>
}
```

## Basic types

| Category | Types |
|---|---|
| Signed integers | `int`, `int8`, `int16`, `int32`, `int64` |
| Unsigned integers | `uint`, `uint8` (= `byte`), `uint16`, `uint32`, `uint64`, `uintptr` |
| Floating point | `float32`, `float64` |
| Complex | `complex64`, `complex128` |
| Text | `string`, `rune` (= `int32`, one Unicode character) |
| Boolean | `bool` |

`int` is 64 bits wide on 64-bit systems. For real numbers, use `float64` unless you have a specific reason not to.

## Explicit conversions

Go does **not** convert between numeric types automatically, not even from `int` to `float64`:

```go,compile_fail
package main

import "fmt"

func main() {
	x := 3
	y := 1.5
	fmt.Println(x * y) // error: mismatched types int and float64
}
```

A conversion looks like a call, `T(value)`:

```go
package main

import "fmt"

func main() {
	x := 3
	y := 1.5
	fmt.Println(float64(x) * y) // 4.5
	fmt.Println(int(y))         // 1: the fractional part is dropped
}
```

## Unused variables are errors

A local variable that is declared but never used stops compilation. The same goes for unused imports. To discard a value on purpose, use the blank identifier `_`:

```go,compile_fail
package main

func main() {
	x := 5 // error: declared and not used: x
}
```

## Constants

Constants are declared with `const` and computed at compile time:

```go
package main

import "fmt"

const Pi = 3.14159

const (
	DaysInWeek = 7
	Greeting   = "Hello"
)

func main() {
	radius := 2.0
	fmt.Println(Pi * radius * radius)
	fmt.Println(Greeting, DaysInWeek)
}
```

## `iota`: enumerations

Inside a `const` block, `iota` starts at 0 and increases by 1 on each line. This is how enumerations are written in Go:

```go
package main

import "fmt"

type Day int

const (
	Monday    Day = iota // 0
	Tuesday              // 1
	Wednesday            // 2
)

func main() {
	fmt.Println(Monday, Tuesday, Wednesday)
}
```

## Summary

- `:=` declares and initializes; `=` only assigns.
- Every variable has a zero value; there is no "undefined".
- Conversions between types are always explicit.
- Unused variables and imports are compile errors.
- `const` and `iota` for fixed values and enumerations.
