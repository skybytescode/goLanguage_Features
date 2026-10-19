# Strings, bytes and runes — Senior

The internal representation of strings, the conversions the compiler performs without copying, zero-copy conversions with `unsafe`, interning with `unique`, and memory leaks through substrings.

## The string header

A string is a two-word struct, with no capacity:

```go,ignore
type stringStruct struct {
	str unsafe.Pointer // the bytes (unmodifiable)
	len int
}
```

Because the bytes never change, strings can **share** memory freely: `s[i:j]` just creates a new header, and assignment copies two words. Literals live in the binary's read-only segment.

## Substrings keep the original alive

`s[i:j]` points into the bytes of `s`. If you keep a small substring of a huge string (one line from a 100 MB file read in full), the whole file stays in memory:

```go
package main

import (
	"fmt"
	"strings"
)

var index = map[string]bool{}

func process(content string) {
	for line := range strings.Lines(content) {
		id, _, _ := strings.Cut(line, ",")
		index[strings.Clone(id)] = true // without Clone, every key would hold all of content
	}
}

func main() {
	process("a1,x\nb2,y\n")
	fmt.Println(len(index))
}
```

`strings.Clone` (Go 1.20) makes an exact copy, so the original can be freed.

## Copy-free conversions done by the compiler

`string(b)` and `[]byte(s)` generally copy. The compiler removes the copy when it can prove the result isn't retained:

| Code | Copies? |
|---|---|
| `m[string(b)]` (map lookup) | no |
| `string(b) == "abc"`, `string(a) < string(b)` | no |
| `"<" + string(b) + ">"` | only the final result |
| `for i, c := range []byte(s)` | no |
| `switch string(b) { case ... }` | no |
| `x := string(b)` retained | yes |

Additionally, conversions of small strings (under 32 bytes) that don't escape the function use a stack buffer, with no heap allocation.

## Zero-copy with `unsafe`

When profiling shows a conversion matters, `unsafe.String` and `unsafe.StringData` (Go 1.20) allow conversions with no copy. The contract: the bytes **must not** change while the string exists.

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
	b := []byte("hello")
	s := bytesToString(b)
	fmt.Println(s)
	b[0] = 'H'     // BREAKS the contract: s changes too
	fmt.Println(s) // Hello — an "immutable" string that changed
}
```

The reverse, `unsafe.Slice(unsafe.StringData(s), len(s))`, gives a `[]byte` you **must not write to**: a literal's memory is read-only, and writing kills the program with a segmentation fault.

## Concatenation

`a + b + c` compiles into a single `runtime.concatstrings` call, which computes the total length and allocates once. The problem only appears in loops, where every iteration performs a separate concatenation; hence the advice to use `strings.Builder`.

`strings.Builder` also avoids the final copy: `String()` returns a string built directly over the internal buffer, with `unsafe.String`. That's why a builder must not be copied after its first write (there's a run-time check).

## Interning: the `unique` package (Go 1.23)

When you have many copies of the same values (country names, tags, cache keys), `unique.Make` keeps a single canonical copy and returns a `Handle[T]`. Comparing two handles is a pointer comparison:

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
	fmt.Println(a == b, a.Value()) // true roro: the same canonical value
}
```

Unused canonical values are freed automatically (the package uses weak pointers internally), so you don't get the memory leak of a global map.

## UTF-8 under the hood

| Code point | Bytes | Format |
|---|---|---|
| U+0000 – U+007F | 1 | `0xxxxxxx` |
| U+0080 – U+07FF | 2 | `110xxxxx 10xxxxxx` |
| U+0800 – U+FFFF | 3 | `1110xxxx 10xxxxxx 10xxxxxx` |
| U+10000 – U+10FFFF | 4 | `11110xxx 10xxxxxx 10xxxxxx 10xxxxxx` |

Romanian diacritics (U+0102–U+021B) take 2 bytes. Important properties: an ASCII byte never appears inside a multi-byte character, so searching for an ASCII separator (`,`, `\n`) directly in the bytes is safe; and continuation bytes (`10xxxxxx`) let you find the start of a character from any position.

Go doesn't validate UTF-8 when a string is created. Invalid sequences become `U+FFFD` only when **decoded** (`range`, `[]rune(s)`, `utf8.DecodeRune`); `strings.ToValidUTF8` replaces them explicitly.

## `[]rune` is costly

`[]rune(s)` allocates 4 bytes per character. To count characters, `utf8.RuneCountInString` allocates nothing; to iterate, `range` decodes on the fly. Save `[]rune` for when you need random access by character position.

## Summary

- String = (pointer, length), immutable; substrings share memory.
- `strings.Clone` breaks the link to a large original.
- The compiler avoids copies in map lookups, comparisons and `range`.
- `unsafe.String` is zero-copy, but requires the bytes never to change.
- `unique.Make` interns repeated values; `[]rune` costs 4 bytes per character.
