# Channels — Junior

Un **channel** e o „țeavă” prin care goroutines își trimit valori. Pe lângă transportul datelor, channel-ul **sincronizează**: trimiterea și primirea pot aștepta una după alta.

## Crearea, trimiterea și primirea

```go
package main

import "fmt"

func main() {
	ch := make(chan string) // channel de string-uri

	go func() {
		ch <- "salut din goroutine" // trimite
	}()

	mesaj := <-ch // primește (așteaptă până vine o valoare)
	fmt.Println(mesaj)
}
```

- `ch <- v` trimite valoarea `v`.
- `v := <-ch` primește o valoare.
- Tipul channel-ului (`chan string`) fixează tipul valorilor.

## Channel fără buffer: întâlnire

Un channel creat cu `make(chan T)` nu are buffer. Trimițătorul **așteaptă** până când un receptor ia valoarea, iar receptorul așteaptă până vine o valoare. E un punct de întâlnire între două goroutines:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	gata := make(chan bool)

	go func() {
		fmt.Println("lucrez...")
		time.Sleep(50 * time.Millisecond)
		fmt.Println("am terminat")
		gata <- true
	}()

	<-gata // main așteaptă semnalul
	fmt.Println("main continuă")
}
```

## Channel cu buffer

`make(chan T, n)` creează un channel cu loc pentru `n` valori. Trimiterea blochează doar când buffer-ul e **plin**; primirea, doar când e **gol**:

```go
package main

import "fmt"

func main() {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3                       // încă nu blochează: buffer-ul are 3 locuri
	fmt.Println(len(ch), cap(ch)) // 3 3

	fmt.Println(<-ch, <-ch, <-ch) // 1 2 3: ordinea se păstrează (FIFO)
}
```

## Închiderea și `range`

Trimițătorul poate **închide** channel-ul cu `close` ca să anunțe că nu mai vin valori. `range` peste un channel citește până la închidere:

```go
package main

import "fmt"

func genereaza(n int) chan int {
	ch := make(chan int)
	go func() {
		for i := 1; i <= n; i++ {
			ch <- i
		}
		close(ch) // gata: nu mai trimitem nimic
	}()
	return ch
}

func main() {
	for v := range genereaza(5) {
		fmt.Print(v, " ") // 1 2 3 4 5
	}
	fmt.Println()
}
```

## A fost închis? Forma comma-ok

După închidere, primirea nu mai blochează: întoarce imediat valoarea zero. Al doilea rezultat spune dacă valoarea e reală:

```go
package main

import "fmt"

func main() {
	ch := make(chan int, 1)
	ch <- 42
	close(ch)

	v, ok := <-ch
	fmt.Println(v, ok) // 42 true: valoare reală, din buffer
	v, ok = <-ch
	fmt.Println(v, ok) // 0 false: channel închis și gol
}
```

## Direcția channel-ului

În semnătura unei funcții poți spune că un channel e doar pentru trimitere (`chan<-`) sau doar pentru primire (`<-chan`). Compilatorul verifică folosirea corectă:

```go
package main

import "fmt"

func producator(out chan<- int) { // doar trimite
	for i := range 3 {
		out <- i
	}
	close(out)
}

func consumator(in <-chan int) { // doar primește
	for v := range in {
		fmt.Println("primit", v)
	}
}

func main() {
	ch := make(chan int)
	go producator(ch)
	consumator(ch)
}
```

## Deadlock

Dacă toate goroutines așteaptă și niciuna nu mai poate avansa, runtime-ul oprește programul:

```go,panic
package main

func main() {
	ch := make(chan int)
	ch <- 1 // nimeni nu primește: main așteaptă la nesfârșit
	// fatal error: all goroutines are asleep - deadlock!
}
```

Cu un channel fără buffer, trimiterea și primirea trebuie să fie în goroutines **diferite**.

## Pe scurt

- `make(chan T)` fără buffer: trimiterea și primirea se așteaptă reciproc.
- `make(chan T, n)` cu buffer: blochează doar când e plin sau gol.
- `close(ch)` anunță sfârșitul; `range ch` citește până la închidere.
- `v, ok := <-ch` spune dacă channel-ul a fost închis.
- `chan<-` și `<-chan` restrâng direcția în semnături.
