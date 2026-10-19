# Variabile, tipuri și constante — Middle

La nivelul acesta ne interesează capcanele: umbrirea variabilelor, diferența dintre tipurile definite și alias-uri, constantele „fără tip” și enumerările care se afișează frumos.

## Umbrirea variabilelor (shadowing)

`:=` declară o variabilă **nouă** în blocul curent, chiar dacă există una cu același nume într-un bloc exterior. E una dintre cele mai frecvente surse de bug-uri în Go:

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	valoare := 0
	if true {
		valoare, err := strconv.Atoi("42") // variabilă NOUĂ, vizibilă doar în if
		if err != nil {
			return
		}
		_ = valoare
	}
	fmt.Println(valoare) // 0, nu 42
}
```

Soluția: declari `err` separat și folosești `=`:

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	valoare := 0
	if true {
		var err error
		valoare, err = strconv.Atoi("42")
		if err != nil {
			return
		}
	}
	fmt.Println(valoare) // 42
}
```

> `go vet` are un analizor opțional, `shadow`, care găsește astfel de cazuri.

Regula pentru `:=` cu mai multe variabile: e permis dacă **cel puțin una** din stânga e nouă în blocul curent; celelalte sunt doar atribuite.

## Tipuri definite și alias-uri

`type Celsius float64` creează un tip **nou**, distinct de `float64`. Nu poți amesteca valorile fără conversie, dar poți adăuga metode:

```go
package main

import "fmt"

type Celsius float64
type Fahrenheit float64

func (c Celsius) ToF() Fahrenheit { return Fahrenheit(c*9/5 + 32) }

type Grade = Celsius // alias: alt nume pentru ACELAȘI tip

func main() {
	var t Celsius = 21.5
	var g Grade = t // merge: Grade și Celsius sunt identice
	fmt.Println(t.ToF(), g.ToF())
	fmt.Printf("%T\n", g) // main.Celsius
}
```

Tipurile definite fac API-urile mai sigure: o funcție care primește `Celsius` nu acceptă din greșeală un `Fahrenheit`.

## Enumerări care se afișează: `String()`

Orice tip cu metoda `String() string` implementează `fmt.Stringer`, iar `fmt` o folosește la afișare:

```go
package main

import "fmt"

type Stare int

const (
	Nou Stare = iota
	InLucru
	Gata
)

func (s Stare) String() string {
	switch s {
	case Nou:
		return "nou"
	case InLucru:
		return "în lucru"
	case Gata:
		return "gata"
	}
	return fmt.Sprintf("Stare(%d)", int(s))
}

func main() {
	fmt.Println(InLucru)  // în lucru
	fmt.Println(Stare(7)) // Stare(7)
}
```

> În proiectele mari, `String()` se generează cu `go generate` și unealta `stringer` (`golang.org/x/tools/cmd/stringer`).

Atenție: în `String()` nu apela `fmt.Sprintf("%v", s)` pe aceeași valoare: ar apela din nou `String()`, la infinit. De aceea am convertit la `int`.

## Trucuri cu `iota`

`iota` e o expresie: o poți combina cu operatori, iar expresia se repetă pe liniile următoare.

```go
package main

import "fmt"

type Permisiune uint8

const (
	Citire   Permisiune = 1 << iota // 1
	Scriere                         // 2
	Executie                        // 4
)

type Marime int64

const (
	_         = iota // sărim peste 0
	KB Marime = 1 << (10 * iota)
	MB
	GB
)

func main() {
	p := Citire | Scriere
	fmt.Println(p&Scriere != 0, p&Executie != 0) // true false
	fmt.Println(KB, MB, GB)
}
```

## Constante fără tip

`const x = 10` nu are tipul `int`: e o constantă **fără tip** (untyped). Primește un tip abia când e folosită, așa că se potrivește în orice context numeric:

```go
package main

import (
	"fmt"
	"time"
)

const timeout = 5 // fără tip

func main() {
	var f float64 = timeout    // ok: devine float64
	d := timeout * time.Second // ok: devine time.Duration
	fmt.Println(f, d)

	n := 5 // variabilă de tip int
	// d2 := n * time.Second         // eroare: int * time.Duration
	d2 := time.Duration(n) * time.Second
	fmt.Println(d2)
}
```

## Depășirea întregilor

La rulare, întregii **se reiau de la capăt** (wrap around) fără eroare. La compilare, constantele care nu încap sunt erori:

```go
package main

import (
	"fmt"
	"math"
)

func main() {
	var x int8 = math.MaxInt8
	x++
	fmt.Println(x) // -128

	var u uint = 0
	u--
	fmt.Println(u == math.MaxUint) // true
}
```

```go,compile_fail
package main

func main() {
	var x int8 = 200 // eroare: constanta 200 nu încape în int8
	_ = x
}
```

## Verbe utile în `fmt`

| Verb | Ce afișează |
|---|---|
| `%v` | valoarea, în formatul implicit |
| `%+v` | la structuri, și numele câmpurilor |
| `%#v` | sintaxa Go a valorii |
| `%T` | tipul |
| `%q` | string între ghilimele, cu escape |
| `%x` | hexazecimal |

## Pe scurt

- `:=` în blocuri interioare poate umbri variabile; verifică cu atenție codul cu `err`.
- `type T U` creează un tip nou; `type T = U` e doar un alt nume.
- Implementează `String()` pentru enumerări.
- Constantele fără tip se adaptează contextului; variabilele nu.
- Depășirea întregilor la rulare nu dă eroare.
