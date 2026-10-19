# Context — Junior

Pachetul `context` rezolvă o problemă comună: cum anunți o operație (și toate goroutines pornite de ea) că trebuie să se oprească — pentru că utilizatorul a închis pagina, a expirat un timp limită sau programul se închide.

## Ce e un `context.Context`

O valoare `ctx` transmisă de la funcție la funcție, care poartă:

- un semnal de **anulare** (`ctx.Done()`, un channel care se închide);
- opțional, un **termen limită** (deadline);
- opțional, **valori** legate de cerere (de exemplu un ID de cerere).

Convenția: contextul e **primul parametru** al funcției și se numește `ctx`.

```go,ignore
func DescarcaPagina(ctx context.Context, url string) ([]byte, error)
```

## Contextul de pornire

`context.Background()` e rădăcina: nu se anulează niciodată. Îl folosești în `main`, în teste și la pornirea serviciilor. `context.TODO()` e identic, dar marchează locurile unde încă nu ai decis ce context să folosești.

## Timp limită: `WithTimeout`

```go
package main

import (
	"context"
	"fmt"
	"time"
)

func operatieLenta(ctx context.Context) error {
	select {
	case <-time.After(200 * time.Millisecond): // „lucrul” durează 200ms
		return nil
	case <-ctx.Done(): // contextul a fost anulat
		return ctx.Err()
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel() // eliberează resursele contextului

	err := operatieLenta(ctx)
	fmt.Println("rezultat:", err) // context deadline exceeded
}
```

`WithTimeout` întoarce un context nou și o funcție `cancel`. **Apelează mereu `cancel`**, de obicei cu `defer`, chiar dacă operația se termină înainte de timeout.

## Anulare manuală: `WithCancel`

```go
package main

import (
	"context"
	"fmt"
	"time"
)

func lucrator(ctx context.Context, rezultate chan<- int) {
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("lucrătorul se oprește:", ctx.Err())
			close(rezultate)
			return
		case rezultate <- i:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // apelul dublu e inofensiv
	rezultate := make(chan int)
	go lucrator(ctx, rezultate)

	for v := range rezultate {
		if v == 3 {
			cancel() // gata, nu mai avem nevoie
		}
	}
	fmt.Println("main a terminat")
}
```

## `ctx.Err()`

După anulare, `ctx.Err()` spune motivul:

| Eroare | Când |
|---|---|
| `context.Canceled` | s-a apelat `cancel()` |
| `context.DeadlineExceeded` | a expirat timpul limită |
| `nil` | contextul nu e (încă) anulat |

## Contextele formează un arbore

Un context creat din altul e **copilul** lui. Anularea părintelui anulează automat toți copiii; anularea unui copil nu-l afectează pe părinte:

```go
package main

import (
	"context"
	"fmt"
)

func main() {
	parinte, cancelParinte := context.WithCancel(context.Background())
	copil, cancelCopil := context.WithCancel(parinte)
	defer cancelCopil()

	cancelParinte()
	<-copil.Done() // copilul a fost anulat odată cu părintele
	fmt.Println("copil:", copil.Err())
}
```

## Contextul în biblioteca standard

Multe funcții primesc deja un context și se opresc singure la anulare:

- `http.NewRequestWithContext(ctx, ...)` — o cerere HTTP;
- `r.Context()` — într-un handler HTTP, contextul se anulează când clientul închide conexiunea;
- `db.QueryContext(ctx, ...)` — o interogare SQL;
- `exec.CommandContext(ctx, ...)` — un proces extern, oprit la anulare.

## Pe scurt

- `ctx` e primul parametru și poartă anularea, termenul limită și valori.
- `context.Background()` e rădăcina.
- `WithTimeout` și `WithCancel` întorc un `cancel`: apelează-l mereu cu `defer`.
- Ascultă `<-ctx.Done()` în bucle și operații lungi.
- Anularea se propagă de la părinte la copii.
