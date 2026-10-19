# Arrays și slices — Senior

Structura internă a unui slice, algoritmul de creștere al lui `append`, conversiile între slices și array-uri și câteva detalii de memorie care contează în codul de performanță.

## Antetul unui slice

Un slice e o structură de trei cuvinte, transmisă **prin valoare**:

```go,ignore
type slice struct {
	array unsafe.Pointer // primul element vizibil
	len   int
	cap   int // câte elemente sunt disponibile de la array încolo
}
```

De aici vin toate comportamentele de la nivelurile anterioare: o funcție care primește un slice primește o copie a antetului (deci vede aceleași elemente), dar modificarea `len` prin `append` rămâne în copia ei.

`unsafe.SliceData(s)` întoarce pointerul la primul element, iar `unsafe.Slice(p, n)` construiește un slice dintr-un pointer și o lungime; amândouă sunt utile la interoperabilitatea cu C sau cu apeluri de sistem.

## Cum crește `append`

Când capacitatea nu ajunge, `runtime.growslice` alocă un array nou și copiază elementele. Noua capacitate:

1. dacă e nevoie de mai mult decât dublul capacității vechi, ia exact cât e nevoie;
2. altfel, sub 256 de elemente, **dublează**;
3. peste 256, crește treptat de la ×2 spre ×1,25 (formula `cap += (cap + 3*256) / 4`);
4. rezultatul e rotunjit în sus la o **clasă de dimensiune** a alocatorului, deci capacitatea finală poate fi puțin mai mare.

```go
package main

import "fmt"

func main() {
	var s []int64
	ultima := -1
	for i := range 2000 {
		s = append(s, int64(i))
		if cap(s) != ultima {
			fmt.Print(cap(s), " ")
			ultima = cap(s)
		}
	}
	fmt.Println()
	// Go 1.26: 4 8 16 32 64 128 256 512 848 1280 1792 2560
}
```

Creșterea geometrică face ca `append` să fie **O(1) amortizat**, dar fiecare realocare copiază tot și lasă array-ul vechi pentru GC. Pentru volume mari cunoscute, prealocarea sau `slices.Grow(s, n)` elimină realocările.

## Conversii slice ↔ array

```go
package main

import "fmt"

func main() {
	s := []byte{1, 2, 3, 4}

	p := (*[4]byte)(s) // Go 1.17: pointer la array, aceeași memorie
	p[0] = 100

	a := [4]byte(s) // Go 1.20: array, copie
	a[1] = 200

	fmt.Println(s, *p, a) // [100 2 3 4] [100 2 3 4] [100 200 3 4]

	// [8]byte(s) ar da panic: lungimea slice-ului (4) e mai mică decât 8
}
```

Un array poate fi transformat în slice cu `arr[:]`, fără copiere. Conversiile sunt utile la hash-uri și chei de lungime fixă (de exemplu `[32]byte` ca cheie de map dintr-un `[]byte`).

## `clear`, `slices.Delete` și referințele uitate

Elementele dintre `len` și `cap` sunt invizibile, dar **încă referite** de array. Dacă sunt pointeri, GC-ul nu le poate elibera:

```go
package main

import (
	"fmt"
	"slices"
)

type Mare struct{ date [1 << 16]byte }

func main() {
	s := []*Mare{{}, {}, {}}
	s = s[:1] // s[1] și s[2] sunt încă în array: memoria nu se eliberează

	t := []*Mare{{}, {}, {}}
	t = slices.Delete(t, 1, 3) // din Go 1.22 pune nil pe pozițiile eliberate
	fmt.Println(len(s), len(t), t[:3][1] == nil)

	u := []int{1, 2, 3}
	clear(u) // Go 1.21: pune valoarea zero în toate elementele; len rămâne 3
	fmt.Println(u)
}
```

Funcțiile din `slices` care micșorează (`Delete`, `DeleteFunc`, `Compact`, `Replace`) curăță din Go 1.22 coada rămasă, ca să nu țină obiecte în viață.

## `append` și goroutines

Două goroutines care fac `append` pe sub-slices ale aceluiași array, cu capacitate liberă, scriu în aceeași memorie: **data race**, chiar dacă fiecare are „propriul” slice. Tiparul periculos:

```go,ignore
baza := make([]int, 0, 10)
go func() { a := append(baza, 1); use(a) }() // ambele scriu în baza[0]
go func() { b := append(baza, 2); use(b) }()
```

Detectorul de curse (`go test -race`) prinde asta. Soluția: fiecare goroutine primește o copie (`slices.Clone`) sau un sub-slice cu capacitate limitată (`baza[:0:0]`).

## Iterarea: index vs. valoare

`for _, v := range s` copiază fiecare element în `v`. Pentru elemente mari (structuri de sute de octeți), `for i := range s` și accesul `s[i].Camp` evită copia. Pentru elemente mici, diferența e neglijabilă, iar compilatorul elimină verificările de limite în ambele variante.

## Array-uri mari pe stivă

Array-urile sunt valori: un `[1 << 20]byte` local poate fi alocat pe stivă, iar trecerea lui ca argument îl copiază integral. Compilatorul mută pe heap variabilele mai mari de 10 MiB (sau 64 KiB pentru `make` implicit); în rest, atenție la copierile ascunse ale array-urilor mari (în `range`, la atribuire, la apeluri).

## Pe scurt

- Slice = (pointer, len, cap), transmis prin valoare.
- `append` dublează sub 256 de elemente, apoi crește spre ×1,25, cu rotunjire la clase de dimensiune.
- `(*[N]T)(s)` partajează memoria, `[N]T(s)` copiază.
- Elementele de după `len` țin obiectele în viață; `slices.Delete` și `clear` ajută.
- `append` concurent pe același array e data race.
