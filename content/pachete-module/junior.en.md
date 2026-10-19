# Packages and modules — Junior

Go code is organized in **packages** (a directory of `.go` files) grouped into **modules** (a project with a `go.mod` file). The module has a version and dependencies; the package is the unit of compilation and visibility.

## The package

Every file starts with `package name`. All files in a directory belong to the same package and see everything declared in the other files, with no imports:

```go,ignore
// geometry/area.go
package geometry

func CircleArea(r float64) float64 { return pi * r * r }
```

```go,ignore
// geometry/constants.go
package geometry

const pi = 3.14159 // visible throughout package geometry, but not outside it
```

The `main` package is special: it produces an executable program, and `func main()` is the entry point.

## Exported or not: the capital letter

A name starting with an **uppercase letter** (`CircleArea`) is **exported**: visible from other packages. With a lowercase letter (`pi`), it's private to the package. There's no `public` or `private` in Go.

## Creating a module

```bash
mkdir hello && cd hello
go mod init github.com/ana/hello
```

This creates `go.mod`:

```text
module github.com/ana/hello

go 1.26
```

The module path (`github.com/ana/hello`) is the prefix of every package in it and, usually, the repository's address.

## Imports

Standard library packages are imported by short name; those from modules by full path:

```go
package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

func main() {
	fmt.Println(strings.ToUpper("go"), rand.IntN(10) < 10)
}
```

The name you use in code is the path's last element (`rand`, not `math/rand/v2`; `v2` is a version suffix, see the Middle level).

## The layout of a small project

```text
hello/
├── go.mod
├── main.go            // package main
└── greetings/
    ├── greetings.go   // package greetings
    └── greetings_test.go
```

```go,ignore
// main.go
package main

import (
	"fmt"

	"github.com/ana/hello/greetings" // module path + directory
)

func main() {
	fmt.Println(greetings.For("Ana"))
}
```

## External dependencies

`go get` adds a dependency to `go.mod` and downloads the code:

```bash
go get github.com/google/uuid@latest
```

```text
module github.com/ana/hello

go 1.26

require github.com/google/uuid v1.6.0
```

The `go.sum` file holds the checksums (hashes) of every dependency. Both files go into git.

## The basic commands

| Command | What it does |
|---|---|
| `go run .` | compiles and runs the package in the current directory |
| `go build` | creates the executable |
| `go test ./...` | runs the tests in every package |
| `go mod tidy` | adds missing dependencies, removes unused ones |
| `go fmt ./...` | formats the code |
| `go vet ./...` | looks for common mistakes |

`./...` means "the current directory and all subdirectories".

## Import aliases

When two packages have the same name, or the name is too long, you can give an alias:

```go
package main

import (
	crand "crypto/rand"
	"fmt"
	mrand "math/rand/v2"
)

func main() {
	b := make([]byte, 4)
	_, _ = crand.Read(b) // cryptographically random
	fmt.Println(len(b), mrand.IntN(6)+1)
}
```

## Summary

- One directory = one package; one `go.mod` = one module.
- Uppercase = exported; lowercase = private to the package.
- `go mod init path` creates the module; `go get` adds dependencies.
- `go.mod` and `go.sum` go into version control.
- `go mod tidy` keeps the dependency list up to date.
