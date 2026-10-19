# Reflection and unsafe — Middle

Modifying values through reflection (the addressability rule), walking structs generically, calling methods, `reflect.DeepEqual` and the real cost of reflection. Then, first steps with `unsafe.Pointer`.

## The three laws of reflection

1. From interface to reflection object: `reflect.ValueOf(x)`.
2. From reflection object back to interface: `v.Interface()`.
3. To **modify** a `reflect.Value`, it must be **settable**: it must come from a pointer.

```go
package main

import (
	"fmt"
	"reflect"
)

func main() {
	x := 3.4
	v := reflect.ValueOf(x)
	fmt.Println(v.CanSet()) // false: v holds a copy of x

	p := reflect.ValueOf(&x).Elem() // Elem: the value the pointer points to
	fmt.Println(p.CanSet())         // true
	p.SetFloat(7.1)
	fmt.Println(x) // 7.1
}
```

`v.SetFloat` on a value that isn't settable **panics**; it doesn't return an error.

## Example: a tag-driven validator

A real use of reflection: a function that works on **any** struct, guided by tags:

```go
package main

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Validate(s any) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return errors.New("expected a struct")
	}
	var errs []error
	t := v.Type()
	for i := range t.NumField() {
		field, value := t.Field(i), v.Field(i)
		for _, rule := range strings.Split(field.Tag.Get("validate"), ",") {
			switch {
			case rule == "required" && value.IsZero():
				errs = append(errs, fmt.Errorf("%s: required", field.Name))
			case strings.HasPrefix(rule, "min="):
				min, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
				if value.Kind() == reflect.Int && value.Int() < int64(min) {
					errs = append(errs, fmt.Errorf("%s: minimum %d", field.Name, min))
				}
			}
		}
	}
	return errors.Join(errs...)
}

type Account struct {
	Name  string `validate:"required"`
	Email string `validate:"required"`
	Age   int    `validate:"min=18"`
}

func main() {
	fmt.Println(Validate(Account{Name: "Ana", Age: 15}))
	fmt.Println(Validate(&Account{Name: "Ion", Email: "i@x.ro", Age: 30})) // <nil>
}
```

## Calling methods

```go
package main

import (
	"fmt"
	"reflect"
)

type Calculator struct{}

func (Calculator) Add(a, b int) int      { return a + b }
func (Calculator) Multiply(a, b int) int { return a * b }

func main() {
	c := reflect.ValueOf(Calculator{})
	for _, name := range []string{"Add", "Multiply", "Divide"} {
		m := c.MethodByName(name)
		if !m.IsValid() {
			fmt.Println(name, ": doesn't exist")
			continue
		}
		res := m.Call([]reflect.Value{reflect.ValueOf(6), reflect.ValueOf(7)})
		fmt.Println(name, "=", res[0].Int())
	}
}
```

Only **exported** methods are visible through reflection.

## `reflect.DeepEqual`

Compares two values recursively, including slices and maps, which `==` doesn't accept:

```go
package main

import (
	"fmt"
	"reflect"
)

func main() {
	a := map[string][]int{"x": {1, 2}}
	b := map[string][]int{"x": {1, 2}}
	fmt.Println(reflect.DeepEqual(a, b)) // true

	var nilSlice []int
	fmt.Println(reflect.DeepEqual(nilSlice, []int{})) // false: nil and empty differ
}
```

In tests, `slices.Equal` and `maps.Equal` are faster and clearer when the types are known; `DeepEqual` remains for arbitrary nested structures (or `github.com/google/go-cmp`, with readable diffs).

## The cost

```go
package main

import (
	"fmt"
	"reflect"
	"testing"
)

type P struct{ X, Y int }

func main() {
	p := P{1, 2}
	direct := testing.Benchmark(func(b *testing.B) {
		for b.Loop() {
			_ = p.X + p.Y
		}
	})
	viaReflect := testing.Benchmark(func(b *testing.B) {
		for b.Loop() {
			v := reflect.ValueOf(p)
			_ = v.Field(0).Int() + v.Field(1).Int()
		}
	})
	fmt.Printf("direct: %v/op, reflect: %v/op\n",
		direct.NsPerOp(), viaReflect.NsPerOp())
}
```

Libraries like `encoding/json` reduce the cost by analyzing a type **only once** and caching the result (field indices, conversion functions).

## `unsafe.Pointer`: first steps

`unsafe.Pointer` is a pointer to anything; it can be converted to and from any other pointer type. A legitimate, common use: reinterpreting memory between types with **the same layout**:

```go
package main

import (
	"fmt"
	"math"
	"unsafe"
)

func main() {
	f := 1.5
	bits := *(*uint64)(unsafe.Pointer(&f)) // the same memory, read as a uint64
	fmt.Printf("%#016x\n", bits)
	fmt.Println(bits == math.Float64bits(f)) // the standard library does exactly this
}
```

For almost every case like this, a safe function already exists in the standard library (`math.Float64bits`, `binary.LittleEndian.Uint32`, `unsafe.String`). Use those.

## Summary

- To modify through reflection, start from a pointer and use `Elem()`.
- Reflection is ideal for functions that work on any struct, guided by tags.
- `MethodByName` + `Call` invoke methods dynamically; exported ones only.
- `DeepEqual` compares recursively, but distinguishes `nil` from empty.
- `unsafe.Pointer` reinterprets memory; prefer the safe standard library functions.
