# Testing, benchmarks and fuzzing — Middle

The `testing` tools that keep tests clean and fast (`t.Helper`, `t.Cleanup`, `t.TempDir`, `t.Parallel`, `t.Context`), "Example" tests, replacing dependencies through interfaces, benchmarks with `b.Loop`, and testing code with time and goroutines through `testing/synctest`.

## Helper functions: `t.Helper`

A shared checking function reports, by default, the line **inside it**. `t.Helper()` makes the error point to the line in the test that called it:

```go,ignore
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
```

## Temporary resources: `TempDir`, `Cleanup`, `Setenv`, `Context`

```go,ignore
func TestExport(t *testing.T) {
	dir := t.TempDir()         // created now, removed automatically at the end
	t.Setenv("APP_MODE", "test") // restored automatically at the end

	db := openDB(t)
	t.Cleanup(func() { db.Close() }) // runs after the test, in reverse order

	ctx := t.Context() // Go 1.24: canceled just before Cleanup
	if err := Export(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
}
```

`t.Cleanup` beats `defer` in helper functions: the cleanup runs at the end of the **test**, not the helper.

## Parallel tests

`t.Parallel()` lets the test run alongside the package's other parallel tests. For sub-tests, each case runs concurrently:

```go,ignore
for _, c := range cases {
	t.Run(c.name, func(t *testing.T) {
		t.Parallel()      // since Go 1.22, c is a new variable per iteration: safe
		check(t, c)
	})
}
```

Parallel tests must not depend on shared global state (global variables, the current directory, environment variables: `t.Setenv` and `t.Chdir` refuse to run in parallel tests).

## Examples that are also tests

An `ExampleXxx` function in a test file shows up in the documentation (`go doc`, pkg.go.dev) and is **verified**: its output must match the `// Output:` comment:

```go,ignore
func ExampleAdd() {
	fmt.Println(Add(2, 3))
	// Output: 5
}
```

## Replacing dependencies through interfaces

Code that depends on a small interface can be tested with a fake implementation, with no mocking library:

```go
package main

import (
	"errors"
	"fmt"
	"time"
)

type Clock interface{ Now() time.Time }

type Sender interface {
	Send(to, message string) error
}

// Reminders is the code under test: it depends only on interfaces.
type Reminders struct {
	clock  Clock
	sender Sender
}

func (r Reminders) Send(email string, due time.Time) error {
	if r.clock.Now().After(due) {
		return errors.New("deadline passed")
	}
	return r.sender.Send(email, "the deadline is "+due.Format("2006-01-02"))
}

// --- fake implementations, as they'd be in the test file ---

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type fakeSender struct{ sent []string }

func (s *fakeSender) Send(to, message string) error {
	s.sent = append(s.sent, to+": "+message)
	return nil
}

func main() {
	today := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	s := &fakeSender{}
	r := Reminders{clock: fixedClock{today}, sender: s}

	fmt.Println(r.Send("ana@x.ro", today.AddDate(0, 0, 3)))  // <nil>
	fmt.Println(r.Send("ion@x.ro", today.AddDate(0, 0, -1))) // deadline passed
	fmt.Println(s.sent)
}
```

## Benchmarks with `b.Loop` (Go 1.24)

```go,ignore
func BenchmarkJoin(b *testing.B) {
	parts := []string{"a", "b", "c", "d"}
	for b.Loop() { // replaces for i := 0; i < b.N; i++
		_ = strings.Join(parts, ",")
	}
}
```

```bash
$ go test -bench=Join -benchmem
BenchmarkJoin-8   	20000000	  55.1 ns/op	  8 B/op	  1 allocs/op
```

`b.Loop` runs the setup before the loop only once, automatically stops the timer outside the loop, and prevents the compiler from eliminating the measured code as "unused". The same mechanism, from an ordinary program:

```go
package main

import (
	"fmt"
	"strings"
	"testing"
)

func main() {
	parts := []string{"a", "b", "c", "d"}
	join := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = strings.Join(parts, ",")
		}
	})
	plus := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			s := ""
			for _, p := range parts {
				s += p + ","
			}
			_ = s
		}
	})
	fmt.Println("Join:", join.AllocsPerOp(), "allocs/op")
	fmt.Println("+=  :", plus.AllocsPerOp(), "allocs/op")
}
```

## Code with time and goroutines: `testing/synctest` (Go 1.25)

Testing a 5-second timeout shouldn't take 5 seconds. `synctest.Test` runs the test in a "bubble" with a **fake clock**: time advances instantly whenever every goroutine in the bubble is blocked.

```go,ignore
func TestExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		time.Sleep(4 * time.Second) // instant
		if ctx.Err() != nil {
			t.Fatal("expired too early")
		}
		time.Sleep(2 * time.Second)
		synctest.Wait() // waits until every goroutine in the bubble is blocked
		if ctx.Err() == nil {
			t.Fatal("should have expired")
		}
	})
}
```

## Separate integration tests

Slow tests, or tests that need external services, are marked so they don't run by default:

```go,ignore
func TestRealDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("slow; run without -short")
	}
	// ...
}
```

Alternatively, a build tag (`//go:build integration`) and `go test -tags=integration`.

## Summary

- `t.Helper`, `t.Cleanup`, `t.TempDir`, `t.Setenv`, `t.Context` keep tests clean.
- `t.Parallel` speeds up independent tests.
- `ExampleXxx` with `// Output:` = verified documentation.
- Small interfaces + fakes instead of mocking libraries.
- `b.Loop` for benchmarks; `synctest` for code with time and goroutines.
