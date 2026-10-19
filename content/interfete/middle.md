# Interfețe — Middle

Interfețele bune în Go sunt **mici** și definite **acolo unde sunt folosite**. Tot aici apare cea mai faimoasă capcană a limbajului: interfața care nu e `nil`, deși conține un pointer `nil`.

## Interfețe mici

Cele mai folosite interfețe din biblioteca standard au o singură metodă:

```go,ignore
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
type Stringer interface { String() string }
type error interface { Error() string }
```

Cu cât interfața e mai mică, cu atât mai multe tipuri o implementează și cu atât e mai ușor de înlocuit în teste. Interfețele mai mari se compun din cele mici:

```go,ignore
type ReadWriter interface {
	Reader
	Writer
}
```

## „Acceptă interfețe, întoarce structuri”

- Parametrii: cere **minimul** de comportament de care ai nevoie (o interfață).
- Rezultatele: întoarce tipul **concret**, ca apelantul să aibă acces la tot.

```go
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// numara primește orice sursă de octeți: fișier, rețea, string, buffer...
func numara(r io.Reader) (int, error) {
	data, err := io.ReadAll(r)
	return len(strings.Fields(string(data))), err
}

func main() {
	n, _ := numara(strings.NewReader("unu doi trei"))
	fmt.Println(n)

	fmt.Fprintln(os.Stdout, "os.Stdout e un io.Writer")
}
```

## Interfețele se definesc la consumator

În Java, interfața stă lângă implementare. În Go, implementarea implicită permite invers: pachetul care **folosește** comportamentul își declară interfața de care are nevoie. Pachetul care îl oferă întoarce doar un tip concret.

```go,ignore
// pachetul notificari: își declară doar ce folosește
type Expeditor interface {
	Trimite(catre, mesaj string) error
}

func Anunta(e Expeditor, utilizatori []string) error { ... }
```

Orice client SMTP, SMS sau mock din teste care are metoda `Trimite` funcționează, fără ca pachetele lor să știe de `notificari`.

## Verificare la compilare

Pentru că implementarea e implicită, e ușor să strici o interfață schimbând o semnătură. O linie ca aceasta oprește compilarea dacă tipul nu mai implementează interfața:

```go
package main

import (
	"fmt"
	"io"
)

type Contor struct{ n int }

func (c *Contor) Write(p []byte) (int, error) {
	c.n += len(p)
	return len(p), nil
}

var _ io.Writer = (*Contor)(nil) // verificare la compilare, fără cost la rulare

func main() {
	c := &Contor{}
	fmt.Fprintf(c, "salut %s", "lume")
	fmt.Println(c.n) // 10
}
```

## Receptor pointer: doar `*T` implementează

Dacă metodele sunt definite pe `*T`, valoarea `T` **nu** implementează interfața:

```go,compile_fail
package main

import "io"

type Contor struct{ n int }

func (c *Contor) Write(p []byte) (int, error) { return len(p), nil }

func main() {
	var w io.Writer = Contor{} // eroare: Write has pointer receiver
	_ = w
}
```

Motivul: o valoare păstrată într-o interfață nu e adresabilă, deci Go nu poate lua adresa ei pentru a apela metoda.

## Capcana: interfața non-nil care conține nil

O interfață are două componente: **tipul** dinamic și **valoarea**. Este `nil` doar dacă **amândouă** lipsesc. Un pointer `nil` pus într-o interfață produce o interfață cu tip, deci **non-nil**:

```go
package main

import "fmt"

type EroareaMea struct{}

func (*EroareaMea) Error() string { return "eroare" }

func verifica(ok bool) error {
	var e *EroareaMea // nil
	if !ok {
		e = &EroareaMea{}
	}
	return e // GREȘIT: întoarce interfață cu tip *EroareaMea și valoare nil
}

func main() {
	err := verifica(true)
	fmt.Println(err == nil) // false!
	fmt.Printf("%T %v\n", err, err == (*EroareaMea)(nil))
}
```

Regula: într-o funcție care întoarce `error`, scrie explicit `return nil` pe drumul fără eroare; nu întoarce o variabilă de tip pointer concret.

## Upgrade de interfață

O funcție poate verifica la rulare dacă valoarea primită oferă **mai mult** decât interfața minimă și poate folosi o cale mai rapidă. Așa face `io.Copy` cu `io.WriterTo` și `io.ReaderFrom`:

```go
package main

import (
	"fmt"
	"strings"
)

type Lungimer interface{ Len() int }

func dimensiune(s fmt.Stringer) int {
	if l, ok := s.(Lungimer); ok {
		return l.Len() // drum rapid
	}
	return len(s.String())
}

type Nume string

func (n Nume) String() string { return string(n) }

func main() {
	var b strings.Builder
	b.WriteString("abc")
	fmt.Println(dimensiune(&b), dimensiune(Nume("Go")))
}
```

## Pe scurt

- Interfețe mici, compuse prin încorporare.
- Parametri interfață, rezultate concrete.
- Interfața se declară în pachetul care o folosește.
- `var _ I = (*T)(nil)` verifică implementarea la compilare.
- Nu întoarce un pointer `nil` concret ca `error`.
