# Select — Junior

`select` așteaptă pe **mai multe** operații cu channels în același timp și o execută pe prima care devine posibilă. E ca un `switch`, dar pentru channels.

## Așteptarea pe mai multe channels

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	rapid := make(chan string)
	lent := make(chan string)

	go func() {
		time.Sleep(10 * time.Millisecond)
		rapid <- "răspuns rapid"
	}()
	go func() {
		time.Sleep(50 * time.Millisecond)
		lent <- "răspuns lent"
	}()

	for range 2 {
		select {
		case m := <-rapid:
			fmt.Println(m)
		case m := <-lent:
			fmt.Println(m)
		}
	}
}
```

`select` blochează până când **un** `case` poate continua, îl execută, apoi se termină. De aceea l-am pus într-o buclă, ca să primim ambele mesaje.

## Timeout cu `time.After`

`time.After(d)` întoarce un channel pe care vine o valoare după durata `d`. Pus într-un `select`, limitează cât aștepți:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	rezultat := make(chan string)
	go func() {
		time.Sleep(200 * time.Millisecond) // o operație prea lentă
		rezultat <- "gata"
	}()

	select {
	case r := <-rezultat:
		fmt.Println(r)
	case <-time.After(50 * time.Millisecond):
		fmt.Println("timeout: am renunțat după 50ms")
	}
}
```

## `default`: fără așteptare

Dacă niciun `case` nu e gata, ramura `default` rulează imediat. Așa faci o trimitere sau o primire **non-blocantă**:

```go
package main

import "fmt"

func main() {
	mesaje := make(chan string, 1)

	select {
	case m := <-mesaje:
		fmt.Println("primit:", m)
	default:
		fmt.Println("niciun mesaj") // channel-ul e gol
	}

	mesaje <- "salut"
	select {
	case mesaje <- "încă unul":
		fmt.Println("trimis")
	default:
		fmt.Println("buffer plin, renunț") // nu blocăm
	}
}
```

## Trimitere și primire în același `select`

Un `case` poate fi și o trimitere:

```go
package main

import "fmt"

func main() {
	intrare := make(chan int, 1)
	iesire := make(chan int, 1)
	intrare <- 5

	select {
	case v := <-intrare:
		fmt.Println("am citit", v)
	case iesire <- 10:
		fmt.Println("am scris 10")
	}
}
```

Aici ambele cazuri sunt gata. Când mai multe sunt gata simultan, `select` alege **aleatoriu** unul dintre ele, deci rezultatul poate diferi de la o rulare la alta.

## Ascultarea unui semnal de oprire

Cel mai frecvent tipar: o goroutine lucrează într-o buclă și verifică la fiecare pas dacă trebuie să se oprească:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	stop := make(chan struct{})
	gata := make(chan struct{})

	go func() {
		defer close(gata)
		pas := 0
		for {
			select {
			case <-stop:
				fmt.Println("oprit după", pas, "pași")
				return
			default:
				pas++
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	time.Sleep(30 * time.Millisecond)
	close(stop)
	<-gata
}
```

În codul real, semnalul vine de obicei de la un `context.Context` (`case <-ctx.Done():`).

## `select {}`

Un `select` fără niciun `case` blochează pentru totdeauna. Uneori apare la finalul lui `main` într-un program în care goroutines fac toată treaba (de exemplu un server), dar de obicei există variante mai clare.

## Pe scurt

- `select` așteaptă pe mai multe channels și execută primul `case` gata.
- `time.After` adaugă un timeout.
- `default` face operațiile non-blocante.
- Când mai multe cazuri sunt gata, alegerea e aleatoare.
- Tiparul `for { select { case <-stop: return; ... } }` oprește goroutines curat.
