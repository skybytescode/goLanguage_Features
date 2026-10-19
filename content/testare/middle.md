# Testare, benchmarks și fuzzing — Middle

Uneltele din `testing` care fac testele curate și rapide (`t.Helper`, `t.Cleanup`, `t.TempDir`, `t.Parallel`, `t.Context`), testele de tip „Example”, înlocuirea dependențelor prin interfețe, benchmark-urile cu `b.Loop` și testarea codului cu timp și goroutines prin `testing/synctest`.

## Funcții ajutătoare: `t.Helper`

O funcție de verificare comună raportează, implicit, linia **din ea**. `t.Helper()` face ca eroarea să indice linia din testul care a apelat-o:

```go,ignore
func verificaEgal[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
```

## Resurse temporare: `TempDir`, `Cleanup`, `Setenv`, `Context`

```go,ignore
func TestExport(t *testing.T) {
	dir := t.TempDir() // creat acum, șters automat la final
	t.Setenv("APP_MOD", "test") // restaurat automat la final

	db := deschideDB(t)
	t.Cleanup(func() { db.Close() }) // rulează după test, în ordine inversă

	ctx := t.Context() // Go 1.24: anulat chiar înainte de Cleanup
	if err := Export(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
}
```

`t.Cleanup` e mai bun decât `defer` în funcțiile ajutătoare: curățarea rulează la sfârșitul **testului**, nu al funcției ajutătoare.

## Teste paralele

`t.Parallel()` permite rularea testului simultan cu alte teste paralele din același pachet. Pentru sub-teste, fiecare caz rulează concurent:

```go,ignore
for _, c := range cazuri {
	t.Run(c.nume, func(t *testing.T) {
		t.Parallel()      // din Go 1.22, c e o variabilă nouă per iterație: sigur
		verifica(t, c)
	})
}
```

Testele paralele nu au voie să depindă de stare globală comună (variabile globale, directorul curent, variabile de mediu: `t.Setenv` și `t.Chdir` refuză să ruleze în teste paralele).

## Exemple care sunt și teste

O funcție `ExampleXxx` din fișierul de test apare în documentație (`go doc`, pkg.go.dev) și e **verificată**: ieșirea trebuie să coincidă cu comentariul `// Output:`:

```go,ignore
func ExampleAduna() {
	fmt.Println(Aduna(2, 3))
	// Output: 5
}
```

## Înlocuirea dependențelor prin interfețe

Codul care depinde de o interfață mică poate fi testat cu o implementare falsă, fără biblioteci de mocking:

```go
package main

import (
	"errors"
	"fmt"
	"time"
)

type Ceas interface{ Acum() time.Time }

type Expeditor interface {
	Trimite(catre, mesaj string) error
}

// Reamintiri e codul testat: depinde doar de interfețe.
type Reamintiri struct {
	ceas Ceas
	exp  Expeditor
}

func (r Reamintiri) Trimite(email string, termen time.Time) error {
	if r.ceas.Acum().After(termen) {
		return errors.New("termen depășit")
	}
	return r.exp.Trimite(email, "termenul e "+termen.Format("2006-01-02"))
}

// --- implementări false, cum ar fi în fișierul de test ---

type ceasFix struct{ t time.Time }

func (c ceasFix) Acum() time.Time { return c.t }

type expeditorFals struct{ trimise []string }

func (e *expeditorFals) Trimite(catre, mesaj string) error {
	e.trimise = append(e.trimise, catre+": "+mesaj)
	return nil
}

func main() {
	azi := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	exp := &expeditorFals{}
	r := Reamintiri{ceas: ceasFix{azi}, exp: exp}

	fmt.Println(r.Trimite("ana@x.ro", azi.AddDate(0, 0, 3)))  // <nil>
	fmt.Println(r.Trimite("ion@x.ro", azi.AddDate(0, 0, -1))) // termen depășit
	fmt.Println(exp.trimise)
}
```

## Benchmarks cu `b.Loop` (Go 1.24)

```go,ignore
func BenchmarkConcatenare(b *testing.B) {
	parti := []string{"a", "b", "c", "d"}
	for b.Loop() { // înlocuiește for i := 0; i < b.N; i++
		_ = strings.Join(parti, ",")
	}
}
```

```bash
$ go test -bench=Concatenare -benchmem
BenchmarkConcatenare-8   	20000000	  55.1 ns/op	  8 B/op	  1 allocs/op
```

`b.Loop` rulează setup-ul de dinaintea buclei o singură dată, oprește automat cronometrul în afara buclei și împiedică compilatorul să elimine codul măsurat ca „nefolosit”. Același mecanism, dintr-un program obișnuit:

```go
package main

import (
	"fmt"
	"strings"
	"testing"
)

func main() {
	parti := []string{"a", "b", "c", "d"}
	join := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = strings.Join(parti, ",")
		}
	})
	plus := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			s := ""
			for _, p := range parti {
				s += p + ","
			}
			_ = s
		}
	})
	fmt.Println("Join:", join.AllocsPerOp(), "alocări/op")
	fmt.Println("+=  :", plus.AllocsPerOp(), "alocări/op")
}
```

## Cod cu timp și goroutines: `testing/synctest` (Go 1.25)

Testarea unui timeout de 5 secunde nu trebuie să dureze 5 secunde. `synctest.Test` rulează testul într-o „bulă” cu un **ceas fals**: timpul avansează instantaneu când toate goroutines din bulă sunt blocate.

```go,ignore
func TestExpirare(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		time.Sleep(4 * time.Second) // instantaneu
		if ctx.Err() != nil {
			t.Fatal("expirat prea devreme")
		}
		time.Sleep(2 * time.Second)
		synctest.Wait() // așteaptă ca toate goroutines din bulă să se blocheze
		if ctx.Err() == nil {
			t.Fatal("trebuia să expire")
		}
	})
}
```

## Teste de integrare separate

Testele lente sau care au nevoie de servicii externe se marchează, ca să nu ruleze implicit:

```go,ignore
func TestBazaDeDateReala(t *testing.T) {
	if testing.Short() {
		t.Skip("lent; rulează fără -short")
	}
	// ...
}
```

Alternativ, un tag de build (`//go:build integrare`) și `go test -tags=integrare`.

## Pe scurt

- `t.Helper`, `t.Cleanup`, `t.TempDir`, `t.Setenv`, `t.Context` țin testele curate.
- `t.Parallel` accelerează testele independente.
- `ExampleXxx` cu `// Output:` = documentație verificată.
- Interfețe mici + implementări false în loc de biblioteci de mocking.
- `b.Loop` pentru benchmarks; `synctest` pentru cod cu timp și goroutines.
