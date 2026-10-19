# Sync și atomic — Senior

Cum e implementat `sync.Mutex` (modul normal și modul de înfometare), ce garantează modelul de memorie pentru operațiile atomice, cum funcționează `sync.Pool` per-P cu „victim cache” și ce e false sharing.

## `sync.Mutex` pe dinăuntru

Un mutex are doar două câmpuri: `state int32` și `sema uint32`. Biții din `state` codifică: blocat, trezit (o goroutine a fost deja trezită), înfometare, plus numărul de goroutines care așteaptă.

**Drumul rapid**: `Lock` încearcă un singur `CompareAndSwap(0, locked)`. Fără contenție, asta e tot — câțiva nanosecunde, inlined.

**Drumul lent**:

1. **Spinning**: dacă mutex-ul e ținut și sunt condiții bune (mai multe nuclee, alte P-uri active, coada locală goală), goroutine-ul se învârte câteva iterații, sperând că deținătorul eliberează curând. Evită costul parcării.
2. **Parcarea**: altfel, goroutine-ul se înscrie ca așteptător și doarme pe semaforul runtime-ului (`runtime_SemacquireMutex`), implementat ca un arbore echilibrat (treap) de cozi indexate după adresă.

## Modul normal vs. modul de înfometare

În **modul normal**, o goroutine trezită concurează pentru lock cu goroutines noi care tocmai au sosit și sunt deja pe CPU. Cele noi câștigă de obicei, ceea ce e bun pentru throughput (nu se face comutare de context), dar poate lăsa o goroutine să aștepte foarte mult.

Dacă un așteptător nu obține lock-ul timp de **peste 1 ms**, mutex-ul trece în **modul de înfometare**: la `Unlock`, proprietatea e predată **direct** primului din coadă; nou-veniții nu mai încearcă să-l ia și se pun la coadă. Mutex-ul revine la modul normal când ultimul așteptător îl obține sau când unul a așteptat sub 1 ms.

Compromisul: modul normal dă performanță, modul de înfometare dă latență predictibilă în coada distribuției.

## `RWMutex`: preferință pentru scriitori

Când un scriitor cheamă `Lock`, el anunță intenția (scade `readerCount` cu o constantă mare); **cititorii noi** se blochează imediat, chiar dacă alți cititori încă citesc. Scriitorul așteaptă doar cititorii existenți. Asta previne înfometarea scriitorilor, dar are o consecință: **`RLock` recursiv poate face deadlock**:

```go,ignore
func (c *Cache) A() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.B() // dacă între timp un scriitor a cerut Lock, RLock-ul din B blochează pentru totdeauna
}

func (c *Cache) B() {
	c.mu.RLock()
	defer c.mu.RUnlock()
}
```

## Modelul de memorie și `atomic`

Din 2022, modelul de memorie Go specifică explicit: operațiile din `sync/atomic` sunt **secvențial consistente** — se comportă ca și cum ar fi executate într-o ordine globală unică, consistentă cu ordinea din fiecare goroutine. Dacă efectul unei operații atomice A e observat de operația atomică B, atunci A se întâmplă înaintea lui B.

Asta înseamnă că un flag atomic poate „publica” date scrise normal:

```go
package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

var (
	date []int       // scris normal
	gata atomic.Bool // publicare
)

func main() {
	go func() {
		date = []int{1, 2, 3}
		gata.Store(true) // scrierea lui date se întâmplă înainte
	}()
	for !gata.Load() {
		time.Sleep(time.Millisecond)
	}
	fmt.Println(date) // garantat [1 2 3]
}
```

Go nu expune ordonări mai slabe (acquire/release, relaxed) ca C++ sau Rust; simplitatea costă puțin pe x86 (unde load-urile sunt oricum ordonate) și ceva mai mult pe ARM.

Tipurile `atomic.Int64` & co. garantează și **alinierea la 64 de biți** pe platformele de 32 de biți — motivul pentru care sunt preferate funcțiilor vechi `atomic.AddInt64(&x, 1)`.

## `sync.Pool`: cache per-P și victim cache

Fiecare P are propriul `poolLocal`: un slot `private` (fără nicio sincronizare) și o coadă `shared` lock-free (o dequeue). `Get` încearcă, în ordine: `private`, capul cozii proprii, **furt** din coada altor P-uri, victim cache, `New`.

La fiecare GC, conținutul pool-ului e mutat în **victim cache**, iar victim cache-ul anterior e aruncat. Un obiect supraviețuiește deci cel puțin un ciclu de GC, ceea ce elimină „cliff”-ul de performanță de dinainte de Go 1.13, când pool-ul era golit complet la fiecare GC.

Consecință: un `sync.Pool` nu e potrivit pentru resurse care trebuie închise (conexiuni, fișiere): obiectele pot dispărea fără nicio notificare.

## `sync.Map` (Go 1.24)

Implementarea inițială folosea două map-uri (unul read-only citit atomic, unul „dirty” protejat de mutex), promovându-l pe al doilea după destule ratări. Din Go 1.24, `sync.Map` e construit pe un **hash-trie concurent** (`HashTrieMap`, același folosit de pachetul `unique`), cu contenție mult mai mică la scrieri pe chei disjuncte și fără promovări costisitoare.

## False sharing

Procesoarele sincronizează memoria la nivel de **linie de cache** (de obicei 64 de octeți). Două variabile atomice independente, aflate în aceeași linie și modificate de pe nuclee diferite, se „bat” pentru linie, deși logic nu au legătură:

```go,ignore
type Contoare struct {
	a atomic.Int64 // același 64 de octeți...
	b atomic.Int64 // ...: fiecare Add invalidează linia pentru celălalt nucleu
}

type ContoareSeparate struct {
	a atomic.Int64
	_ [56]byte // padding: b ajunge pe altă linie de cache
	b atomic.Int64
}
```

Runtime-ul folosește exact acest padding (`cpu.CacheLinePad`) pentru structurile per-P. În codul aplicației, merită doar după ce un profil arată contenție pe astfel de câmpuri.

## Sharding

Când un singur mutex devine punctul fierbinte, împarte datele în N bucăți, fiecare cu mutex-ul ei, alese după hash-ul cheii. Contenția scade aproximativ de N ori, cu prețul unor operații globale (numărare, iterare) mai complicate.

## Pe scurt

- `Mutex`: CAS pe drumul rapid, spinning, apoi parcare pe semafor.
- Modul de înfometare după 1 ms de așteptare: predare directă, latență predictibilă.
- `RWMutex` preferă scriitorii; `RLock` recursiv poate bloca.
- Atomicele sunt secvențial consistente și pot publica date.
- `sync.Pool` e per-P, cu victim cache; atenție la false sharing.
