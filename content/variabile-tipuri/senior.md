# Variabile, tipuri și constante — Senior

Aici ne uităm la regulile din specificație care explică comportamentul de la nivelurile anterioare: aritmetica exactă a constantelor, identitatea și asignabilitatea tipurilor, ordinea de inițializare și cum arată valorile în memorie.

## Constantele au precizie arbitrară

Constantele fără tip sunt calculate de compilator cu precizie (practic) nelimitată. Doar rezultatul final trebuie să încapă în tipul în care ajunge:

```go
package main

import "fmt"

const (
	Mare   = 1 << 100   // nu încape în niciun tip întreg
	Mic    = Mare >> 98 // 4: încape
	Treime = 1.0 / 3
)

func main() {
	fmt.Println(Mic)
	fmt.Println(Treime*3 == 1) // true: calculat exact, la compilare
}
```

Fiecare tip de constantă fără tip are un **tip implicit**, folosit când contextul nu cere altul (de exemplu în `x := c`):

| Constantă | Tip implicit |
|---|---|
| `42` | `int` |
| `4.2` | `float64` |
| `'x'` | `rune` |
| `2i` | `complex128` |
| `"s"` | `string` |
| `true` | `bool` |

De aceea `fmt.Println(Mare)` nu compilează: `Mare` ar deveni `int`, iar `1<<100` nu e reprezentabil în `int`.

## Conversii: constante vs. valori la rulare

Aceeași conversie se comportă diferit în funcție de operand:

```go
package main

import "fmt"

func main() {
	v := 300
	fmt.Println(uint8(v)) // 44: la rulare se păstrează doar biții de jos
	// fmt.Println(uint8(300)) // eroare de compilare: constanta nu e reprezentabilă

	f := 3.99
	fmt.Println(int(f))  // 3: trunchiere spre zero
	fmt.Println(int(-f)) // -3
}
```

Conversia unui `float64` care nu încape în tipul întreg țintă (de exemplu `int8(1e10)` pe o variabilă) are rezultat **dependent de implementare**; nu te baza pe el.

## Identitate și asignabilitate

Fiecare tip are un **tip de bază** (underlying type). Un tip **definit** (`type T ...`) e diferit de orice alt tip; tipurile literale (`[]int`, `map[string]int`, `func(int) error`) sunt identice dacă au aceeași structură.

O valoare de tip `V` poate fi atribuită unei variabile de tip `T` dacă, printre altele:

- `V` și `T` sunt identice;
- au același tip de bază **și** cel puțin unul dintre ele nu e tip definit;
- `T` e interfață și `V` o implementează.

```go
package main

import "fmt"

type IDuri []int
type Lista []int

func main() {
	brut := []int{1, 2, 3}
	var a IDuri = brut // ok: []int nu e tip definit
	// var b Lista = a // eroare: IDuri și Lista sunt ambele tipuri definite
	b := Lista(a) // conversia explicită e permisă: același tip de bază
	fmt.Println(a, b)
}
```

Regula explică de ce poți trece un `func(int) int` literal unde se cere un `type Handler func(int) int`, fără conversie.

## Alias-uri generice (Go 1.24)

Alias-urile pot avea parametri de tip, ceea ce ajută la mutarea tipurilor între pachete fără a rupe API-ul:

```go
package main

import "fmt"

type Set[T comparable] = map[T]struct{}

func main() {
	s := Set[string]{"a": {}, "b": {}}
	var m map[string]struct{} = s // identic: e doar un alias
	fmt.Println(len(m))
}
```

## `new` cu expresie (Go 1.26)

Începând cu Go 1.26, `new` acceptă și o expresie, nu doar un tip. E util pentru câmpurile opționale de tip pointer (de exemplu în structuri pentru JSON):

```go
package main

import "fmt"

type Optiuni struct {
	Limita *int
}

func main() {
	o := Optiuni{Limita: new(100)} // înainte: v := 100; &v
	fmt.Println(*o.Limita)
}
```

## Ordinea de inițializare

Variabilele de la nivel de pachet sunt inițializate **în ordinea dependențelor**, nu neapărat în ordinea din fișier. Abia după ele rulează funcțiile `init()`, în ordinea în care apar (fișierele sunt date compilatorului sortate după nume de comanda `go`):

```go
package main

import "fmt"

var (
	a = b + 1 // depinde de b, deci b se inițializează primul
	b = f()
)

func f() int {
	fmt.Println("inițializez b")
	return 41
}

func init() {
	fmt.Println("init: a =", a)
}

func main() {
	fmt.Println("main")
}
```

Pachetele importate sunt inițializate complet înaintea pachetului care le importă. Ciclurile de inițializare sunt erori de compilare.

## Cum arată valorile în memorie

Valorile au dimensiune și aliniere fixe. Câmpurile unei structuri sunt aliniate, iar compilatorul adaugă *padding* între ele; ordinea câmpurilor contează:

```go
package main

import (
	"fmt"
	"unsafe"
)

type Risipitor struct {
	a bool  // 1 octet + 7 padding
	b int64 // 8
	c bool  // 1 octet + 7 padding
}

type Compact struct {
	b int64 // 8
	a bool  // 1
	c bool  // 1 + 6 padding
}

func main() {
	fmt.Println(unsafe.Sizeof(Risipitor{}), unsafe.Sizeof(Compact{})) // 24 16
	fmt.Println(unsafe.Sizeof(""), unsafe.Sizeof([]int{}))            // 16 24
}
```

Un `string` e un antet de 2 cuvinte (pointer + lungime); un slice e de 3 cuvinte (pointer, lungime, capacitate). Analizorul `fieldalignment` (din `golang.org/x/tools`) sugerează ordinea optimă a câmpurilor.

## Virgula mobilă

`float64` urmează IEEE 754, cu toate consecințele:

```go
package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Println(0.1+0.2 == 0.3) // true: constante, calculate exact la compilare
	a, b := 0.1, 0.2
	fmt.Println(a+b == 0.3) // false: float64, la rulare
	fmt.Println(a + b)      // 0.30000000000000004

	nan := math.NaN()
	fmt.Println(nan == nan) // false: NaN nu e egal cu nimic

	m := map[float64]int{}
	m[nan] = 1
	m[nan] = 2
	fmt.Println(len(m)) // 2: cheile NaN nu pot fi găsite niciodată
}
```

Pentru bani folosește întregi (de exemplu bani/cenți) sau `math/big`.

## Pe scurt

- Constantele sunt exacte; doar valoarea finală trebuie să fie reprezentabilă.
- Asignabilitatea depinde de tipurile definite și de tipul de bază.
- Inițializarea pachetului urmează dependențele, apoi `init()`.
- Ordinea câmpurilor schimbă dimensiunea structurilor.
- Nu compara `float64` cu `==` după calcule.
