# Pointeri — Middle

Întrebarea practică nu e „ce e un pointer”, ci **când** să folosești unul. Răspunsul ține de mutabilitate, de dimensiunea valorilor și de semantica pe care o vrei pentru tipurile tale.

## Valoare sau pointer?

| Folosește un pointer când… | Folosește o valoare când… |
|---|---|
| funcția trebuie să modifice argumentul | valoarea e mică și imutabilă (ex. `time.Time`, un `Punct{X, Y}`) |
| structura e mare și copierea costă | vrei ca fiecare copie să fie independentă |
| tipul conține un `sync.Mutex` sau alt câmp care nu trebuie copiat | tipul e un tip de bază, slice, map, channel sau funcție |
| ai nevoie de „lipsa valorii” (`nil`) | |

Slices, maps, channels și funcțiile conțin deja pointeri interni. Un `*[]int` sau `*map[string]int` e rareori necesar: modificarea elementelor se vede oricum la apelant.

```go
package main

import "fmt"

func seteaza(m map[string]int) {
	m["a"] = 1 // vizibil la apelant, fără pointer
}

func adauga(s []int) {
	s = append(s, 4) // NU e vizibil: lungimea apelantului nu se schimbă
	_ = s
}

func main() {
	m := map[string]int{}
	seteaza(m)
	fmt.Println(m) // map[a:1]

	s := []int{1, 2, 3}
	adauga(s)
	fmt.Println(s) // [1 2 3]
}
```

Pentru `append`, convenția Go e să întorci slice-ul nou (`s = adauga(s)`), nu să primești `*[]int`.

## Poți întoarce adresa unei variabile locale

În C, asta ar fi un bug grav. În Go e complet sigur: compilatorul observă că adresa „scapă” din funcție și alocă variabila pe heap.

```go
package main

import "fmt"

type Config struct {
	Port int
}

func implicit() *Config {
	c := Config{Port: 8080}
	return &c // sigur: c trăiește cât timp există pointeri spre el
}

func main() {
	fmt.Println(implicit().Port)
}
```

## Câmpuri opționale

Un pointer face diferența între „valoarea zero” și „nesetat”. E util la configurări parțiale sau la JSON:

```go
package main

import "fmt"

type Actualizare struct {
	Nume   *string
	Varsta *int
}

func aplica(nume *string, varsta *int, a Actualizare) {
	if a.Nume != nil {
		*nume = *a.Nume
	}
	if a.Varsta != nil {
		*varsta = *a.Varsta
	}
}

func main() {
	nume, varsta := "Ana", 30
	zero := 0
	aplica(&nume, &varsta, Actualizare{Varsta: &zero}) // setăm vârsta la 0, numele rămâne
	fmt.Println(nume, varsta)
}
```

## Capcană: pointeri la elementele unui slice

`&s[i]` indică spre elementul din array-ul de sub slice. Dacă un `append` ulterior realocă array-ul, pointerul rămâne spre **vechiul** array:

```go
package main

import "fmt"

func main() {
	s := make([]int, 1, 1)
	p := &s[0]
	s = append(s, 2) // capacitate depășită: array nou
	*p = 100
	fmt.Println(s[0]) // 0: p indică spre array-ul vechi
}
```

Același lucru e valabil pentru pointerii la elementele unui map: în Go nici nu poți lua adresa unui element de map (`&m[k]` nu compilează), tocmai pentru că map-ul își mută elementele.

## Pointerii nil și metodele

O metodă cu receptor pointer **poate** fi apelată pe un pointer `nil`. Uneori e chiar util, ca valoare implicită sigură:

```go
package main

import "fmt"

type Nod struct {
	Val int
	Urm *Nod
}

func (n *Nod) Suma() int {
	if n == nil {
		return 0 // lista goală
	}
	return n.Val + n.Urm.Suma()
}

func main() {
	lista := &Nod{1, &Nod{2, &Nod{3, nil}}}
	fmt.Println(lista.Suma()) // 6
	var gol *Nod
	fmt.Println(gol.Suma()) // 0, fără panic
}
```

## Comparația pointerilor

Doi pointeri sunt egali dacă indică spre aceeași variabilă (sau sunt ambii `nil`), nu dacă valorile sunt egale:

```go
package main

import "fmt"

func main() {
	a, b := 1, 1
	pa, pb := &a, &b
	fmt.Println(pa == pb, *pa == *pb) // false true
}
```

## Pe scurt

- Pointer pentru mutabilitate, structuri mari și tipuri care nu trebuie copiate.
- Maps, slices și channels nu au nevoie de pointeri pentru a modifica elementele.
- Întoarcerea adresei unei variabile locale e sigură.
- Pointerii în slices pot rămâne spre un array vechi după `append`.
- Metodele pe pointeri pot trata explicit cazul `nil`.
