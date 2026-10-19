# Variables, types and constants — Middle

At this level we care about the traps: variable shadowing, the difference between defined types and aliases, "untyped" constants, and enumerations that print nicely.

## Variable shadowing

`:=` declares a **new** variable in the current block, even if one with the same name exists in an outer block. It is one of the most common sources of bugs in Go:

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	value := 0
	if true {
		value, err := strconv.Atoi("42") // a NEW variable, visible only inside the if
		if err != nil {
			return
		}
		_ = value
	}
	fmt.Println(value) // 0, not 42
}
```

The fix: declare `err` separately and use `=`:

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	value := 0
	if true {
		var err error
		value, err = strconv.Atoi("42")
		if err != nil {
			return
		}
	}
	fmt.Println(value) // 42
}
```

> `go vet` has an optional analyzer, `shadow`, that finds these cases.

The rule for `:=` with several variables: it's allowed if **at least one** on the left is new in the current block; the others are simply assigned.

## Defined types and aliases

`type Celsius float64` creates a **new** type, distinct from `float64`. You can't mix the values without a conversion, but you can add methods:

```go
package main

import "fmt"

type Celsius float64
type Fahrenheit float64

func (c Celsius) ToF() Fahrenheit { return Fahrenheit(c*9/5 + 32) }

type Degrees = Celsius // alias: another name for the SAME type

func main() {
	var t Celsius = 21.5
	var d Degrees = t // works: Degrees and Celsius are identical
	fmt.Println(t.ToF(), d.ToF())
	fmt.Printf("%T\n", d) // main.Celsius
}
```

Defined types make APIs safer: a function that takes `Celsius` won't accept a `Fahrenheit` by mistake.

## Enumerations that print: `String()`

Any type with a `String() string` method implements `fmt.Stringer`, and `fmt` uses it when printing:

```go
package main

import "fmt"

type State int

const (
	New State = iota
	InProgress
	Done
)

func (s State) String() string {
	switch s {
	case New:
		return "new"
	case InProgress:
		return "in progress"
	case Done:
		return "done"
	}
	return fmt.Sprintf("State(%d)", int(s))
}

func main() {
	fmt.Println(InProgress) // in progress
	fmt.Println(State(7))   // State(7)
}
```

> In larger projects, `String()` is generated with `go generate` and the `stringer` tool (`golang.org/x/tools/cmd/stringer`).

Careful: inside `String()`, don't call `fmt.Sprintf("%v", s)` on the same value: it would call `String()` again, forever. That's why we converted to `int`.

## `iota` tricks

`iota` is an expression: you can combine it with operators, and the expression repeats on the following lines.

```go
package main

import "fmt"

type Permission uint8

const (
	Read    Permission = 1 << iota // 1
	Write                          // 2
	Execute                        // 4
)

type Size int64

const (
	_       = iota // skip 0
	KB Size = 1 << (10 * iota)
	MB
	GB
)

func main() {
	p := Read | Write
	fmt.Println(p&Write != 0, p&Execute != 0) // true false
	fmt.Println(KB, MB, GB)
}
```

## Untyped constants

`const x = 10` does not have type `int`: it is an **untyped** constant. It only gets a type when used, so it fits any numeric context:

```go
package main

import (
	"fmt"
	"time"
)

const timeout = 5 // untyped

func main() {
	var f float64 = timeout    // ok: becomes float64
	d := timeout * time.Second // ok: becomes time.Duration
	fmt.Println(f, d)

	n := 5 // a variable of type int
	// d2 := n * time.Second  // error: int * time.Duration
	d2 := time.Duration(n) * time.Second
	fmt.Println(d2)
}
```

## Integer overflow

At run time, integers **wrap around** silently. At compile time, constants that don't fit are errors:

```go
package main

import (
	"fmt"
	"math"
)

func main() {
	var x int8 = math.MaxInt8
	x++
	fmt.Println(x) // -128

	var u uint = 0
	u--
	fmt.Println(u == math.MaxUint) // true
}
```

```go,compile_fail
package main

func main() {
	var x int8 = 200 // error: constant 200 overflows int8
	_ = x
}
```

## Useful `fmt` verbs

| Verb | Prints |
|---|---|
| `%v` | the value, default format |
| `%+v` | for structs, field names too |
| `%#v` | Go syntax for the value |
| `%T` | the type |
| `%q` | quoted, escaped string |
| `%x` | hexadecimal |

## Summary

- `:=` in inner blocks can shadow variables; read code around `err` carefully.
- `type T U` creates a new type; `type T = U` is just another name.
- Implement `String()` for enumerations.
- Untyped constants adapt to their context; variables don't.
- Integer overflow at run time is not an error.
