# net/http — Senior

Cum e construit serverul HTTP al Go (o goroutine per conexiune, buffering, `Content-Length` vs. chunked), regulile de precedență ale `ServeMux`, HTTP/2, controlul fin al răspunsului cu `ResponseController` și cum funcționează pool-ul de conexiuni al clientului.

## Anatomia serverului

`Server.Serve` acceptă conexiuni într-o buclă și pornește **o goroutine per conexiune** (`conn.serve`). Pentru HTTP/1.1 cu keep-alive, aceeași goroutine citește cererile succesive de pe conexiune și le servește **secvențial**; concurența vine din conexiuni diferite. Pentru HTTP/2, fiecare **stream** primește propria goroutine, deci cererile multiplexate pe aceeași conexiune rulează în paralel.

Fiecare conexiune are un `bufio.Reader` și un `bufio.Writer` (refolosite prin pool-uri). `ResponseWriter` e un `*http.response` care buffer-ează începutul corpului:

- dacă handler-ul se termină și corpul a încăput în buffer (câțiva KB), serverul știe lungimea și trimite `Content-Length`;
- dacă buffer-ul se umple înainte de final (sau apelezi `Flush`), antetele sunt trimise fără lungime și corpul folosește **`Transfer-Encoding: chunked`**;
- dacă nu setezi `Content-Type`, serverul îl **ghicește** din primii 512 octeți (`http.DetectContentType`).

Un al doilea `WriteHeader` e ignorat cu mesajul `http: superfluous response.WriteHeader call` în log — de obicei semnul unui handler care nu face `return` după `http.Error`.

## Precedența în `ServeMux` (Go 1.22)

Când mai multe tipare se potrivesc, câștigă cel **mai specific**: un tipar e mai specific dacă se potrivește cu o submulțime strictă a cererilor celuilalt.

```go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func main() {
	mux := http.NewServeMux()
	raspunde := func(text string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, text) }
	}
	mux.HandleFunc("/", raspunde("rădăcină (orice altceva)"))
	mux.HandleFunc("/fisiere/", raspunde("prefix /fisiere/"))
	mux.HandleFunc("GET /fisiere/{nume}", raspunde("un fișier"))
	mux.HandleFunc("GET /fisiere/readme.md", raspunde("readme, exact"))
	mux.HandleFunc("/fisiere/{dir}/{cale...}", raspunde("subdirector + restul căii"))
	mux.HandleFunc("GET /{$}", raspunde("doar /"))

	for _, p := range []string{"/", "/altceva", "/fisiere/a.txt", "/fisiere/readme.md", "/fisiere/x/y/z"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		fmt.Printf("%-20s → %s\n", p, rec.Body.String())
	}
}
```

Două tipare care se suprapun fără ca unul să fie mai specific (de exemplu `GET /a/{x}` și `/{y}/b`) produc un **panic la înregistrare**, nu un comportament ambiguu la rulare. La fel și două tipare echivalente: `/fisiere/` și `/fisiere/{cale...}` se potrivesc cu exact aceleași cereri (`/fisiere/{cale...} matches the same requests as /fisiere/`). Un tipar cu `GET` acceptă automat și `HEAD`.

## HTTP/2

Serverul și clientul vorbesc HTTP/2 **automat** peste TLS (negociat prin ALPN). Din Go 1.24, `Server.Protocols` și `Transport.Protocols` controlează explicit protocoalele, inclusiv HTTP/2 necriptat (h2c), util în spatele unui load balancer care termină TLS:

```go,ignore
var p http.Protocols
p.SetHTTP1(true)
p.SetUnencryptedHTTP2(true)
srv := &http.Server{Addr: ":8080", Handler: mux, Protocols: &p}
```

## `ResponseController` (Go 1.20)

Funcționalitățile opționale ale unui `ResponseWriter` erau expuse prin interfețe (`http.Flusher`, `http.Hijacker`), pierdute de orice middleware care învelea writer-ul. `http.NewResponseController(w)` le găsește chiar și prin wrapper-e, dacă acestea implementează `Unwrap() http.ResponseWriter`:

```go
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

func evenimente(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	rc := http.NewResponseController(w)
	for i := range 3 {
		fmt.Fprintf(w, "data: pasul %d\n\n", i)
		if err := rc.Flush(); err != nil { // trimite imediat, fără să aștepte buffer-ul
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(evenimente))
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
			fmt.Println(sc.Text()) // sosesc pe rând, nu toate la final
		}
	}
}
```

`ResponseController` oferă și `SetReadDeadline`/`SetWriteDeadline` per cerere, `Hijack` (preluarea conexiunii brute, pentru WebSocket) și `EnableFullDuplex` (citirea corpului cererii în timp ce scrii răspunsul, dezactivată implicit pentru HTTP/1).

## Pool-ul de conexiuni al clientului

`http.Transport` păstrează conexiunile inactive într-un pool indexat după `(schemă, gazdă, proxy)`:

1. o cerere caută întâi o conexiune inactivă pentru cheia ei;
2. dacă nu există, pornește un dial; dacă între timp se eliberează o conexiune, o folosește pe aceea și dial-ul continuă pentru viitor;
3. după răspuns, conexiunea revine în pool **doar dacă** corpul a fost citit complet și închis.

Limite relevante: `MaxIdleConns` (100), `MaxIdleConnsPerHost` (**2**), `MaxConnsPerHost` (nelimitat implicit), `IdleConnTimeout` (90 s). Un client care face 50 de cereri concurente către același host și are `MaxIdleConnsPerHost = 2` deschide și închide constant conexiuni: TCP + TLS la fiecare. Cu HTTP/2, o singură conexiune multiplexează toate cererile.

Cererile cu corp care eșuează pe o conexiune refolosită (închisă între timp de server) sunt reîncercate automat **doar** dacă sunt idempotente sau au `GetBody` setat.

## Timeout-urile, precis

```text
Client.Timeout ──────────────────────────────────────────────────────────►
  │ dial │ TLS handshake │ trimite cererea │ așteaptă antete │ citește corpul │
         └ TLSHandshakeTimeout          ResponseHeaderTimeout
Server:  ReadHeaderTimeout ─► ReadTimeout ─────────► WriteTimeout ─────────►
```

`Client.Timeout` acoperă **tot**, inclusiv citirea corpului. Pentru descărcări mari sau streaming, folosește un context cu termen pe cerere și timeout-urile granulare din `Transport`, nu `Client.Timeout`.

## Pe scurt

- O goroutine per conexiune (HTTP/1) sau per stream (HTTP/2).
- Corp mic → `Content-Length`; corp mare sau `Flush` → chunked.
- `ServeMux`: cel mai specific tipar câștigă; conflictele dau panic la înregistrare.
- `ResponseController` pentru flush, deadline-uri, hijack — și prin middleware.
- Pool-ul clientului cere corpuri citite complet; `MaxIdleConnsPerHost` e 2 implicit.
