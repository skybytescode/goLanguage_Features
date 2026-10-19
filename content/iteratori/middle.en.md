# Range and iterators — Middle

Iterators become truly useful when you **compose** them: filtering, mapping, limiting, with no intermediate slices. Also here: iterators for your own structures, "pull" iterators and the rules a correct iterator must follow.

## Adapters: `Filter`, `Map`, `Take`

An adapter takes an iterator and returns another. Values pass through the chain one at a time, so nothing is computed needlessly:

```go
package main

import (
	"fmt"
	"iter"
	"slices"
)

func Filter[V any](seq iter.Seq[V], ok func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if ok(v) && !yield(v) {
				return
			}
		}
	}
}

func Map[V, W any](seq iter.Seq[V], f func(V) W) iter.Seq[W] {
	return func(yield func(W) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	return func(yield func(V) bool) {
		if n <= 0 {
			return
		}
		i := 0
		for v := range seq {
			if !yield(v) {
				return
			}
			i++
			if i == n {
				return
			}
		}
	}
}

// Naturals is an infinite iterator: 1, 2, 3, ...
func Naturals(yield func(int) bool) {
	for i := 1; ; i++ {
		if !yield(i) {
			return
		}
	}
}

func main() {
	evenSquares := Take(Map(Filter(Naturals, func(n int) bool { return n%2 == 0 }),
		func(n int) int { return n * n }), 4)
	fmt.Println(slices.Collect(evenSquares)) // [4 16 36 64]
}
```

`Naturals` is infinite, but `Take` stops the chain after 4 values: `yield` returns `false`, and each level stops in turn.

## Iterators for your own structures

The convention: an `All()` method (or `Values()`, `Keys()`) returning an `iter.Seq`/`iter.Seq2`. A binary tree walked in order:

```go
package main

import (
	"fmt"
	"iter"
)

type Tree struct {
	Left, Right *Tree
	Val         int
}

func (t *Tree) Insert(v int) *Tree {
	if t == nil {
		return &Tree{Val: v}
	}
	if v < t.Val {
		t.Left = t.Left.Insert(v)
	} else {
		t.Right = t.Right.Insert(v)
	}
	return t
}

func (t *Tree) All() iter.Seq[int] {
	return func(yield func(int) bool) {
		t.walk(yield)
	}
}

// walk returns false if the loop stopped, so the recursion stops right away.
func (t *Tree) walk(yield func(int) bool) bool {
	if t == nil {
		return true
	}
	return t.Left.walk(yield) && yield(t.Val) && t.Right.walk(yield)
}

func main() {
	var t *Tree
	for _, v := range []int{5, 3, 8, 1, 4, 9} {
		t = t.Insert(v)
	}
	for v := range t.All() {
		if v > 5 {
			break
		}
		fmt.Print(v, " ") // 1 3 4 5
	}
	fmt.Println()
}
```

Before Go 1.23, the same thing required a callback (`Walk(func(int))`), with no way to `break`, or a channel and a goroutine, which is much slower.

## The rules of a correct iterator

1. **Check `yield`'s result.** If it returns `false`, return immediately. Calling `yield` after `false` panics at run time.
2. **Don't keep `yield`** after the iterator returns, and don't call it from another goroutine.
3. **Release resources with `defer`** in the iterator: it runs both on `break` and on normal completion.

```go
package main

import (
	"fmt"
	"iter"
)

func Lines(name string) iter.Seq[string] {
	return func(yield func(string) bool) {
		fmt.Println("opening", name)
		defer fmt.Println("closing", name) // runs on break too
		for _, l := range []string{"a", "b", "c"} {
			if !yield(l) {
				return
			}
		}
	}
}

func main() {
	for l := range Lines("data.txt") {
		fmt.Println(l)
		if l == "b" {
			break
		}
	}
}
```

## Iterators in the standard library

| Function | Produces |
|---|---|
| `slices.All(s)`, `slices.Values(s)`, `slices.Backward(s)` | index/value, values, reversed |
| `slices.Chunk(s, n)` | sub-slices of `n` each |
| `maps.All(m)`, `maps.Keys(m)`, `maps.Values(m)` | pairs, keys, values |
| `strings.Lines`, `SplitSeq`, `FieldsSeq` | pieces of text |
| `slices.Collect`, `slices.Sorted`, `maps.Collect` | consume an iterator |
| `slices.AppendSeq`, `maps.Insert` | add from an iterator |

## "Pull" iterators: `iter.Pull`

The iterators above are "push": they call the loop body. Sometimes you need to ask for values **yourself**, one at a time — for example to walk two iterators in parallel. `iter.Pull` turns a push iterator into a `next`/`stop` pair:

```go
package main

import (
	"fmt"
	"iter"
	"slices"
)

func Zip[A, B any](a iter.Seq[A], b iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		nextB, stop := iter.Pull(b)
		defer stop() // mandatory: releases iterator b
		for va := range a {
			vb, ok := nextB()
			if !ok || !yield(va, vb) {
				return
			}
		}
	}
}

func main() {
	names := slices.Values([]string{"Ana", "Ion", "Eva"})
	grades := slices.Values([]int{9, 7})
	for n, g := range Zip(names, grades) {
		fmt.Println(n, g) // Ana 9, Ion 7
	}
}
```

Forget `stop()` and the iterator stays suspended, holding its resources.

## Summary

- Adapters (`Filter`, `Map`, `Take`) compose iterators without intermediate slices.
- Expose an `All()` returning `iter.Seq` for your collections.
- Always check `yield`'s result; `defer` for cleanup.
- `iter.Pull` when you need to ask for values yourself; always call `stop`.
