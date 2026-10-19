# Context — Middle

Folosirea corectă a contextului într-o aplicație reală: propagarea prin straturi, valorile din context (și când să nu le folosești), cauzele anulării, `AfterFunc`, `WithoutCancel` și cum scrii funcții care respectă anularea.

## Propagarea prin straturi

Contextul vine de sus (handler HTTP, `main`, consumatorul unei cozi) și trece prin **fiecare** funcție care poate bloca sau dura. Fiecare strat poate restrânge termenul, dar nu-l poate prelungi:

```go
package main

import (
	"context"
	"fmt"
	"time"
)

func interogareDB(ctx context.Context) error {
	// stratul de date își impune propria limită, mai mică
	ctx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("interogare: %w", ctx.Err())
	}
}

func serviciu(ctx context.Context) error {
	if err := interogareDB(ctx); err != nil {
		return fmt.Errorf("serviciu: %w", err)
	}
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := serviciu(ctx)
	fmt.Println(err)
	fmt.Println(err != nil && ctx.Err() == nil) // true: doar stratul de jos a expirat
}
```

Termenul efectiv e întotdeauna **cel mai apropiat** dintre al părintelui și al copilului.

## Scrie funcții care respectă anularea

Anularea e **cooperativă**: contextul nu oprește nimic singur, doar închide un channel. Codul tău trebuie să-l verifice:

- în bucle lungi, cu `ctx.Err()` sau un `select` cu `default`;
- la orice așteptare (channel, timer), într-un `select` cu `<-ctx.Done()`;
- transmițându-l funcțiilor care îl acceptă.

```go
package main

import (
	"context"
	"fmt"
	"time"
)

func proceseazaToate(ctx context.Context, elemente []int) (int, error) {
	procesate := 0
	for _, e := range elemente {
		if err := ctx.Err(); err != nil { // verificare ieftină la fiecare pas
			return procesate, err
		}
		time.Sleep(time.Duration(e) * time.Millisecond)
		procesate++
	}
	return procesate, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	n, err := proceseazaToate(ctx, []int{10, 10, 10, 10, 10})
	fmt.Println(n, err) // 2 sau 3, context deadline exceeded
}
```

## Valori în context

`context.WithValue` atașează o valoare. Cheia trebuie să fie de un **tip neexportat**, ca să nu existe coliziuni între pachete:

```go
package main

import (
	"context"
	"fmt"
)

type cheieCtx int

const cheieIDCerere cheieCtx = 0

func CuIDCerere(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, cheieIDCerere, id)
}

func IDCerere(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(cheieIDCerere).(string)
	return id, ok
}

func main() {
	ctx := CuIDCerere(context.Background(), "req-42")
	if id, ok := IDCerere(ctx); ok {
		fmt.Println("procesez cererea", id)
	}
}
```

Ce **să** pui în context: date legate de cerere care traversează granițe de API — ID-ul cererii, informații de tracing, identitatea utilizatorului autentificat.

Ce **să nu** pui: parametri opționali ai funcțiilor, conexiuni la baze de date, loggere, configurații. Acestea fac dependențele invizibile; transmite-le explicit.

## Cauza anulării (Go 1.20+)

`ctx.Err()` spune doar *că* s-a anulat. `WithCancelCause` și `WithTimeoutCause` păstrează și *de ce*:

```go
package main

import (
	"context"
	"errors"
	"fmt"
)

var ErrClientPlecat = errors.New("clientul a închis conexiunea")

func main() {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(ErrClientPlecat)

	fmt.Println(ctx.Err())          // context canceled
	fmt.Println(context.Cause(ctx)) // clientul a închis conexiunea
}
```

## `AfterFunc`, `WithoutCancel`, `WithDeadline`

```go
package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// AfterFunc (Go 1.21): rulează o funcție la anulare, fără o goroutine care așteaptă
	stop := context.AfterFunc(ctx, func() {
		fmt.Println("curățare după anulare")
	})
	defer stop()

	// WithoutCancel (Go 1.21): păstrează valorile, dar nu și anularea
	// util pentru lucru care trebuie să continue după răspuns (audit, metrici)
	detasat := context.WithoutCancel(ctx)

	cancel()
	time.Sleep(10 * time.Millisecond)
	fmt.Println(ctx.Err(), detasat.Err()) // context canceled <nil>

	// WithDeadline: ca WithTimeout, dar cu un moment absolut
	dl, cancelDL := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	defer cancelDL()
	d, _ := dl.Deadline()
	fmt.Println(time.Until(d).Round(time.Hour))
}
```

## Reguli

1. Contextul e **primul parametru**, numit `ctx`; nu-l păstra într-o structură (excepții: tipuri care reprezintă chiar o operație, ca `http.Request`).
2. Nu transmite un context `nil`; folosește `context.TODO()`.
3. Apelează **întotdeauna** funcția `cancel` (`go vet` verifică asta prin analizorul `lostcancel`).
4. Un context poate fi folosit simultan din mai multe goroutines.
5. Valorile din context sunt pentru date de cerere, nu pentru dependențe.

## Pe scurt

- Fiecare strat poate scurta termenul, nu-l poate prelungi.
- Anularea e cooperativă: verifică `ctx.Err()` / `ctx.Done()`.
- `WithValue` cu chei de tip neexportat, doar pentru date de cerere.
- `WithCancelCause` + `context.Cause` explică de ce s-a anulat.
- `AfterFunc` pentru curățare, `WithoutCancel` pentru lucru detașat.
