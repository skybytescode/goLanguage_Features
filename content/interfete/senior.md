# Interfețe — Senior

Cum arată o interfață în memorie, ce e un `itab`, cât costă un apel dinamic și o conversie la interfață, și când comparația a două interfețe dă panic.

## Două reprezentări: `eface` și `iface`

Runtime-ul are două structuri, de câte două cuvinte fiecare:

```go,ignore
// interfața goală (any)
type eface struct {
	_type *_type         // descrierea tipului dinamic
	data  unsafe.Pointer // pointer la valoare
}

// interfață cu metode
type iface struct {
	tab  *itab           // tipul dinamic + tabela de metode
	data unsafe.Pointer
}

type itab struct {
	inter *interfacetype // interfața
	_type *_type         // tipul concret
	hash  uint32         // copie a hash-ului tipului, pentru type switch
	fun   [1]uintptr     // adresele metodelor; de fapt are lungime variabilă
}
```

Un `itab` e calculat **o singură dată** pentru fiecare pereche (interfață, tip concret). Perechile cunoscute la compilare sunt generate static; cele descoperite la rulare (de exemplu prin type assertion la o interfață) sunt calculate și păstrate într-un tabel hash global.

Câmpul `data` e mereu un pointer. Ce se întâmplă cu valorile care nu sunt pointeri e subiectul secțiunii următoare.

## Conversia la interfață poate aloca

Când pui o valoare non-pointer într-o interfață, runtime-ul are nevoie de un pointer la o copie a ei. De obicei asta înseamnă o **alocare pe heap** (`runtime.convT64`, `convTstring` etc.). Excepții, fără alocare:

- valori de tip pointer (pointerul e pus direct în `data`);
- valori de dimensiune zero;
- întregi de un octet și întregi mici (0–255), care indică într-un tabel static din runtime;
- constante și valori pe care compilatorul le poate pune în memoria read-only;
- cazurile în care escape analysis dovedește că interfața nu scapă (atunci copia stă pe stivă).

```go
package main

import (
	"fmt"
	"testing"
)

var (
	sink      any
	mic, mare = 200, 100000 // variabile, nu constante
)

func main() {
	micAllocs := testing.AllocsPerRun(1000, func() { sink = mic })
	mareAllocs := testing.AllocsPerRun(1000, func() { sink = mare })
	fmt.Println(micAllocs, mareAllocs) // 0 1
}
```

În bucle fierbinți, `fmt.Println(x)` și logarea cu argumente `any` pot genera multe alocări mici tocmai din cauza aceasta.

## Costul unui apel prin interfață

Un apel `r.Read(p)` printr-o interfață înseamnă: încarcă `itab`, încarcă adresa din `fun[i]`, apel indirect. Pe procesoarele moderne, cu predicție bună, diferența față de un apel direct e de ordinul nanosecundelor. Costul real e altul: un apel indirect **nu poate fi inlined**, deci compilatorul pierde și optimizările care ar fi urmat (inclusiv escape analysis: argumentele unui apel prin interfață sunt considerate, de regulă, că scapă).

Compilatorul **devirtualizează** apelul când știe tipul concret (de exemplu, imediat după `var r io.Reader = &buf`), iar cu PGO o face și pe baza profilului: generează `if tip == *bytes.Buffer { apel direct } else { apel indirect }`.

## Type assertion și type switch

`x.(T)` cu `T` concret compară doar un pointer de tip (sau `itab`) — foarte ieftin. `x.(I)` cu `I` interfață trebuie să găsească un `itab`, ceea ce implică tabelul hash global (cu un cache per locul apelului din Go 1.22). Un type switch folosește câmpul `hash` din `itab` pentru a sări rapid la ramura potrivită; cu multe ramuri concrete, compilatorul generează căutare binară după hash.

## Comparația interfețelor poate da panic

Două interfețe sunt egale dacă au același tip dinamic și valori egale. Dacă tipul dinamic **nu e comparabil** (slice, map, funcție), comparația dă **panic la rulare**, chiar dacă la compilare totul e în regulă:

```go,panic
package main

import "fmt"

func main() {
	var a, b any = []int{1}, []int{1}
	fmt.Println(a == b) // panic: runtime error: comparing uncomparable type []int
}
```

Același panic apare la folosirea unei asemenea valori drept cheie într-un `map[any]T`. Constrângerea generică `comparable` acceptă interfețe (din Go 1.20) tocmai cu acest risc.

## Interfețe cu type sets

De la generics (Go 1.18), o interfață poate conține și **termeni de tip**:

```go,ignore
type Numar interface {
	~int | ~int64 | ~float64
}
```

Asemenea interfețe pot fi folosite **doar** ca constrângeri pentru parametri de tip, nu ca tipuri de variabile. Detalii în [Generics](../../generics/middle/).

## Metode neexportate: interfețe „sigilate”

O interfață care conține o metodă **neexportată** nu poate fi implementată în afara pachetului ei. E modul Go de a obține un set închis de implementări (un „sum type” restrâns):

```go,ignore
package forma

type Forma interface {
	Arie() float64
	sigilat() // doar tipurile din pachetul forma o pot implementa
}
```

Un tip din alt pachet o poate „implementa” doar încorporând o valoare care deja o implementează.

## Pe scurt

- `any` e `eface` (tip + date); interfețele cu metode sunt `iface` (itab + date).
- `itab`-urile se calculează o dată per pereche și se păstrează.
- Conversia valorilor non-pointer la interfață alocă de obicei.
- Apelurile prin interfață blochează inlining-ul; PGO le poate devirtualiza.
- `==` pe interfețe cu tip dinamic necomparabil dă panic.
