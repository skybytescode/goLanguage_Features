# Stringuri, bytes și runes — Senior

Reprezentarea internă a stringurilor, conversiile pe care compilatorul le face fără copiere, conversiile zero-copy cu `unsafe`, internarea cu `unique` și scurgerile de memorie prin sub-stringuri.

## Antetul unui string

Un string e o structură de două cuvinte, fără capacitate:

```go,ignore
type stringStruct struct {
	str unsafe.Pointer // octeții (nemodificabili)
	len int
}
```

Pentru că octeții nu se schimbă niciodată, stringurile pot **împărți** memoria fără grijă: `s[i:j]` creează doar un antet nou, iar atribuirea copiază două cuvinte. Literalele stau în segmentul read-only al binarului.

## Sub-stringurile țin în viață originalul

`s[i:j]` indică în octeții lui `s`. Dacă păstrezi un sub-string mic dintr-un string uriaș (o linie dintr-un fișier de 100 MB citit integral), tot fișierul rămâne în memorie:

```go
package main

import (
	"fmt"
	"strings"
)

var index = map[string]bool{}

func proceseaza(continut string) {
	for linie := range strings.Lines(continut) {
		id, _, _ := strings.Cut(linie, ",")
		index[strings.Clone(id)] = true // fără Clone, fiecare cheie ar ține tot continut
	}
}

func main() {
	proceseaza("a1,x\nb2,y\n")
	fmt.Println(len(index))
}
```

`strings.Clone` (Go 1.20) face o copie exactă, ca originalul să poată fi eliberat.

## Conversii fără copiere, făcute de compilator

`string(b)` și `[]byte(s)` copiază în general. Compilatorul elimină copia când poate dovedi că rezultatul nu e păstrat:

| Cod | Copiază? |
|---|---|
| `m[string(b)]` (căutare în map) | nu |
| `string(b) == "abc"`, `string(a) < string(b)` | nu |
| `"<" + string(b) + ">"` | doar rezultatul final |
| `for i, c := range []byte(s)` | nu |
| `switch string(b) { case ... }` | nu |
| `x := string(b)` păstrat | da |

În plus, conversiile de stringuri mici (sub 32 de octeți) care nu scapă din funcție folosesc un buffer de pe stivă, fără alocare pe heap.

## Zero-copy cu `unsafe`

Când profilarea arată că o conversie contează, `unsafe.String` și `unsafe.StringData` (Go 1.20) permit conversii fără copie. Contractul: octeții **nu au voie** să se mai schimbe cât timp stringul există.

```go
package main

import (
	"fmt"
	"unsafe"
)

func bytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func main() {
	b := []byte("salut")
	s := bytesToString(b)
	fmt.Println(s)
	b[0] = 'S'     // ÎNCALCĂ contractul: s se schimbă și el
	fmt.Println(s) // Salut — un string „imutabil” care s-a modificat
}
```

Varianta inversă, `unsafe.Slice(unsafe.StringData(s), len(s))`, produce un `[]byte` în care **nu ai voie să scrii**: memoria unui literal e read-only, iar scrierea oprește programul cu segmentation fault.

## Concatenarea

`a + b + c` e compilat într-un singur apel `runtime.concatstrings`, care calculează lungimea totală și alocă o singură dată. Problema apare doar în bucle, unde fiecare iterație face o concatenare separată; de aici recomandarea pentru `strings.Builder`.

`strings.Builder` evită și copia finală: `String()` întoarce un string construit direct peste buffer-ul intern, cu `unsafe.String`. De aceea builder-ul nu are voie să fie copiat după prima scriere (are o verificare la rulare).

## Internare: pachetul `unique` (Go 1.23)

Când ai multe copii ale acelorași valori (nume de țări, tag-uri, chei de cache), `unique.Make` păstrează o singură copie canonică și întoarce un `Handle[T]`. Compararea a două handle-uri e o comparație de pointeri:

```go
package main

import (
	"fmt"
	"strings"
	"unique"
)

func main() {
	a := unique.Make(strings.Repeat("ro", 2))
	b := unique.Make("roro")
	fmt.Println(a == b, a.Value()) // true roro: aceeași valoare canonică
}
```

Valorile canonice nefolosite sunt eliberate automat (pachetul folosește pointeri slabi intern), deci nu ai o scurgere de memorie ca la un map global.

## UTF-8 pe dinăuntru

| Cod Unicode | Octeți | Format |
|---|---|---|
| U+0000 – U+007F | 1 | `0xxxxxxx` |
| U+0080 – U+07FF | 2 | `110xxxxx 10xxxxxx` |
| U+0800 – U+FFFF | 3 | `1110xxxx 10xxxxxx 10xxxxxx` |
| U+10000 – U+10FFFF | 4 | `11110xxx 10xxxxxx 10xxxxxx 10xxxxxx` |

Diacriticele românești (U+0102–U+021B) ocupă 2 octeți. Proprietăți importante: un octet ASCII nu apare niciodată în interiorul unui caracter multi-octet, deci căutarea unui separator ASCII (`,`, `\n`) direct în octeți e sigură; iar octeții de continuare (`10xxxxxx`) permit găsirea începutului unui caracter din orice poziție.

Go nu validează UTF-8 la crearea unui string. Secvențele invalide devin `U+FFFD` doar când sunt **decodate** (`range`, `[]rune(s)`, `utf8.DecodeRune`); `strings.ToValidUTF8` le înlocuiește explicit.

## `[]rune` costă

`[]rune(s)` alocă 4 octeți per caracter. Pentru a număra caractere, `utf8.RuneCountInString` nu alocă nimic; pentru a parcurge, `range` decodează din mers. Rezervă `[]rune` pentru cazurile în care ai nevoie de acces aleatoriu după poziția caracterului.

## Pe scurt

- String = (pointer, lungime), imutabil; sub-stringurile împart memoria.
- `strings.Clone` rupe legătura cu un original mare.
- Compilatorul evită copia la căutări în map, comparații și `range`.
- `unsafe.String` e zero-copy, dar cere ca octeții să nu se mai schimbe.
- `unique.Make` internează valori repetate; `[]rune` alocă 4 octeți per caracter.
