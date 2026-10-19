# Errors, defer, panic and recover — Middle

In real code, an error passes through several layers. Each layer adds context, and the code at the top must be able to find out what happened at the bottom. That's what **wrapping**, `errors.Is` and `errors.As` are for.

## Wrapping with `%w`

`fmt.Errorf` with the `%w` verb creates a new error that **contains** the original one:

```go
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func readConfig(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %q: %w", path, err)
	}
	return data, nil
}

func main() {
	_, err := readConfig("/does/not/exist.yaml")
	fmt.Println(err)
	fmt.Println(errors.Is(err, fs.ErrNotExist)) // true: found in the chain
}
```

With `%v` instead of `%w` the message would be identical, but the link to the original error would be lost.

## Sentinel errors and `errors.Is`

A **sentinel error** is an exported variable you compare against: `io.EOF`, `fs.ErrNotExist`, `sql.ErrNoRows`. `errors.Is` searches the whole wrapping chain, so use it instead of `==`:

```go
package main

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

func find(id int) error {
	if id != 1 {
		return fmt.Errorf("user %d: %w", id, ErrNotFound)
	}
	return nil
}

func main() {
	err := find(7)
	fmt.Println(err == ErrNotFound)          // false: it's wrapped
	fmt.Println(errors.Is(err, ErrNotFound)) // true
}
```

## Error types and `errors.As` / `errors.AsType`

When the caller needs **data** from the error (a code, a field), define a type. `errors.As` finds the first error of that type in the chain. Since Go 1.26, the generic `errors.AsType` is shorter and safer:

```go
package main

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Reason }

func validate(age int) error {
	if age < 0 {
		return fmt.Errorf("registration: %w", &ValidationError{"age", "negative"})
	}
	return nil
}

func main() {
	err := validate(-3)

	var ve *ValidationError
	if errors.As(err, &ve) {
		fmt.Println("invalid field:", ve.Field)
	}

	if ve, ok := errors.AsType[*ValidationError](err); ok { // Go 1.26
		fmt.Println("reason:", ve.Reason)
	}
}
```

## Several errors: `errors.Join`

When you validate several things, you can return all the errors at once. `errors.Is` and `errors.As` search each of them:

```go
package main

import (
	"errors"
	"fmt"
)

var (
	ErrName  = errors.New("missing name")
	ErrEmail = errors.New("invalid email")
)

func validate(name, email string) error {
	var errs []error
	if name == "" {
		errs = append(errs, ErrName)
	}
	if email == "" {
		errs = append(errs, ErrEmail)
	}
	return errors.Join(errs...) // nil if the list is empty
}

func main() {
	err := validate("", "")
	fmt.Println(err)
	fmt.Println(errors.Is(err, ErrEmail)) // true
}
```

## What to choose

| Situation | Solution |
|---|---|
| the caller doesn't react differently by cause | `fmt.Errorf("context: %w", err)` |
| the caller checks for a specific case | sentinel error + `errors.Is` |
| the caller needs details | error type + `errors.As` / `errors.AsType` |
| you don't want callers to depend on the internal error | `%v` instead of `%w` (no wrapping) |

Wrapping makes the internal error part of your **API**: callers will start depending on it.

## Context, not repetition

Each layer adds **what it was doing**, not "error while…" or "failed to…". The result reads like a chain:

```text
starting server: loading config: reading config "app.yaml": open app.yaml: no such file or directory
```

And handle an error **once**: either log it or return it. Not both, or it shows up several times in the log.

## `defer` that modifies the result

A `defer` can read and modify **named** results. It's useful for adding context to every error in a function, or for errors from `Close`:

```go
package main

import (
	"errors"
	"fmt"
)

type resource struct{}

func (resource) Close() error { return errors.New("flush failed") }

func process() (err error) {
	r := resource{}
	defer func() {
		if cerr := r.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing: %w", cerr) // don't lose the error from Close
		}
	}()
	return nil
}

func main() {
	fmt.Println(process())
}
```

For files opened for **writing**, the error from `Close` matters: the data may not have been written.

## `panic` in libraries

A library shouldn't panic on bad input; it returns an `error`. The accepted exception: `MustX` functions (`regexp.MustCompile`, `template.Must`), used during initialization with constant values, where an error is a bug anyway.

## Summary

- `%w` wraps; `errors.Is` and `errors.As` search the whole chain.
- Sentinels for cases, types for details, `errors.Join` for lists.
- Each layer adds short context; handle an error only once.
- Named results + `defer` catch errors from `Close`.
