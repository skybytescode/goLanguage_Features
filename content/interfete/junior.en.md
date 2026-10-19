# Interfaces — Junior

An **interface** describes a behavior: a list of methods. Any type that has all those methods implements it **automatically**, without declaring so. There's no `implements` keyword in Go.

## Defining and implementing an interface

```go
package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
}

type Rectangle struct{ W, H float64 }
type Circle struct{ R float64 }

func (r Rectangle) Area() float64 { return r.W * r.H }
func (c Circle) Area() float64    { return math.Pi * c.R * c.R }

func main() {
	shapes := []Shape{Rectangle{2, 3}, Circle{1}}
	total := 0.0
	for _, s := range shapes {
		total += s.Area() // the call reaches the concrete type's method
	}
	fmt.Printf("total area: %.2f\n", total)
}
```

`Rectangle` and `Circle` know nothing about `Shape`. Having the `Area() float64` method is enough.

## Functions that take interfaces

A function that takes an interface works with any type that implements it, including types written later:

```go
package main

import "fmt"

type Greeter interface {
	Greet() string
}

type Romanian struct{}
type English struct{}

func (Romanian) Greet() string { return "Bună ziua!" }
func (English) Greet() string  { return "Hello!" }

func introduce(g Greeter) {
	fmt.Println(g.Greet())
}

func main() {
	introduce(Romanian{})
	introduce(English{})
}
```

## `fmt.Stringer`

The best-known small interface in the standard library. If your type has `String() string`, `fmt` uses it when printing:

```go
package main

import "fmt"

type Temperature float64

func (t Temperature) String() string {
	return fmt.Sprintf("%.1f°C", float64(t))
}

func main() {
	var t Temperature = 21.456
	fmt.Println(t) // 21.5°C
}
```

## The empty interface: `any`

`any` (an alias for `interface{}`) requires no methods, so every value implements it. It's useful when you really don't know the type (as in `fmt.Println`), but you lose compile-time type checking. Use it sparingly.

```go
package main

import "fmt"

func main() {
	var x any
	x = 42
	fmt.Println(x)
	x = "text"
	fmt.Println(x)
	x = []int{1, 2}
	fmt.Println(x)
}
```

## Type assertion: getting the concrete type

`v.(T)` extracts the value of type `T` from an interface. The two-result form doesn't panic if the type is different:

```go
package main

import "fmt"

func main() {
	var x any = "hello"

	s, ok := x.(string)
	fmt.Println(s, ok) // hello true

	n, ok := x.(int)
	fmt.Println(n, ok) // 0 false

	// x.(int) without ok would panic: interface conversion
}
```

## Type switch

When there are several possible types, a **type switch** is clearer than a chain of assertions:

```go
package main

import "fmt"

func describe(x any) string {
	switch v := x.(type) {
	case int:
		return fmt.Sprintf("integer: %d", v)
	case string:
		return fmt.Sprintf("text of %d bytes", len(v))
	case bool:
		return fmt.Sprintf("boolean: %t", v)
	case nil:
		return "nothing"
	default:
		return fmt.Sprintf("other type: %T", v)
	}
}

func main() {
	fmt.Println(describe(7))
	fmt.Println(describe("Go"))
	fmt.Println(describe(true))
	fmt.Println(describe(nil))
	fmt.Println(describe(3.14))
}
```

In each branch, `v` has the type from the `case`.

## Summary

- An interface is a list of methods; implementation is implicit.
- Functions that take interfaces work with any matching type.
- `String() string` controls printing with `fmt`.
- `any` accepts anything, but you lose compile-time checks.
- `v, ok := x.(T)` and `switch x.(type)` recover the concrete type.
