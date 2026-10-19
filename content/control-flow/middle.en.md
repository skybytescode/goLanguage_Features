# Control flow — Middle

Control statements have a few details that matter in real code: labels for nested loops, `fallthrough`, loop variable semantics and the "return early" style.

## Labels: `break` and `continue` out of nested loops

A plain `break` only leaves the innermost loop. With a label, you can leave the outer one:

```go
package main

import "fmt"

func main() {
	matrix := [][]int{{1, 2, 3}, {4, -1, 6}, {7, 8, 9}}

search:
	for i, row := range matrix {
		for j, v := range row {
			if v < 0 {
				fmt.Printf("negative value at [%d][%d]\n", i, j)
				break search
			}
		}
	}
}
```

A common trap: `break` inside a `switch` or `select` only leaves the `switch`/`select`, **not** the loop around it:

```go
package main

import "fmt"

func main() {
	commands := []string{"a", "stop", "b"}
loop:
	for _, c := range commands {
		switch c {
		case "stop":
			break loop // without the label, it would only leave the switch
		default:
			fmt.Println("processing", c)
		}
	}
}
```

## `fallthrough`

`fallthrough` moves **unconditionally** into the body of the next `case`, without checking its condition. It's rarely used; a list of values (`case a, b:`) is usually clearer.

```go
package main

import "fmt"

func main() {
	level := 2
	switch level {
	case 3:
		fmt.Println("archive access")
		fallthrough
	case 2:
		fmt.Println("reports access")
		fallthrough
	case 1:
		fmt.Println("dashboard access")
	}
}
```

## `switch` with an init statement

Like `if`, `switch` accepts a short statement before the expression:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	switch env := os.Getenv("APP_ENV"); env {
	case "prod":
		fmt.Println("production")
	case "":
		fmt.Println("unset, using dev")
	default:
		fmt.Println("environment:", env)
	}
}
```

## Loop variables (Go 1.22+)

Before Go 1.22, the variable of a `for` loop was **a single one** for the whole loop. Closures and goroutines capturing it all saw the last value. Since Go 1.22, every iteration gets its own variable:

```go
package main

import "fmt"

func main() {
	var funcs []func()
	for i := range 3 {
		funcs = append(funcs, func() { fmt.Print(i, " ") })
	}
	for _, f := range funcs {
		f() // Go 1.22+: 0 1 2   (before: 3 3 3)
	}
	fmt.Println()
}
```

The behavior depends on the `go` version in `go.mod`: modules that declare `go 1.21` or older keep the old semantics. If you see old code with `i := i` inside a loop, that was the fix for this problem; it can now be deleted.

## Return early (guard clauses)

Idiomatic Go avoids `else` after a `return`. You handle error cases first, and the "happy path" stays at minimal indentation:

```go
package main

import (
	"errors"
	"fmt"
)

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func main() {
	r, err := divide(10, 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(r)
}
```

## `range` over different types

| Expression | First value | Second value |
|---|---|---|
| array, slice, pointer to array | `int` index | the element |
| string | byte index | `rune` (Unicode character) |
| map | key | value |
| channel | element | — |
| integer `n` | `0 … n-1` | — |
| iterator function | what `yield` gets | what `yield` gets |

The last row (iterator functions, Go 1.23) has its own topic: [Range and iterators](../../iteratori/middle/).

## `goto`

Go has `goto`, but it's used almost exclusively in generated code. It can't jump over variable declarations or into a block. In hand-written code, prefer loops and functions.

## Summary

- Labels solve leaving nested loops, and leaving a loop from inside `switch`/`select`.
- `fallthrough` ignores the next `case`'s condition.
- Since Go 1.22, each iteration has its own variables; you no longer need `i := i`.
- Handle errors first and return early.
