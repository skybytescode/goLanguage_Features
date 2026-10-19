# Maps — Middle

Maps au câteva reguli care surprind: elementele lor nu sunt adresabile, sunt tipuri-referință, nu sunt sigure pentru acces concurent. Pachetul `maps` și câteva tipare acoperă restul nevoilor uzuale.

## Map-urile sunt referințe

O variabilă map conține un pointer la structura internă. Copierea variabilei sau trimiterea ei ca argument **nu** copiază datele:

```go
package main

import "fmt"

func adauga(m map[string]int) {
	m["nou"] = 1
}

func main() {
	a := map[string]int{"x": 1}
	b := a // același map
	b["y"] = 2
	adauga(a)
	fmt.Println(len(a), len(b)) // 3 3
}
```

Pentru o copie independentă, folosește `maps.Clone` (copie superficială: valorile de tip pointer sau slice sunt partajate în continuare).

## Elementele unui map nu sunt adresabile

Nu poți modifica direct un câmp al unei structuri stocate ca valoare:

```go,compile_fail
package main

type Punct struct{ X, Y int }

func main() {
	m := map[string]Punct{"a": {1, 2}}
	m["a"].X = 10 // eroare: cannot assign to struct field m["a"].X in map
}
```

Două soluții:

```go
package main

import "fmt"

type Punct struct{ X, Y int }

func main() {
	// 1. citești, modifici, scrii înapoi
	m := map[string]Punct{"a": {1, 2}}
	p := m["a"]
	p.X = 10
	m["a"] = p

	// 2. păstrezi pointeri
	mp := map[string]*Punct{"a": {1, 2}}
	mp["a"].X = 10

	fmt.Println(m["a"], *mp["a"])
}
```

## Seturi

Go nu are un tip set. Se folosește un map cu valori `struct{}` (zero octeți) sau `bool` (mai lizibil la verificare):

```go
package main

import "fmt"

func main() {
	vazut := map[string]struct{}{}
	for _, s := range []string{"a", "b", "a", "c", "b"} {
		if _, ok := vazut[s]; ok {
			fmt.Println("duplicat:", s)
			continue
		}
		vazut[s] = struct{}{}
	}
	fmt.Println(len(vazut)) // 3
}
```

## Pachetul `maps`

```go
package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	m := map[string]int{"b": 2, "a": 1, "c": 3}

	chei := slices.Sorted(maps.Keys(m)) // chei sortate (Go 1.23)
	fmt.Println(chei)                   // [a b c]

	copie := maps.Clone(m)
	copie["d"] = 4
	fmt.Println(maps.Equal(m, copie)) // false

	maps.DeleteFunc(copie, func(k string, v int) bool { return v%2 == 0 })
	fmt.Println(copie) // map[a:1 c:3]

	// map din perechi
	inv := maps.Collect(func(yield func(int, string) bool) {
		for k, v := range m {
			if !yield(v, k) {
				return
			}
		}
	})
	fmt.Println(inv[2]) // b
}
```

`maps.Keys` și `maps.Values` întorc **iteratori** (`iter.Seq`), nu slices. `slices.Collect` sau `slices.Sorted` îi transformă în slices. Detalii în [Range și iteratori](../../iteratori/middle/).

## Prealocarea

Dacă știi câte elemente vor fi, `make(map[K]V, n)` evită creșterile repetate:

```go
package main

import "fmt"

func main() {
	n := 10000
	patrate := make(map[int]int, n)
	for i := range n {
		patrate[i] = i * i
	}
	fmt.Println(len(patrate))
}
```

## `clear`

`clear(m)` (Go 1.21) șterge toate elementele, păstrând memoria alocată pentru reutilizare. Înainte, același efect se obținea cu o buclă de `delete`.

## Map-urile nu sunt sigure pentru goroutines

Citirile simultane sunt în regulă. O **scriere** simultană cu orice alt acces e o cursă de date, iar runtime-ul o detectează adesea și oprește programul cu `fatal error: concurrent map writes` (nu poate fi prinsă cu `recover`).

Soluția obișnuită e un mutex:

```go
package main

import (
	"fmt"
	"sync"
)

type Contoare struct {
	mu sync.Mutex
	m  map[string]int
}

func (c *Contoare) Inc(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[k]++
}

func main() {
	c := Contoare{m: map[string]int{}}
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() { c.Inc("vizite") })
	}
	wg.Wait()
	fmt.Println(c.m["vizite"]) // 100
}
```

`sync.RWMutex` ajută când citirile domină. `sync.Map` e optimizat pentru două cazuri anume: chei scrise o dată și citite de multe ori, sau goroutines care lucrează pe seturi disjuncte de chei. În rest, un map + mutex e de obicei mai rapid și mai clar.

## Chei compuse

O structură comparabilă e cheia ideală pentru mai multe dimensiuni; evită concatenarea de string-uri:

```go
package main

import "fmt"

type poz struct{ r, c int }

func main() {
	tabla := map[poz]string{{0, 0}: "X", {1, 1}: "O"}
	fmt.Println(tabla[poz{1, 1}])
}
```

## Pe scurt

- Map-urile sunt referințe; `maps.Clone` pentru copii.
- Elementele nu sunt adresabile: scrie înapoi sau stochează pointeri.
- `map[T]struct{}` pentru seturi.
- `maps.Keys` dă un iterator; `slices.Sorted` îl sortează.
- Accesul concurent cu scrieri cere mutex (sau `sync.Map` în cazuri speciale).
