# Reflection and unsafe — Junior

**Reflection** means a program can inspect its own types and values at run time: what type a variable has, what fields a struct has, what tags they carry. The `reflect` package makes this possible. It's powerful, but you'll rarely use it directly; `fmt`, `encoding/json` and many other libraries use it behind the scenes.

## The type of a value

```go
package main

import (
	"fmt"
	"reflect"
)

type Point struct{ X, Y int }

func main() {
	values := []any{42, "hello", 3.14, []int{1, 2}, Point{1, 2}, &Point{}}
	for _, v := range values {
		t := reflect.TypeOf(v)
		fmt.Printf("%-12v kind=%v\n", t, t.Kind())
	}
}
```

- **`Type`** is the exact type (`main.Point`, `[]int`).
- **`Kind`** is the basic category: `Int`, `String`, `Slice`, `Struct`, `Ptr`, etc. Many different types can share the same `Kind`.

`fmt.Printf("%T", v)` uses exactly this mechanism.

## The value: `reflect.ValueOf`

`reflect.Value` lets you read the value, whatever its type:

```go
package main

import (
	"fmt"
	"reflect"
)

func describe(x any) {
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Int, reflect.Int64:
		fmt.Println("integer:", v.Int())
	case reflect.String:
		fmt.Println("text of length", v.Len())
	case reflect.Slice:
		fmt.Println("slice with", v.Len(), "elements")
	default:
		fmt.Println("other type:", v.Kind())
	}
}

func main() {
	describe(7)
	describe("Go")
	describe([]string{"a", "b", "c"})
	describe(true)
}
```

For cases like this, a **type switch** (see [Interfaces](../../interfete/junior/)) is usually simpler. Reflection becomes necessary when you don't know the types in advance: for example, any user-defined struct.

## Struct fields and tags

This is how packages like `encoding/json` read tags:

```go
package main

import (
	"fmt"
	"reflect"
)

type User struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email"`
	Age   int    `json:"age,omitempty"`
}

func main() {
	t := reflect.TypeFor[User]() // Go 1.22; same as reflect.TypeOf(User{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Printf("%-6s %-7v json=%q validate=%q\n",
			f.Name, f.Type, f.Tag.Get("json"), f.Tag.Get("validate"))
	}
}
```

## When to use reflection

Rarely. Reflection:

- is **slower** than ordinary code (by tens of times, in many cases);
- moves errors from compile time to **run time** (a panic instead of a compiler error);
- makes code harder to read.

Use it for general-purpose libraries (serialization, validation, ORMs, dependency injection). For everything else, interfaces and generics are better.

## What `unsafe` is

The `unsafe` package allows operations Go's type system forbids: reading memory as another type, pointer arithmetic. The name says it all: code using it can corrupt memory and no longer benefits from Go's compatibility guarantees.

The harmless part, useful for learning: functions that tell you how much memory a type occupies.

```go
package main

import (
	"fmt"
	"unsafe"
)

type Example struct {
	A bool
	B int64
	C bool
}

func main() {
	fmt.Println(unsafe.Sizeof(int32(0)))      // 4
	fmt.Println(unsafe.Sizeof(""))            // 16: pointer + length
	fmt.Println(unsafe.Sizeof([]int{}))       // 24: pointer + len + cap
	fmt.Println(unsafe.Sizeof(Example{}))     // 24: includes padding
	fmt.Println(unsafe.Offsetof(Example{}.B)) // 8: B starts after 7 bytes of padding
}
```

The values are for a 64-bit platform.

## Summary

- `reflect.TypeOf` gives the type, `reflect.ValueOf` the value; `Kind` is the category.
- Struct fields and tags are read with `Type.Field(i)`.
- Reflection is slow and moves errors to run time: use it only in general libraries.
- `unsafe` bypasses the type system; `Sizeof` and `Offsetof` help you understand memory.
