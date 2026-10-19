# I/O: Reader and Writer — Senior

The exact contract of `Read` and `Write`, the zero-copy optimizations in `io.Copy` (`sendfile`, `splice`), how files and sockets interact with the runtime, and how to write to disk so data survives a power failure.

## The `Read` contract

The `io.Reader` documentation has precise rules that every implementation and every caller must follow:

- `Read` may return **fewer** bytes than `len(p)`, even before the end. Don't assume the buffer is full; for exactly `n` bytes use `io.ReadFull`.
- It may return `n > 0` **and** a non-nil error (including `io.EOF`) in the same call. The caller processes the `n` bytes first.
- Returning `0, nil` is discouraged (it doesn't mean EOF); callers treat it as "nothing new".
- The implementation must not retain `p` after returning.

```go
package main

import (
	"fmt"
	"io"
	"strings"
)

func main() {
	r := strings.NewReader("abc")
	buf := make([]byte, 5)

	_, err := io.ReadFull(r, buf) // asks for exactly 5
	fmt.Println(err)              // unexpected EOF: there were only 3

	r = strings.NewReader("abcdef")
	n, err := io.ReadAtLeast(r, buf, 2)
	fmt.Println(n, err) // 5 <nil>
}
```

## The `Write` contract

`Write` must return a non-nil error if `n < len(p)`. Unlike POSIX `write(2)`, an `io.Writer` does **not** do partial writes without an error, so callers don't need to retry in a loop. The implementation must not modify `p`, not even temporarily.

## `io.Copy` and the "upgrade" interfaces

`io.Copy(dst, src)` doesn't always copy through its own 32 KB buffer. It checks first:

1. whether `src` implements `io.WriterTo` → calls `src.WriteTo(dst)`;
2. whether `dst` implements `io.ReaderFrom` → calls `dst.ReadFrom(src)`;
3. only then, the buffered loop.

Standard library types use these hooks for kernel-level **zero-copy**:

| Copy | Mechanism on Linux |
|---|---|
| file → TCP socket | `sendfile(2)`: data never passes through user space |
| socket → socket, pipe → socket | `splice(2)` |
| file → file | `copy_file_range(2)` |

A server sending static files with `http.ServeContent` or a proxy doing `io.Copy(conn1, conn2)` benefits automatically. Wrapping an `*os.File` or `*net.TCPConn` in your own type (for metrics, say) **hides** these interfaces and disables the optimization; implement `ReadFrom`/`WriteTo` yourself by delegating, or expose the original type.

`io.CopyBuffer` lets you reuse a buffer (for example from a `sync.Pool`), but it's ignored when one of the paths above applies.

## Files, sockets and the runtime

**Sockets** and pipes are put in non-blocking mode and registered with the **netpoller**: a read that would block parks the goroutine without blocking the thread (see [Goroutines](../../goroutines/senior/)).

**Regular files** on Linux can't be "waited on" through epoll (they're always "ready"), so `Read`/`Write` on them are blocking system calls: the thread (M) blocks, and the P is handed to another thread. A program reading thousands of files concurrently can create many threads; a semaphore limiting simultaneous disk operations keeps the count under control.

`os.File` supports deadlines (`SetReadDeadline`) only for descriptors that go through the netpoller (pipes, terminals, some special files); on regular files they return `os.ErrNoDeadline`.

## Durability: atomic writes to disk

`f.Write` + `f.Close` doesn't guarantee the data is on disk: it sits in the kernel's page cache. A power failure can leave an empty or truncated file. The robust pattern for replacing a file:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*") // same directory: rename is only atomic within one FS
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // a no-op if the rename succeeded

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil { // fsync: the data reaches the disk
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil { // atomic replacement
		return err
	}
	d, err := os.Open(dir) // fsync the directory: the rename becomes durable
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func main() {
	dir, _ := os.MkdirTemp("", "atomic")
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "config.json")
	fmt.Println(writeAtomic(path, []byte(`{"v":1}`)))
	b, _ := os.ReadFile(path)
	fmt.Println(string(b))
}
```

Readers see either the complete old file or the complete new one, never an intermediate state.

## `bufio` under the hood

- `bufio.Reader` reads from the source in buffer-sized chunks; if the `p` in `Read(p)` is larger than the buffer and the buffer is empty, it reads **directly** into `p`, avoiding a double copy.
- `bufio.Writer` does the same when writing: a write larger than the free buffer goes straight through, after the buffer is flushed.
- `bufio.Scanner` limits the size of a token (64 KB by default). Longer lines yield `bufio.ErrTooLong`; `scanner.Buffer(buf, max)` raises it.

## Summary

- `Read` may return less than you asked for, and data + an error at once; `io.ReadFull` for exactly `n`.
- `Write` with `n < len(p)` always returns an error.
- `io.Copy` uses `WriterTo`/`ReaderFrom` → `sendfile`, `splice`, `copy_file_range`.
- Sockets go through the netpoller; regular files block threads.
- Durable writes: a temp file, `Sync`, `Rename`, `Sync` on the directory.
