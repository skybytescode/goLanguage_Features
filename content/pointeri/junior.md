# Pointeri — Junior

Un **pointer** păstrează adresa de memorie a unei valori. Prin el poți citi sau modifica valoarea de la acea adresă. Go are pointeri, dar fără aritmetică pe pointeri, așa că sunt mult mai simpli (și mai siguri) decât în C.

## `&` și `*`

- `&x` înseamnă „adresa lui `x`”.
- `*p` înseamnă „valoarea de la adresa `p`”.
- `*int` e tipul „pointer la `int`”.

```go
package main

import "fmt"

func main() {
	x := 10
	p := &x // p are tipul *int și indică spre x

	fmt.Println(*p) // 10: citim prin pointer
	*p = 20         // scriem prin pointer
	fmt.Println(x)  // 20: x s-a schimbat
}
```

## De ce avem nevoie de pointeri: funcțiile primesc copii

În Go, argumentele sunt mereu **copiate**. O funcție care primește un `int` lucrează pe o copie și nu poate schimba variabila celui care o apelează. Cu un pointer, poate:

```go
package main

import "fmt"

func dubleazaCopie(n int) {
	n *= 2 // modifică doar copia locală
}

func dubleaza(n *int) {
	*n *= 2 // modifică valoarea originală
}

func main() {
	a := 5
	dubleazaCopie(a)
	fmt.Println(a) // 5

	dubleaza(&a)
	fmt.Println(a) // 10
}
```

## `nil`

Valoarea zero a unui pointer e `nil`: nu indică nicăieri. Citirea prin `nil` oprește programul cu un **panic**:

```go,panic
package main

import "fmt"

func main() {
	var p *int
	fmt.Println(p == nil) // true
	fmt.Println(*p)       // panic: invalid memory address or nil pointer dereference
}
```

Verifică pointerii care pot fi `nil` înainte să-i folosești.

## Pointeri la structuri

Pentru structuri, Go face dereferențierea automat: scrii `p.Nume`, nu `(*p).Nume`.

```go
package main

import "fmt"

type Persoana struct {
	Nume   string
	Varsta int
}

func aniversare(p *Persoana) {
	p.Varsta++ // echivalent cu (*p).Varsta++
}

func main() {
	ana := Persoana{Nume: "Ana", Varsta: 30}
	aniversare(&ana)
	fmt.Println(ana.Varsta) // 31

	ion := &Persoana{Nume: "Ion"} // & direct pe un literal de structură
	ion.Varsta = 25
	fmt.Println(*ion)
}
```

## `new`

`new(T)` alocă o valoare zero de tipul `T` și întoarce un pointer la ea:

```go
package main

import "fmt"

func main() {
	p := new(int) // *int, indică spre un 0
	*p = 42
	fmt.Println(*p)
}
```

În practică, `&T{...}` e mult mai folosit decât `new(T)`, pentru că poți da și valori câmpurilor.

## Nu există aritmetică pe pointeri

În Go nu poți scrie `p++` ca să treci la următorul element din memorie. Asta elimină o clasă întreagă de bug-uri (citirea în afara memoriei alocate). Pentru colecții folosești slices.

```go,compile_fail
package main

func main() {
	x := [3]int{1, 2, 3}
	p := &x[0]
	p++ // eroare: invalid operation: p++ (non-numeric type *int)
}
```

## Pe scurt

- `&x` ia adresa, `*p` citește sau scrie valoarea de la adresă.
- Argumentele sunt copiate; un pointer permite modificarea originalului.
- Un pointer `nil` dereferențiat produce panic.
- La structuri, `p.Camp` funcționează direct pe pointer.
- Nu există aritmetică pe pointeri.
