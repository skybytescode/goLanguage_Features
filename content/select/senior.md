# Select — Senior

Cum implementează runtime-ul `select` (funcția `selectgo`), de unde vine alegerea aleatoare, cum evită deadlock-urile între lock-uri, cum optimizează compilatorul cazurile simple și cum construiești un `select` cu număr variabil de cazuri.

## Ce face compilatorul înainte de runtime

Nu orice `select` ajunge la `selectgo`:

| Formă | Ce generează compilatorul |
|---|---|
| `select {}` | `block()`: parchează goroutine-ul pentru totdeauna |
| un singur `case`, fără `default` | o operație obișnuită pe channel (`chansend`/`chanrecv`) |
| un `case` + `default` | o operație **non-blocantă** (`selectnbsend`/`selectnbrecv`) |
| restul | `runtime.selectgo` |

De aceea tiparul `select { case v := <-ch: ...; default: ... }` e foarte ieftin: un singur apel, adesea fără lock (drumul rapid care verifică atomic dacă channel-ul e gol).

## `selectgo`, pas cu pas

1. **Ordinea de verificare (`pollorder`)**: o permutare **aleatoare** a cazurilor (Fisher–Yates cu generatorul rapid al runtime-ului). De aici vine echitatea: între cazuri gata simultan, fiecare are șanse egale.
2. **Ordinea de blocare (`lockorder`)**: cazurile sortate după **adresa** channel-ului (heap sort). Toate channels implicate sunt blocate în această ordine globală, ceea ce exclude deadlock-urile între două `select`-uri care blochează aceleași channels în ordini diferite. Un channel care apare de mai multe ori e blocat o singură dată.
3. **Trecerea 1**: în ordinea `pollorder`, caută un caz gata (receptor/trimițător care așteaptă, buffer cu loc/date, channel închis). Primul găsit e executat; se deblochează totul și gata.
4. Dacă nu e niciunul și există `default`: deblochează și execută `default`.
5. **Trecerea 2**: creează câte un `sudog` pentru **fiecare** caz și îl adaugă în coada channel-ului respectiv; apoi parchează goroutine-ul (`gopark`), eliberând lock-urile.
6. Când o altă goroutine operează pe unul dintre channels, găsește `sudog`-ul, finalizează transferul și trezește goroutine-ul. Un câmp atomic (`selectDone`) asigură că doar **un** caz poate „câștiga”, chiar dacă două channels devin gata simultan.
7. **Trecerea 3**: după trezire, goroutine-ul blochează din nou toate channels și își scoate `sudog`-urile din toate celelalte cozi.

Costul e deci **liniar în numărul de cazuri**, de trei ori: un `select` cu 2–4 cazuri e ieftin, unul cu 50 de cazuri face multă muncă la fiecare execuție.

## Echitatea are o limită

Alegerea e uniformă **doar între cazurile gata în același moment**. Un caz care e mereu gata (de exemplu un channel cu buffer mereu plin) va fi ales de aproximativ `1/n` ori; dacă o sursă produce mult mai repede decât alta, ea va domina. Echitatea între producători se rezolvă în arhitectură (cozi separate, limitare), nu în `select`.

## `select` cu număr variabil de cazuri: `reflect.Select`

Numărul de cazuri al unui `select` e fix la compilare. Pentru N channels cunoscute doar la rulare există `reflect.Select`:

```go
package main

import (
	"fmt"
	"reflect"
)

func main() {
	surse := make([]chan int, 3)
	cazuri := make([]reflect.SelectCase, len(surse))
	for i := range surse {
		surse[i] = make(chan int, 1)
		cazuri[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(surse[i])}
	}
	surse[1] <- 42

	ales, valoare, ok := reflect.Select(cazuri)
	fmt.Println(ales, valoare.Int(), ok) // 1 42 true
}
```

E de câteva ori mai lent decât un `select` static (reflection și alocări). Pentru multe surse, un fan-in într-un singur channel (vezi [Channels](../../channels/middle/)) e de obicei mai bun.

## Timerele în Go 1.23

Implementarea timerelor a fost rescrisă în Go 1.23, cu efecte directe asupra `select`:

- channel-ul unui timer (`timer.C`, `time.After`) e acum **sincron** (fără buffer, din perspectiva programului): o valoare nu „stă” în channel după `Stop` sau `Reset`;
- timerele sunt legate de channel, iar un timer neoprit, la care nu mai există referințe, e **colectat de GC**; `time.After` într-o buclă nu mai produce scurgeri de memorie.

Comportamentul nou e activ pentru modulele cu `go 1.23` sau mai nou în `go.mod`; `GODEBUG=asynctimerchan=1` îl revine la cel vechi.

## Interacțiunea cu scheduler-ul

O goroutine parcată într-un `select` nu consumă CPU și nu ocupă un P. Dacă toate goroutines sunt parcate pe channels și niciun timer sau eveniment de rețea nu mai poate apărea, runtime-ul detectează **deadlock-ul global** și oprește programul. Detecția nu vede blocajele parțiale (o parte din goroutines blocate, altele încă active): acelea apar ca scurgeri și se găsesc cu profilul `goroutine` din `pprof`.

## Pe scurt

- Compilatorul transformă `select`-urile cu un caz (+`default`) în operații simple.
- `selectgo`: permutare aleatoare pentru verificare, ordine după adresă pentru blocare.
- Costul e liniar în numărul de cazuri; un singur caz câștigă, atomic.
- Echitatea e doar între cazurile gata simultan.
- `reflect.Select` pentru N dinamic; din Go 1.23, timerele sunt sincrone și colectabile.
