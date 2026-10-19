# Channels — Senior

Structura `hchan` din runtime, ce se întâmplă exact la o trimitere și la o primire, garanțiile din modelul de memorie Go și criteriile pentru a alege între channels și mutex-uri.

## Structura internă: `hchan`

`make(chan T, n)` alocă pe heap o structură `runtime.hchan`; variabila de tip channel e doar un pointer la ea (de aceea channels se pot copia și compara).

```go,ignore
type hchan struct {
	qcount   uint           // câte elemente sunt acum în buffer
	dataqsiz uint           // capacitatea buffer-ului (n)
	buf      unsafe.Pointer // buffer circular de dataqsiz elemente
	elemsize uint16
	closed   uint32
	elemtype *_type
	sendx    uint  // unde scrie următoarea trimitere
	recvx    uint  // de unde citește următoarea primire
	recvq    waitq // goroutines blocate la primire (listă de sudog)
	sendq    waitq // goroutines blocate la trimitere
	lock     mutex // protejează toate câmpurile de mai sus
}
```

Un `sudog` reprezintă o goroutine care așteaptă pe un channel: pointer la G, pointer la valoarea de trimis sau la locul unde trebuie scrisă valoarea primită, plus legături în listă. O goroutine blocată într-un `select` pe 5 channels are 5 `sudog`-uri, câte unul în fiecare coadă.

## O trimitere, pas cu pas

`ch <- v` apelează `runtime.chansend`:

1. Channel `nil` → goroutine-ul se parchează pentru totdeauna.
2. Ia `lock`. Dacă e închis → deblochează și **panic**.
3. Dacă există un receptor în `recvq`: îl scoate și **copiază valoarea direct pe stiva receptorului** (fără să treacă prin buffer), apoi îl marchează ca gata de rulare (`goready`). Este singurul loc din runtime unde o goroutine scrie pe stiva alteia.
4. Altfel, dacă e loc în buffer: copiază valoarea în `buf[sendx]`, avansează `sendx`, crește `qcount`.
5. Altfel: creează un `sudog`, îl pune în `sendq` și **parchează** goroutine-ul (`gopark`), eliberând lock-ul. Va fi trezit de un receptor, care va lua valoarea direct din `sudog`.

Primirea (`chanrecv`) e simetrică. Un detaliu: când buffer-ul e plin și există trimițători blocați, receptorul ia elementul din capul buffer-ului și mută valoarea primului trimițător blocat în coada buffer-ului, ca ordinea FIFO să se păstreze.

Există și un „drum rapid” fără lock: o primire non-blocantă dintr-un channel gol și deschis (de exemplu `select` cu `default`) poate decide citind atomic doar `qcount` și `closed`.

## `close`

`closechan` ia lock-ul, setează `closed`, apoi trezește **toate** goroutines din `recvq` (primesc valoarea zero și `ok == false`) și din `sendq` (acestea vor da panic). De aceea închiderea e un broadcast ieftin, iar trimițătorii blocați pe un channel închis de altcineva sunt un bug grav.

## Ce costă

O operație fără așteptare costă un lock, o copiere și un unlock: zeci de nanosecunde. Una care blochează adaugă parcarea și trezirea unei goroutine prin scheduler: sute de nanosecunde, plus o posibilă comutare de context pe alt P. Sub contenție mare (mulți trimițători pe același channel), lock-ul devine punctul fierbinte.

Pentru elemente mari, valoarea e copiată de două ori când trece prin buffer (trimițător → buffer → receptor). Trimite pointeri sau structuri mici.

## Modelul de memorie: ce garantează un channel

Modelul de memorie Go definește relația **happens-before**. Pentru channels:

1. O trimitere **se întâmplă înaintea** terminării primirii corespunzătoare.
2. Închiderea unui channel se întâmplă înaintea unei primiri care întoarce valoarea zero din cauza închiderii.
3. Pentru un channel **fără buffer**, primirea se întâmplă înaintea terminării trimiterii.
4. Pentru un channel cu capacitatea `C`, a `k`-a primire se întâmplă înaintea terminării trimiterii `k+C`.

Consecința regulii 1: tot ce a scris trimițătorul **înainte** de trimitere e vizibil receptorului **după** primire, fără alte sincronizări:

```go
package main

import "fmt"

var config map[string]string

func main() {
	gata := make(chan struct{})
	go func() {
		config = map[string]string{"mediu": "prod"} // scris înainte de close
		close(gata)
	}()
	<-gata
	fmt.Println(config["mediu"]) // garantat „prod”: fără cursă de date
}
```

Regula 4 e baza semaforului cu channel cu buffer: cel mult `C` goroutines pot fi simultan între „trimitere” și „primire”.

## `len` și `cap` sunt doar indicii

`len(ch)` citește `qcount` fără lock. Valoarea poate fi depășită de realitate imediat după citire. Nu lua decizii de sincronizare pe baza ei (`if len(ch) > 0 { <-ch }` poate bloca); folosește un `select` cu `default`.

## Channels sau mutex?

| Channels, când… | Mutex, când… |
|---|---|
| transferi **proprietatea** asupra datelor între goroutines | protejezi **starea** internă a unei structuri |
| coordonezi etape (pipeline, worker pool) | operația e scurtă: citește/modifică un câmp |
| semnalizezi evenimente (gata, oprire, timeout) | ai nevoie de performanță sub contenție mare |
| combini cu `select` pentru timeout și anulare | logica e mai clară ca o secțiune critică |

Un cache sau un contor protejat cu un channel și o goroutine „proprietar” e de obicei mai lent și mai complicat decât același lucru cu un `sync.Mutex`. Proverbul „partajează memoria comunicând” e despre arhitectură, nu o interdicție pentru mutex-uri.

## Pe scurt

- `hchan`: buffer circular, două cozi de `sudog`, un lock.
- Trimiterea către un receptor care așteaptă scrie direct pe stiva lui.
- `close` trezește toți așteptătorii; trimițătorii vor da panic.
- O trimitere se întâmplă înaintea primirii corespunzătoare: datele scrise înainte devin vizibile.
- Channels pentru transfer și coordonare, mutex pentru stare partajată.
