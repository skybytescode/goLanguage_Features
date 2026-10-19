# Sync și atomic — Junior

Când mai multe goroutines folosesc **aceeași** variabilă și cel puțin una o modifică, ai nevoie de sincronizare. Pachetul `sync` oferă unelte simple pentru asta: `Mutex`, `WaitGroup`, `Once`. Pachetul `sync/atomic` oferă operații atomice pe numere.

## Problema

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	sold := 0
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			sold += 10 // mai multe goroutines scriu simultan: rezultat imprevizibil
		})
	}
	wg.Wait()
	fmt.Println(sold) // ar trebui 10000, dar adesea e mai puțin
}
```

`sold += 10` înseamnă de fapt trei pași: citește, adună, scrie. Două goroutines pot citi aceeași valoare veche și una dintre scrieri se pierde.

## `sync.Mutex`

Un **mutex** (mutual exclusion) lasă o singură goroutine să intre într-o secțiune de cod. Celelalte așteaptă la `Lock`:

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var (
		mu   sync.Mutex
		sold int
		wg   sync.WaitGroup
	)
	for range 1000 {
		wg.Go(func() {
			mu.Lock()
			sold += 10
			mu.Unlock()
		})
	}
	wg.Wait()
	fmt.Println(sold) // 10000, garantat
}
```

## `defer Unlock`

Cel mai sigur tipar: `Unlock` amânat imediat după `Lock`. Mutex-ul e eliberat chiar dacă funcția iese mai devreme sau dă panic:

```go
package main

import (
	"errors"
	"fmt"
	"sync"
)

type Cont struct {
	mu   sync.Mutex
	sold int
}

func (c *Cont) Retrage(suma int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if suma > c.sold {
		return errors.New("fonduri insuficiente") // Unlock rulează și aici
	}
	c.sold -= suma
	return nil
}

func (c *Cont) Depune(suma int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sold += suma
}

func main() {
	var c Cont
	c.Depune(100)
	fmt.Println(c.Retrage(30))  // <nil>
	fmt.Println(c.Retrage(500)) // fonduri insuficiente
	fmt.Println(c.sold)         // 70
}
```

Convenția: mutex-ul stă în structură, **lângă** datele pe care le protejează, iar valoarea zero e gata de folosit.

## `sync.Once`: o singură dată

`Once.Do(f)` rulează `f` exact o dată, oricâte goroutines ar apela simultan. E ideal pentru inițializări leneșe:

```go
package main

import (
	"fmt"
	"sync"
)

var (
	once   sync.Once
	config map[string]string
)

func Config() map[string]string {
	once.Do(func() {
		fmt.Println("încarc configurația...") // apare o singură dată
		config = map[string]string{"port": "8080"}
	})
	return config
}

func main() {
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() { _ = Config() })
	}
	wg.Wait()
	fmt.Println(Config()["port"])
}
```

## `sync/atomic` pentru contoare

Pentru un simplu contor, un tip atomic e mai simplu și mai rapid decât un mutex:

```go
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {
	var vizite atomic.Int64
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			vizite.Add(1)
		})
	}
	wg.Wait()
	fmt.Println(vizite.Load()) // 1000
}
```

Tipurile atomice (`atomic.Int32`, `Int64`, `Uint64`, `Bool`, `Pointer[T]`) au metodele `Load`, `Store`, `Add` (la numere), `Swap` și `CompareAndSwap`.

## Ce să alegi

| Situație | Unealtă |
|---|---|
| un contor sau un flag | `atomic.Int64`, `atomic.Bool` |
| mai multe câmpuri care trebuie modificate împreună | `sync.Mutex` |
| inițializare o singură dată | `sync.Once` |
| aștepți ca un grup de goroutines să termine | `sync.WaitGroup` |
| transmiți date între goroutines | channels |

## Pe scurt

- Accesul concurent cu scrieri cere sincronizare.
- `mu.Lock()` + `defer mu.Unlock()` e tiparul de bază.
- Mutex-ul stă lângă datele protejate.
- `sync.Once` pentru inițializări, `atomic` pentru contoare.
- Verifică totul cu `go test -race`.
