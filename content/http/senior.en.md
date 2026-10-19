# net/http — Senior

How Go's HTTP server is built (one goroutine per connection, buffering, `Content-Length` vs. chunked), `ServeMux` precedence rules, HTTP/2, fine-grained response control with `ResponseController`, and how the client's connection pool works.

## Anatomy of the server

`Server.Serve` accepts connections in a loop and starts **one goroutine per connection** (`conn.serve`). For HTTP/1.1 with keep-alive, the same goroutine reads successive requests from the connection and serves them **sequentially**; concurrency comes from different connections. For HTTP/2, each **stream** gets its own goroutine, so requests multiplexed over one connection run in parallel.

Each connection has a `bufio.Reader` and a `bufio.Writer` (reused through pools). The `ResponseWriter` is an `*http.response` that buffers the start of the body:

- if the handler returns and the body fit in the buffer (a few KB), the server knows the length and sends `Content-Length`;
- if the buffer fills before the end (or you call `Flush`), headers are sent without a length and the body uses **`Transfer-Encoding: chunked`**;
- if you don't set `Content-Type`, the server **sniffs** it from the first 512 bytes (`http.DetectContentType`).

A second `WriteHeader` is ignored with `http: superfluous response.WriteHeader call` in the log — usually the sign of a handler that doesn't `return` after `http.Error`.

## Precedence in `ServeMux` (Go 1.22)

When several patterns match, the **most specific** wins: one pattern is more specific than another if it matches a strict subset of the other's requests.

```go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func main() {
	mux := http.NewServeMux()
	reply := func(text string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, text) }
	}
	mux.HandleFunc("/", reply("root (anything else)"))
	mux.HandleFunc("/files/", reply("prefix /files/"))
	mux.HandleFunc("GET /files/{name}", reply("a file"))
	mux.HandleFunc("GET /files/readme.md", reply("readme, exact"))
	mux.HandleFunc("/files/{dir}/{path...}", reply("subdirectory + rest of the path"))
	mux.HandleFunc("GET /{$}", reply("only /"))

	for _, p := range []string{"/", "/other", "/files/a.txt", "/files/readme.md", "/files/x/y/z"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		fmt.Printf("%-18s → %s\n", p, rec.Body.String())
	}
}
```

Two overlapping patterns where neither is more specific (for example `GET /a/{x}` and `/{y}/b`) cause a **panic at registration**, not ambiguous behavior at run time. So do two equivalent patterns: `/files/` and `/files/{path...}` match exactly the same requests (`/files/{path...} matches the same requests as /files/`). A `GET` pattern automatically accepts `HEAD` too.

## HTTP/2

The server and client speak HTTP/2 **automatically** over TLS (negotiated via ALPN). Since Go 1.24, `Server.Protocols` and `Transport.Protocols` control the protocols explicitly, including unencrypted HTTP/2 (h2c), useful behind a load balancer that terminates TLS:

```go,ignore
var p http.Protocols
p.SetHTTP1(true)
p.SetUnencryptedHTTP2(true)
srv := &http.Server{Addr: ":8080", Handler: mux, Protocols: &p}
```

## `ResponseController` (Go 1.20)

A `ResponseWriter`'s optional features used to be exposed through interfaces (`http.Flusher`, `http.Hijacker`), lost by any middleware that wrapped the writer. `http.NewResponseController(w)` finds them even through wrappers, as long as they implement `Unwrap() http.ResponseWriter`:

```go
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

func events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	rc := http.NewResponseController(w)
	for i := range 3 {
		fmt.Fprintf(w, "data: step %d\n\n", i)
		if err := rc.Flush(); err != nil { // send now, without waiting for the buffer
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(events))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if sc.Text() != "" {
			fmt.Println(sc.Text()) // they arrive one by one, not all at the end
		}
	}
}
```

`ResponseController` also offers per-request `SetReadDeadline`/`SetWriteDeadline`, `Hijack` (taking over the raw connection, for WebSocket) and `EnableFullDuplex` (reading the request body while writing the response, disabled by default for HTTP/1).

## The client's connection pool

`http.Transport` keeps idle connections in a pool keyed by `(scheme, host, proxy)`:

1. a request first looks for an idle connection for its key;
2. if there's none, it starts a dial; if a connection frees up meanwhile, it uses that one and the dial carries on for future use;
3. after the response, the connection goes back to the pool **only if** the body was fully read and closed.

Relevant limits: `MaxIdleConns` (100), `MaxIdleConnsPerHost` (**2**), `MaxConnsPerHost` (unlimited by default), `IdleConnTimeout` (90 s). A client making 50 concurrent requests to the same host with `MaxIdleConnsPerHost = 2` constantly opens and closes connections: TCP + TLS every time. With HTTP/2, a single connection multiplexes every request.

Requests with a body that fail on a reused connection (closed by the server meanwhile) are retried automatically **only** if they're idempotent or have `GetBody` set.

## Timeouts, precisely

```text
Client.Timeout ───────────────────────────────────────────────────────────►
  │ dial │ TLS handshake │ send the request │ await headers │ read the body │
         └ TLSHandshakeTimeout          ResponseHeaderTimeout
Server:  ReadHeaderTimeout ─► ReadTimeout ─────────► WriteTimeout ──────────►
```

`Client.Timeout` covers **everything**, including reading the body. For large downloads or streaming, use a per-request context with a deadline and the granular `Transport` timeouts, not `Client.Timeout`.

## Summary

- One goroutine per connection (HTTP/1) or per stream (HTTP/2).
- Small body → `Content-Length`; large body or `Flush` → chunked.
- `ServeMux`: the most specific pattern wins; conflicts panic at registration.
- `ResponseController` for flushing, deadlines, hijacking — even through middleware.
- The client pool needs fully read bodies; `MaxIdleConnsPerHost` is 2 by default.
