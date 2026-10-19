# Reflection and unsafe — Senior

How a `reflect.Value` is represented, what reflection can and can't do (unexported fields, creating new types), the exact rules for `unsafe.Pointer`, and the tools that check them: `go vet` and `-d=checkptr`.

## What a `reflect.Value` looks like

```go,ignore
type Value struct {
	typ_ *abi.Type      // the runtime type descriptor (the same one interfaces use)
	ptr  unsafe.Pointer // a pointer to the data, or the data itself if it fits in a pointer
	flag                // the Kind, plus bits: addressable, read-only, indirect, method
}
```

`reflect.ValueOf(x)` takes an `any`, so `x` is first converted to an interface (possibly with an allocation, see [Interfaces](../../interfete/senior/)); then the `Value` copies the type and pointer from the interface. The bits in `flag` explain the previous level's rules:

- **flagAddr**: the value is addressable (it came through an `Elem()` on a pointer or through a field of an addressable value) → `CanSet` can be true;
- **flagRO**: the value was obtained through an **unexported** field → it can be read, but `Set` panics, and so does `Interface()`.

```go
package main

import (
	"fmt"
	"reflect"
)

type Account struct {
	Public string
	secret string
}

func main() {
	a := Account{"visible", "hidden"}
	v := reflect.ValueOf(&a).Elem()

	fmt.Println(v.Field(1).String())       // "hidden": reading is allowed
	fmt.Println(v.Field(1).CanSet())       // false: unexported field
	fmt.Println(v.Field(0).CanSet())       // true
	fmt.Println(v.Field(1).CanInterface()) // false: Interface() would panic
}
```

## What reflection can create at run time

- new values of any type: `reflect.New(t)`, `reflect.MakeSlice`, `MakeMap`, `MakeChan`;
- **new composite types**: `reflect.SliceOf(t)`, `MapOf(k, v)`, `PointerTo(t)`, `FuncOf(...)`, even `StructOf(fields)`;
- new functions: `reflect.MakeFunc(type, impl)` creates a real, callable function whose implementation receives its arguments as `[]reflect.Value`.

What it **can't** do: create types with new **methods** (`StructOf` can't add methods, so created types can't implement interfaces beyond those promoted through embedded fields), or instantiate generic functions (generics are resolved at compile time).

## Iterators in `reflect` (Go 1.23)

`Value.Seq()` and `Value.Seq2()` return iterators for any value you can `range` over: slices, maps, channels, strings, integers and iterator functions:

```go
package main

import (
	"fmt"
	"reflect"
)

func main() {
	m := map[string]int{"a": 1}
	for k, v := range reflect.ValueOf(m).Seq2() {
		fmt.Println(k.String(), v.Int())
	}
}
```

## The rules for `unsafe.Pointer`

The `unsafe` package documentation lists **six** valid patterns. Anything else is undefined behavior, even if it "works" today:

1. **Converting `*T1` → `unsafe.Pointer` → `*T2`**, if `T2` is no larger than `T1` and they have compatible layouts (e.g. `math.Float64bits`).
2. **`unsafe.Pointer` → `uintptr`** (with no conversion back), only for printing or hashing addresses.
3. **`unsafe.Pointer` → `uintptr` → arithmetic → `unsafe.Pointer`**, in a **single expression**:

```go,ignore
// VALID: all in one expression
p := unsafe.Pointer(uintptr(unsafe.Pointer(&s)) + unsafe.Offsetof(s.field))

// INVALID: between the two lines, the GC may move or free the object
u := uintptr(unsafe.Pointer(&s))
p := unsafe.Pointer(u + unsafe.Offsetof(s.field))
```

   Simpler and safe since Go 1.17: `unsafe.Add(unsafe.Pointer(&s), off)`.

4. **A `uintptr` as an argument to `syscall.Syscall`**, converted directly in the argument list.
5. **The result of `reflect.Value.Pointer`/`UnsafeAddr`** converted immediately to `unsafe.Pointer`.
6. **Converting `reflect.SliceHeader`/`StringHeader`** — **deprecated**; replaced by `unsafe.Slice`, `unsafe.SliceData`, `unsafe.String`, `unsafe.StringData` (Go 1.17–1.20).

The rule behind them all: a `uintptr` is an **integer**. The GC doesn't see it as a reference, and stacks can move at any time (see [Functions](../../functii/senior/)). Between the moment an address becomes a `uintptr` and the moment it becomes a pointer again, there must be no point where the runtime can step in.

## Checks: `go vet` and `checkptr`

- `go vet` (the `unsafeptr` analyzer) flags `uintptr` → `unsafe.Pointer` conversions that don't follow the patterns above.
- `-gcflags=all=-d=checkptr` instruments `unsafe.Pointer` conversions at run time: it checks alignment, that the resulting pointer stays within the same allocated object, and that arithmetic doesn't leave it. It's **enabled automatically** with `-race` and `-msan`.

The `checkptr` checks concern heap objects: a conversion or arithmetic that ends up outside the allocated object stops the program with a fatal `checkptr: ...` error. Pointers into the stack or into memory outside the Go heap can't be checked, so the absence of an error doesn't prove correctness.

## The compatibility promise doesn't cover `unsafe`

Code that depends on the internal layout of runtime types (a map's structure, the fields of a `reflect.Value`, offsets in `runtime.g`) can break with any new Go release. That's what happened when maps switched to Swiss tables (Go 1.24) to libraries that read the `hmap` struct directly. The `//go:linkname` restrictions in Go 1.23 serve the same purpose (see [Packages and modules](../../pachete-module/senior/)).

## cgo and pointers

For C calls, the cgo rules require that Go memory passed to C contain no Go pointers (C isn't tracked by the GC), and that C not keep Go pointers after the call. `GODEBUG=cgocheck=1` (the default) checks this partially at run time; `runtime.Pinner` (Go 1.21) pins Go objects so C can reference them for longer.

## Summary

- `reflect.Value` = type + pointer + flags (addressable, read-only).
- Unexported fields can be read, but not written or extracted as an interface.
- Reflection can create composite types and functions, but not types with new methods.
- Only the six documented patterns are valid for `unsafe.Pointer`; `unsafe.Add` and `unsafe.Slice` replace the fragile ones.
- `go vet` and `checkptr` (automatic with `-race`) catch bad conversions.
