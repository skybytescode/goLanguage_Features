# Tooling și profiling — Middle

Profilarea cu `pprof` (CPU, heap, goroutines, blocare, mutex), profilarea serviciilor în producție cu `net/http/pprof`, trace-ul de execuție, `go generate`, informațiile de versiune din binar și `go fix`.

## Tipurile de profil

| Profil | Ce arată | Cum îl obții |
|---|---|---|
| **CPU** | unde se consumă timpul de procesor (eșantionare la 100 Hz) | `-cpuprofile`, `pprof.StartCPUProfile` |
| **heap** | memoria ocupată acum (`inuse`) și cea alocată total (`alloc`) | `-memprofile`, `pprof.WriteHeapProfile` |
| **allocs** | ca heap, dar implicit pe alocări totale | `/debug/pprof/allocs` |
| **goroutine** | stack-urile tuturor goroutine-urilor | `/debug/pprof/goroutine` |
| **block** | unde stau goroutines blocate (channels, select) | `runtime.SetBlockProfileRate` |
| **mutex** | unde se așteaptă la mutex-uri | `runtime.SetMutexProfileFraction` |

## Profil din cod

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
)

func lucruGreu() int {
	total := 0
	for i := range 200_000 {
		total += len(strings.Repeat("x", i%100))
	}
	return total
}

func main() {
	cale := filepath.Join(os.TempDir(), "cpu.out")
	f, err := os.Create(cale)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.Remove(cale)

	pprof.StartCPUProfile(f)
	rezultat := lucruGreu()
	pprof.StopCPUProfile()
	f.Close()

	info, _ := os.Stat(cale)
	fmt.Println("rezultat:", rezultat, "| profil scris:", info.Size() > 0)
	// apoi: go tool pprof -http=:8080 cpu.out
}
```

## Citirea unui profil

```bash
go tool pprof -http=:8080 cpu.out    # interfață web: flame graph, graf, sursă
go tool pprof cpu.out                # interactiv în terminal
(pprof) top 10                       # funcțiile cu cel mai mult timp propriu
(pprof) top -cum                     # după timpul cumulat (inclusiv apelurile)
(pprof) list lucruGreu               # sursa, cu timpul pe fiecare linie
```

- **flat**: timpul petrecut în funcția însăși;
- **cum**: timpul în funcție **și** în tot ce apelează.

În flame graph, lățimea e timpul; caută „platourile” largi din vârf (unde se consumă efectiv timpul), nu turnurile înalte.

Pentru memorie, `-sample_index=alloc_space` arată cine **alocă** mult (presiune pe GC), iar `inuse_space` cine **ține** memoria acum (scurgeri).

## Profilare în producție: `net/http/pprof`

```go,ignore
import _ "net/http/pprof" // înregistrează /debug/pprof/* pe http.DefaultServeMux

func main() {
	go func() {
		// DOAR pe o interfață internă, niciodată expus public
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// ... aplicația ...
}
```

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30   # CPU, 30 s
go tool pprof http://localhost:6060/debug/pprof/heap
curl http://localhost:6060/debug/pprof/goroutine?debug=2             # toate stack-urile
```

Profilul CPU costă puțin (câteva procente) cât rulează, deci poate fi colectat în producție. Profilul de goroutines e prima unealtă pentru scurgeri de goroutines și blocaje.

## Trace-ul de execuție

Profilurile agregă; **trace-ul** arată ce s-a întâmplat **în timp**: pe ce P a rulat fiecare goroutine, blocările, GC-ul, apelurile de sistem.

```bash
go test -trace=trace.out ./...
go tool trace trace.out
curl -o trace.out http://localhost:6060/debug/pprof/trace?seconds=5
```

E unealta potrivită pentru latență (de ce o cerere a durat 200 ms deși CPU-ul a fost liber?), nu pentru throughput.

## `go generate`

`go generate` rulează comenzi scrise în comentarii speciale. Nu rulează automat la build: e un pas explicit, iar rezultatul (codul generat) se pune în git.

```go,ignore
//go:generate go tool stringer -type=Stare
//go:generate protoc --go_out=. api.proto

type Stare int
```

```bash
go generate ./...
```

Fișierele generate au prin convenție primul comentariu `// Code generated ... DO NOT EDIT.`, pe care uneltele (linters, `gopls`, GitHub) îl recunosc.

## Versiunea în binar

Comanda `go` include automat informații despre build: modulele, versiunile, revizia git (`vcs.revision`), dacă arborele era modificat:

```go
package main

import (
	"fmt"
	"runtime/debug"
)

func main() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("fără informații de build")
		return
	}
	fmt.Println("versiunea Go:", info.GoVersion)
	fmt.Println("modul:", info.Main.Path, info.Main.Version)
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" || s.Key == "GOOS" || s.Key == "GOARCH" {
			fmt.Println(s.Key, "=", s.Value)
		}
	}
}
```

Pentru valori proprii, `-ldflags "-X main.versiune=v1.2.3"` setează o variabilă string la link.

```bash
go version -m ./app     # aceleași informații, citite dintr-un binar existent
```

## `go fix` și modernizarea (Go 1.26)

Din Go 1.26, `go fix` rulează un set de **modernizatori**: analizoare care rescriu automat tipare vechi în forma idiomatică actuală — de exemplu bucle `for i := 0; i < n; i++` → `for i := range n`, `interface{}` → `any`, bucle manuale de căutare → `slices.Contains`, `sort.Slice` → `slices.Sort`, `i := i` inutil în bucle.

```bash
go fix -diff ./...   # vezi ce s-ar schimba
go fix ./...         # aplică
```

## Pe scurt

- Șase profiluri: CPU, heap, allocs, goroutine, block, mutex.
- `go tool pprof -http` + `top`, `list`, flame graph; `flat` vs. `cum`.
- `net/http/pprof` pe o interfață internă pentru producție.
- Trace-ul pentru latență și comportament în timp.
- `go generate` explicit; `debug.ReadBuildInfo` pentru versiune; `go fix` modernizează codul.
