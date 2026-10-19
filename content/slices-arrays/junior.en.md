# Arrays and slices — Junior

Go has two types for sequences of elements: the **array**, with a fixed length, and the **slice**, with a variable length. In practice you'll almost always use slices.

## Arrays

The length is part of the type: `[3]int` and `[4]int` are different types.

```go
package main

import "fmt"

func main() {
	var a [3]int // [0 0 0]
	a[0] = 10
	b := [3]string{"a", "b", "c"}
	c := [...]int{1, 2, 3, 4} // the compiler counts: [4]int

	fmt.Println(a, b, c, len(c))
}
```

## Slices

A slice is a "window" over an array, with a variable length. It's written without a length in the brackets: `[]int`.

```go
package main

import "fmt"

func main() {
	numbers := []int{1, 2, 3} // a slice literal
	numbers = append(numbers, 4, 5)
	fmt.Println(numbers, len(numbers)) // [1 2 3 4 5] 5

	var empty []int // nil, length 0: append works right away
	empty = append(empty, 42)
	fmt.Println(empty)
}
```

**Important:** `append` returns the new slice. Always assign the result: `s = append(s, x)`.

## `make`: a slice with length and capacity

`make([]T, length, capacity)` creates a slice of zero elements. The capacity is space already reserved:

```go
package main

import "fmt"

func main() {
	s := make([]int, 3) // [0 0 0], len 3, cap 3
	fmt.Println(s, len(s), cap(s))

	t := make([]int, 0, 10) // empty, but with room for 10 elements
	for i := range 5 {
		t = append(t, i*i)
	}
	fmt.Println(t, len(t), cap(t)) // [0 1 4 9 16] 5 10
}
```

## Sub-slices

`s[a:b]` takes the elements from index `a` up to `b`, **excluding** `b`. Either end can be omitted:

```go
package main

import "fmt"

func main() {
	s := []string{"a", "b", "c", "d", "e"}
	fmt.Println(s[1:3]) // [b c]
	fmt.Println(s[:2])  // [a b]
	fmt.Println(s[3:])  // [d e]
	fmt.Println(s[:])   // everything
}
```

## Iterating

```go
package main

import "fmt"

func main() {
	grades := []int{9, 7, 10}
	sum := 0
	for _, g := range grades {
		sum += g
	}
	fmt.Printf("average: %.2f\n", float64(sum)/float64(len(grades)))
}
```

## `copy`

`copy(dst, src)` copies elements and returns how many it copied (the minimum of the two lengths):

```go
package main

import "fmt"

func main() {
	src := []int{1, 2, 3}
	dst := make([]int, 2)
	n := copy(dst, src)
	fmt.Println(dst, n) // [1 2] 2
}
```

## The `slices` package

Since Go 1.21, the `slices` package has the functions you'd otherwise write by hand:

```go
package main

import (
	"fmt"
	"slices"
)

func main() {
	s := []int{5, 2, 8, 1}
	fmt.Println(slices.Contains(s, 8)) // true
	fmt.Println(slices.Index(s, 2))    // 1
	fmt.Println(slices.Max(s))         // 8

	slices.Sort(s)
	fmt.Println(s) // [1 2 5 8]
}
```

## Multi-dimensional slices

```go
package main

import "fmt"

func main() {
	board := [][]string{
		{"X", "O", "X"},
		{" ", "X", "O"},
		{"O", " ", "X"},
	}
	for _, row := range board {
		fmt.Println(row)
	}
}
```

## Summary

- Array: fixed length, part of the type. Slice: variable length.
- `s = append(s, x)`: always assign the result.
- `make([]T, len, cap)` reserves space up front.
- `s[a:b]` includes `a`, excludes `b`.
- The `slices` package has sorting, searching and much more.
