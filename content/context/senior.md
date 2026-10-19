# Context — Senior

Cum e implementat pachetul `context`: tipurile concrete din spatele interfeței, cum se propagă anularea în arbore, ce costă `Value`, cum sunt integrate contextele străine și cum își implementezi propriul tip de context.

## Interfața

```go,ignore
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}
```

Fiecare funcție `With...` întoarce un tip concret care învelește părintele:

| Constructor | Tip intern | Ce adaugă |
|---|---|---|
| `Background()`, `TODO()` | `backgroundCtx`, `todoCtx` | nimic: `Done()` întoarce `nil` |
| `WithCancel`, `WithCancelCause` | `*cancelCtx` | channel `done`, lista copiilor, eroarea |
| `WithTimeout`, `WithDeadline` | `*timerCtx` | un `cancelCtx` + un `time.Timer` |
| `WithValue` | `*valueCtx` | o pereche cheie/valoare |
| `WithoutCancel` | `withoutCancelCtx` | ascunde anularea părintelui |
| `AfterFunc` | `*afterFuncCtx` | o funcție de rulat la anulare |

## `cancelCtx` și propagarea

```go,ignore
type cancelCtx struct {
	Context                        // părintele
	mu       sync.Mutex
	done     atomic.Value          // chan struct{}, creat leneș
	children map[canceler]struct{} // copiii care trebuie anulați odată cu acesta
	err      atomic.Value
	cause    error
}
```

La crearea unui `cancelCtx`, funcția `propagateCancel` îl leagă de părinte:

1. Dacă părintele nu poate fi anulat niciodată (`Done() == nil`, cum e `Background`), nu face nimic.
2. Dacă părintele e deja anulat, copilul e anulat imediat.
3. Dacă părintele e (sau învelește) un `*cancelCtx` din pachetul `context`, copilul e adăugat direct în `children` — fără goroutines.
4. Dacă părintele implementează `AfterFunc` (Go 1.21+), se folosește acel mecanism.
5. Altfel — un tip de context **străin**, scris de altcineva — pachetul pornește o **goroutine** care așteaptă pe `parent.Done()` și anulează copilul. Fiecare context derivat dintr-un context străin costă deci o goroutine până la anulare.

La `cancel()`: se setează eroarea, se închide `done`, se anulează recursiv fiecare copil, iar contextul se scoate din `children` al părintelui (altfel părintele ar ține referințe la toți copiii anulați — o scurgere de memorie; de aceea `cancel` trebuie apelat **mereu**).

## `Done()` e creat leneș

Channel-ul `done` e creat abia la primul apel `Done()`. Dacă e anulat înainte ca cineva să-l ceară, se folosește un channel global, deja închis (`closedchan`), fără alocare. Contextele pe care nimeni nu le ascultă nu alocă niciun channel.

## `timerCtx`

`WithTimeout(d)` e doar `WithDeadline(time.Now().Add(d))`. Dacă deadline-ul părintelui e deja mai devreme, nu se creează niciun timer: se întoarce un simplu `cancelCtx` (termenul efectiv e oricum al părintelui). Altfel, un `time.AfterFunc` apelează `cancel` cu `DeadlineExceeded` la expirare. `cancel()` oprește timerul, de aceea e important să-l apelezi și când operația se termină devreme: altfel timerul ar ține contextul în viață până la expirare.

## `Value` e o listă înlănțuită

`WithValue` nu creează un map: fiecare `valueCtx` ține **o singură** pereche și un pointer la părinte. `ctx.Value(k)` urcă pe lanț până găsește cheia sau ajunge la rădăcină:

```go
package main

import (
	"context"
	"fmt"
)

type cheie int

func main() {
	ctx := context.Background()
	for i := range 100 {
		ctx = context.WithValue(ctx, cheie(i), i)
	}
	// căutarea cheii 0 traversează 100 de niveluri: O(adâncime)
	fmt.Println(ctx.Value(cheie(0)))
}
```

În practică, lanțurile au câteva niveluri și costul e neglijabil. Dar explică de ce contextul nu e potrivit ca „sac” general de dependențe: căutările sunt liniare, netipizate și invizibile în semnături.

Comparația cheilor se face cu `==` pe interfețe, deci tipul cheii contează: `cheie(0)` și `int(0)` sunt chei diferite. Din același motiv, cheia nu trebuie să fie un tip necomparabil (`WithValue` dă panic).

## Implementarea unui tip de context propriu

Rar necesar, dar uneori util (de exemplu un context legat de ciclul de viață al unui obiect). Înglobează un context existent și suprascrie doar ce trebuie. Dacă tipul tău expune anularea prin ceva diferit de un `cancelCtx` intern, implementează și `AfterFunc`, ca derivatele să nu aibă nevoie de goroutine.

```go
package main

import (
	"context"
	"fmt"
)

// ctxCuEticheta adaugă o metodă, păstrând tot comportamentul părintelui.
type ctxCuEticheta struct {
	context.Context
	eticheta string
}

func (c ctxCuEticheta) String() string { return "ctx[" + c.eticheta + "]" }

func main() {
	baza, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := ctxCuEticheta{baza, "import-zilnic"}

	copil, cancelCopil := context.WithCancel(ctx) // părintele e învelit: tot fără goroutine
	defer cancelCopil()
	cancel()
	<-copil.Done()
	fmt.Println(ctx, copil.Err())
}
```

Pentru că `ctxCuEticheta` încorporează un `*cancelCtx`, `propagateCancel` îl recunoaște (prin `parent.Value(&cancelCtxKey)`) și leagă copilul direct, fără goroutine.

## Contextul în `net/http`

Serverul HTTP creează un context per conexiune și unul per cerere, anulat când clientul închide conexiunea, când handler-ul `ServeHTTP` se termină sau la `Server.Shutdown`. Pentru HTTP/2, anularea unui stream anulează doar contextul cererii respective. Lucrul pornit dintr-un handler și care trebuie să continue după răspuns are nevoie de `context.WithoutCancel(r.Context())`.

## Pe scurt

- Fiecare `With...` adaugă un nivel: `cancelCtx`, `timerCtx`, `valueCtx`.
- Anularea se propagă prin harta `children`, fără goroutines pentru tipurile standard.
- Contextele străine costă o goroutine per derivat, dacă nu implementează `AfterFunc`.
- `Done()` e creat leneș; `cancel()` eliberează timerul și legătura cu părintele.
- `Value` e o căutare liniară într-o listă înlănțuită.
