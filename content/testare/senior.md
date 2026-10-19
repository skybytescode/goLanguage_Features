# Testare, benchmarks și fuzzing — Senior

Fuzzing-ul ghidat de acoperire, metodologia corectă pentru benchmarks (zgomot, `benchstat`, eliminarea codului mort), cum funcționează bula din `synctest`, cache-ul de teste, acoperirea pentru binare întregi și organizarea testelor interne vs. externe.

## Fuzzing (Go 1.18)

Un test de fuzzing primește intrări generate automat și verifică **proprietăți** care trebuie să fie adevărate pentru orice intrare:

```go,ignore
func FuzzInversare(f *testing.F) {
	f.Add("salut")    // corpus inițial (seed)
	f.Add("ăîșț")
	f.Fuzz(func(t *testing.T, s string) {
		inv := Inverseaza(s)
		dublu := Inverseaza(inv)
		if utf8.ValidString(s) && s != dublu {
			t.Errorf("Inverseaza(Inverseaza(%q)) = %q", s, dublu)
		}
		if utf8.ValidString(s) && !utf8.ValidString(inv) {
			t.Errorf("Inverseaza(%q) a produs UTF-8 invalid: %q", s, inv)
		}
	})
}
```

```bash
go test -fuzz=FuzzInversare -fuzztime=30s
```

Cum funcționează:

- fuzzer-ul compilează pachetul cu **instrumentare de acoperire** (contoare pe ramuri, în stilul libFuzzer);
- pornește mai multe procese lucrătoare care **mută** intrările din corpus (inversează biți, inserează octeți, combină intrări) și le păstrează pe cele care ating ramuri noi;
- la o eroare, **minimizează** intrarea și o salvează în `testdata/fuzz/FuzzInversare/<hash>`;
- fișierul salvat devine un caz de regresie: un `go test` obișnuit (fără `-fuzz`) rulează mereu corpusul din `testdata`.

Tipurile acceptate ca argumente sunt cele de bază: `string`, `[]byte`, numere, `bool`, `rune`. Proprietăți bune de fuzzing: ida-întors (`Decode(Encode(x)) == x`), comparația cu o implementare de referință, absența panic-urilor pe input arbitrar (parsere!).

## Benchmarks corecte

Un benchmark e o **măsurătoare statistică**, nu un număr. Sursele de zgomot: frecvența variabilă a procesorului, alte procese, GC-ul, alinierea codului în memorie. Metodologia:

```bash
go test -bench=. -count=10 -benchmem > vechi.txt
# ... schimbarea ...
go test -bench=. -count=10 -benchmem > nou.txt
benchstat vechi.txt nou.txt          # golang.org/x/perf/cmd/benchstat
```

```text
          │  vechi.txt  │              nou.txt               │
          │   sec/op    │   sec/op     vs base               │
Parse-8     1.204µ ± 2%   0.981µ ± 1%  -18.52% (p=0.000 n=10)
```

`benchstat` raportează mediana, intervalul de variație și dacă diferența e semnificativă statistic. Fără `-count` și `benchstat`, „10% mai rapid” poate fi doar zgomot.

**Eliminarea codului mort.** Compilatorul poate observa că rezultatul unei funcții pure nu e folosit și poate elimina apelul, iar benchmark-ul măsoară nimic. Cu bucla clasică `for i := 0; i < b.N; i++` se foloseau variabile „sink” globale. `b.Loop` (Go 1.24) împiedică această optimizare pentru apelurile din corpul buclei și, în plus, rulează funcția de benchmark o singură dată (cu `b.N` era rulată de mai multe ori, cu setup-ul repetat).

Alte reguli: `b.ReportAllocs()` sau `-benchmem` pentru alocări, `-cpu=1,4,8` pentru scalare, `b.RunParallel` pentru cod concurent, `b.SetBytes(n)` pentru throughput în MB/s.

## Bula din `synctest` pe dinăuntru

`synctest.Test` pornește funcția într-o goroutine nouă, marcată ca aparținând unei **bule**. Toate goroutines pornite din ea aparțin aceleiași bule. Bula are propriul **ceas**, care pornește la miezul nopții UTC din 1 ianuarie 2000 și nu avansează singur.

Runtime-ul urmărește dacă toate goroutines din bulă sunt **blocate durabil** — blocate în operații pe care doar alte goroutines din bulă le pot debloca: operații pe channels create în bulă, `time.Sleep`, `sync.WaitGroup.Wait`, `sync.Cond.Wait`, `select` pe astfel de channels. Când se întâmplă asta:

- dacă există timere în bulă, ceasul **sare** direct la următorul timer, care se declanșează;
- dacă nu există timere, testul e într-un **deadlock** și eșuează cu un mesaj explicit.

Operațiile care așteaptă ceva din afara bulei (I/O de rețea reală, mutex-uri, channels create în afara bulei) **nu** sunt blocări durabile: bula nu poate avansa timpul cât timp o goroutine stă în ele. De aceea, codul testat cu `synctest` folosește `net.Pipe` sau implementări în memorie în locul rețelei reale.

## Cache-ul de teste

`go test` păstrează rezultatele testelor care au trecut și le refolosește (`(cached)` în output) dacă nu s-a schimbat nimic: sursele, flag-urile, variabilele de mediu **citite** de test și fișierele **deschise** de test (urmărite prin `os.Open`, `os.Stat` etc.). Rezultatele se cache-uiesc doar în modul listă de pachete (`go test ./...`), nu în modul director curent (`go test` simplu). `-count=1` dezactivează cache-ul.

## Acoperirea pentru binare întregi (Go 1.20)

Acoperirea nu e limitată la teste unitare. Un binar construit cu `-cover` scrie date de acoperire la ieșire, util pentru teste de integrare sau end-to-end:

```bash
go build -cover -o app ./cmd/app
GOCOVERDIR=./acoperire ./app --scenariu=complet
go tool covdata percent -i=./acoperire
go tool covdata textfmt -i=./acoperire -o profil.txt   # apoi go tool cover -html
```

## Teste interne și externe

- `package calcul` (în `calcul_test.go`) — test **intern**: acces la tot, inclusiv la identificatorii neexportați.
- `package calcul_test` — test **extern**: doar API-ul public, exact ca un utilizator. Rupe și ciclurile de import (un test care are nevoie de un pachet care importă `calcul`).

Când un test extern are totuși nevoie de ceva intern, convenția e un fișier `export_test.go` în pachetul intern, care expune doar pentru teste:

```go,ignore
// export_test.go
package calcul

var ParseazaIntern = parseaza // vizibil doar pentru calcul_test
```

## Pe scurt

- Fuzzing ghidat de acoperire; erorile găsite devin cazuri de regresie în `testdata/fuzz`.
- Benchmarks: `-count=10` + `benchstat`; `b.Loop` previne eliminarea codului mort.
- `synctest` avansează un ceas fals când toate goroutines din bulă sunt blocate durabil.
- Cache-ul de teste ține cont de fișierele și variabilele de mediu citite.
- `-cover` și `GOCOVERDIR` pentru acoperirea binarelor; `export_test.go` pentru teste externe.
