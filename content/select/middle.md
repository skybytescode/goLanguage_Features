# Select — Middle

Tiparele din codul real: bucle de evenimente cu `select`, timere și tickere folosite corect, channels `nil` pentru a dezactiva cazuri, prioritizarea și limitarea ratei.

## Bucla de evenimente

O goroutine care reacționează la mai multe surse (date, timer, oprire) e aproape mereu un `for` + `select`:

```go
package main

import (
	"context"
	"fmt"
	"time"
)

func agregator(ctx context.Context, intrare <-chan int) {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	suma := 0
	for {
		select {
		case v := <-intrare:
			suma += v
		case <-ticker.C:
			fmt.Println("raport periodic, suma =", suma)
		case <-ctx.Done():
			fmt.Println("oprire:", ctx.Err())
			return
		}
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()

	intrare := make(chan int)
	go func() {
		for i := 1; ; i++ {
			select {
			case intrare <- i:
				time.Sleep(5 * time.Millisecond)
			case <-ctx.Done():
				return
			}
		}
	}()
	agregator(ctx, intrare)
}
```

Observă că și **trimițătorul** ascultă `ctx.Done()`. Fără asta, după ce agregatorul se oprește, trimițătorul ar rămâne blocat pentru totdeauna pe `intrare <- i`.

## Timere: `time.After` vs. `time.NewTimer`

`time.After` creează un timer nou la fiecare apel. Într-o buclă, fiecare iterație face unul nou. Din Go 1.23, timerele neoprite și nereferite sunt colectate de GC, deci nu mai e o scurgere de memorie, dar rămâne o alocare per iterație. Pentru un timeout **per inactivitate**, un singur timer resetat e mai clar:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	evenimente := make(chan string)
	go func() {
		for _, e := range []string{"a", "b", "c"} {
			evenimente <- e
			time.Sleep(10 * time.Millisecond)
		}
	}()

	const inactivitate = 50 * time.Millisecond
	timer := time.NewTimer(inactivitate)
	defer timer.Stop()

	for {
		select {
		case e := <-evenimente:
			fmt.Println("eveniment", e)
			timer.Reset(inactivitate) // Go 1.23+: Reset e sigur fără a goli channel-ul
		case <-timer.C:
			fmt.Println("nicio activitate de", inactivitate)
			return
		}
	}
}
```

Înainte de Go 1.23, `Reset` pe un timer care expirase deja cerea golirea manuală a lui `timer.C`, altfel bucla putea primi o valoare veche. Din 1.23, channel-ul timerului e sincron și `Reset`/`Stop` garantează că nu mai vine nicio valoare veche (comportamentul depinde de versiunea `go` din `go.mod`).

## Dezactivarea unui caz cu un channel `nil`

Operațiile pe un channel `nil` blochează pentru totdeauna, deci un `case` cu channel `nil` **nu e ales niciodată**. Îl poți „opri” punând variabila pe `nil`. Tiparul clasic: combinarea a două channels până se închid amândouă:

```go
package main

import "fmt"

func sursa(vals ...int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for _, v := range vals {
			ch <- v
		}
	}()
	return ch
}

func main() {
	a, b := sursa(1, 2, 3), sursa(10, 20)
	suma := 0
	for a != nil || b != nil {
		select {
		case v, ok := <-a:
			if !ok {
				a = nil // închis: nu mai alegem acest caz
				continue
			}
			suma += v
		case v, ok := <-b:
			if !ok {
				b = nil
				continue
			}
			suma += v
		}
	}
	fmt.Println(suma) // 36
}
```

Fără `a = nil`, un channel închis ar fi „mereu gata” (întoarce imediat zero), iar bucla s-ar învârti în gol.

## Prioritate

`select` alege aleatoriu între cazurile gata; nu există prioritate. Dacă oprirea trebuie să aibă întâietate, verific-o întâi separat:

```go,ignore
for {
	select {
	case <-ctx.Done(): // verificare prioritară, non-blocantă
		return
	default:
	}

	select {
	case <-ctx.Done():
		return
	case job := <-joburi:
		proceseaza(job)
	}
}
```

## Limitarea ratei cu un ticker

Un `time.Ticker` livrează un „jeton” la interval fix. Așteptând jetonul înainte de fiecare operație, nu depășești o anumită rată:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	cereri := []string{"a", "b", "c", "d"}
	limita := time.NewTicker(10 * time.Millisecond) // cel mult 100 de cereri/secundă
	defer limita.Stop()

	start := time.Now()
	for _, c := range cereri {
		<-limita.C
		fmt.Printf("%s la %v\n", c, time.Since(start).Round(10*time.Millisecond))
	}
}
```

Pentru limitare cu „rafale” (burst), folosește `golang.org/x/time/rate`.

## Trimitere cu timeout

Același tipar funcționează și la trimitere, ca un producător să nu blocheze la nesfârșit un consumator lent:

```go
package main

import (
	"errors"
	"fmt"
	"time"
)

var ErrPlin = errors.New("coadă plină")

func trimite(ch chan<- int, v int, asteptare time.Duration) error {
	select {
	case ch <- v:
		return nil
	case <-time.After(asteptare):
		return ErrPlin
	}
}

func main() {
	coada := make(chan int, 1)
	fmt.Println(trimite(coada, 1, 10*time.Millisecond)) // <nil>
	fmt.Println(trimite(coada, 2, 10*time.Millisecond)) // coadă plină
}
```

## Pe scurt

- `for` + `select` cu `ctx.Done()` e bucla standard de evenimente.
- Ambele capete (trimițător și receptor) trebuie să asculte oprirea.
- Un timer resetat pentru timeout-uri de inactivitate; `Reset` e sigur din Go 1.23.
- Un channel `nil` dezactivează un caz.
- `select` nu are prioritate; verifică separat ce e urgent.
