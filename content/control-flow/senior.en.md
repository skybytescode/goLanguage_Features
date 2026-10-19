# Control flow — Senior

What exactly `range` does with the expression it walks, how Go compiles a `switch`, what per-iteration loop variables cost, and how to help the compiler eliminate bounds checks.

## `range` evaluates its expression once

The expression after `range` is evaluated **once**, before the first iteration. For slices, the length is fixed at that point; appends inside the loop don't extend it:

```go
package main

import "fmt"

func main() {
	s := []int{1, 2, 3}
	for i, v := range s {
		if i == 0 {
			s = append(s, 99) // s now points to a different, longer array
		}
		fmt.Print(v, " ")
	}
	fmt.Println("\nlen:", len(s)) // the loop ran 3 times, but len is 4
}
```

For an **array** (not a slice), `range` walks a **copy** of it. Changes made inside the loop don't show up in the values you receive:

```go
package main

import "fmt"

func main() {
	arr := [3]int{1, 2, 3}
	for i, v := range arr { // a copy of the array
		arr[2] = 100
		fmt.Print(i, ":", v, " ") // 0:1 1:2 2:3
	}
	fmt.Println()
	for i, v := range &arr { // pointer: no copy
		arr[2] = 200
		fmt.Print(i, ":", v, " ") // 0:1 1:2 2:200
	}
	fmt.Println()
}
```

The value `v` is always a copy of the element. To modify the elements of a slice of structs, use the index: `s[i].Field = x`. For large arrays of structs, `for i := range s` also avoids copying every element.

## Map order is random on purpose

The runtime picks a random starting point for every `range` over a map, so nobody can depend on an order. Deleting a key during iteration is safe; a key added during iteration **may or may not** show up.

```go
package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	for _, k := range slices.Sorted(maps.Keys(m)) { // stable order
		fmt.Println(k, m[k])
	}
}
```

## `range` over a string decodes UTF-8

The steps are characters, not bytes. Invalid bytes produce `U+FFFD` and advance by a single byte:

```go
package main

import "fmt"

func main() {
	for i, r := range "aș\xffb" {
		fmt.Printf("%d:%q ", i, r) // 0:'a' 1:'ș' 3:'�' 4:'b'
	}
	fmt.Println()
}
```

## How `switch` compiles

Cases are evaluated **top to bottom, left to right**, until the first match. Non-constant case expressions are only evaluated if reached, so their side effects only happen when needed.

When all cases are integer or string constants, the compiler no longer tests them one by one: it sorts the values and emits a **binary search**, and for dense integers on amd64 and arm64, a **jump table** (since Go 1.19). A `switch` with many constant cases is not slower than a map; it's usually faster.

```go
package main

import "fmt"

func class(c byte) string {
	switch c { // constant cases: jump table / binary search
	case 'a', 'e', 'i', 'o', 'u':
		return "vowel"
	case ' ', '\t', '\n':
		return "space"
	}
	return "other"
}

func main() {
	for _, c := range []byte("ai x") {
		fmt.Print(class(c), " ")
	}
	fmt.Println()
}
```

## What per-iteration variables cost

The Go 1.22 change doesn't mean an allocation on every step. The compiler creates a new variable per iteration only **semantically**; if the variable isn't captured by a closure and doesn't escape (escape analysis), it stays in a register, exactly as before.

For the three-clause form (`for i := 0; i < n; i++`), at the end of every iteration the variable is copied into a fresh one **before** `i++`. So a closure captured in one iteration doesn't see the increment:

```go
package main

import "fmt"

func main() {
	var ps []*int
	for i := 0; i < 3; i++ {
		ps = append(ps, &i) // different addresses since Go 1.22
	}
	for _, p := range ps {
		fmt.Print(*p, " ") // 0 1 2
	}
	fmt.Println()
}
```

The `bisect` tool (`golang.org/x/tools/cmd/bisect`) can pinpoint the exact loop whose behavior changed when migrating an old module to `go 1.22`.

## Bounds check elimination

Every `s[i]` carries a bounds check that panics when out of range. The compiler removes them when it can prove the index is valid; `for i := range s` is the ideal case. You can help with an explicit check up front:

```go
package main

import "fmt"

func sum4(s []int) int {
	_ = s[3] // one check; the four below are eliminated
	return s[0] + s[1] + s[2] + s[3]
}

func main() {
	fmt.Println(sum4([]int{1, 2, 3, 4}))
}
```

The remaining checks show up with:

```bash
go build -gcflags='-d=ssa/check_bce/debug=1' .
```

## `defer` in loops

`defer` runs when the **function** returns, not at the end of the iteration. In a loop that opens files, they all stay open until the end:

```go,ignore
for _, name := range files {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close() // piles up: they all close only at return
	// ...
}
```

The fix is to move the loop body into a separate function, where `defer` runs on every call.

## Summary

- `range` evaluates its expression once; arrays are copied, slices aren't.
- Map order is random on purpose; sort the keys when it matters.
- `switch` with constant cases becomes a binary search or a jump table.
- Per-iteration variables cost nothing if they don't escape the loop.
- `for i := range s` and "hint" checks eliminate bounds checks.
