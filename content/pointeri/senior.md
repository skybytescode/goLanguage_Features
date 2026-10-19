# Pointeri — Senior

Pointerii determină unde trăiește o valoare (stivă sau heap) și cât lucru are garbage collector-ul. La nivelul acesta ne uităm la escape analysis, la pointerii interiori, la pointerii slabi (`weak`) și la funcțiile de curățare.

## Escape analysis: stivă sau heap?

Specificația Go nu vorbește despre stivă și heap. Compilatorul decide: o variabilă stă pe stiva funcției dacă poate dovedi că **nu e folosită după ce funcția se termină**. Altfel, „scapă” (escapes) pe heap, unde o va elibera GC-ul.

Alocările pe stivă sunt practic gratuite; cele pe heap costă la alocare și, mai ales, mai târziu, la colectare. Decizia se vede cu `-gcflags=-m`:

```go,ignore
package main

type Punct struct{ X, Y int }

func peStiva() int {
	p := Punct{1, 2} // nu scapă: rămâne pe stivă
	return p.X + p.Y
}

func peHeap() *Punct {
	p := Punct{1, 2} // moved to heap: p
	return &p
}

func prinInterfata(v any) {}

func main() {
	x := 42
	prinInterfata(x) // x escapes to heap (de obicei: valoarea e copiată într-o interfață)
}
```

```bash
$ go build -gcflags='-m' .
./main.go:10:2: moved to heap: p
./main.go:18:16: x escapes to heap
```

Cauze frecvente de escape:

- întoarcerea unui pointer la o variabilă locală;
- stocarea unui pointer într-o variabilă globală, un map, un channel sau un câmp de pe heap;
- conversia la o interfață (de ex. argumente pentru `fmt.Println`), dacă compilatorul nu poate demonstra că valoarea nu e păstrată;
- closure-uri care capturează prin referință și trăiesc mai mult decât funcția;
- dimensiuni necunoscute la compilare (`make([]byte, n)` cu `n` variabil) sau valori foarte mari.

Optimizarea tipică într-o buclă fierbinte: reutilizezi un buffer în loc să aloci unul nou la fiecare iterație, sau primești buffer-ul de la apelant (cum face `io.Reader.Read(p []byte)`).

## Pointerii interiori țin în viață tot obiectul

GC-ul Go acceptă pointeri **în interiorul** unui obiect (la un câmp sau la un element de array). Cât timp există un astfel de pointer, **tot** obiectul rămâne alocat:

```go
package main

import "fmt"

type Mare struct {
	date [1 << 20]byte // 1 MiB
	id   int
}

func idDin() *int {
	m := &Mare{id: 7}
	return &m.id // 8 octeți folosiți, 1 MiB ținut în viață
}

func main() {
	fmt.Println(*idDin())
}
```

Același efect apare cu sub-slice-urile: `mare[:10]` ține în viață tot array-ul de sub `mare`. Soluția e o copie: `slices.Clone(mare[:10])`.

## Pointeri slabi: pachetul `weak` (Go 1.24)

Un `weak.Pointer[T]` indică spre un obiect fără să-l țină în viață. După ce obiectul devine inaccesibil și e colectat, `Value()` întoarce `nil`. E baza pentru cache-uri care nu împiedică eliberarea memoriei:

```go
package main

import (
	"fmt"
	"runtime"
	"weak"
)

type Imagine struct{ pixeli []byte }

func main() {
	img := &Imagine{pixeli: make([]byte, 1<<20)}
	wp := weak.Make(img)

	fmt.Println(wp.Value() != nil) // true: img e încă folosit
	runtime.KeepAlive(img)

	img = nil
	runtime.GC()
	fmt.Println(wp.Value() == nil) // true: obiectul a fost colectat
}
```

Doi pointeri slabi creați din același obiect sunt egali (`==`), deci pot fi chei de map; pachetul `unique` (Go 1.23) folosește exact acest mecanism pentru internarea valorilor.

## Curățare: `runtime.AddCleanup` (Go 1.24)

`runtime.AddCleanup(ptr, f, arg)` programează `f(arg)` după ce obiectul de la `ptr` devine inaccesibil. Înlocuiește `runtime.SetFinalizer`, care avea capcane serioase (învia obiectul, nu mergea pe cicluri, un singur finalizer per obiect):

```go
package main

import (
	"fmt"
	"runtime"
	"time"
)

type Resursa struct{ id int }

func main() {
	r := &Resursa{id: 1}
	runtime.AddCleanup(r, func(id int) {
		fmt.Println("curăț resursa", id)
	}, r.id) // argumentul NU trebuie să fie r: l-ar ține în viață

	r = nil
	runtime.GC()
	time.Sleep(10 * time.Millisecond) // curățarea rulează pe o goroutine separată
}
```

Curățările nu au garanție de execuție (de exemplu, la ieșirea din program). Nu le folosi pentru resurse care **trebuie** eliberate; pentru asta există `Close()` și `defer`.

## `runtime.KeepAlive`

Un obiect poate fi colectat **înainte** de ultima linie în care apare variabila, dacă compilatorul vede că nu mai e folosit. Când un obiect deține o resursă externă (un descriptor de fișier, memorie C) și ai o curățare atașată, `runtime.KeepAlive(x)` marchează punctul până la care `x` trebuie să rămână în viață.

## `uintptr` nu e pointer

`uintptr` e un întreg. GC-ul nu-l urmărește: dacă păstrezi adresa unui obiect într-un `uintptr`, obiectul poate fi colectat sau mutat (stivele se mută când cresc). Conversiile între `unsafe.Pointer` și `uintptr` sunt sigure doar în tiparele precise descrise în documentația pachetului `unsafe`; detalii în subiectul [Reflection și unsafe](../../reflect-unsafe/senior/).

## Write barriers

GC-ul Go rulează în paralel cu programul. Ca să nu-i scape pointerii mutați în timp ce marchează, compilatorul inserează o **write barrier** la scrierea unui pointer în memorie de pe heap, cât timp GC-ul e activ. E unul dintre motivele pentru care structurile fără pointeri (de exemplu `[]int` față de `[]*int`) sunt mai ieftine pentru GC: nu trebuie scanate și nu declanșează bariere.

## Pe scurt

- Escape analysis decide stiva sau heap-ul; verifică cu `-gcflags=-m`.
- Un pointer interior ține în viață tot obiectul; copiază ce ai nevoie.
- `weak.Pointer` pentru cache-uri, `runtime.AddCleanup` în loc de `SetFinalizer`.
- `uintptr` nu ține obiectele în viață.
- Mai puțini pointeri înseamnă mai puțin lucru pentru GC.
