# Control flow — Senior

Ce face exact `range` cu expresia pe care o parcurge, cum compilează Go un `switch`, ce costă variabilele de buclă per-iterație și cum ajuți compilatorul să elimine verificările de limite.

## `range` evaluează expresia o singură dată

Expresia de după `range` e evaluată **o singură dată**, înainte de prima iterație. Pentru slices, lungimea e fixată atunci; adăugările din buclă nu prelungesc parcurgerea:

```go
package main

import "fmt"

func main() {
	s := []int{1, 2, 3}
	for i, v := range s {
		if i == 0 {
			s = append(s, 99) // s indică acum alt array, mai lung
		}
		fmt.Print(v, " ")
	}
	fmt.Println("\nlen:", len(s)) // bucla a făcut 3 pași, dar len e 4
}
```

Pentru un **array** (nu slice), `range` parcurge o **copie** a lui. Modificările din buclă nu se văd în valorile primite:

```go
package main

import "fmt"

func main() {
	arr := [3]int{1, 2, 3}
	for i, v := range arr { // copie a array-ului
		arr[2] = 100
		fmt.Print(i, ":", v, " ") // 0:1 1:2 2:3
	}
	fmt.Println()
	for i, v := range &arr { // pointer: fără copie
		arr[2] = 200
		fmt.Print(i, ":", v, " ") // 0:1 1:2 2:200
	}
	fmt.Println()
}
```

Valoarea `v` e întotdeauna o copie a elementului. Pentru a modifica elementele unui slice de structuri, folosește indexul: `s[i].Camp = x`. Pentru array-uri mari de structuri, `for i := range s` evită și copierea fiecărui element.

## Ordinea în maps e aleatoare intenționat

Runtime-ul alege un punct de start aleatoriu la fiecare `range` peste un map, ca nimeni să nu depindă de o ordine. Ștergerea unei chei în timpul parcurgerii e sigură; o cheie adăugată în timpul parcurgerii **poate** să apară sau nu.

```go
package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	for _, k := range slices.Sorted(maps.Keys(m)) { // ordine stabilă
		fmt.Println(k, m[k])
	}
}
```

## `range` peste string decodează UTF-8

Pașii nu sunt de câte un octet, ci de câte un caracter. Octeții invalizi produc `U+FFFD` și avansează cu un singur octet:

```go
package main

import "fmt"

func main() {
	for i, r := range "aș\xffb" {
		fmt.Printf("%d:%q ", i, r) // 0:'a' 1:'ș' 3:'�' 4:'b'
	}
	fmt.Println()
}
```

## Cum compilează `switch`

Ramurile sunt evaluate **de sus în jos și de la stânga la dreapta**, până la prima potrivire. Expresiile ne-constante din `case` se evaluează doar dacă se ajunge la ele, deci pot avea efecte secundare doar când e nevoie.

Când toate ramurile sunt constante întregi sau string-uri, compilatorul nu mai testează pe rând: sortează valorile și generează o **căutare binară**, iar pentru întregi denși, pe amd64 și arm64, o **tabelă de salt** (din Go 1.19). Un `switch` cu multe ramuri constante nu e mai lent decât un map; de obicei e mai rapid.

```go
package main

import "fmt"

func clasa(c byte) string {
	switch c { // ramuri constante: tabelă de salt / căutare binară
	case 'a', 'e', 'i', 'o', 'u':
		return "vocală"
	case ' ', '\t', '\n':
		return "spațiu"
	}
	return "altceva"
}

func main() {
	for _, c := range []byte("ai x") {
		fmt.Print(clasa(c), " ")
	}
	fmt.Println()
}
```

## Ce costă variabilele per-iterație

Schimbarea din Go 1.22 nu înseamnă o alocare la fiecare pas. Compilatorul creează o variabilă nouă per iterație doar **semantic**; dacă variabila nu e capturată de un closure și nu „scapă” (escape analysis), rămâne în registru, exact ca înainte.

Pentru forma cu trei clauze (`for i := 0; i < n; i++`), la sfârșitul fiecărei iterații variabila e copiată într-una nouă **înainte** de `i++`. Asta înseamnă că un closure capturat într-o iterație nu vede incrementarea:

```go
package main

import "fmt"

func main() {
	var ps []*int
	for i := 0; i < 3; i++ {
		ps = append(ps, &i) // adrese diferite din Go 1.22
	}
	for _, p := range ps {
		fmt.Print(*p, " ") // 0 1 2
	}
	fmt.Println()
}
```

Instrumentul `bisect` (`golang.org/x/tools/cmd/bisect`) poate găsi exact bucla al cărei comportament s-a schimbat la migrarea unui modul vechi la `go 1.22`.

## Eliminarea verificărilor de limite

Fiecare `s[i]` are o verificare de limite (bounds check) care dă panic la depășire. Compilatorul le elimină când poate demonstra că indexul e valid; `for i := range s` e cazul ideal. Poți ajuta cu o verificare explicită la început:

```go
package main

import "fmt"

func suma4(s []int) int {
	_ = s[3] // o singură verificare; cele patru de mai jos sunt eliminate
	return s[0] + s[1] + s[2] + s[3]
}

func main() {
	fmt.Println(suma4([]int{1, 2, 3, 4}))
}
```

Verificările rămase se văd cu:

```bash
go build -gcflags='-d=ssa/check_bce/debug=1' .
```

## `defer` în bucle

`defer` rulează la ieșirea din **funcție**, nu din iterație. Într-o buclă care deschide fișiere, toate rămân deschise până la final:

```go,ignore
for _, nume := range fisiere {
	f, err := os.Open(nume)
	if err != nil {
		return err
	}
	defer f.Close() // se acumulează: toate se închid abia la return
	// ...
}
```

Soluția e să muți corpul buclei într-o funcție separată, unde `defer` rulează la fiecare apel.

## Pe scurt

- `range` evaluează expresia o dată; array-urile sunt copiate, slices nu.
- Ordinea în maps e aleatoare intenționat; sortează cheile când contează.
- `switch` cu ramuri constante devine căutare binară sau tabelă de salt.
- Variabilele per-iterație nu costă nimic dacă nu scapă din buclă.
- `for i := range s` și verificările „hint” elimină bounds checks.
