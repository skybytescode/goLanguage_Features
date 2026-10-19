# Testing, benchmarks and fuzzing — Senior

Coverage-guided fuzzing, sound benchmark methodology (noise, `benchstat`, dead-code elimination), how the `synctest` bubble works, the test cache, coverage for whole binaries, and organizing internal vs. external tests.

## Fuzzing (Go 1.18)

A fuzz test receives automatically generated inputs and checks **properties** that must hold for any input:

```go,ignore
func FuzzReverse(f *testing.F) {
	f.Add("hello") // initial corpus (seed)
	f.Add("ăîșț")
	f.Fuzz(func(t *testing.T, s string) {
		rev := Reverse(s)
		double := Reverse(rev)
		if utf8.ValidString(s) && s != double {
			t.Errorf("Reverse(Reverse(%q)) = %q", s, double)
		}
		if utf8.ValidString(s) && !utf8.ValidString(rev) {
			t.Errorf("Reverse(%q) produced invalid UTF-8: %q", s, rev)
		}
	})
}
```

```bash
go test -fuzz=FuzzReverse -fuzztime=30s
```

How it works:

- the fuzzer builds the package with **coverage instrumentation** (branch counters, libFuzzer-style);
- it starts several worker processes that **mutate** corpus inputs (flipping bits, inserting bytes, combining inputs) and keep those that reach new branches;
- on a failure, it **minimizes** the input and saves it to `testdata/fuzz/FuzzReverse/<hash>`;
- the saved file becomes a regression case: a plain `go test` (without `-fuzz`) always runs the corpus in `testdata`.

The accepted argument types are the basic ones: `string`, `[]byte`, numbers, `bool`, `rune`. Good fuzzing properties: round-tripping (`Decode(Encode(x)) == x`), comparison against a reference implementation, no panics on arbitrary input (parsers!).

## Sound benchmarks

A benchmark is a **statistical measurement**, not a number. Sources of noise: variable CPU frequency, other processes, the GC, code alignment in memory. The methodology:

```bash
go test -bench=. -count=10 -benchmem > old.txt
# ... the change ...
go test -bench=. -count=10 -benchmem > new.txt
benchstat old.txt new.txt            # golang.org/x/perf/cmd/benchstat
```

```text
          │   old.txt   │              new.txt               │
          │   sec/op    │   sec/op     vs base               │
Parse-8     1.204µ ± 2%   0.981µ ± 1%  -18.52% (p=0.000 n=10)
```

`benchstat` reports the median, the variation interval and whether the difference is statistically significant. Without `-count` and `benchstat`, "10% faster" may just be noise.

**Dead-code elimination.** The compiler can notice that a pure function's result is unused and drop the call, so the benchmark measures nothing. With the classic `for i := 0; i < b.N; i++` loop, people used global "sink" variables. `b.Loop` (Go 1.24) prevents this optimization for calls in the loop body and, in addition, runs the benchmark function only once (with `b.N` it ran several times, repeating the setup).

Other rules: `b.ReportAllocs()` or `-benchmem` for allocations, `-cpu=1,4,8` for scaling, `b.RunParallel` for concurrent code, `b.SetBytes(n)` for MB/s throughput.

## Inside the `synctest` bubble

`synctest.Test` starts the function in a new goroutine marked as belonging to a **bubble**. Every goroutine started from it belongs to the same bubble. The bubble has its own **clock**, which starts at midnight UTC on January 1, 2000 and doesn't advance on its own.

The runtime tracks whether every goroutine in the bubble is **durably blocked** — blocked in operations only other goroutines in the bubble can unblock: operations on channels created in the bubble, `time.Sleep`, `sync.WaitGroup.Wait`, `sync.Cond.Wait`, `select` on such channels. When that happens:

- if there are timers in the bubble, the clock **jumps** straight to the next timer, which fires;
- if there are no timers, the test is **deadlocked** and fails with an explicit message.

Operations waiting on something outside the bubble (real network I/O, mutexes, channels created outside the bubble) are **not** durable blocking: the bubble can't advance time while a goroutine sits in them. That's why code tested with `synctest` uses `net.Pipe` or in-memory implementations instead of the real network.

## The test cache

`go test` keeps the results of passing tests and reuses them (`(cached)` in the output) if nothing changed: the sources, the flags, the environment variables **read** by the test and the files **opened** by the test (tracked through `os.Open`, `os.Stat`, etc.). Results are only cached in package-list mode (`go test ./...`), not in local-directory mode (plain `go test`). `-count=1` disables the cache.

## Coverage for whole binaries (Go 1.20)

Coverage isn't limited to unit tests. A binary built with `-cover` writes coverage data on exit, useful for integration or end-to-end tests:

```bash
go build -cover -o app ./cmd/app
GOCOVERDIR=./coverage ./app --scenario=full
go tool covdata percent -i=./coverage
go tool covdata textfmt -i=./coverage -o profile.txt   # then go tool cover -html
```

## Internal and external tests

- `package calc` (in `calc_test.go`) — an **internal** test: access to everything, including unexported identifiers.
- `package calc_test` — an **external** test: only the public API, exactly like a user. It also breaks import cycles (a test that needs a package that imports `calc`).

When an external test still needs something internal, the convention is an `export_test.go` file in the internal package that exposes it for tests only:

```go,ignore
// export_test.go
package calc

var ParseInternal = parse // visible only to calc_test
```

## Summary

- Coverage-guided fuzzing; found failures become regression cases in `testdata/fuzz`.
- Benchmarks: `-count=10` + `benchstat`; `b.Loop` prevents dead-code elimination.
- `synctest` advances a fake clock when every goroutine in the bubble is durably blocked.
- The test cache accounts for the files and environment variables a test reads.
- `-cover` and `GOCOVERDIR` for binary coverage; `export_test.go` for external tests.
