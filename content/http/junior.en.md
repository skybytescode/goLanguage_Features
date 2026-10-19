# net/http — Junior

Go's standard library contains a production-ready HTTP server and client. Many Go services use no framework at all: just `net/http`.

## Your first server

```go,ignore
package main

import (
	"fmt"
	"log"
	"net/http"
)

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, world!")
}

func main() {
	http.HandleFunc("/", hello)
	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

A **handler** receives:

- `w http.ResponseWriter` — where you write the response (it's an `io.Writer`);
- `r *http.Request` — the request: method, URL, headers, body.

Each request is handled in its own **goroutine**, so the server serves several clients at once automatically.

> The examples on this page use `httptest`, which starts a temporary server inside the same program, so they can run anywhere. In your application, you use `http.ListenAndServe` as above.

## Routes with methods and parameters (Go 1.22)

`http.ServeMux` accepts patterns with the **HTTP method** and **parameters** in the path:

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "product %s\n", r.PathValue("id"))
	})
	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, "created")
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/products/42")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Print(resp.StatusCode, " ", string(body)) // 200 product 42

	resp, _ = http.Post(srv.URL+"/products/42", "text/plain", nil)
	resp.Body.Close()
	fmt.Println(resp.StatusCode) // 405: POST isn't allowed on this route
}
```

## Query string parameters

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

func search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing parameter q", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "results for %q\n", q)
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(search))
	defer srv.Close()

	for _, url := range []string{"/?q=golang", "/"} {
		resp, _ := http.Get(srv.URL + url)
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Print(resp.StatusCode, " ", string(b))
	}
}
```

`http.Error` sets the status code and writes the message. Don't forget the `return` after it.

## A JSON response

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

type Health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") // headers BEFORE the body
	json.NewEncoder(w).Encode(Health{Status: "ok", Version: "1.0"})
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(health))
	defer srv.Close()

	resp, _ := http.Get(srv.URL)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	fmt.Print(resp.Header.Get("Content-Type"), " ", string(b))
}
```

Order matters: headers, then `WriteHeader` (optional, 200 by default), then the body. Once you've written to the body, headers can no longer be changed.

## The HTTP client

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "pong")
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second} // ALWAYS set a timeout
	resp, err := client.Get(srv.URL)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close() // mandatory: otherwise the connection isn't reused

	if resp.StatusCode != http.StatusOK {
		fmt.Println("unexpected status:", resp.Status)
		return
	}
	b, _ := io.ReadAll(resp.Body)
	fmt.Println(string(b))
}
```

Careful: `http.Get` returns `err == nil` for a 404 or 500 too. The error only appears when no response could be obtained at all; you check the status code yourself.

## Static files

```go,ignore
http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))
```

## Summary

- A handler receives a `ResponseWriter` and a `*Request`; each request gets its own goroutine.
- `ServeMux` supports `"GET /path/{param}"` since Go 1.22; `r.PathValue` reads the parameter.
- Headers are set before the body; `http.Error` + `return` for errors.
- The client needs a `Timeout` and `defer resp.Body.Close()`.
- A 404 isn't a network error: check `resp.StatusCode`.
