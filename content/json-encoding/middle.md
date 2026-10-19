# JSON și encoding — Middle

Controlul fin al serializării: `omitempty` vs. `omitzero`, metode `MarshalJSON`/`UnmarshalJSON`, decodare în flux cu `Decoder`, validare strictă, `json.RawMessage` pentru JSON polimorf și celelalte encodări din biblioteca standard.

## `omitempty` și `omitzero` (Go 1.24)

`omitempty` omite câmpul dacă valoarea e „goală”: `false`, `0`, `""`, `nil`, slice/map de lungime 0. Problema: **structurile** nu sunt niciodată considerate goale, deci un `time.Time` zero apare mereu. `omitzero` omite câmpul dacă e **valoarea zero** a tipului (sau dacă metoda `IsZero()` întoarce `true`):

```go
package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Eveniment struct {
	Nume       string    `json:"nume"`
	Note       []string  `json:"note,omitempty"`
	Creat      time.Time `json:"creat,omitempty"` // NU e omis: structură
	Modificat  time.Time `json:"modificat,omitzero"`
	Prioritate int       `json:"prioritate,omitzero"`
}

func main() {
	b, _ := json.Marshal(Eveniment{Nume: "lansare"})
	fmt.Println(string(b))
	// {"nume":"lansare","creat":"0001-01-01T00:00:00Z"}
}
```

Atenție: cu `omitempty` sau `omitzero`, un `0` real și „lipsă” devin imposibil de deosebit. Dacă diferența contează (de exemplu la un PATCH), folosește un pointer (`*int`).

## Tipuri proprii: `MarshalJSON` și `UnmarshalJSON`

Un tip își poate controla complet reprezentarea:

```go
package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Culoare int

const (
	Rosu Culoare = iota
	Verde
)

var nume = []string{"rosu", "verde"}

func (c Culoare) MarshalJSON() ([]byte, error) {
	return json.Marshal(nume[c])
}

func (c *Culoare) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	for i, n := range nume {
		if strings.EqualFold(n, s) {
			*c = Culoare(i)
			return nil
		}
	}
	return fmt.Errorf("culoare necunoscută %q", s)
}

type Masina struct {
	Model   string  `json:"model"`
	Culoare Culoare `json:"culoare"`
}

func main() {
	b, _ := json.Marshal(Masina{"Dacia", Verde})
	fmt.Println(string(b)) // {"model":"Dacia","culoare":"verde"}

	var m Masina
	err := json.Unmarshal([]byte(`{"model":"Logan","culoare":"ROSU"}`), &m)
	fmt.Println(m, err)
}
```

`UnmarshalJSON` are receptor **pointer** (modifică valoarea); `MarshalJSON` are de obicei receptor valoare, ca să funcționeze și pentru valori, nu doar pentru pointeri. Pentru tipuri care sunt „text” (inclusiv chei de map), implementează `encoding.TextMarshaler`/`TextUnmarshaler` în loc.

## Decodare în flux: `json.Decoder`

Când datele vin de la un `io.Reader` (corpul unei cereri HTTP, un fișier), `Decoder` citește direct din el, fără să încarce totul în memorie, și poate procesa mai multe valori succesive:

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Log struct {
	Nivel string `json:"nivel"`
	Mesaj string `json:"mesaj"`
}

func main() {
	// JSON Lines: câte un obiect pe linie
	flux := strings.NewReader(`{"nivel":"info","mesaj":"pornit"}
{"nivel":"eroare","mesaj":"disc plin"}
{"nivel":"info","mesaj":"oprit"}`)

	dec := json.NewDecoder(flux)
	for {
		var l Log
		err := dec.Decode(&l)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		if l.Nivel == "eroare" {
			fmt.Println("!", l.Mesaj)
		}
	}
}
```

## Validare strictă

Implicit, câmpurile necunoscute sunt ignorate în tăcere: o greșeală de tipar în numele unui câmp trece neobservată. Pentru API-uri și configurații, fii strict:

```go
package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Config struct {
	Port int `json:"port"`
}

func main() {
	dec := json.NewDecoder(strings.NewReader(`{"port": 80, "prot": 443}`))
	dec.DisallowUnknownFields()
	var c Config
	fmt.Println(dec.Decode(&c)) // json: unknown field "prot"
}
```

Validarea semantică (porturi în interval, câmpuri obligatorii) rămâne în sarcina ta, după decodare.

## `json.RawMessage`: decodare amânată

Pentru mesaje polimorfe, cu un câmp care spune tipul restului, decodezi întâi „plicul” și păstrezi partea variabilă ca octeți bruți:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Plic struct {
	Tip  string          `json:"tip"`
	Date json.RawMessage `json:"date"`
}

type Plata struct{ Suma float64 }
type Livrare struct{ Adresa string }

func main() {
	mesaje := []string{
		`{"tip":"plata","date":{"Suma":99.5}}`,
		`{"tip":"livrare","date":{"Adresa":"Str. Lungă 1"}}`,
	}
	for _, m := range mesaje {
		var p Plic
		json.Unmarshal([]byte(m), &p)
		switch p.Tip {
		case "plata":
			var pl Plata
			json.Unmarshal(p.Date, &pl)
			fmt.Println("plată:", pl.Suma)
		case "livrare":
			var l Livrare
			json.Unmarshal(p.Date, &l)
			fmt.Println("livrare la:", l.Adresa)
		}
	}
}
```

## Numere mari: `json.Number`

Decodate în `any`, numerele devin `float64`, care pierde precizia peste 2⁵³. ID-urile mari (de exemplu de pe Twitter/X) se corup. `dec.UseNumber()` le păstrează ca string (`json.Number`), iar opțiunea de tag `,string` le serializează ca string JSON.

## Alte encodări din biblioteca standard

| Pachet | Pentru |
|---|---|
| `encoding/csv` | fișiere CSV (cu ghilimele, separatori, linii multiple) |
| `encoding/xml` | XML, cu tag-uri `xml:"..."` similare |
| `encoding/base64`, `encoding/hex` | date binare ca text |
| `encoding/binary` | numere în format binar, cu ordinea octeților aleasă |
| `encoding/gob` | serializare Go↔Go, eficientă, cu tipuri |
| `encoding/pem` | blocuri PEM (chei, certificate) |

## Pe scurt

- `omitzero` (Go 1.24) omite și structurile zero, ca `time.Time{}`.
- `MarshalJSON`/`UnmarshalJSON` sau `TextMarshaler` pentru reprezentări proprii.
- `json.Decoder` pentru fluxuri; `DisallowUnknownFields` pentru strictețe.
- `json.RawMessage` amână decodarea părților polimorfe.
- `UseNumber` sau `,string` pentru numere mari.
