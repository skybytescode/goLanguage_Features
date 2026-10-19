# Goroutines — Junior

O **goroutine** e o funcție care rulează concurent cu restul programului. Pornirea uneia costă foarte puțin (câțiva kilobytes de memorie), așa că un program Go poate avea zeci de mii sau chiar milioane de goroutines simultan.

## Cuvântul cheie `go`

Pui `go` în fața unui apel de funcție, iar funcția pornește într-o goroutine nouă. Apelul nu așteaptă terminarea ei:

```go
package main

import (
	"fmt"
	"time"
)

func saluta(nume string) {
	fmt.Println("salut,", nume)
}

func main() {
	go saluta("Ana") // pornește concurent
	go saluta("Ion")
	time.Sleep(100 * time.Millisecond) // DOAR pentru demonstrație: așteptăm grosier
	fmt.Println("gata")
}
```

Ordinea mesajelor „salut” poate diferi la fiecare rulare.

## Când `main` se termină, totul se oprește

Programul se termină când `main` se întoarce, **fără** să aștepte celelalte goroutines:

```go
package main

import "fmt"

func main() {
	go fmt.Println("probabil nu apuc să rulez")
	fmt.Println("main s-a terminat")
}
```

`time.Sleep` din primul exemplu e o soluție proastă: nu știi cât să aștepți. Soluția corectă e `sync.WaitGroup`.

## `sync.WaitGroup`: așteaptă să termine

Un `WaitGroup` numără goroutines active. Din Go 1.25, metoda `Go` pornește goroutine-ul și îl numără automat; `Wait` blochează până când toate au terminat:

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	for i := range 3 {
		wg.Go(func() {
			fmt.Println("lucrătorul", i, "a terminat")
		})
	}
	wg.Wait() // așteaptă toate cele 3
	fmt.Println("toți au terminat")
}
```

În cod mai vechi vei vedea forma manuală, cu `Add` și `Done`:

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	for i := range 3 {
		wg.Add(1) // înainte de go!
		go func() {
			defer wg.Done()
			fmt.Println("lucrătorul", i)
		}()
	}
	wg.Wait()
}
```

`Add` trebuie apelat **înainte** de `go`, altfel `Wait` ar putea vedea contorul la 0 și s-ar întoarce prea devreme.

## Goroutines anonime

Cel mai des, goroutine-ul e o funcție anonimă. Din Go 1.22, fiecare iterație a buclei are propria variabilă `i`, deci capturarea ei e sigură.

## Concurență nu înseamnă paralelism

- **Concurență**: mai multe sarcini progresează în aceeași perioadă (structura programului).
- **Paralelism**: mai multe sarcini rulează exact în același moment, pe nuclee diferite.

Go rulează goroutines în paralel pe toate nucleele disponibile (`runtime.GOMAXPROCS`), dar programul tău trebuie gândit corect și dacă ar exista un singur nucleu.

```go
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("nuclee:", runtime.NumCPU())
	fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0)) // 0 = doar citește valoarea
}
```

## Comunicarea între goroutines

Goroutines nu întorc valori. Ca să trimiți rezultate înapoi, folosești **channels** (subiectul următor) sau o variabilă protejată de un mutex. Proverbul Go:

> Nu comunica prin memorie partajată; partajează memoria comunicând.

```go
package main

import "fmt"

func main() {
	rezultate := make(chan int)
	for i := 1; i <= 3; i++ {
		go func() {
			rezultate <- i * i // trimite rezultatul
		}()
	}
	suma := 0
	for range 3 {
		suma += <-rezultate // primește câte un rezultat
	}
	fmt.Println("suma pătratelor:", suma) // 14
}
```

## Pe scurt

- `go f()` pornește `f` concurent, fără să aștepte.
- Programul se oprește când `main` se termină.
- `sync.WaitGroup` (`wg.Go` + `wg.Wait`) așteaptă goroutines.
- Goroutines sunt ieftine: poți avea foarte multe.
- Rezultatele se trimit prin channels.
