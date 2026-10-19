# Go Language Features

Site web despre funcționalitățile limbajului Go, organizat pe 4 niveluri de adâncime:

| Nivel | Nume | Întrebarea la care răspunde |
|---|---|---|
| 1 | Fundamente | Ce este și cum îl folosesc |
| 2 | Practician | Cum îl folosesc corect |
| 3 | Avansat | De ce funcționează așa |
| 4 | Expert | Cum funcționează pe dinăuntru |

Fiecare dintre cele 25 de subiecte acoperă toate cele 4 niveluri. Site-ul e disponibil în română și în engleză (butonul **EN / RO** din bara laterală).

## Tech stack

- Un generator de site static scris în Go: [goldmark](https://github.com/yuin/goldmark) pentru Markdown și [chroma](https://github.com/alecthomas/chroma) pentru colorarea codului
- HTML static + puțin JavaScript (căutare și copiere), fără framework
- GitHub Actions + GitHub Pages pentru publicare

## Rulare locală

```bash
go run . -serve :8080          # http://localhost:8080/
go test ./...                  # compilează și rulează fiecare exemplu din conținut
```

## Conținut

Textele sunt în `content/<subiect>/<nivel>.md` (română) și `content/<subiect>/<nivel>.en.md` (engleză), cu `<nivel>` = `1`, `2`, `3` sau `4`. Lista subiectelor și titlurile lor sunt în `content.go`, iar textele interfeței în `i18n.go`.

URL-urile în română au forma `/<subiect>/<nivel>/`, iar cele în engleză `/en/<subiect>/<nivel>/`. Adresele din prima versiune (`junior`, `middle`, `senior`) redirecționează spre nivelurile 1, 2 și 3.

Blocurile de cod ` ```go ` sunt programe complete. Testele (`examples_test.go`) verifică pentru fiecare că e formatat cu gofmt, că se compilează și că rulează fără eroare. Variante:

- ` ```go,compile_fail ` marchează un exemplu care nu compilează intenționat (testul verifică asta);
- ` ```go,panic ` marchează un exemplu care eșuează intenționat la rulare (panic sau deadlock);
- ` ```go,ignore ` e un fragment: nu e verificat și nu are buton de rulare.

Butonul „Rulează” deschide exemplul în [Go Playground](https://go.dev/play). Playground-ul nu primește codul în URL, așa că ID-urile sunt păstrate în `playground.json`. După ce adaugi sau modifici exemple:

```bash
go run . -fmt      # formatează exemplele cu gofmt
go run . -share    # trimite exemplele noi în Playground și actualizează playground.json
```
