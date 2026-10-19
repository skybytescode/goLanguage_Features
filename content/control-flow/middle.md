# Control flow — Middle

Instrucțiunile de control au câteva detalii care contează în codul real: etichete pentru bucle imbricate, `fallthrough`, semantica variabilelor de buclă și stilul „return devreme”.

## Etichete: `break` și `continue` din bucle imbricate

Un `break` simplu iese doar din bucla cea mai apropiată. Cu o etichetă, poți ieși din bucla exterioară:

```go
package main

import "fmt"

func main() {
	matrice := [][]int{{1, 2, 3}, {4, -1, 6}, {7, 8, 9}}

cautare:
	for i, rand := range matrice {
		for j, v := range rand {
			if v < 0 {
				fmt.Printf("valoare negativă la [%d][%d]\n", i, j)
				break cautare
			}
		}
	}
}
```

Capcană frecventă: `break` în interiorul unui `switch` sau `select` iese doar din `switch`/`select`, **nu** din bucla care îl conține:

```go
package main

import "fmt"

func main() {
	comenzi := []string{"a", "stop", "b"}
bucla:
	for _, c := range comenzi {
		switch c {
		case "stop":
			break bucla // fără etichetă, ar ieși doar din switch
		default:
			fmt.Println("procesez", c)
		}
	}
}
```

## `fallthrough`

`fallthrough` trece **necondiționat** în corpul următorului `case`, fără să-i mai verifice condiția. E rar folosit; de obicei o listă de valori (`case a, b:`) e mai clară.

```go
package main

import "fmt"

func main() {
	nivel := 2
	switch nivel {
	case 3:
		fmt.Println("acces la arhivă")
		fallthrough
	case 2:
		fmt.Println("acces la rapoarte")
		fallthrough
	case 1:
		fmt.Println("acces la panou")
	}
}
```

## `switch` cu inițializare

Ca `if`, `switch` acceptă o instrucțiune scurtă înainte de expresie:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	switch mediu := os.Getenv("APP_ENV"); mediu {
	case "prod":
		fmt.Println("producție")
	case "":
		fmt.Println("nesetat, folosesc dev")
	default:
		fmt.Println("mediu:", mediu)
	}
}
```

## Variabilele de buclă (Go 1.22+)

Înainte de Go 1.22, variabila unei bucle `for` era **una singură** pentru toată bucla. Closure-urile și goroutine-urile care o capturau vedeau toate ultima valoare. De la Go 1.22, fiecare iterație are propria variabilă:

```go
package main

import "fmt"

func main() {
	var functii []func()
	for i := range 3 {
		functii = append(functii, func() { fmt.Print(i, " ") })
	}
	for _, f := range functii {
		f() // Go 1.22+: 0 1 2   (înainte: 3 3 3)
	}
	fmt.Println()
}
```

Comportamentul depinde de versiunea `go` din `go.mod`: modulele care declară `go 1.21` sau mai vechi păstrează semantica veche. Dacă vezi cod vechi cu `i := i` în buclă, era soluția pentru problema asta; acum poate fi șters.

## Return devreme (guard clauses)

Codul Go idiomatic evită `else` după un `return`. Tratezi cazurile de eroare întâi, iar „drumul fericit” rămâne la indentarea minimă:

```go
package main

import (
	"errors"
	"fmt"
)

func imparte(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("împărțire la zero")
	}
	return a / b, nil
}

func main() {
	r, err := imparte(10, 0)
	if err != nil {
		fmt.Println("eroare:", err)
		return
	}
	fmt.Println(r)
}
```

## `range` peste diverse tipuri

| Expresie | Prima valoare | A doua valoare |
|---|---|---|
| array, slice, pointer la array | index `int` | elementul |
| string | indexul octetului | `rune` (caracter Unicode) |
| map | cheia | valoarea |
| channel | elementul | — |
| întreg `n` | `0 … n-1` | — |
| funcție iterator | ce dă `yield` | ce dă `yield` |

Ultimul rând (funcțiile iterator, Go 1.23) are propriul subiect: [Range și iteratori](../../iteratori/middle/).

## `goto`

Go are `goto`, dar e folosit aproape exclusiv în cod generat. Nu poate sări peste declarații de variabile și nici în interiorul unui bloc. În codul scris de mână, preferă bucle și funcții.

## Pe scurt

- Etichetele rezolvă ieșirea din bucle imbricate și din `switch`/`select` în buclă.
- `fallthrough` ignoră condiția următorului `case`.
- Din Go 1.22 fiecare iterație are variabile proprii; nu mai ai nevoie de `i := i`.
- Tratează erorile întâi și ieși devreme.
