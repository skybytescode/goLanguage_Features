# Packages and modules — Senior

How the `go` command picks versions (Minimal Version Selection), how dependencies are verified (proxy, checksum database), how `toolchain` and `GOTOOLCHAIN` work, build constraints, and what the linker does with unused code.

## Minimal Version Selection (MVS)

Most package managers (npm, Cargo, pip) resolve dependencies by looking for the **newest** version satisfying every constraint, with a SAT solver and a lock file. Go does something else:

> For each module, pick the **highest** of the **minimum** versions required by anyone in the dependency graph.

```text
app        requires  A v1.2, B v1.0
A v1.2     requires  C v1.3
B v1.0     requires  C v1.5

→ C v1.5 is used (the maximum of the requirements), even if C v1.9 exists
```

Consequences:

- **Reproducible without a lock file**: the same `go.mod` files always yield the same build; a new version published tomorrow changes nothing until you upgrade explicitly.
- **No solver**: the algorithm is a linear graph walk.
- **No upper bounds**: you can't say "`< v1.6`". Incompatible versions are excluded with `exclude` in the main module, or retracted by their author with `retract`.

Since Go 1.17, `go.mod` lists **every** module needed for the imported packages (graph pruning), so not every `go.mod` in the graph has to be read. That's why a modern `go.mod` has two `require` blocks, the second with `// indirect` dependencies.

## Proxy and checksum database

The `go` command doesn't download straight from git. By default:

```bash
GOPROXY=https://proxy.golang.org,direct
GOSUMDB=sum.golang.org
```

- The **proxy** serves immutable zip archives for each `module@version`. A tag deleted or moved in git doesn't change what others receive.
- The **checksum database** is a transparent log (a Merkle tree, like Certificate Transparency) with the hash of every public version. On first download, the hash is checked against the log and written to `go.sum`. An attacker can't serve different content for the same version without being detected.
- For private modules: `GOPRIVATE=github.com/company/*` excludes them from the proxy and the checksum database (but `go.sum` still verifies them).

## `toolchain` and `GOTOOLCHAIN` (Go 1.21)

The `go 1.26` line in `go.mod` is a **minimum requirement**, not a suggestion: an older toolchain refuses the module. If you have Go 1.24 installed and build a module requiring 1.26, the `go` command **automatically downloads** the right toolchain (as a module, through the proxy) and uses it:

```text
module github.com/ana/app

go 1.26
toolchain go1.26.4
```

```bash
GOTOOLCHAIN=local     # never download; fail if the version is too old
GOTOOLCHAIN=go1.26.4  # use exactly this version
go get go@1.26        # raise the module's minimum requirement
```

The `go` line also controls **language semantics**: for example, per-iteration loop variables (Go 1.22) only apply to packages in modules with `go 1.22` or newer. Many standard library behavior changes are tied to this version through default `GODEBUG` settings.

## Build constraints

A file can be included or excluded by OS, architecture or tags:

```go,ignore
//go:build linux && (amd64 || arm64) && !nocgo

package system
```

Filename suffixes apply constraints automatically too: `file_linux.go`, `file_windows_amd64.go`, `file_test.go`. Custom tags are enabled with `go build -tags=integration`. The `goexperiment.X` tag reflects the toolchain's active experiments.

## The unit of compilation: the package

The compiler (`go tool compile`) receives **all of a package's files** at once and produces an object file with **export data**: the types, the inlinable functions and the escape information of the exported API. Packages importing it read only that data, not the sources. Hence:

- packages compile in parallel, in the topological order of imports;
- import cycles are impossible by construction;
- a function from another package can be inlined (its body is in the export data).

Results are kept in the **build cache** (`go env GOCACHE`), keyed by the hash of every input (sources, flags, toolchain version, dependencies). A repeated build with no changes recompiles nothing; `go clean -cache` empties it.

## The linker and dead code

The linker starts from `main.main` and the `init` functions and includes only **reachable** symbols. Unused functions disappear from the binary. Exceptions that keep code alive:

- when the program uses `reflect.Value.Method`/`MethodByName` with names unknown at compile time, the linker must keep **every** exported method of every reachable type (that's why `text/template` can bloat binaries);
- `init` functions and the globals they initialize.

```bash
go build -ldflags='-s -w'   # no symbol table and no DWARF: a smaller binary
go version -m ./app         # the modules and versions embedded in the binary
```

## `//go:linkname`

The directive allows access to unexported symbols from another package (for example the runtime). Many popular libraries used it, blocking internal runtime changes. Since Go 1.23, the linker **rejects** `linkname` references to standard library symbols that aren't explicitly marked as allowed (`-checklinkname=0` disables the check temporarily). The message is clear: don't build on runtime internals.

## `retract`

An author can retract a mistakenly published version without deleting it (the proxy is immutable):

```text
retract (
	v1.4.0 // contains a serious regression in Parse
	[v1.5.0, v1.5.3]
)
```

`go get @latest` skips retracted versions, and `go list -m -u` flags them for those already using them.

## Summary

- MVS: the maximum of the minimum versions; reproducible without a lock file.
- An immutable proxy + a transparent checksum database = a verifiable supply chain.
- The `go` line is a requirement and controls semantics; `GOTOOLCHAIN` fetches the right version.
- The package is the unit of compilation; export data enables cross-package inlining.
- The linker drops unreachable code; `linkname` into the runtime is restricted since 1.23.
