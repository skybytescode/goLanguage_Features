# Arrays and slices — Senior

The internal structure of a slice, `append`'s growth algorithm, conversions between slices and arrays, and a few memory details that matter in performance-sensitive code.

## The slice header

A slice is a three-word struct, passed **by value**:

```go,ignore
type slice struct {
	array unsafe.Pointer // the first visible element
	len   int
	cap   int // how many elements are available from array onward
}
```

All the behavior from the previous levels follows from this: a function receiving a slice gets a copy of the header (so it sees the same elements), but changing `len` via `append` stays in its copy.

`unsafe.SliceData(s)` returns the pointer to the first element, and `unsafe.Slice(p, n)` builds a slice from a pointer and a length; both are useful for interop with C or system calls.

## How `append` grows

When the capacity runs out, `runtime.growslice` allocates a new array and copies the elements. The new capacity:

1. if more than double the old capacity is needed, take exactly what's needed;
2. otherwise, below 256 elements, **double**;
3. above 256, grow gradually from ×2 toward ×1.25 (the formula `cap += (cap + 3*256) / 4`);
4. the result is rounded up to an allocator **size class**, so the final capacity can be slightly larger.

```go
package main

import "fmt"

func main() {
	var s []int64
	last := -1
	for i := range 2000 {
		s = append(s, int64(i))
		if cap(s) != last {
			fmt.Print(cap(s), " ")
			last = cap(s)
		}
	}
	fmt.Println()
	// Go 1.26: 4 8 16 32 64 128 256 512 848 1280 1792 2560
}
```

Geometric growth makes `append` **amortized O(1)**, but each reallocation copies everything and leaves the old array to the GC. For large known volumes, preallocation or `slices.Grow(s, n)` eliminates reallocations.

## Slice ↔ array conversions

```go
package main

import "fmt"

func main() {
	s := []byte{1, 2, 3, 4}

	p := (*[4]byte)(s) // Go 1.17: pointer to array, same memory
	p[0] = 100

	a := [4]byte(s) // Go 1.20: array, a copy
	a[1] = 200

	fmt.Println(s, *p, a) // [100 2 3 4] [100 2 3 4] [100 200 3 4]

	// [8]byte(s) would panic: the slice's length (4) is less than 8
}
```

An array can become a slice with `arr[:]`, with no copy. Conversions are useful for hashes and fixed-length keys (for example a `[32]byte` map key from a `[]byte`).

## `clear`, `slices.Delete` and forgotten references

Elements between `len` and `cap` are invisible but **still referenced** by the array. If they're pointers, the GC can't free them:

```go
package main

import (
	"fmt"
	"slices"
)

type Big struct{ data [1 << 16]byte }

func main() {
	s := []*Big{{}, {}, {}}
	s = s[:1] // s[1] and s[2] are still in the array: the memory isn't freed

	t := []*Big{{}, {}, {}}
	t = slices.Delete(t, 1, 3) // since Go 1.22, sets the vacated slots to nil
	fmt.Println(len(s), len(t), t[:3][1] == nil)

	u := []int{1, 2, 3}
	clear(u) // Go 1.21: sets every element to its zero value; len stays 3
	fmt.Println(u)
}
```

The `slices` functions that shrink (`Delete`, `DeleteFunc`, `Compact`, `Replace`) have cleared the leftover tail since Go 1.22, so they don't keep objects alive.

## `append` and goroutines

Two goroutines `append`ing to sub-slices of the same array with spare capacity write to the same memory: a **data race**, even though each has "its own" slice. The dangerous pattern:

```go,ignore
base := make([]int, 0, 10)
go func() { a := append(base, 1); use(a) }() // both write to base[0]
go func() { b := append(base, 2); use(b) }()
```

The race detector (`go test -race`) catches this. The fix: each goroutine gets a copy (`slices.Clone`) or a sub-slice with limited capacity (`base[:0:0]`).

## Iterating: index vs. value

`for _, v := range s` copies each element into `v`. For large elements (structs of hundreds of bytes), `for i := range s` with `s[i].Field` avoids the copy. For small elements the difference is negligible, and the compiler eliminates bounds checks in both forms.

## Large arrays on the stack

Arrays are values: a local `[1 << 20]byte` may be allocated on the stack, and passing it as an argument copies it in full. The compiler moves variables larger than 10 MiB (or 64 KiB for implicit `make`) to the heap; otherwise, watch out for hidden copies of large arrays (in `range`, on assignment, in calls).

## Summary

- Slice = (pointer, len, cap), passed by value.
- `append` doubles below 256 elements, then grows toward ×1.25, rounded to size classes.
- `(*[N]T)(s)` shares memory, `[N]T(s)` copies.
- Elements past `len` keep objects alive; `slices.Delete` and `clear` help.
- Concurrent `append` on the same array is a data race.
