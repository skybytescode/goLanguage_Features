# JSON și encoding — Senior

Cum funcționează `encoding/json` pe dinăuntru (reflection și cache de encodere), unde se duce timpul și memoria, capcanele de securitate ale comportamentului implicit și ce aduce `encoding/json/v2`.

## Reflection și cache-ul de encodere

La primul `Marshal` al unui tip, pachetul îl inspectează prin `reflect`: câmpurile, tag-urile, metodele `MarshalJSON`/`MarshalText`, câmpurile încorporate. Rezultatul e o **funcție de encodare** specifică tipului, păstrată într-un `sync.Map` global. Apelurile următoare pentru același tip sar peste analiză.

Pentru structuri, analiza produce lista câmpurilor cu numele deja escapat, indexul de acces (inclusiv prin structuri încorporate) și funcția de encodare a tipului fiecărui câmp. La decodare, căutarea câmpului după numele din JSON folosește un map, plus o comparație case-insensitive ca fallback.

## Unde se duce timpul

- **Reflection** la fiecare câmp (`reflect.Value.Field(i)`, interfețe): costul principal față de cod generat.
- **Alocări**: `Marshal` lucrează într-un buffer intern refolosit prin `sync.Pool`, dar rezultatul final e mereu o alocare nouă; `Unmarshal` alocă stringuri pentru fiecare valoare string și, pentru `any`, map-uri și slice-uri.
- **Validarea în doi pași**: `Unmarshal` scanează întâi tot input-ul ca să verifice că e JSON valid, apoi îl decodează.
- **Metode proprii**: `MarshalJSON` întoarce `[]byte` care trebuie apoi **revalidat și compactat** de pachet; o metodă proprie poate fi mai lentă decât encodarea implicită.

Când JSON-ul e pe calea critică, opțiunile sunt: `json.Decoder` cu tipuri concrete (nu `any`), refolosirea structurilor de destinație, generatoare de cod (easyjson, ffjson) sau `encoding/json/v2`.

## Capcane de securitate ale comportamentului implicit

**Potrivirea numelor e case-insensitive.** `{"ADMIN": true}` populează câmpul cu tag-ul `json:"admin"`. Dacă un proxy validează JSON-ul cu reguli case-sensitive (`"admin"` interzis) și transmite cererea mai departe unui serviciu Go, verificarea poate fi ocolită.

**Cheile duplicate sunt acceptate**, iar ultima câștigă:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Cerere struct {
	Rol string `json:"rol"`
}

func main() {
	var c Cerere
	json.Unmarshal([]byte(`{"rol": "user", "ROL": "admin"}`), &c)
	fmt.Println(c.Rol) // admin: duplicat + potrivire case-insensitive
}
```

Diferite parsere (în limbaje diferite) pot alege valori diferite pentru același document — o sursă cunoscută de vulnerabilități de tip „parser differential”.

**UTF-8 invalid** e înlocuit în tăcere cu `U+FFFD` la decodare, în loc să fie respins.

**Escaparea HTML**: implicit, `<`, `>` și `&` sunt scrise ca `<` etc., ca JSON-ul să poată fi inclus în pagini HTML. Pentru API-uri pure, `enc.SetEscapeHTML(false)` dă un output mai lizibil.

## Unmarshal în structuri existente

`Unmarshal` **nu** resetează destinația: câmpurile absente din JSON își păstrează valoarea anterioară, map-urile existente primesc chei noi (nu sunt înlocuite), iar slice-urile sunt resetate la lungime 0 și refolosite. Refolosirea unei structuri pentru mai multe documente poate amesteca date între ele:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Profil struct {
	Nume  string            `json:"nume"`
	Email string            `json:"email"`
	Meta  map[string]string `json:"meta"`
}

func main() {
	var p Profil
	json.Unmarshal([]byte(`{"nume":"Ana","email":"ana@x.ro","meta":{"a":"1"}}`), &p)
	json.Unmarshal([]byte(`{"nume":"Ion","meta":{"b":"2"}}`), &p)
	fmt.Printf("%+v\n", p) // email-ul Anei și meta[a] au rămas
}
```

## `encoding/json/v2` (experimental din Go 1.25)

Cu `GOEXPERIMENT=jsonv2`, sunt disponibile pachetele `encoding/json/v2` (API semantic) și `encoding/json/jsontext` (tokenizer/encoder sintactic, la nivel de flux). Cu experimentul activ, chiar și `encoding/json` (v1) e reimplementat peste v2, păstrând comportamentul vechi prin opțiuni.

Diferențele principale în v2, implicit:

| Comportament | v1 | v2 |
|---|---|---|
| potrivirea numelor | case-insensitive | case-sensitive |
| chei duplicate | ultima câștigă | eroare |
| UTF-8 invalid | înlocuit | eroare |
| slice/map `nil` | `null` | `[]` / `{}` |
| `omitempty` | după valoarea Go | după valoarea JSON goală |
| performanță la decodare | — | semnificativ mai rapidă, decodare într-un singur pas |

v2 adaugă și opțiuni de tag (`format:` pentru timp și durate, `inline`, `unknown`), funcții `MarshalWrite`/`UnmarshalRead` pentru fluxuri și metode `MarshalJSONTo`/`UnmarshalJSONFrom` care scriu direct într-un `jsontext.Encoder`, fără `[]byte` intermediar.

Recomandarea pentru cod nou care are nevoie de strictețe azi: `Decoder` cu `DisallowUnknownFields`, validare explicită a cheilor duplicate unde contează, și pregătirea pentru v2 prin tag-uri cu nume exacte.

## `encoding/gob` vs. JSON

`gob` transmite întâi descrierea tipului, apoi valorile în format binar; pentru fluxuri lungi între programe Go e mai compact și mai rapid decât JSON. Dar e legat de Go, nu e un format stabil pentru stocare pe termen lung și nu e potrivit pentru date nesigure (decodarea poate consuma multă memorie). Pentru comunicare între limbaje, Protocol Buffers (`google.golang.org/protobuf`) e alegerea uzuală.

## Pe scurt

- Encoderele sunt construite prin reflection o dată per tip și păstrate în cache.
- Costul vine din reflection, alocări și validarea în doi pași.
- v1 e permisiv: nume case-insensitive, chei duplicate, UTF-8 invalid acceptat.
- `Unmarshal` nu resetează destinația.
- `encoding/json/v2` (experiment) e strict, mai rapid și are API de flux.
