# Testare, benchmarks și fuzzing — Junior

Go are testarea inclusă în limbaj și în unelte: pachetul `testing` și comanda `go test`. Nu ai nevoie de nicio bibliotecă externă ca să începi.

## Primul test

Testele stau în fișiere care se termină în `_test.go`, lângă codul testat, în același pachet:

```go,ignore
// calcul.go
package calcul

func Aduna(a, b int) int {
	return a + b
}
```

```go,ignore
// calcul_test.go
package calcul

import "testing"

func TestAduna(t *testing.T) {
	got := Aduna(2, 3)
	if got != 5 {
		t.Errorf("Aduna(2, 3) = %d, vreau 5", got)
	}
}
```

Regulile:

- funcția începe cu `Test`, urmat de o literă mare;
- primește un singur parametru, `t *testing.T`;
- un test **trece** dacă nu raportează nicio eroare.

```bash
$ go test
PASS
ok      exemplu/calcul  0.002s

$ go test -v ./...        # toate pachetele, cu detalii
=== RUN   TestAduna
--- PASS: TestAduna (0.00s)
```

## `Errorf` vs. `Fatalf`

- `t.Errorf` raportează eroarea și **continuă** testul (vezi toate problemele odată).
- `t.Fatalf` raportează și **oprește** testul imediat (când nu are sens să continui, de exemplu după o eroare la deschiderea unui fișier).

```go,ignore
func TestCitire(t *testing.T) {
	date, err := os.ReadFile("testdata/intrare.txt")
	if err != nil {
		t.Fatalf("nu pot citi fișierul: %v", err) // fără date, restul n-are sens
	}
	if len(date) == 0 {
		t.Error("fișierul e gol")
	}
}
```

Fișierele folosite de teste stau în directorul `testdata/`, ignorat de compilator.

## Teste pe bază de tabel

Tiparul cel mai folosit în Go: o listă de cazuri și o buclă. Adaugi un caz nou scriind o linie:

```go,ignore
func TestEstePar(t *testing.T) {
	cazuri := []struct {
		nume  string
		n     int
		vreau bool
	}{
		{"zero", 0, true},
		{"pozitiv par", 4, true},
		{"pozitiv impar", 7, false},
		{"negativ par", -2, true},
	}
	for _, c := range cazuri {
		t.Run(c.nume, func(t *testing.T) { // un sub-test per caz
			if got := EstePar(c.n); got != c.vreau {
				t.Errorf("EstePar(%d) = %v, vreau %v", c.n, got, c.vreau)
			}
		})
	}
}
```

`t.Run` creează **sub-teste** cu nume. Poți rula doar unul:

```bash
go test -run 'TestEstePar/negativ' -v
```

Aceeași idee, într-un program pe care îl poți rula direct:

```go
package main

import "fmt"

func EstePar(n int) bool { return n%2 == 0 }

func main() {
	cazuri := []struct {
		n     int
		vreau bool
	}{{0, true}, {4, true}, {7, false}, {-2, true}, {-3, false}}

	esecuri := 0
	for _, c := range cazuri {
		if got := EstePar(c.n); got != c.vreau {
			fmt.Printf("EȘEC: EstePar(%d) = %v, vreau %v\n", c.n, got, c.vreau)
			esecuri++
		}
	}
	fmt.Printf("%d cazuri, %d eșecuri\n", len(cazuri), esecuri)
}
```

## Mesaje de eroare utile

Un mesaj bun spune **ce** ai apelat, **ce** ai primit și **ce** așteptai: `Aduna(2, 3) = 6, vreau 5`. Convenția Go: întâi valoarea obținută (`got`), apoi cea dorită (`want`).

## Acoperirea codului

```bash
go test -cover                        # procentul de linii acoperite
go test -coverprofile=c.out
go tool cover -html=c.out             # raport în browser, linie cu linie
```

## Comenzi utile

| Comandă | Ce face |
|---|---|
| `go test ./...` | toate pachetele |
| `go test -v` | afișează fiecare test |
| `go test -run TestNume` | doar testele care se potrivesc cu expresia |
| `go test -count=1` | ignoră cache-ul de teste |
| `go test -race` | detectează cursele de date |
| `go test -short` | sare peste testele lungi (care verifică `testing.Short()`) |

## Pe scurt

- Teste în `*_test.go`, funcții `TestXxx(t *testing.T)`.
- `t.Errorf` continuă, `t.Fatalf` oprește.
- Testele pe bază de tabel + `t.Run` sunt tiparul standard.
- Mesaje de forma `F(x) = got, vreau want`.
- `go test -cover` și `-race` în mod regulat.
