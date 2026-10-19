# net/http — Middle

What turns a sample server into a production one: middleware, timeouts, request body limits, graceful shutdown, a properly configured client and handler testing.

## Middleware

A middleware is a function that takes an `http.Handler` and returns another, adding behavior before or after:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
)

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, r *http.Request) { panic("bug") })

	srv := httptest.NewServer(logging(recovery(mux))) // order: outermost to innermost
	defer srv.Close()

	for _, p := range []string{"/ok", "/boom"} {
		resp, _ := http.Get(srv.URL + p)
		resp.Body.Close()
		fmt.Println(p, resp.StatusCode)
	}
}
```

The HTTP server already has an internal `recover` that keeps the process alive, but it closes the connection with no response. The middleware above sends a clean 500.

## Server timeouts

`http.ListenAndServe` uses a server with **no timeouts**: a slow (or malicious) client can hold connections open forever. In production, configure `http.Server`:

```go,ignore
srv := &http.Server{
	Addr:              ":8080",
	Handler:           mux,
	ReadHeaderTimeout: 5 * time.Second,  // protection against Slowloris
	ReadTimeout:       15 * time.Second, // the whole request, body included
	WriteTimeout:      30 * time.Second, // from the end of header reading to the end
	IdleTimeout:       2 * time.Minute,  // idle keep-alive connections
	MaxHeaderBytes:    1 << 20,
}
```

For a handler that needs longer (a big upload, a stream), `http.NewResponseController(w).SetWriteDeadline(...)` (Go 1.20) changes the deadline for that request only.

## Limiting and decoding the body

```go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // at most 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var u User
	if err := dec.Decode(&u); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if u.Email == "" {
		http.Error(w, "email required", http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "created %s\n", u.Name)
}

func main() {
	for _, body := range []string{`{"name":"Ana","email":"a@x.ro"}`, `{"name":"Ion"}`, `{"nme":"x"}`} {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
		rec := httptest.NewRecorder()
		create(rec, req)
		fmt.Print(rec.Code, " ", rec.Body.String())
	}
}
```

Without `MaxBytesReader`, a client can send a body of several GB and exhaust memory.

## The request context

`r.Context()` is canceled when the client closes the connection. Pass it along (database, other services), so useless work stops:

```go,ignore
func report(w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(r.Context(), "SELECT ...")
	if errors.Is(err, context.Canceled) {
		return // the client left; nobody is waiting for the response
	}
	// ...
}
```

## Graceful shutdown

On a stop signal (a deploy, scaling), the server must stop accepting new connections but finish the requests in progress:

```go,ignore
func main() {
	srv := &http.Server{Addr: ":8080", Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done() // the signal arrived
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("forced shutdown:", err)
	}
}
```

## The client, configured properly

- **A single `http.Client`** reused across the whole application: it holds the connection pool (keep-alive). A new client per request means new TCP connections and TLS handshakes every time.
- A **timeout** on the client, or a context per request (`http.NewRequestWithContext`).
- The **response body** must be read **to the end** and closed, so the connection returns to the pool. `io.Copy(io.Discard, resp.Body)` before `Close` if you don't need it.
- `Transport.MaxIdleConnsPerHost` (2 by default) is too small for a client making many concurrent requests to the same service.

## Testing handlers

`httptest.NewRecorder` captures the response without a network; `httptest.NewServer` starts a real loopback server for end-to-end tests (including for clients):

```go,ignore
func TestHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d, want 200", rec.Code)
	}
}
```

## Summary

- Middleware = `func(http.Handler) http.Handler`, composed outermost to innermost.
- Configure `http.Server` with timeouts; a plain `ListenAndServe` has none.
- `MaxBytesReader` + `DisallowUnknownFields` + validation for JSON bodies.
- `Shutdown` for graceful stops; `r.Context()` for cancellation.
- One client, with a timeout, with the body read and closed.
