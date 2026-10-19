# I/O: Reader și Writer — Junior

Aproape tot ce citește sau scrie date în Go — fișiere, rețea, buffere în memorie, compresie — folosește două interfețe mici din pachetul `io`: `io.Reader` și `io.Writer`.

## Cele două interfețe

```go,ignore
type Reader interface {
	Read(p []byte) (n int, err error) // umple p cu până la len(p) octeți
}

type Writer interface {
	Write(p []byte) (n int, err error) // scrie octeții din p
}
```

Un fișier, o conexiune de rețea, un `strings.Reader`, un `bytes.Buffer` sau `os.Stdout` le implementează. O funcție scrisă pentru un `io.Reader` merge cu toate.

## Citirea și scrierea unui fișier întreg

Pentru fișiere mici, cele mai simple funcții sunt `os.WriteFile` și `os.ReadFile`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	cale := filepath.Join(os.TempDir(), "salut.txt")

	err := os.WriteFile(cale, []byte("Salut din Go!\n"), 0o644)
	if err != nil {
		fmt.Println("scriere:", err)
		return
	}

	date, err := os.ReadFile(cale)
	if err != nil {
		fmt.Println("citire:", err)
		return
	}
	fmt.Print(string(date))
	os.Remove(cale)
}
```

`0o644` sunt permisiunile fișierului (citire/scriere pentru proprietar, citire pentru ceilalți).

## Deschiderea și închiderea

Pentru fișiere mari sau citire treptată, deschizi fișierul și îl închizi cu `defer`:

```go
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	cale := filepath.Join(os.TempDir(), "date.txt")
	os.WriteFile(cale, []byte("abcdefghij"), 0o644)
	defer os.Remove(cale)

	f, err := os.Open(cale) // doar pentru citire
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	buf := make([]byte, 4)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			fmt.Printf("am citit %q\n", buf[:n])
		}
		if err == io.EOF { // sfârșitul fișierului
			break
		}
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
```

`io.EOF` nu e o eroare propriu-zisă: înseamnă „nu mai sunt date”.

## Citirea pe linii: `bufio.Scanner`

```go
package main

import (
	"bufio"
	"fmt"
	"strings"
)

func main() {
	text := "prima linie\na doua linie\na treia linie"
	scanner := bufio.NewScanner(strings.NewReader(text))
	nr := 0
	for scanner.Scan() {
		nr++
		fmt.Printf("%d: %s\n", nr, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("eroare:", err)
	}
}
```

Același cod funcționează cu un fișier (`bufio.NewScanner(f)`) sau cu intrarea standard (`bufio.NewScanner(os.Stdin)`).

## Scrierea cu `fmt.Fprintf`

`fmt.Fprintf` scrie formatat în **orice** `io.Writer`:

```go
package main

import (
	"bytes"
	"fmt"
	"os"
)

func raport(w *bytes.Buffer, nume string, scor int) {
	fmt.Fprintf(w, "%-6s %3d\n", nume, scor)
}

func main() {
	var buf bytes.Buffer
	raport(&buf, "Ana", 95)
	raport(&buf, "Ion", 87)

	fmt.Fprint(os.Stdout, buf.String()) // os.Stdout e și el un io.Writer
	fmt.Fprintln(os.Stderr, "mesajele de eroare merg pe stderr")
}
```

## Crearea unui fișier

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	cale := filepath.Join(os.TempDir(), "lista.txt")
	f, err := os.Create(cale) // creează sau golește fișierul
	if err != nil {
		fmt.Println(err)
		return
	}
	for i := 1; i <= 3; i++ {
		fmt.Fprintf(f, "elementul %d\n", i)
	}
	if err := f.Close(); err != nil { // la scriere, verifică eroarea de la Close
		fmt.Println(err)
		return
	}

	date, _ := os.ReadFile(cale)
	fmt.Print(string(date))
	os.Remove(cale)
}
```

## Căi de fișiere

Folosește `path/filepath` pentru căi pe disc: alege separatorul corect (`/` sau `\`) pentru sistemul curent.

```go
package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	cale := filepath.Join("date", "2026", "raport.csv")
	fmt.Println(cale)
	fmt.Println(filepath.Base(cale), filepath.Ext(cale), filepath.Dir(cale))
}
```

## Pe scurt

- `io.Reader` și `io.Writer` sunt peste tot: fișiere, rețea, memorie.
- `os.ReadFile`/`os.WriteFile` pentru fișiere mici.
- `os.Open` + `defer f.Close()` pentru citire treptată; `io.EOF` marchează sfârșitul.
- `bufio.Scanner` citește pe linii.
- `fmt.Fprintf` scrie formatat în orice `Writer`.
