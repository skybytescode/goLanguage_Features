# Tooling and profiling — Senior

Profile-guided optimization (PGO), what you can see inside the compiler (`-m`, SSA, assembly), reproducible and static builds, the "flight recorder" for production traces, debugging with Delve, and writing your own static analyzers.

## PGO: profile-guided optimization (Go 1.21)

The compiler can use a CPU profile from production to decide where aggressive optimizations pay off:

```bash
# 1. collect a representative profile from production
curl -o default.pgo 'http://service:6060/debug/pprof/profile?seconds=60'
# 2. put it next to the main package, as default.pgo, and commit it
mv default.pgo ./cmd/server/
# 3. the build picks it up automatically (-pgo=auto is the default)
go build ./cmd/server
```

What it does with it: raises the **inlining** budget for hot calls and **devirtualizes** interface calls when the profile shows a dominant concrete type. The typical gain reported by the Go team: 2–14% CPU. The process is iterative and stable: a profile taken from a PGO-built binary is still valid for the next build.

## Inside the compiler

```bash
go build -gcflags='-m' .        # inlining and escape analysis
go build -gcflags='-m=2' .      # with the reasons behind decisions
go build -gcflags='-S' . 2>asm.txt          # the generated assembly
go build -gcflags='-d=ssa/check_bce/debug=1' .  # the remaining bounds checks
GOSSAFUNC=FuncName go build .               # ssa.html: every step of SSA compilation
go tool objdump -s 'main\.FuncName' ./app   # disassembly of the final binary
```

`ssa.html` shows the function after each of the SSA back end's ~50 passes (dead-code elimination, constant propagation, register allocation). It's the best way to understand **why** a function isn't optimized the way you expected.

Relevant compiler directives: `//go:noinline`, `//go:nosplit` (no stack check in the prologue; runtime only), `//go:noescape` (for functions implemented in assembly), `//go:build`, `//go:embed`, `//go:generate`, `//go:debug`.

## Static and reproducible binaries

```bash
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w -buildid=' -o app ./cmd/app
```

- **`CGO_ENABLED=0`**: a fully static binary, with no libc dependency; it runs in a `scratch` or `distroless` image. (The `net` and `os/user` packages have pure-Go implementations used then.)
- **`-trimpath`**: removes absolute paths from the build machine from the binary.
- **`-s -w`**: no symbol table and no DWARF info (a smaller binary, but no debugging).

The Go toolchain has been **reproducible** since Go 1.21: the same sources, version and flags produce a bit-for-bit identical binary on any machine. Even Go's official binaries can be verified this way.

## Flight recorder (Go 1.25)

A full trace over hours is huge. The problem: when a latency spike happens, it's already too late to start tracing. `trace.FlightRecorder` **continuously** keeps the last few seconds of trace in an in-memory ring buffer, and you save it only when you detect the problem:

```go
package main

import (
	"bytes"
	"fmt"
	"runtime/trace"
	"time"
)

func main() {
	fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge:   2 * time.Second, // keep at least the last 2 seconds
		MaxBytes: 16 << 20,
	})
	if err := fr.Start(); err != nil {
		fmt.Println(err)
		return
	}
	defer fr.Stop()

	start := time.Now()
	time.Sleep(30 * time.Millisecond) // "a slow request"
	if time.Since(start) > 20*time.Millisecond {
		var buf bytes.Buffer
		if _, err := fr.WriteTo(&buf); err != nil { // save the recent trace
			fmt.Println(err)
			return
		}
		fmt.Println("trace saved:", buf.Len() > 0)
		// in production: write buf to a file and analyze it with go tool trace
	}
}
```

## Debugging: Delve

`dlv` (github.com/go-delve/delve) is the debugger that understands goroutines, interfaces and the Go runtime:

```bash
dlv debug ./cmd/server -- --port=8080
(dlv) break server.go:42
(dlv) continue
(dlv) goroutines              # every goroutine, with its current location
(dlv) goroutine 17 stack
(dlv) print req.Header
dlv attach <pid>              # to a running process
dlv core ./app core.1234      # post-mortem, on a core dump (GOTRACEBACK=crash)
```

Optimizations make debugging harder (variables eliminated, functions inlined); `-gcflags='all=-N -l'` disables them. `dlv debug` does this automatically.

## Your own static analyzers

The `golang.org/x/tools/go/analysis` framework is what `go vet`, `gopls` and the `go fix` modernizers are built on. An analyzer receives the syntax tree **with types resolved** and reports diagnostics, optionally with automatic fixes:

```go,ignore
var Analyzer = &analysis.Analyzer{
	Name: "nofmtprint",
	Doc:  "forbids fmt.Print* in production packages; use log/slog",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func); ok &&
				fn.Pkg() != nil && fn.Pkg().Path() == "fmt" && strings.HasPrefix(fn.Name(), "Print") {
				pass.Reportf(call.Pos(), "fmt.%s in production code", fn.Name())
			}
			return true
		})
	}
	return nil, nil
}
```

Run it with `singlechecker.Main(Analyzer)` as a standalone tool, or as a `golangci-lint` plugin. For simple checks, `go vet -vettool=...` accepts any binary built with `unitchecker`.

## Summary

- PGO: `default.pgo` next to `main` → production-guided inlining and devirtualization.
- `-gcflags=-m`, `GOSSAFUNC`, `-S` and `objdump` show what the compiler did.
- `CGO_ENABLED=0 -trimpath` for static binaries; Go builds are reproducible.
- `trace.FlightRecorder` (Go 1.25) saves the trace only when the problem occurs.
- Delve for debugging; `go/analysis` for your own analyzers.
