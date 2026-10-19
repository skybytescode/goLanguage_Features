# Range și iteratori — Senior

Cum transformă compilatorul o buclă `range` peste o funcție, cum sunt tratate `break`, `return` și `defer` din corpul buclei, ce se întâmplă când un iterator încalcă regulile și cum e implementat `iter.Pull` cu corutine în runtime.

## Rescrierea buclei

Compilatorul transformă corpul buclei într-un closure sintetizat, transmis ca `yield`:

```go,ignore
// ce scrii
for x := range seq {
	if x > 10 {
		break
	}
	fmt.Println(x)
}

// aproximativ ce generează compilatorul
seq(func(x int) bool {
	if x > 10 {
		return false // break
	}
	fmt.Println(x)
	return true // continuă
})
```

`continue` devine `return true`, `break` devine `return false`. Cazurile complicate sunt instrucțiunile care **ies din funcția exterioară**: `return`, `break`/`continue` cu etichetă spre o buclă exterioară, `goto`. Pentru ele, compilatorul folosește o variabilă de stare: closure-ul o setează și întoarce `false`, iar după ce iteratorul se întoarce, codul generat verifică starea și execută `return`-ul real (cu valorile salvate).

## `defer` în corpul buclei

Un `defer` scris în corpul unei bucle `range` peste funcție aparține **funcției exterioare**, ca în orice altă buclă — deși tehnic se execută într-un closure. Runtime-ul are suport special pentru asta (`deferrangefunc`), ca `defer`-ul să fie atașat cadrului corect.

## Verificări la rulare

Pentru că iteratorul e cod scris de oricine, compilatorul adaugă verificări în jurul lui `yield`. Încălcările dau panic cu mesaje explicite:

```go,panic
package main

import "fmt"

// Gresit ignoră rezultatul lui yield.
func Gresit(yield func(int) bool) {
	yield(1)
	yield(2) // apelat după ce bucla a făcut break
}

func main() {
	for x := range Gresit {
		fmt.Println(x)
		break
	}
	// panic: runtime error: range function continued iteration after function for loop body returned false
}
```

Alte situații detectate: apelarea lui `yield` după ce iteratorul s-a întors, sau un iterator care „înghite” un panic din corpul buclei și continuă.

## Performanța

Un iterator push e un apel de funcție pe element, prin closure — teoretic mai lent decât o buclă directă. În practică, când iteratorul și corpul sunt suficient de mici, compilatorul **inlinează** ambele, iar codul final e echivalent cu bucla scrisă de mână. Iteratorii recursivi (ca arborele de la nivelul Middle) sau cei care trec prin interfețe nu pot fi inlined complet, dar rămân mult mai rapizi decât alternativa cu channels și goroutines:

```text
// pe un laptop obișnuit, parcurgerea a 1.000.000 de elemente
BenchmarkSliceLoop     ~ 0.3 ms
BenchmarkIterSeq       ~ 0.3 ms   // inlined: identic
BenchmarkChannel       ~ 60  ms   // o sincronizare per element
```

Cifrele variază cu mașina; raportul e ce contează. Verifică pentru codul tău cu `go test -bench`.

## `iter.Pull` și corutinele

Transformarea unui iterator push în pull cere ca iteratorul să fie **suspendat** între două apeluri `next()`. Fără suport din runtime, singura cale ar fi o goroutine și un channel, cu costul unei programări prin scheduler la fiecare valoare.

`iter.Pull` folosește în schimb **corutine** (`runtime.coro`): iteratorul rulează pe o goroutine proprie, dar transferul de control între ea și apelant se face cu `coroswitch`, o comutare directă care **ocolește scheduler-ul**. Doar una dintre cele două rulează la un moment dat, deci nu există paralelism și nici nevoie de sincronizare. Costul unei comutări e de ordinul zecilor de nanosecunde, de câteva ori mai puțin decât o trimitere pe channel.

Consecințe practice:

- `next` și `stop` nu au voie să fie apelate concurent din mai multe goroutines;
- un `stop()` uitat lasă goroutine-ul iteratorului blocat pentru totdeauna (o scurgere);
- dacă iteratorul dă panic, panic-ul e propagat în goroutine-ul care a apelat `next`.

## Iteratori de unică folosință

`iter.Seq` nu spune dacă iteratorul poate fi parcurs de mai multe ori. Cei construiți peste colecții (`slices.Values`, `maps.Keys`) pot fi reluați; cei construiți peste un flux (linii citite dintr-un `io.Reader`, rezultatele unei interogări SQL) de obicei nu. Convenția din biblioteca standard e să documentezi explicit iteratorii de unică folosință.

## Erori în iteratori

Un `iter.Seq[V]` nu are unde să întoarcă o eroare. Două tipare uzuale:

```go,ignore
// 1. Seq2 cu eroare pe fiecare element
func Randuri(db *sql.DB) iter.Seq2[Rand, error]

// 2. iterator + metodă Err(), ca bufio.Scanner
it := NouCititor(r)
for v := range it.All() { ... }
if err := it.Err(); err != nil { ... }
```

## Pe scurt

- Corpul buclei devine un closure `yield`; `break` = `return false`.
- `return` și etichetele spre exterior sunt tratate printr-o variabilă de stare.
- Încălcările protocolului dau panic cu mesaje explicite.
- Iteratorii mici sunt inlined și ajung la viteza buclelor obișnuite.
- `iter.Pull` folosește corutine din runtime, fără scheduler; apelează mereu `stop`.
