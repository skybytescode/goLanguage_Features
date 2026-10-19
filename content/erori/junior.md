# Erori, defer, panic și recover — Junior

Go nu are excepții. O funcție care poate eșua întoarce o **eroare** ca valoare obișnuită, iar apelantul o verifică explicit. Pe lângă asta, `defer` programează cod de curățare, iar `panic` oprește programul în situații cu adevărat neașteptate.

## Tipul `error`

`error` e o interfață cu o singură metodă:

```go,ignore
type error interface {
	Error() string
}
```

O funcție care poate eșua întoarce, prin convenție, eroarea ca **ultim** rezultat. `nil` înseamnă „fără eroare”.

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	n, err := strconv.Atoi("12a")
	if err != nil {
		fmt.Println("conversie eșuată:", err)
		return
	}
	fmt.Println("număr:", n)
}
```

`if err != nil` e cel mai frecvent tipar din Go. E repetitiv, dar face vizibil fiecare loc în care ceva poate eșua.

## Crearea erorilor

```go
package main

import (
	"errors"
	"fmt"
)

func retrage(sold, suma int) (int, error) {
	if suma <= 0 {
		return sold, errors.New("suma trebuie să fie pozitivă")
	}
	if suma > sold {
		return sold, fmt.Errorf("fonduri insuficiente: ai %d, ceri %d", sold, suma)
	}
	return sold - suma, nil
}

func main() {
	sold, err := retrage(100, 150)
	if err != nil {
		fmt.Println("eroare:", err)
	}
	fmt.Println("sold:", sold)
}
```

- `errors.New` pentru un mesaj fix.
- `fmt.Errorf` pentru un mesaj cu valori.

Mesajele de eroare încep cu literă mică și nu se termină cu punct: ele ajung de obicei în mijlocul altor mesaje.

## Nu ignora erorile

O eroare ignorată ascunde problema și o mută mai departe, unde e greu de găsit. Dacă ignori intenționat o eroare, fă asta vizibil cu `_`:

```go,ignore
_ = fisier.Close() // știm că eroarea nu ne interesează aici
```

## `defer`: rulează la ieșirea din funcție

`defer` amână un apel până când funcția curentă se termină, indiferent pe ce drum iese. E locul firesc pentru curățare: închiderea fișierelor, deblocarea mutex-urilor.

```go
package main

import "fmt"

func main() {
	fmt.Println("început")
	defer fmt.Println("curățare") // rulează ultima
	fmt.Println("lucru")
}
```

Mai multe `defer`-uri rulează în ordine **inversă** (ultimul intrat, primul ieșit):

```go
package main

import "fmt"

func main() {
	for i := range 3 {
		defer fmt.Println("defer", i)
	}
	fmt.Println("gata")
	// gata, defer 2, defer 1, defer 0
}
```

Tiparul tipic: deschizi o resursă, verifici eroarea, apoi imediat programezi închiderea:

```go,ignore
f, err := os.Open("date.txt")
if err != nil {
	return err
}
defer f.Close()
// folosești f...
```

## `panic`

`panic` oprește execuția normală. Funcțiile `defer` încă rulează, apoi programul se termină cu un mesaj și un stack trace. Runtime-ul însuși dă panic la erori de programare: index în afara limitelor, pointer `nil`, împărțire întreagă la zero.

```go,panic
package main

import "fmt"

func main() {
	defer fmt.Println("defer-ul rulează și la panic")
	s := []int{1, 2, 3}
	i := 5
	fmt.Println(s[i]) // panic: runtime error: index out of range [5] with length 3
}
```

Regula: **erorile** sunt pentru situații așteptate (fișier lipsă, input invalid, rețea căzută). **panic** e pentru bug-uri și situații imposibile.

## `recover`

`recover`, apelat într-o funcție `defer`, oprește un panic și întoarce valoarea lui. Programul continuă după funcția care a dat panic:

```go
package main

import "fmt"

func sigur(f func()) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("am prins:", r)
		}
	}()
	f()
}

func main() {
	sigur(func() { panic("ceva grav") })
	fmt.Println("programul continuă")
}
```

Îl vei folosi rar: de exemplu, un server HTTP îl folosește ca o cerere care dă panic să nu oprească tot serverul.

## Pe scurt

- Erorile sunt valori; verifică-le cu `if err != nil`.
- `errors.New` și `fmt.Errorf` creează erori.
- `defer` rulează la ieșirea din funcție, în ordine inversă.
- `panic` pentru bug-uri, nu pentru erori obișnuite.
- `recover` oprește un panic, doar dintr-un `defer`.
