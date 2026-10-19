# I/O: Reader and Writer — Junior

Almost everything that reads or writes data in Go — files, the network, in-memory buffers, compression — uses two small interfaces from the `io` package: `io.Reader` and `io.Writer`.

## The two interfaces

```go,ignore
type Reader interface {
	Read(p []byte) (n int, err error) // fills p with up to len(p) bytes
}

type Writer interface {
	Write(p []byte) (n int, err error) // writes the bytes in p
}
```

A file, a network connection, a `strings.Reader`, a `bytes.Buffer` or `os.Stdout` all implement them. A function written for an `io.Reader` works with all of them.

## Reading and writing a whole file

For small files, the simplest functions are `os.WriteFile` and `os.ReadFile`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	path := filepath.Join(os.TempDir(), "hello.txt")

	err := os.WriteFile(path, []byte("Hello from Go!\n"), 0o644)
	if err != nil {
		fmt.Println("write:", err)
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Print(string(data))
	os.Remove(path)
}
```

`0o644` are the file permissions (read/write for the owner, read for everyone else).

## Opening and closing

For large files or gradual reading, open the file and close it with `defer`:

```go
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	path := filepath.Join(os.TempDir(), "data.txt")
	os.WriteFile(path, []byte("abcdefghij"), 0o644)
	defer os.Remove(path)

	f, err := os.Open(path) // read-only
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	buf := make([]byte, 4)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			fmt.Printf("read %q\n", buf[:n])
		}
		if err == io.EOF { // end of file
			break
		}
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
```

`io.EOF` isn't really an error: it means "no more data".

## Reading line by line: `bufio.Scanner`

```go
package main

import (
	"bufio"
	"fmt"
	"strings"
)

func main() {
	text := "first line\nsecond line\nthird line"
	scanner := bufio.NewScanner(strings.NewReader(text))
	n := 0
	for scanner.Scan() {
		n++
		fmt.Printf("%d: %s\n", n, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("error:", err)
	}
}
```

The same code works with a file (`bufio.NewScanner(f)`) or standard input (`bufio.NewScanner(os.Stdin)`).

## Writing with `fmt.Fprintf`

`fmt.Fprintf` writes formatted output to **any** `io.Writer`:

```go
package main

import (
	"bytes"
	"fmt"
	"os"
)

func report(w *bytes.Buffer, name string, score int) {
	fmt.Fprintf(w, "%-6s %3d\n", name, score)
}

func main() {
	var buf bytes.Buffer
	report(&buf, "Ana", 95)
	report(&buf, "Ion", 87)

	fmt.Fprint(os.Stdout, buf.String()) // os.Stdout is an io.Writer too
	fmt.Fprintln(os.Stderr, "error messages go to stderr")
}
```

## Creating a file

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	path := filepath.Join(os.TempDir(), "list.txt")
	f, err := os.Create(path) // creates or truncates the file
	if err != nil {
		fmt.Println(err)
		return
	}
	for i := 1; i <= 3; i++ {
		fmt.Fprintf(f, "item %d\n", i)
	}
	if err := f.Close(); err != nil { // when writing, check the error from Close
		fmt.Println(err)
		return
	}

	data, _ := os.ReadFile(path)
	fmt.Print(string(data))
	os.Remove(path)
}
```

## File paths

Use `path/filepath` for paths on disk: it picks the right separator (`/` or `\`) for the current system.

```go
package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	path := filepath.Join("data", "2026", "report.csv")
	fmt.Println(path)
	fmt.Println(filepath.Base(path), filepath.Ext(path), filepath.Dir(path))
}
```

## Summary

- `io.Reader` and `io.Writer` are everywhere: files, the network, memory.
- `os.ReadFile`/`os.WriteFile` for small files.
- `os.Open` + `defer f.Close()` for gradual reading; `io.EOF` marks the end.
- `bufio.Scanner` reads line by line.
- `fmt.Fprintf` writes formatted output to any `Writer`.
