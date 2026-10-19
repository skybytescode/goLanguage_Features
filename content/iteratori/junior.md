# Range și iteratori — Junior

`for ... range` e modul standard de a parcurge orice colecție în Go. Din Go 1.22 merge și peste numere, iar din Go 1.23 și peste **funcții iterator**, ceea ce îți permite să-ți faci propriile colecții parcurgibile cu `range`.

## Recapitulare: ce poate parcurge `range`

```go
package main

import "fmt"

func main() {
	for i, v := range []string{"a", "b"} { // slice: index, valoare
		fmt.Println(i, v)
	}
	for k, v := range map[string]int{"x": 1} { // map: cheie, valoare
		fmt.Println(k, v)
	}
	for i, r := range "hé" { // string: index octet, caracter
		fmt.Println(i, string(r))
	}
	for i := range 3 { // întreg (Go 1.22): 0, 1, 2
		fmt.Println(i)
	}
	ch := make(chan int, 2)
	ch <- 10
	ch <- 20
	close(ch)
	for v := range ch { // channel: până la închidere
		fmt.Println(v)
	}
}
```

## Iteratori din biblioteca standard

Multe funcții din `slices`, `maps` și `strings` întorc **iteratori**: valori pe care le poți parcurge cu `range`, fără să creeze un slice intermediar.

```go
package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	nume := []string{"Ana", "Ion", "Eva"}

	for i, n := range slices.Backward(nume) { // de la coadă la început
		fmt.Println(i, n)
	}

	varste := map[string]int{"Ana": 30, "Ion": 25}
	for _, k := range slices.Sorted(maps.Keys(varste)) { // chei sortate
		fmt.Println(k, varste[k])
	}
}
```

## Primul tău iterator

Un iterator e o funcție care primește o altă funcție, numită de obicei `yield`. Iteratorul apelează `yield` pentru fiecare valoare. Dacă `yield` întoarce `false`, înseamnă că bucla s-a oprit (cu `break`) și iteratorul trebuie să se oprească:

```go
package main

import "fmt"

// numerePare produce numerele pare de la 0 până la max.
func numerePare(max int) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		for i := 0; i <= max; i += 2 {
			if !yield(i) {
				return // bucla a făcut break
			}
		}
	}
}

func main() {
	for n := range numerePare(10) {
		if n > 6 {
			break
		}
		fmt.Println(n) // 0 2 4 6
	}
}
```

Corpul buclei `for` devine, practic, funcția `yield`.

## Tipurile `iter.Seq` și `iter.Seq2`

Pachetul `iter` dă nume celor două forme de iteratori:

- `iter.Seq[V]` — produce o valoare pe pas (ca `for v := range`);
- `iter.Seq2[K, V]` — produce două valori pe pas (ca `for k, v := range`).

```go
package main

import (
	"fmt"
	"iter"
)

func Numara(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 1; i <= n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func Enumera(cuvinte []string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, c := range cuvinte {
			if !yield(i+1, c) {
				return
			}
		}
	}
}

func main() {
	for n := range Numara(3) {
		fmt.Print(n, " ")
	}
	fmt.Println()
	for nr, c := range Enumera([]string{"unu", "doi"}) {
		fmt.Println(nr, c)
	}
}
```

## De la iterator la slice

`slices.Collect` adună toate valorile unui iterator într-un slice:

```go
package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	m := map[string]bool{"go": true, "rust": true}
	limbaje := slices.Collect(maps.Keys(m))
	slices.Sort(limbaje)
	fmt.Println(limbaje) // [go rust]
}
```

## Pe scurt

- `range` merge peste slices, arrays, maps, stringuri, channels, întregi și funcții.
- Un iterator e o funcție care apelează `yield` pentru fiecare valoare.
- Când `yield` întoarce `false`, iteratorul se oprește.
- `iter.Seq` și `iter.Seq2` sunt tipurile standard pentru iteratori.
- `slices.Collect` transformă un iterator în slice.
