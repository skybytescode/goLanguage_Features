# Runtime: scheduler, memorie, GC — Middle

Cum controlezi GC-ul (`GOGC`, `GOMEMLIMIT`), cum citești ce face (`gctrace`, `runtime/metrics`) și cum reduci presiunea pe el scriind cod care alocă mai puțin.

## `GOGC`: compromisul memorie–CPU

GC-ul pornește un nou ciclu când heap-ul a crescut cu un anumit procent față de heap-ul **viu** (ce a supraviețuit ultimei colectări). Procentul e `GOGC`, implicit **100**:

| `GOGC` | Heap viu 100 MiB → următorul GC la | Efect |
|---|---|---|
| 50 | ~150 MiB | mai puțină memorie, mai mult CPU pe GC |
| 100 | ~200 MiB | implicit |
| 400 | ~500 MiB | mai multă memorie, mai puțin CPU pe GC |
| `off` | niciodată (doar cu `GOMEMLIMIT`) | — |

Dublarea lui `GOGC` dublează aproximativ memoria suplimentară și înjumătățește aproximativ costul CPU al GC-ului. Se setează din mediu (`GOGC=200 ./app`) sau din cod (`debug.SetGCPercent(200)`).

## `GOMEMLIMIT`: limita de memorie (Go 1.19)

`GOGC` se raportează la heap-ul viu, deci nu știe cât e limita containerului. Dacă heap-ul viu e 600 MiB și containerul are 1 GiB, `GOGC=100` ar lăsa heap-ul să crească spre 1,2 GiB → OOM kill.

`GOMEMLIMIT` e o limită **„moale”** pentru toată memoria gestionată de runtime: când se apropie de ea, GC-ul rulează mai des, indiferent de `GOGC`.

```bash
# container cu 1 GiB: lași o marjă pentru stive, buffere din afara heap-ului etc.
GOMEMLIMIT=900MiB ./app

# heap mic, dar vrei să folosești memoria disponibilă fără GC-uri inutile:
GOGC=off GOMEMLIMIT=900MiB ./app
```

Limita e moale: dacă heap-ul viu chiar depășește limita, runtime-ul nu oprește programul, ci limitează CPU-ul consumat de GC (cel mult ~50%), ca să evite „death spiral”-ul în care programul doar colectează.

```go
package main

import (
	"fmt"
	"runtime/debug"
)

func main() {
	vechi := debug.SetMemoryLimit(512 << 20)            // 512 MiB; întoarce valoarea anterioară
	fmt.Println("limita anterioară:", vechi == 1<<63-1) // implicit: fără limită
	fmt.Println("GOGC anterior:", debug.SetGCPercent(150))
}
```

## Ce face GC-ul: `gctrace`

```bash
$ GODEBUG=gctrace=1 ./app
gc 7 @2.104s 3%: 0.031+4.2+0.044 ms clock, 0.25+1.1/8.3/12+0.35 ms cpu, 61->63->32 MB, 64 MB goal, 0 MB stacks, 0 MB globals, 8 P
```

| Fragment | Înseamnă |
|---|---|
| `gc 7 @2.104s` | al 7-lea ciclu, la 2,1 s de la pornire |
| `3%` | procentul de CPU folosit de GC de la pornire |
| `0.031+4.2+0.044 ms clock` | pauză STW la start + marcare concurentă + pauză STW la final |
| `61->63->32 MB` | heap la start → heap la final → heap viu marcat |
| `64 MB goal` | ținta pentru următorul ciclu |

Pauzele stop-the-world sunt de obicei **sub o milisecundă**; marcarea rulează concurent cu programul.

## `runtime/metrics`

Pentru monitorizare în producție, `runtime/metrics` oferă metrici stabile, ieftine de citit (fără stop-the-world, spre deosebire de `ReadMemStats`):

```go
package main

import (
	"fmt"
	"runtime/metrics"
)

func main() {
	esantioane := []metrics.Sample{
		{Name: "/gc/heap/live:bytes"},
		{Name: "/gc/cycles/total:gc-cycles"},
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/memory/classes/total:bytes"},
	}
	metrics.Read(esantioane)
	for _, s := range esantioane {
		fmt.Printf("%-32s %d\n", s.Name, s.Value.Uint64())
	}
}
```

Exportoarele Prometheus și OpenTelemetry pentru Go le citesc pe acestea.

## Mai puține alocări = mai puțin GC

Costul GC-ului crește cu **rata de alocare** și cu numărul de **pointeri** de scanat. Tehnici concrete:

- **Prealocare**: `make([]T, 0, n)`, `make(map[K]V, n)`, `strings.Builder.Grow`.
- **Refolosirea buffer-elor**: `sync.Pool`, sau buffere primite de la apelant (`Read(p []byte)`).
- **Evită conversiile inutile**: `[]byte` ↔ `string`, valori mici transformate în `any` (de exemplu argumente de logare în bucle fierbinți).
- **Structuri fără pointeri**: un `[]Punct` cu `Punct{X, Y int}` nu e scanat de GC; un `[]*Punct` are un pointer per element.
- **Valori, nu pointeri**, pentru structuri mici (stau pe stivă).

Măsoară întâi: `go test -bench . -benchmem` arată `allocs/op`, iar profilul de alocări (`-memprofile`, vezi [Tooling](../../tooling/middle/)) arată **unde** se aloca.

## `GODEBUG`

`GODEBUG` controlează opțiuni de runtime și de compatibilitate: `gctrace=1`, `schedtrace=1000`, `madvdontneed=1`, plus setări care păstrează comportamente vechi după schimbări în biblioteca standard (de exemplu `http2client=0`). Setările implicite depind de linia `go` din `go.mod`; le poți fixa și cu directive `//go:debug` în pachetul `main`.

## Pe scurt

- `GOGC` schimbă memoria pe CPU; implicit 100.
- `GOMEMLIMIT` e esențial în containere; setează-l puțin sub limită.
- `GODEBUG=gctrace=1` și `runtime/metrics` arată ce face GC-ul.
- Mai puține alocări și mai puțini pointeri = GC mai ieftin.
- Măsoară cu `-benchmem` și profiluri înainte să optimizezi.
