# Generics — Middle

Constrângeri proprii, operatorul `~`, constrângeri cu metode și, mai ales, întrebarea practică: când merită generics și când e mai bună o interfață.

## Constrângeri proprii: uniuni și `~`

O constrângere e o interfață. Pe lângă metode, poate conține o **uniune** de tipuri. `~T` înseamnă „orice tip al cărui tip de bază e `T`”, inclusiv tipurile definite de utilizator:

```go
package main

import "fmt"

type Numar interface {
	~int | ~int64 | ~float64
}

func Suma[T Numar](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

type Lei float64 // tip definit, cu tipul de bază float64

func main() {
	fmt.Println(Suma([]int{1, 2, 3}))
	fmt.Println(Suma([]Lei{9.99, 0.01})) // funcționează datorită lui ~float64
}
```

Fără `~`, `Lei` n-ar satisface constrângerea: `Lei` și `float64` sunt tipuri diferite. Pachetul `golang.org/x/exp/constraints` are constrângeri gata făcute (`Integer`, `Float`, `Signed`).

## Constrângeri cu metode

O constrângere poate cere metode, ca o interfață obișnuită. Combinația „tipuri + metode” e posibilă, dar rară:

```go
package main

import (
	"fmt"
	"strings"
)

func Uneste[T fmt.Stringer](elemente []T, sep string) string {
	parti := make([]string, len(elemente))
	for i, e := range elemente {
		parti[i] = e.String()
	}
	return strings.Join(parti, sep)
}

type Culoare int

func (c Culoare) String() string { return [...]string{"roșu", "verde", "albastru"}[c] }

func main() {
	fmt.Println(Uneste([]Culoare{0, 2, 1}, ", "))
}
```

De ce nu `[]fmt.Stringer` direct? Pentru că un `[]Culoare` **nu** e un `[]fmt.Stringer`; ar trebui convertit element cu element. Varianta generică primește direct `[]Culoare`.

## Generics sau interfață?

| Folosește generics când… | Folosește o interfață când… |
|---|---|
| scrii structuri de date (liste, cozi, seturi, cache-uri) | ai nevoie de comportament diferit pe tipuri (polimorfism) |
| aceeași logică merge pe slices/maps de orice tip | implementările au cod diferit, doar semnătura comună |
| vrei să eviți conversiile `any` și type assertion-urile | apelantul pune tipuri diferite în aceeași colecție |
| folosești operatori (`<`, `+`) pe tipuri diferite | metoda e tot ce contează |

Regula lui Ian Lance Taylor: dacă te trezești scriind exact același cod de mai multe ori, diferind doar prin tipuri, ia în considerare generics. Dacă doar **apelezi metode**, o interfață e de obicei suficientă.

## Tipuri generice cu constrângeri

```go
package main

import (
	"fmt"
	"maps"
	"slices"
)

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable](elem ...T) *Set[T] {
	s := &Set[T]{m: make(map[T]struct{}, len(elem))}
	for _, e := range elem {
		s.Add(e)
	}
	return s
}

func (s *Set[T]) Add(v T)      { s.m[v] = struct{}{} }
func (s *Set[T]) Has(v T) bool { _, ok := s.m[v]; return ok }
func (s *Set[T]) Len() int     { return len(s.m) }

func Intersectie[T comparable](a, b *Set[T]) *Set[T] {
	rez := NewSet[T]()
	for v := range a.m {
		if b.Has(v) {
			rez.Add(v)
		}
	}
	return rez
}

func main() {
	a := NewSet(1, 2, 3, 4)
	b := NewSet(3, 4, 5)
	i := Intersectie(a, b)
	fmt.Println(slices.Sorted(maps.Keys(i.m))) // [3 4]
}
```

## Metodele nu pot avea parametri de tip proprii

O metodă poate folosi parametrii tipului ei, dar **nu** poate declara alții noi. Operațiile care schimbă tipul (de exemplu `Map` de la `T` la `U`) se scriu ca funcții:

```go,compile_fail
package main

type Lista[T any] struct{ elem []T }

func (l Lista[T]) Map[U any](f func(T) U) Lista[U] { // eroare: methods cannot have type parameters
	return Lista[U]{}
}

func main() {}
```

## Deducerea parțială

Poți da explicit doar primii parametri de tip, iar restul se deduc:

```go
package main

import "fmt"

func Converteste[Tinta, Sursa ~int | ~float64](s []Sursa) []Tinta {
	rez := make([]Tinta, len(s))
	for i, v := range s {
		rez[i] = Tinta(v)
	}
	return rez
}

func main() {
	f := Converteste[float64]([]int{1, 2, 3}) // Sursa = int, dedus
	fmt.Println(f)
}
```

De aceea, la proiectarea unei funcții, pune **primii** parametrii de tip care nu pot fi deduși din argumente.

## Valoarea zero și comparația cu ea

`var zero T` merge pentru orice `T`. Comparația `v == zero` cere însă `comparable`. Pentru un `T any`, varianta e `reflect.ValueOf(&v).Elem().IsZero()`, dar de obicei e un semn că ai nevoie de constrângerea `comparable`.

## Pe scurt

- `~T` include tipurile definite peste `T`; uniunile listează tipuri permise.
- Constrângerile pot cere metode, ca interfețele.
- Generics pentru structuri de date și algoritmi; interfețe pentru comportament.
- Metodele nu pot avea parametri de tip proprii: folosește funcții.
- Pune primii parametrii de tip care nu se pot deduce.
