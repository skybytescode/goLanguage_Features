# JSON și encoding — Junior

Pachetul `encoding/json` transformă structurile Go în JSON (**marshal**) și invers (**unmarshal**). E folosit în aproape orice API web scris în Go.

## Din Go în JSON: `json.Marshal`

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Utilizator struct {
	Nume   string
	Varsta int
	Activ  bool
}

func main() {
	u := Utilizator{Nume: "Ana", Varsta: 30, Activ: true}
	date, err := json.Marshal(u)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(date)) // {"Nume":"Ana","Varsta":30,"Activ":true}
}
```

`json.Marshal` întoarce `[]byte`. Doar câmpurile **exportate** (cu literă mare) apar în JSON.

## Tag-uri: numele din JSON

De obicei vrei nume în stilul JSON (`nume`, `created_at`). Le setezi cu **tag-uri**:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Produs struct {
	ID       int      `json:"id"`
	Nume     string   `json:"nume"`
	Pret     float64  `json:"pret"`
	Etichete []string `json:"etichete"`
	Intern   string   `json:"-"` // niciodată în JSON
}

func main() {
	p := Produs{ID: 7, Nume: "Carte", Pret: 49.9, Etichete: []string{"nou"}, Intern: "secret"}
	date, _ := json.MarshalIndent(p, "", "  ") // formatat frumos
	fmt.Println(string(date))
}
```

## Din JSON în Go: `json.Unmarshal`

Îi dai octeții și un **pointer** la variabila în care să scrie:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Comanda struct {
	ID      int      `json:"id"`
	Client  string   `json:"client"`
	Produse []string `json:"produse"`
	Platita bool     `json:"platita"`
}

func main() {
	text := `{"id": 101, "client": "Ion", "produse": ["pâine", "lapte"], "platita": true, "extra": 1}`

	var c Comanda
	if err := json.Unmarshal([]byte(text), &c); err != nil {
		fmt.Println("JSON invalid:", err)
		return
	}
	fmt.Printf("%+v\n", c)
}
```

Câmpurile din JSON care nu există în structură (`extra`) sunt **ignorate**. Câmpurile din structură care lipsesc din JSON rămân la valoarea zero.

## Structuri imbricate și liste

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Adresa struct {
	Oras   string `json:"oras"`
	Strada string `json:"strada"`
}

type Persoana struct {
	Nume      string   `json:"nume"`
	Adresa    Adresa   `json:"adresa"`
	Telefoane []string `json:"telefoane"`
}

func main() {
	text := `[
		{"nume": "Ana", "adresa": {"oras": "Cluj", "strada": "Unirii"}, "telefoane": ["0711"]},
		{"nume": "Ion", "adresa": {"oras": "Iași", "strada": "Păcurari"}}
	]`
	var persoane []Persoana
	if err := json.Unmarshal([]byte(text), &persoane); err != nil {
		fmt.Println(err)
		return
	}
	for _, p := range persoane {
		fmt.Println(p.Nume, "din", p.Adresa.Oras, p.Telefoane)
	}
}
```

## JSON necunoscut: `map[string]any`

Dacă nu știi structura dinainte, poți decoda într-un map. Numerele devin `float64`, obiectele `map[string]any`, listele `[]any`:

```go
package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var date map[string]any
	json.Unmarshal([]byte(`{"nume": "Go", "versiune": 1.26, "tags": ["rapid", "simplu"]}`), &date)

	fmt.Println(date["nume"])
	fmt.Printf("%T\n", date["versiune"]) // float64
	for _, t := range date["tags"].([]any) {
		fmt.Println("-", t)
	}
}
```

Codul devine plin de type assertions; preferă structurile când știi forma datelor.

## Tipuri Go și tipuri JSON

| Go | JSON |
|---|---|
| `bool` | `true` / `false` |
| numere (`int`, `float64`…) | număr |
| `string` | string |
| `[]T`, array | listă (`[]`); slice `nil` → `null` |
| `struct`, `map[string]T` | obiect (`{}`) |
| pointer `nil`, interfață `nil` | `null` |
| `time.Time` | string RFC 3339 (`"2026-10-19T12:00:00Z"`) |
| `[]byte` | string base64 |

## Pe scurt

- `json.Marshal` / `json.MarshalIndent` din Go în JSON.
- `json.Unmarshal(date, &v)` din JSON în Go; dă-i un pointer.
- Doar câmpurile exportate sunt serializate; tag-urile dau numele.
- `json:"-"` exclude un câmp.
- `map[string]any` pentru JSON necunoscut, dar preferă structurile.
