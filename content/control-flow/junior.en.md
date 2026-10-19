# Control flow — Junior

Go has few control statements, but each is flexible: `if`, `for` (the only loop in the language) and `switch`.

## `if`

The condition has no parentheses, but the braces are mandatory:

```go
package main

import "fmt"

func main() {
	age := 20
	if age >= 18 {
		fmt.Println("adult")
	} else if age >= 14 {
		fmt.Println("teenager")
	} else {
		fmt.Println("child")
	}
}
```

The condition must be a `bool`: `if 1 { }` doesn't compile, unlike in C or JavaScript.

## `if` with an init statement

Before the condition you can put a short statement. A variable declared there exists only inside the `if` and its `else` branches:

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	if n, err := strconv.Atoi("123"); err != nil {
		fmt.Println("not a number:", err)
	} else {
		fmt.Println("the number is", n)
	}
	// n and err no longer exist here
}
```

This form is used all the time for error handling.

## `for`: the only loop

Go has no `while` or `do-while`. `for` covers them all:

```go
package main

import "fmt"

func main() {
	// the classic form
	for i := 0; i < 3; i++ {
		fmt.Println("i =", i)
	}

	// like a "while"
	n := 1
	for n < 100 {
		n *= 2
	}
	fmt.Println("n =", n)

	// an infinite loop, stopped with break
	k := 0
	for {
		k++
		if k == 5 {
			break
		}
	}
	fmt.Println("k =", k)
}
```

## `for` with `range`

`range` walks the elements of a collection. You get the index (or key) and the value:

```go
package main

import "fmt"

func main() {
	fruits := []string{"apple", "pear", "plum"}
	for i, f := range fruits {
		fmt.Println(i, f)
	}

	// value only: ignore the index with _
	for _, f := range fruits {
		fmt.Println(f)
	}

	// since Go 1.22: range over a number
	for i := range 3 {
		fmt.Println("step", i) // 0, 1, 2
	}
}
```

## `continue` and `break`

`continue` jumps to the next iteration, `break` leaves the loop:

```go
package main

import "fmt"

func main() {
	for i := range 10 {
		if i%2 == 0 {
			continue // skip even numbers
		}
		if i > 7 {
			break
		}
		fmt.Println(i) // 1 3 5 7
	}
}
```

## `switch`

`switch` is cleaner than a chain of `if-else`. Unlike C, it **does not fall through** to the next case: you don't need `break`.

```go
package main

import "fmt"

func main() {
	day := "saturday"
	switch day {
	case "saturday", "sunday": // several values in one case
		fmt.Println("weekend")
	case "monday":
		fmt.Println("start of the week")
	default:
		fmt.Println("weekday")
	}
}
```

## `switch` with no expression

Without an expression, each `case` is a condition. It's the equivalent of a long `if-else if`:

```go
package main

import "fmt"

func main() {
	grade := 8
	switch {
	case grade >= 9:
		fmt.Println("excellent")
	case grade >= 7:
		fmt.Println("good")
	case grade >= 5:
		fmt.Println("pass")
	default:
		fmt.Println("fail")
	}
}
```

## Summary

- `if` has no parentheses but always has braces; it can have an init statement.
- `for` is the only loop: classic, "while", infinite or with `range`.
- `switch` doesn't need `break` and can list several values in one `case`.
