# Generics — Senior

Cum compilează Go codul generic (GC shape stenciling cu dicționare), ce consecințe are asta pentru performanță, ce sunt type set-urile, și ce limitări are designul actual.

## Implementarea: GC shape stenciling + dicționare

Există două strategii clasice:

- **monomorfizare** (C++, Rust): o copie a codului pentru fiecare tip; rapid, dar binare mari și compilare lentă;
- **boxing** (Java): o singură copie, totul prin pointeri și dispatch dinamic; compact, dar lent.

Go a ales un compromis. Compilatorul generează câte o copie pentru fiecare **GC shape** — tipurile cu aceeași reprezentare în memorie din perspectiva alocatorului și a GC-ului. `int` și `int64` au aceeași formă pe 64 de biți; un tip definit `type ID int` are forma lui `int`; și, important, **toate tipurile pointer au aceeași formă**.

Fiecare copie primește un **dicționar** ascuns, ca prim argument, cu informațiile specifice tipului concret: descriptorul de tip, `itab`-urile pentru metodele din constrângere, funcțiile de egalitate și hash.

```go,ignore
// ce scrii
func Suma[T Numar](s []T) T

// aproximativ ce generează compilatorul
func Suma[go.shape.int](dict *dictionary, s []int) int
func Suma[go.shape.float64](dict *dictionary, s []float64) float64
```

## Consecința: metodele pe tipuri pointer trec prin dicționar

Pentru tipuri valoare (`int`, `float64`, structuri), fiecare formă are cod specializat; operațiile aritmetice și comparațiile sunt directe. Pentru tipuri pointer, toate împart **aceeași** copie, iar un apel de metodă din constrângere devine un apel indirect, prin dicționar — similar cu un apel prin interfață, dar cu un nivel de indirectare în plus:

```go,ignore
func Afiseaza[T fmt.Stringer](v T) string {
	return v.String() // pentru T = *Persoana: apel indirect prin dicționar, fără inlining
}
```

Concluzia practică: generics sunt excelente pentru **structuri de date** și algoritmi pe tipuri valoare. Pentru cod care doar apelează metode, o interfață simplă e la fel de rapidă sau mai rapidă, iar compilatorul o poate devirtualiza mai ușor. Măsoară cu benchmark-uri înainte de a „optimiza” trecând la generics.

## Type sets

Din Go 1.18, o interfață definește un **type set**: mulțimea tipurilor care o satisfac.

- `interface{ String() string }` — toate tipurile cu metoda `String`;
- `interface{ ~int | ~string }` — tipurile cu tipul de bază `int` sau `string`;
- `interface{ ~int; String() string }` — intersecția celor două.

Un tip satisface o constrângere dacă e în type set-ul ei. Operațiile permise în funcția generică sunt cele valabile pentru **toate** tipurile din set: `+` e permis pe `~int | ~float64`, dar nu pe `~int | ~[]byte`.

## Tipul de bază comun și `range`

Unele operații (`range`, `make`, operații pe channels) cer ca toate tipurile din set să aibă **același tip de bază**. Până la Go 1.24 specificația numea acest tip comun **core type**; din Go 1.25 noțiunea a fost scoasă din specificație, iar regulile sunt descrise separat pentru fiecare operație, cu același efect în practică. Indexarea e mai permisivă: e suficient ca toate tipurile să dea același tip de element:

```go
package main

import "fmt"

type Bytes interface {
	~[]byte | ~string
}

func Primul[T Bytes](s T) byte {
	return s[0] // indexarea e permisă: ambele dau byte
}

func main() {
	fmt.Println(Primul("abc"), Primul([]byte{7}))
}
```

`for range s` pe un `T` cu set-ul `~[]byte | ~string` nu compilează (`[]byte and string have different underlying types`): `range` produce valori diferite pentru cele două (octeți vs. rune).

## `comparable` și interfețele

Din Go 1.20, tipurile interfață satisfac `comparable`, chiar dacă comparația lor poate da panic la rulare (vezi [Interfețe](../../interfete/senior/)). Spec-ul distinge între tipuri **strict comparabile** (unde `==` nu poate da panic) și comparabile în sens larg.

## Constrângeri care se referă la ele însele (Go 1.26)

Un tip generic poate apărea în propria listă de parametri de tip. Asta permite constrângeri de forma „T trebuie să aibă o metodă care primește un T”:

```go
package main

import "fmt"

type Adunabil[A Adunabil[A]] interface {
	Aduna(A) A
}

func SumaToate[A Adunabil[A]](xs ...A) A {
	var total A
	for _, x := range xs {
		total = total.Aduna(x)
	}
	return total
}

type Vector struct{ X, Y int }

func (v Vector) Aduna(w Vector) Vector { return Vector{v.X + w.X, v.Y + w.Y} }

func main() {
	fmt.Println(SumaToate(Vector{1, 2}, Vector{3, 4}, Vector{5, 6})) // {9 12}
}
```

Înainte de Go 1.26, același lucru cerea o constrângere scrisă direct în listă, de forma `[A interface{ Aduna(A) A }]`.

## Limitările designului

- **Fără parametri de tip pe metode.** Ar cere generarea de cod la rulare sau ar strica legătura dintre metode și interfețe.
- **Fără specializare.** Nu poți scrie o variantă separată pentru `T = string`; se folosește un type switch pe `any(v)`, pe care compilatorul îl simplifică de obicei la instanțiere.
- **Fără parametri variadici de tip** (`func F[T ...any]`) și fără valori constante ca parametri (`Array[T, N]`).
- **Inferența** nu folosește tipul rezultatului așteptat, doar argumentele (cu câteva excepții pentru funcții transmise ca argumente, din Go 1.21).

## Pe scurt

- O copie de cod per GC shape, plus un dicționar cu informațiile tipului.
- Toate tipurile pointer împart aceeași copie: metodele sunt apelate indirect.
- Generics pentru structuri de date pe tipuri valoare; măsoară înainte să optimizezi.
- Type set-urile și tipurile de bază comune decid ce operații sunt permise.
- Din Go 1.26, constrângerile pot fi auto-referențiale.
