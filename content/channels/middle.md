# Channels — Middle

Channels devin un instrument de arhitectură prin câteva tipare clasice: worker pool, pipeline, fan-out/fan-in și semnalizarea opririi. Tot aici: regulile de închidere, care previn cele mai multe panic-uri cu channels.

## Regulile închiderii

| Operație | Channel `nil` | Channel deschis | Channel închis |
|---|---|---|---|
| trimitere `ch <- v` | blochează pentru totdeauna | trimite sau blochează | **panic** |
| primire `<-ch` | blochează pentru totdeauna | primește sau blochează | valoarea zero, imediat |
| `close(ch)` | **panic** | închide | **panic** |

De aici regulile practice:

1. **Doar trimițătorul închide** channel-ul, niciodată receptorul.
2. Dacă sunt **mai mulți** trimițători, închide un coordonator, după ce toți au terminat (de obicei după un `WaitGroup`).
3. Închiderea nu e obligatorie: e un **semnal** pentru receptori („nu mai vine nimic”), nu eliberează memorie. Un channel neînchis și nereferit e colectat de GC.

```go,panic
package main

func main() {
	ch := make(chan int, 1)
	close(ch)
	ch <- 1 // panic: send on closed channel
}
```

## Worker pool

Un număr fix de goroutines consumă sarcini dintr-un channel și scriu rezultate în altul. Concurența e limitată natural la numărul de lucrători:

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

type Sarcina struct{ ID, N int }
type Rezultat struct{ ID, Patrat int }

func lucrator(sarcini <-chan Sarcina, rezultate chan<- Rezultat) {
	for s := range sarcini { // se oprește când sarcini e închis
		time.Sleep(5 * time.Millisecond) // „lucru”
		rezultate <- Rezultat{s.ID, s.N * s.N}
	}
}

func main() {
	sarcini := make(chan Sarcina)
	rezultate := make(chan Rezultat)

	var wg sync.WaitGroup
	for range 3 { // 3 lucrători
		wg.Go(func() { lucrator(sarcini, rezultate) })
	}

	go func() { // producătorul
		for i := 1; i <= 9; i++ {
			sarcini <- Sarcina{ID: i, N: i}
		}
		close(sarcini)
	}()

	go func() { // coordonatorul închide rezultatele după ce toți lucrătorii termină
		wg.Wait()
		close(rezultate)
	}()

	suma := 0
	for r := range rezultate {
		suma += r.Patrat
	}
	fmt.Println("suma:", suma) // 285
}
```

Rezultatele sosesc în **orice ordine**. Dacă ordinea contează, păstrează `ID`-ul și sortează la final, sau scrie în slice pe poziția `ID`.

## Pipeline

Fiecare etapă primește un channel și întoarce altul. Etapele rulează concurent, iar valorile curg prin ele:

```go
package main

import "fmt"

func genereaza(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

func patrat(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

func filtreazaPare(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			if n%2 == 0 {
				out <- n
			}
		}
	}()
	return out
}

func main() {
	for v := range filtreazaPare(patrat(genereaza(1, 2, 3, 4, 5, 6))) {
		fmt.Print(v, " ") // 4 16 36
	}
	fmt.Println()
}
```

Fiecare etapă închide channel-ul ei de ieșire cu `defer close(out)`: închiderea se propagă prin tot pipeline-ul.

## Fan-out, fan-in

**Fan-out**: mai multe goroutines citesc din același channel (ca în worker pool). **Fan-in**: valorile din mai multe channels sunt combinate într-unul:

```go
package main

import (
	"fmt"
	"sync"
)

func combina[T any](surse ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	for _, s := range surse {
		wg.Go(func() {
			for v := range s {
				out <- v
			}
		})
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func numere(de, pana int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := de; i <= pana; i++ {
			out <- i
		}
	}()
	return out
}

func main() {
	total := 0
	for v := range combina(numere(1, 10), numere(11, 20), numere(21, 30)) {
		total += v
	}
	fmt.Println(total) // 465
}
```

## Semnal de oprire: channel `done`

Închiderea unui channel e văzută de **toți** receptorii simultan, așa că e mecanismul perfect pentru „opriți-vă toți”. Un `chan struct{}` nu transportă date, doar semnalul:

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func lucreaza(id int, done <-chan struct{}) {
	for {
		select {
		case <-done:
			fmt.Println("lucrătorul", id, "se oprește")
			return
		default:
			time.Sleep(5 * time.Millisecond) // o bucată de lucru
		}
	}
}

func main() {
	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := range 3 {
		wg.Go(func() { lucreaza(i, done) })
	}
	time.Sleep(20 * time.Millisecond)
	close(done) // toți primesc semnalul
	wg.Wait()
}
```

În codul real, `context.Context` (care folosește intern exact acest mecanism, prin `ctx.Done()`) e varianta standard. Vezi [Context](../../context/middle/) și [Select](../../select/junior/).

## Mărimea buffer-ului

- **0 (fără buffer)**: sincronizare garantată; trimițătorul știe că valoarea a fost primită.
- **1**: util pentru semnale care nu trebuie să blocheze trimițătorul (de exemplu un rezultat unic).
- **N cunoscut**: când știi exact câte valori vor fi (un rezultat per goroutine), ca nicio goroutine să nu rămână blocată.
- **Mare, „ca să nu blocheze”**: aproape mereu o greșeală; ascunde o problemă de viteză între producător și consumator până când buffer-ul se umple oricum.

## Pe scurt

- Trimiterea pe un channel închis și închiderea dublă dau panic.
- Doar trimițătorul închide; cu mai mulți trimițători, un coordonator.
- Worker pool, pipeline și fan-in/fan-out sunt tiparele de bază.
- `close(done)` oprește toți receptorii odată.
- Alege mărimea buffer-ului pentru semantică, nu pentru a ascunde blocaje.
