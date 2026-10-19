# Erori, defer, panic și recover — Senior

Semantica exactă a lui `defer`, regulile stricte pentru `recover`, diferența dintre panic și erorile fatale ale runtime-ului și cum funcționează lanțurile de erori pe dinăuntru.

## Argumentele lui `defer` sunt evaluate imediat

La instrucțiunea `defer f(x)`, valoarea funcției și argumentele sunt evaluate **acum**; doar apelul e amânat. Un closure, în schimb, citește variabilele la execuție:

```go
package main

import "fmt"

func main() {
	x := 1
	defer fmt.Println("argument evaluat imediat:", x)              // 1
	defer func() { fmt.Println("closure citește la final:", x) }() // 2
	x = 2
}
```

Același lucru e valabil pentru receptor: `defer obiect.Metoda()` evaluează `obiect` imediat. Un tipar util care se bazează pe asta:

```go
package main

import (
	"fmt"
	"time"
)

func cronometreaza(nume string) func() {
	start := time.Now()
	return func() { fmt.Printf("%s: %v\n", nume, time.Since(start).Round(time.Millisecond)) }
}

func main() {
	defer cronometreaza("main")() // cronometreaza rulează acum, funcția întoarsă la final
	time.Sleep(20 * time.Millisecond)
}
```

## Regulile lui `recover`

`recover` oprește un panic **doar** dacă e apelat **direct** dintr-o funcție amânată cu `defer`, în goroutine-ul care a dat panic. În orice alt loc întoarce `nil`:

```go,panic
package main

import "fmt"

func ajutor() {
	if r := recover(); r != nil { // NU funcționează: nu e apelat direct de funcția din defer
		fmt.Println("prins", r)
	}
}

func main() {
	defer func() {
		ajutor()
	}()
	panic("bum")
}
```

`defer ajutor()` (fără closure-ul intermediar) ar fi funcționat: atunci `ajutor` e chiar funcția amânată.

Alte reguli:

- **Fiecare goroutine e pe cont propriu.** Un panic într-o goroutine fără `recover` oprește **tot programul**, oricât de multe `recover` ar avea `main`. Codul care pornește goroutines într-un server trebuie să-și pună propriul `defer recover()` acolo unde e cazul.
- Din Go 1.21, `panic(nil)` produce un `*runtime.PanicNilError`, ca `recover()` să nu întoarcă `nil` pentru un panic real.
- Un panic apărut într-un `defer` în timpul altui panic îl înlocuiește; ambele apar în mesajul final.
- După `recover`, funcția care conține `defer`-ul se întoarce normal, cu rezultatele cu nume așa cum le-a lăsat `defer`-ul. Așa transformi un panic în eroare:

```go
package main

import (
	"fmt"
)

func imparte(a, b int) (rez int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recuperat: %v", r)
		}
	}()
	return a / b, nil
}

func main() {
	fmt.Println(imparte(1, 0))
}
```

## Erori fatale: nu pot fi recuperate

Unele situații nu sunt panic-uri, ci **erori fatale** ale runtime-ului; `recover` nu le vede, iar programul se oprește imediat:

| Eroare fatală | Cauză |
|---|---|
| `concurrent map writes` | scrieri simultane într-un map, fără sincronizare |
| `all goroutines are asleep - deadlock!` | nicio goroutine nu mai poate avansa |
| `stack overflow` | recursie prea adâncă |
| `out of memory` | alocare imposibilă |

```go,panic
package main

func main() {
	defer func() { recover() }() // inutil: deadlock-ul e fatal
	ch := make(chan int)
	<-ch // fatal error: all goroutines are asleep - deadlock!
}
```

## Ieșiri care ocolesc `defer`

- `os.Exit` termină procesul **imediat**: `defer`-urile nu rulează. De aceea `log.Fatal` (care apelează `os.Exit(1)`) în `main` poate lăsa fișiere netrimise pe disc. Tiparul obișnuit e un `run() error` apelat din `main`, cu un singur `os.Exit` la final.
- `runtime.Goexit` termină goroutine-ul curent **după** ce rulează `defer`-urile lui. E ce folosește `t.FailNow()` în teste.

## Cum funcționează lanțurile de erori

Wrapping-ul se bazează doar pe convenții de metode:

- `Unwrap() error` — un singur „părinte” (ce creează `fmt.Errorf` cu un `%w`);
- `Unwrap() []error` — mai mulți (ce creează `errors.Join` sau `fmt.Errorf` cu mai multe `%w`).

`errors.Is` parcurge arborele **în adâncime**, iar la fiecare nod verifică `==`, apoi o metodă opțională `Is(error) bool`. Asta permite egalitate „semantică”:

```go
package main

import (
	"errors"
	"fmt"
)

type EroareHTTP struct{ Cod int }

func (e EroareHTTP) Error() string { return fmt.Sprintf("http %d", e.Cod) }

// Is: orice 5xx se potrivește cu ErrServer
func (e EroareHTTP) Is(tinta error) bool {
	return tinta == ErrServer && e.Cod >= 500
}

var ErrServer = errors.New("eroare de server")

func main() {
	err := fmt.Errorf("apel API: %w", EroareHTTP{Cod: 503})
	fmt.Println(errors.Is(err, ErrServer)) // true
}
```

`errors.As` poate fi personalizat analog, cu o metodă `As(any) bool`.

## Costul erorilor

Crearea unei erori cu `errors.New` sau `fmt.Errorf` alocă (și `fmt.Errorf` formatează). Pe drumuri fierbinți în care eroarea e frecventă și „normală” (de exemplu un parser care încearcă mai multe variante), folosește santinele pre-alocate în loc de erori noi la fiecare apel. Un `if err != nil` pe drumul fără eroare costă practic nimic: e o comparație cu zero, iar predicția de salt o ghicește aproape mereu.

## Pe scurt

- `defer f(x)` evaluează `f` și `x` imediat.
- `recover` funcționează doar apelat direct din funcția amânată, în aceeași goroutine.
- Un panic nerecuperat în orice goroutine oprește tot programul.
- Erorile fatale (map concurent, deadlock) nu pot fi prinse.
- `os.Exit` ocolește `defer`; `Unwrap`, `Is` și `As` definesc lanțurile.
