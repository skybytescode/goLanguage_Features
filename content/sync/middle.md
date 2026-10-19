# Sync și atomic — Middle

Uneltele mai specializate din `sync`: `RWMutex`, familia `OnceFunc`/`OnceValue`, `sync.Pool`, `sync.Map`, `sync.Cond`, plus tiparele care țin secțiunile critice scurte și corecte.

## `sync.RWMutex`: mulți cititori, un scriitor

Când citirile sunt mult mai dese decât scrierile, `RWMutex` permite **mai mulți cititori simultan**; un scriitor are acces exclusiv:

```go
package main

import (
	"fmt"
	"sync"
)

type Cache struct {
	mu   sync.RWMutex
	date map[string]string
}

func (c *Cache) Get(k string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.date[k]
	return v, ok
}

func (c *Cache) Set(k, v string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.date[k] = v
}

func main() {
	c := &Cache{date: map[string]string{}}
	c.Set("limba", "go")
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { c.Get("limba") }) // citesc în paralel
	}
	wg.Wait()
	fmt.Println(c.Get("limba"))
}
```

`RWMutex` are un cost mai mare decât `Mutex`. Merită doar dacă citirile domină clar și secțiunile critice nu sunt minuscule; altfel un `Mutex` simplu e la fel de rapid sau mai rapid. Măsoară.

## `OnceFunc`, `OnceValue`, `OnceValues` (Go 1.21)

Variante mai comode ale lui `sync.Once`, care întorc funcții:

```go
package main

import (
	"fmt"
	"os"
	"sync"
)

var incarcaConfig = sync.OnceValues(func() (string, error) {
	fmt.Println("citesc o singură dată")
	b, err := os.ReadFile("/etc/hostname")
	return string(b), err
})

func main() {
	for range 3 {
		_, err := incarcaConfig() // același rezultat la fiecare apel
		fmt.Println("eroare:", err != nil)
	}
}
```

Spre deosebire de `Once.Do`, dacă funcția dă panic, fiecare apel ulterior dă **același** panic, în loc să întoarcă în tăcere o valoare neinițializată.

## Tipuri atomice și `CompareAndSwap`

`CompareAndSwap(vechi, nou)` scrie `nou` doar dacă valoarea curentă e încă `vechi`. E baza algoritmilor fără lock:

```go
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// maximAtomic actualizează m dacă v e mai mare, fără mutex.
func maximAtomic(m *atomic.Int64, v int64) {
	for {
		curent := m.Load()
		if v <= curent || m.CompareAndSwap(curent, v) {
			return
		}
		// altcineva a modificat între timp: reîncercăm
	}
}

func main() {
	var maxim atomic.Int64
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() { maximAtomic(&maxim, int64(i)) })
	}
	wg.Wait()
	fmt.Println(maxim.Load()) // 99
}
```

`atomic.Pointer[T]` permite înlocuirea atomică a unei structuri întregi: cititorii văd mereu fie versiunea veche, fie cea nouă, niciodată o stare amestecată. E tiparul „copy-on-write” pentru configurații reîncărcate la cald:

```go
package main

import (
	"fmt"
	"sync/atomic"
)

type Config struct {
	Port  int
	Debug bool
}

var config atomic.Pointer[Config]

func main() {
	config.Store(&Config{Port: 8080})
	c := config.Load() // cititorii: fără lock

	nou := *c // copie, modificare, publicare
	nou.Debug = true
	config.Store(&nou)

	fmt.Println(c.Debug, config.Load().Debug) // false true
}
```

## `sync.Pool`: refolosirea obiectelor temporare

`sync.Pool` păstrează obiecte care pot fi refolosite, reducând alocările și presiunea pe GC. Obiectele din pool pot fi eliberate oricând de GC, deci pool-ul e un **cache**, nu un depozit:

```go
package main

import (
	"bytes"
	"fmt"
	"sync"
)

var buffere = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func formateaza(nume string) string {
	b := buffere.Get().(*bytes.Buffer)
	defer func() {
		b.Reset() // curăță înainte de a-l da înapoi
		buffere.Put(b)
	}()
	b.WriteString("salut, ")
	b.WriteString(nume)
	return b.String()
}

func main() {
	fmt.Println(formateaza("Ana"))
	fmt.Println(formateaza("Ion"))
}
```

Atenție la buffere care cresc foarte mari: refolosirea lor ține memoria ocupată. `fmt` și `encoding/json` aruncă buffer-ele mai mari de un prag în loc să le pună înapoi.

## `sync.Map`

Un map sigur pentru acces concurent, optimizat pentru două cazuri: chei scrise o dată și citite des (cache-uri care doar cresc), sau goroutines care lucrează pe chei disjuncte. API-ul nu e tipizat (`any`), iar în alte cazuri un `map` + `Mutex` e mai rapid.

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var m sync.Map
	m.Store("a", 1)
	v, incarcat := m.LoadOrStore("a", 2) // există deja: întoarce valoarea existentă
	fmt.Println(v, incarcat)             // 1 true
	m.Range(func(k, v any) bool {
		fmt.Println(k, v)
		return true
	})
}
```

## `sync.Cond`

O variabilă de condiție: goroutines așteaptă (`Wait`) până când alta le anunță (`Signal`/`Broadcast`) că o condiție s-ar putea fi schimbat. E rar necesară în Go: channels și `context` acoperă aproape toate cazurile, mai clar.

## Reguli pentru secțiuni critice

1. **Scurte.** Nu face I/O, apeluri de rețea sau calcule lungi cu lock-ul ținut. Copiază ce îți trebuie, eliberează, apoi lucrează.
2. **Fără callback-uri sub lock.** Codul necunoscut poate încerca să ia același lock: deadlock (mutex-urile Go **nu** sunt reentrante).
3. **Ordine fixă** când iei mai multe lock-uri, ca să eviți deadlock-urile între goroutines.
4. **Nu copia** structurile care conțin un mutex (`go vet` te ajută).

```go,panic
package main

import "sync"

var mu sync.Mutex

func a() {
	mu.Lock()
	defer mu.Unlock()
	b() // b încearcă să ia același lock
}

func b() {
	mu.Lock() // deadlock: mutex-urile nu sunt reentrante
	defer mu.Unlock()
}

func main() { a() }
```

## Pe scurt

- `RWMutex` doar când citirile domină clar; altfel `Mutex`.
- `OnceValue(s)` pentru inițializări leneșe care întorc valori.
- `CompareAndSwap` și `atomic.Pointer` pentru actualizări fără lock.
- `sync.Pool` reduce alocările temporare; `sync.Map` doar în cazurile lui.
- Secțiuni critice scurte, fără I/O, fără reentranță.
