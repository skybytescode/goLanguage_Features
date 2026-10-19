# Variabile, tipuri și constante — Junior

Go e un limbaj **tipizat static**: fiecare variabilă are un tip cunoscut la compilare, iar compilatorul nu te lasă să amesteci tipurile din greșeală. În schimb, de cele mai multe ori nu trebuie să scrii tipul: compilatorul îl deduce.

## Primul program

```go
package main

import "fmt"

func main() {
	fmt.Println("Salut, Go!")
}
```

Orice program executabil are un pachet `main` și o funcție `main`. `fmt` e pachetul pentru afișare și formatare.

## Declararea variabilelor

Sunt trei forme uzuale:

```go
package main

import "fmt"

func main() {
	var a int = 10 // tip și valoare
	var b = 20     // tipul e dedus din valoare: int
	c := 30        // declarare scurtă, doar în interiorul funcțiilor

	var nume string // fără valoare: primește „valoarea zero”
	fmt.Println(a, b, c, nume == "")
}
```

`:=` e forma pe care o vei folosi cel mai des. Ea **declară** o variabilă nouă; pentru a schimba valoarea unei variabile existente folosești `=`.

## Valoarea zero

În Go nu există variabile neinițializate. Fiecare tip are o **valoare zero**:

| Tip | Valoarea zero |
|---|---|
| `int`, `float64` | `0` |
| `bool` | `false` |
| `string` | `""` |
| pointeri, slices, maps, channels, funcții, interfețe | `nil` |

```go
package main

import "fmt"

func main() {
	var n int
	var ok bool
	var s string
	var p *int
	fmt.Printf("%d %t %q %v\n", n, ok, s, p) // 0 false "" <nil>
}
```

## Tipurile de bază

| Categorie | Tipuri |
|---|---|
| Întregi cu semn | `int`, `int8`, `int16`, `int32`, `int64` |
| Întregi fără semn | `uint`, `uint8` (= `byte`), `uint16`, `uint32`, `uint64`, `uintptr` |
| Virgulă mobilă | `float32`, `float64` |
| Complexe | `complex64`, `complex128` |
| Text | `string`, `rune` (= `int32`, un caracter Unicode) |
| Logic | `bool` |

`int` are 64 de biți pe sistemele de 64 de biți. Pentru numere reale folosește `float64` dacă nu ai un motiv anume.

## Conversii explicite

Go **nu** convertește automat între tipuri numerice, nici măcar de la `int` la `float64`:

```go,compile_fail
package main

import "fmt"

func main() {
	x := 3
	y := 1.5
	fmt.Println(x * y) // eroare: tipuri nepotrivite int și float64
}
```

Conversia se scrie ca un apel `T(valoare)`:

```go
package main

import "fmt"

func main() {
	x := 3
	y := 1.5
	fmt.Println(float64(x) * y) // 4.5
	fmt.Println(int(y))         // 1: partea fracționară e tăiată
}
```

## Variabilele nefolosite sunt erori

O variabilă locală declarată și nefolosită oprește compilarea. Același lucru e valabil pentru importurile nefolosite. Dacă vrei să ignori o valoare, folosește identificatorul gol `_`:

```go,compile_fail
package main

func main() {
	x := 5 // eroare: declared and not used: x
}
```

## Constante

Constantele se declară cu `const` și sunt calculate la compilare:

```go
package main

import "fmt"

const Pi = 3.14159

const (
	ZileInSaptamana = 7
	Salut           = "Bună"
)

func main() {
	raza := 2.0
	fmt.Println(Pi * raza * raza)
	fmt.Println(Salut, ZileInSaptamana)
}
```

## `iota`: enumerări

În interiorul unui bloc `const`, `iota` începe de la 0 și crește cu 1 la fiecare linie. Așa se scriu enumerările în Go:

```go
package main

import "fmt"

type Zi int

const (
	Luni     Zi = iota // 0
	Marti              // 1
	Miercuri           // 2
)

func main() {
	fmt.Println(Luni, Marti, Miercuri)
}
```

## Pe scurt

- `:=` declară și inițializează; `=` doar atribuie.
- Orice variabilă are o valoare zero; nu există „nedefinit”.
- Conversiile între tipuri sunt mereu explicite.
- Variabilele și importurile nefolosite sunt erori de compilare.
- `const` și `iota` pentru valori fixe și enumerări.
