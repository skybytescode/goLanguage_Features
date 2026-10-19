# Pachete și module — Junior

Codul Go e organizat în **pachete** (un director cu fișiere `.go`) grupate în **module** (un proiect cu un fișier `go.mod`). Modulul are o versiune și dependențe; pachetul e unitatea de compilare și de vizibilitate.

## Pachetul

Fiecare fișier începe cu `package nume`. Toate fișierele dintr-un director aparțin aceluiași pachet și văd tot ce e declarat în celelalte fișiere, fără importuri:

```go,ignore
// geometrie/arie.go
package geometrie

func ArieCerc(r float64) float64 { return pi * r * r }
```

```go,ignore
// geometrie/constante.go
package geometrie

const pi = 3.14159 // vizibil în tot pachetul geometrie, dar nu în afara lui
```

Pachetul `main` e special: produce un program executabil, iar `func main()` e punctul de intrare.

## Exportat sau nu: litera mare

Un nume care începe cu **literă mare** (`ArieCerc`) e **exportat**: vizibil din alte pachete. Cu literă mică (`pi`), e privat pachetului. Nu există `public` sau `private` în Go.

## Crearea unui modul

```bash
mkdir salut && cd salut
go mod init github.com/ana/salut
```

Se creează `go.mod`:

```text
module github.com/ana/salut

go 1.26
```

Calea modulului (`github.com/ana/salut`) e prefixul tuturor pachetelor din el și, de obicei, adresa repository-ului.

## Importuri

Pachetele din biblioteca standard se importă după nume scurt; cele din module după calea completă:

```go
package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

func main() {
	fmt.Println(strings.ToUpper("go"), rand.IntN(10) < 10)
}
```

Numele cu care folosești pachetul în cod e ultimul element din cale (`rand`, nu `math/rand/v2`; `v2` e un sufix de versiune, vezi nivelul Middle).

## Structura unui proiect mic

```text
salut/
├── go.mod
├── main.go            // package main
└── salutari/
    ├── salutari.go    // package salutari
    └── salutari_test.go
```

```go,ignore
// main.go
package main

import (
	"fmt"

	"github.com/ana/salut/salutari" // calea modulului + directorul
)

func main() {
	fmt.Println(salutari.Pentru("Ana"))
}
```

## Dependențe externe

`go get` adaugă o dependență în `go.mod` și descarcă codul:

```bash
go get github.com/google/uuid@latest
```

```text
module github.com/ana/salut

go 1.26

require github.com/google/uuid v1.6.0
```

Fișierul `go.sum` conține sumele de control (hash-uri) ale fiecărei dependențe. Ambele fișiere se pun în git.

## Comenzile de bază

| Comandă | Ce face |
|---|---|
| `go run .` | compilează și rulează pachetul din directorul curent |
| `go build` | creează executabilul |
| `go test ./...` | rulează testele din toate pachetele |
| `go mod tidy` | adaugă dependențele lipsă, le șterge pe cele nefolosite |
| `go fmt ./...` | formatează codul |
| `go vet ./...` | caută greșeli frecvente |

`./...` înseamnă „directorul curent și toate subdirectoarele”.

## Alias pentru import

Când două pachete au același nume, sau numele e prea lung, poți da un alias:

```go
package main

import (
	crand "crypto/rand"
	"fmt"
	mrand "math/rand/v2"
)

func main() {
	b := make([]byte, 4)
	_, _ = crand.Read(b) // aleator criptografic
	fmt.Println(len(b), mrand.IntN(6)+1)
}
```

## Pe scurt

- Un director = un pachet; un `go.mod` = un modul.
- Literă mare = exportat; literă mică = privat pachetului.
- `go mod init cale` creează modulul; `go get` adaugă dependențe.
- `go.mod` și `go.sum` se pun în controlul versiunilor.
- `go mod tidy` ține lista de dependențe la zi.
