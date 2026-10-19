# JSON and encoding — Senior

How `encoding/json` works under the hood (reflection and an encoder cache), where time and memory go, the security pitfalls of the default behavior, and what `encoding/json/v2` brings.

## Reflection and the encoder cache

On the first `Marshal` of a type, the package inspects it through `reflect`: the fields, the tags, `MarshalJSON`/`MarshalText` methods, embedded fields. The result is a type-specific **encoding function**, kept in a global `sync.Map`. Later calls for the same type skip the analysis.

For structs, the analysis produces the list of fields with their names already escaped, the access index (including through embedded structs) and each field type's encoding function. When decoding, looking up a field by its JSON name uses a map, plus a case-insensitive comparison as a fallback.

## Where the time goes

- **Reflection** on every field (`reflect.Value.Field(i)`, interfaces): the main cost compared to generated code.
- **Allocations**: `Marshal` works in an internal buffer reused through `sync.Pool`, but the final result is always a new allocation; `Unmarshal` allocates strings for every string value and, for `any`, maps and slices.
- **Two-pass validation**: `Unmarshal` first scans the entire input to check it's valid JSON, then decodes it.
- **Custom methods**: `MarshalJSON` returns a `[]byte` the package must then **revalidate and compact**; a custom method can be slower than the default encoding.

When JSON is on the critical path, the options are: `json.Decoder` with concrete types (not `any`), reusing destination structs, code generators (easyjson, ffjson) or `encoding/json/v2`.

## Security pitfalls of the default behavior

**Name matching is case-insensitive.** `{"ADMIN": true}` fills the field tagged `json:"admin"`. If a proxy validates the JSON with case-sensitive rules (`"admin"` forbidden) and forwards the request to a Go service, the check can be bypassed.

**Duplicate keys are accepted**, and the last one wins:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Request struct {
	Role string `json:"role"`
}

func main() {
	var r Request
	json.Unmarshal([]byte(`{"role": "user", "ROLE": "admin"}`), &r)
	fmt.Println(r.Role) // admin: a duplicate + case-insensitive matching
}
```

Different parsers (in different languages) may pick different values for the same document — a known source of "parser differential" vulnerabilities.

**Invalid UTF-8** is silently replaced with `U+FFFD` when decoding, instead of being rejected.

**HTML escaping**: by default, `<`, `>` and `&` are written as `<` etc., so the JSON can be embedded in HTML pages. For pure APIs, `enc.SetEscapeHTML(false)` gives more readable output.

## Unmarshal into existing structs

`Unmarshal` does **not** reset the destination: fields absent from the JSON keep their previous value, existing maps get new keys (they aren't replaced), and slices are reset to length 0 and reused. Reusing a struct for several documents can mix data between them:

```go
package main

import (
	"encoding/json"
	"fmt"
)

type Profile struct {
	Name  string            `json:"name"`
	Email string            `json:"email"`
	Meta  map[string]string `json:"meta"`
}

func main() {
	var p Profile
	json.Unmarshal([]byte(`{"name":"Ana","email":"ana@x.ro","meta":{"a":"1"}}`), &p)
	json.Unmarshal([]byte(`{"name":"Ion","meta":{"b":"2"}}`), &p)
	fmt.Printf("%+v\n", p) // Ana's email and meta[a] are still there
}
```

## `encoding/json/v2` (experimental since Go 1.25)

With `GOEXPERIMENT=jsonv2`, the packages `encoding/json/v2` (semantic API) and `encoding/json/jsontext` (a syntactic, streaming tokenizer/encoder) are available. With the experiment on, even `encoding/json` (v1) is reimplemented on top of v2, preserving the old behavior through options.

The main default differences in v2:

| Behavior | v1 | v2 |
|---|---|---|
| name matching | case-insensitive | case-sensitive |
| duplicate keys | last one wins | error |
| invalid UTF-8 | replaced | error |
| `nil` slice/map | `null` | `[]` / `{}` |
| `omitempty` | by the Go value | by the empty JSON value |
| decoding performance | — | significantly faster, single-pass decoding |

v2 also adds tag options (`format:` for times and durations, `inline`, `unknown`), `MarshalWrite`/`UnmarshalRead` functions for streams, and `MarshalJSONTo`/`UnmarshalJSONFrom` methods that write straight into a `jsontext.Encoder`, with no intermediate `[]byte`.

The recommendation for new code that needs strictness today: a `Decoder` with `DisallowUnknownFields`, explicit duplicate-key validation where it matters, and preparing for v2 with exact names in tags.

## `encoding/gob` vs. JSON

`gob` sends the type description first, then the values in binary form; for long streams between Go programs it's more compact and faster than JSON. But it's tied to Go, isn't a stable format for long-term storage, and isn't suitable for untrusted data (decoding can consume a lot of memory). For cross-language communication, Protocol Buffers (`google.golang.org/protobuf`) is the usual choice.

## Summary

- Encoders are built via reflection once per type and cached.
- The cost comes from reflection, allocations and two-pass validation.
- v1 is lenient: case-insensitive names, duplicate keys, invalid UTF-8 accepted.
- `Unmarshal` doesn't reset the destination.
- `encoding/json/v2` (experimental) is strict, faster and has a streaming API.
