# Pachete și module — Senior

Cum alege comanda `go` versiunile (Minimal Version Selection), cum sunt verificate dependențele (proxy, checksum database), cum funcționează `toolchain` și `GOTOOLCHAIN`, constrângerile de build și ce face linker-ul cu codul nefolosit.

## Minimal Version Selection (MVS)

Majoritatea managerilor de pachete (npm, Cargo, pip) rezolvă dependențele căutând **cea mai nouă** versiune care satisface toate constrângerile, cu un solver SAT și un fișier lock. Go face altceva:

> Pentru fiecare modul, se alege **cea mai mare** dintre versiunile **minime** cerute de oricine în graful de dependențe.

```text
aplicatie  requires  A v1.2, B v1.0
A v1.2     requires  C v1.3
B v1.0     requires  C v1.5

→ se folosește C v1.5 (maximul cerințelor), chiar dacă există C v1.9
```

Consecințe:

- **Reproductibil fără lock file**: aceleași `go.mod`-uri dau mereu același build; o versiune nouă publicată mâine nu schimbă nimic până nu actualizezi explicit.
- **Fără solver**: algoritmul e o parcurgere de graf, liniară.
- **Fără limite superioare**: nu poți spune „`< v1.6`”. Versiunile incompatibile se exclud cu `exclude` în modulul principal sau sunt retrase de autor cu `retract`.

Din Go 1.17, `go.mod` listează **toate** modulele necesare pentru pachetele importate (graph pruning), ca să nu fie nevoie de citirea fiecărui `go.mod` din graf. De aceea un `go.mod` modern are două blocuri `require`, al doilea cu dependențe `// indirect`.

## Proxy și checksum database

Comanda `go` nu descarcă direct din git. Implicit:

```bash
GOPROXY=https://proxy.golang.org,direct
GOSUMDB=sum.golang.org
```

- **Proxy-ul** servește arhive zip imutabile pentru fiecare `modul@versiune`. Un tag șters sau mutat în git nu schimbă ce primesc ceilalți.
- **Checksum database** e un log transparent (arbore Merkle, ca Certificate Transparency) cu hash-ul fiecărei versiuni publice. La prima descărcare, hash-ul e verificat în log și scris în `go.sum`. Un atacator nu poate servi conținut diferit pentru aceeași versiune fără să fie detectat.
- Pentru module private: `GOPRIVATE=github.com/firma/*` le exclude din proxy și din checksum database (dar `go.sum` le verifică în continuare).

## `toolchain` și `GOTOOLCHAIN` (Go 1.21)

Linia `go 1.26` din `go.mod` e o **cerință minimă**, nu o sugestie: un toolchain mai vechi refuză modulul. Dacă instalezi Go 1.24 și compilezi un modul care cere 1.26, comanda `go` **descarcă automat** toolchain-ul potrivit (ca modul, prin proxy) și îl folosește:

```text
module github.com/ana/app

go 1.26
toolchain go1.26.4
```

```bash
GOTOOLCHAIN=local     # nu descărca niciodată; eșuează dacă versiunea e prea veche
GOTOOLCHAIN=go1.26.4  # folosește exact această versiune
go get go@1.26        # ridică cerința minimă a modulului
```

Linia `go` controlează și **semantica limbajului**: de exemplu, variabilele de buclă per-iterație (Go 1.22) se aplică doar pachetelor din module cu `go 1.22` sau mai nou. Multe schimbări de comportament din biblioteca standard sunt legate de această versiune prin setări `GODEBUG` implicite.

## Constrângeri de build

Un fișier poate fi inclus sau exclus după sistem, arhitectură sau tag-uri:

```go,ignore
//go:build linux && (amd64 || arm64) && !nocgo

package sistem
```

Și sufixele de nume aplică automat constrângeri: `fisier_linux.go`, `fisier_windows_amd64.go`, `fisier_test.go`. Tag-urile proprii se activează cu `go build -tags=integrare`. Tag-ul `goexperiment.X` reflectă experimentele active ale toolchain-ului.

## Unitatea de compilare: pachetul

Compilatorul (`go tool compile`) primește **toate fișierele unui pachet** odată și produce un fișier obiect cu **datele de export**: tipurile, funcțiile inlinabile și informațiile de escape ale API-ului exportat. Pachetele care îl importă citesc doar aceste date, nu sursele. De aceea:

- pachetele se compilează în paralel, în ordinea topologică a importurilor;
- ciclurile de import sunt imposibile prin construcție;
- o funcție dintr-un alt pachet poate fi inlined (corpul ei e în datele de export).

Rezultatele sunt păstrate în **build cache** (`go env GOCACHE`), indexate după hash-ul tuturor intrărilor (surse, flag-uri, versiunea toolchain-ului, dependențe). Un build repetat fără schimbări nu recompilează nimic; `go clean -cache` îl golește.

## Linker-ul și codul mort

Linker-ul pornește de la `main.main` și funcțiile `init` și include doar simbolurile **accesibile**. Funcțiile nefolosite dispar din binar. Excepții care țin cod în viață:

- când programul folosește `reflect.Value.Method`/`MethodByName` cu nume necunoscute la compilare, linker-ul trebuie să păstreze **toate** metodele exportate ale tuturor tipurilor accesibile (de aceea `text/template` poate umfla binarele);
- funcțiile `init` și variabilele globale inițializate de ele.

```bash
go build -ldflags='-s -w'   # fără tabelă de simboluri și DWARF: binar mai mic
go version -m ./app         # modulele și versiunile incluse în binar
```

## `//go:linkname`

Directiva permite accesul la simboluri neexportate din alt pachet (de exemplu din runtime). Multe biblioteci populare o foloseau, blocând schimbările interne ale runtime-ului. Din Go 1.23, linker-ul **refuză** referințele `linkname` spre simboluri din biblioteca standard care nu sunt marcate explicit ca permise (`-checklinkname=0` dezactivează temporar verificarea). E un semnal clar: nu construi pe interiorul runtime-ului.

## `retract`

Un autor își poate retrage o versiune publicată greșit, fără s-o șteargă (proxy-ul e imutabil):

```text
retract (
	v1.4.0 // conține o regresie gravă în Parse
	[v1.5.0, v1.5.3]
)
```

`go get @latest` sare peste versiunile retrase, iar `go list -m -u` le semnalează celor care le folosesc deja.

## Pe scurt

- MVS: maximul versiunilor minime; reproductibil fără lock file.
- Proxy imutabil + checksum database transparent = lanț de aprovizionare verificabil.
- Linia `go` e o cerință și controlează semantica; `GOTOOLCHAIN` descarcă versiunea potrivită.
- Pachetul e unitatea de compilare; datele de export permit inlining între pachete.
- Linker-ul elimină codul inaccesibil; `linkname` spre runtime e restricționat din 1.23.
