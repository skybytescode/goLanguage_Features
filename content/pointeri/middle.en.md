# Pointers — Middle

The practical question isn't "what is a pointer" but **when** to use one. The answer depends on mutability, the size of values and the semantics you want for your types.

## Value or pointer?

| Use a pointer when… | Use a value when… |
|---|---|
| the function must modify the argument | the value is small and immutable (e.g. `time.Time`, a `Point{X, Y}`) |
| the struct is large and copying is costly | you want every copy to be independent |
| the type contains a `sync.Mutex` or another field that must not be copied | the type is a basic type, slice, map, channel or function |
| you need "absence of value" (`nil`) | |

Slices, maps, channels and functions already contain internal pointers. A `*[]int` or `*map[string]int` is rarely needed: changes to the elements are visible to the caller anyway.

```go
package main

import "fmt"

func set(m map[string]int) {
	m["a"] = 1 // visible to the caller, no pointer needed
}

func add(s []int) {
	s = append(s, 4) // NOT visible: the caller's length doesn't change
	_ = s
}

func main() {
	m := map[string]int{}
	set(m)
	fmt.Println(m) // map[a:1]

	s := []int{1, 2, 3}
	add(s)
	fmt.Println(s) // [1 2 3]
}
```

For `append`, the Go convention is to return the new slice (`s = add(s)`), not to take a `*[]int`.

## You can return the address of a local variable

In C, this would be a serious bug. In Go it's completely safe: the compiler notices that the address "escapes" the function and allocates the variable on the heap.

```go
package main

import "fmt"

type Config struct {
	Port int
}

func defaults() *Config {
	c := Config{Port: 8080}
	return &c // safe: c lives as long as there are pointers to it
}

func main() {
	fmt.Println(defaults().Port)
}
```

## Optional fields

A pointer distinguishes "zero value" from "not set". It's useful for partial updates or JSON:

```go
package main

import "fmt"

type Update struct {
	Name *string
	Age  *int
}

func apply(name *string, age *int, u Update) {
	if u.Name != nil {
		*name = *u.Name
	}
	if u.Age != nil {
		*age = *u.Age
	}
}

func main() {
	name, age := "Ana", 30
	zero := 0
	apply(&name, &age, Update{Age: &zero}) // set the age to 0, the name stays
	fmt.Println(name, age)
}
```

## Trap: pointers to slice elements

`&s[i]` points to the element in the array behind the slice. If a later `append` reallocates the array, the pointer still points to the **old** array:

```go
package main

import "fmt"

func main() {
	s := make([]int, 1, 1)
	p := &s[0]
	s = append(s, 2) // capacity exceeded: new array
	*p = 100
	fmt.Println(s[0]) // 0: p points to the old array
}
```

Maps go further: in Go you can't even take the address of a map element (`&m[k]` doesn't compile), precisely because the map moves its elements around.

## Nil pointers and methods

A method with a pointer receiver **can** be called on a `nil` pointer. Sometimes that's even useful, as a safe default:

```go
package main

import "fmt"

type Node struct {
	Val  int
	Next *Node
}

func (n *Node) Sum() int {
	if n == nil {
		return 0 // empty list
	}
	return n.Val + n.Next.Sum()
}

func main() {
	list := &Node{1, &Node{2, &Node{3, nil}}}
	fmt.Println(list.Sum()) // 6
	var empty *Node
	fmt.Println(empty.Sum()) // 0, no panic
}
```

## Comparing pointers

Two pointers are equal if they point to the same variable (or are both `nil`), not if the values are equal:

```go
package main

import "fmt"

func main() {
	a, b := 1, 1
	pa, pb := &a, &b
	fmt.Println(pa == pb, *pa == *pb) // false true
}
```

## Summary

- Pointers for mutability, large structs and types that must not be copied.
- Maps, slices and channels don't need pointers to modify elements.
- Returning the address of a local variable is safe.
- Pointers into slices may keep pointing to an old array after `append`.
- Methods on pointers can handle the `nil` case explicitly.
