# I/O: Reader și Writer — Senior

Contractul exact al lui `Read` și `Write`, optimizările zero-copy din `io.Copy` (`sendfile`, `splice`), cum interacționează fișierele și socket-urile cu runtime-ul și cum scrii pe disc astfel încât datele să supraviețuiască unei căderi de curent.

## Contractul lui `Read`

Documentația lui `io.Reader` are reguli precise, pe care orice implementare și orice apelant trebuie să le respecte:

- `Read` poate întoarce **mai puțini** octeți decât `len(p)`, chiar dacă nu s-a ajuns la final. Nu presupune că bufferul e plin; pentru exact `n` octeți folosește `io.ReadFull`.
- Poate întoarce `n > 0` **și** o eroare non-nil (inclusiv `io.EOF`) în același apel. Apelantul procesează întâi cei `n` octeți.
- Întoarcerea `0, nil` e descurajată (nu înseamnă EOF); apelantul o tratează ca „nimic nou”.
- Implementarea nu are voie să păstreze `p` după întoarcere.

```go
package main

import (
	"fmt"
	"io"
	"strings"
)

func main() {
	r := strings.NewReader("abc")
	buf := make([]byte, 5)

	_, err := io.ReadFull(r, buf) // cere exact 5
	fmt.Println(err)              // unexpected EOF: au fost doar 3

	r = strings.NewReader("abcdef")
	n, err := io.ReadAtLeast(r, buf, 2)
	fmt.Println(n, err) // 5 <nil>
}
```

## Contractul lui `Write`

`Write` trebuie să întoarcă o eroare non-nil dacă `n < len(p)`. Spre deosebire de `write(2)` din POSIX, un `io.Writer` **nu** face scrieri parțiale fără eroare, deci apelantul nu trebuie să reîncerce în buclă. Implementarea nu are voie să modifice `p`, nici temporar.

## `io.Copy` și interfețele de „upgrade”

`io.Copy(dst, src)` nu copiază mereu prin propriul buffer de 32 KB. Verifică întâi:

1. dacă `src` implementează `io.WriterTo` → apelează `src.WriteTo(dst)`;
2. dacă `dst` implementează `io.ReaderFrom` → apelează `dst.ReadFrom(src)`;
3. abia apoi, bucla cu buffer.

Tipurile din biblioteca standard folosesc aceste puncte pentru **zero-copy** la nivel de kernel:

| Copiere | Mecanism pe Linux |
|---|---|
| fișier → socket TCP | `sendfile(2)`: datele nu trec prin user space |
| socket → socket, pipe → socket | `splice(2)` |
| fișier → fișier | `copy_file_range(2)` |

Un server care trimite fișiere statice cu `http.ServeContent` sau un proxy care face `io.Copy(conn1, conn2)` beneficiază automat. Învelirea unui `*os.File` sau `*net.TCPConn` într-un tip propriu (de exemplu pentru metrici) **ascunde** aceste interfețe și dezactivează optimizarea; implementează și tu `ReadFrom`/`WriteTo` delegând, sau expune tipul original.

`io.CopyBuffer` permite refolosirea unui buffer (de exemplu dintr-un `sync.Pool`), dar e ignorat când se aplică una dintre căile de mai sus.

## Fișiere, socket-uri și runtime-ul

**Socket-urile** și pipe-urile sunt puse în mod non-blocant și înregistrate în **netpoller**: o citire care ar bloca parchează goroutine-ul, fără să blocheze firul (vezi [Goroutines](../../goroutines/senior/)).

**Fișierele obișnuite** pe Linux nu pot fi „așteptate” prin epoll (sunt mereu „gata”), așa că `Read`/`Write` pe ele sunt apeluri de sistem blocante: firul (M) se blochează, iar P-ul e predat altui fir. Un program care citește concurent mii de fișiere poate crea multe fire; un semafor care limitează operațiile simultane pe disc ține numărul sub control.

`os.File` are deadline-uri (`SetReadDeadline`) doar pentru descriptorii care trec prin netpoller (pipe-uri, terminale, unele fișiere speciale); pe fișiere obișnuite întorc `os.ErrNoDeadline`.

## Durabilitate: scrieri atomice pe disc

`f.Write` + `f.Close` nu garantează că datele sunt pe disc: ele stau în page cache-ul kernelului. O cădere de curent poate lăsa un fișier gol sau trunchiat. Tiparul robust pentru înlocuirea unui fișier:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func scrieAtomic(cale string, date []byte) error {
	dir := filepath.Dir(cale)
	tmp, err := os.CreateTemp(dir, ".tmp-*") // în același director: rename e atomic doar pe același FS
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // fără efect dacă rename a reușit

	if _, err := tmp.Write(date); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil { // fsync: datele ajung pe disc
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), cale); err != nil { // înlocuire atomică
		return err
	}
	d, err := os.Open(dir) // fsync pe director: rename-ul devine durabil
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func main() {
	dir, _ := os.MkdirTemp("", "atomic")
	defer os.RemoveAll(dir)
	cale := filepath.Join(dir, "config.json")
	fmt.Println(scrieAtomic(cale, []byte(`{"v":1}`)))
	b, _ := os.ReadFile(cale)
	fmt.Println(string(b))
}
```

Cititorii văd fie fișierul vechi complet, fie pe cel nou complet, niciodată o stare intermediară.

## `bufio` pe dinăuntru

- `bufio.Reader` citește din sursă în bucăți de mărimea bufferului; dacă `p` din `Read(p)` e mai mare decât bufferul și bufferul e gol, citește **direct** în `p`, fără copiere dublă.
- `bufio.Writer` face același lucru la scriere: o scriere mai mare decât bufferul liber trece direct, după ce bufferul e golit.
- `bufio.Scanner` are o limită pentru mărimea unui token (64 KB implicit). Liniile mai lungi dau `bufio.ErrTooLong`; `scanner.Buffer(buf, max)` o mărește.

## Pe scurt

- `Read` poate întoarce mai puțin decât ai cerut și date + eroare simultan; `io.ReadFull` pentru exact `n`.
- `Write` cu `n < len(p)` întoarce mereu eroare.
- `io.Copy` folosește `WriterTo`/`ReaderFrom` → `sendfile`, `splice`, `copy_file_range`.
- Socket-urile trec prin netpoller; fișierele obișnuite blochează fire.
- Scriere durabilă: fișier temporar, `Sync`, `Rename`, `Sync` pe director.
