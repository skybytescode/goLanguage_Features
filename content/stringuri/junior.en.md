# Strings, bytes and runes — Junior

A Go `string` is an **immutable sequence of bytes**, usually UTF-8 encoded text. The difference between bytes and characters is essential for any text that isn't plain ASCII, including accented letters.

## Literals

```go
package main

import "fmt"

func main() {
	s := "Hello,\n\"world\"!" // interpreted: \n and \" are escape sequences
	r := `C:\path\no\escapes
on several lines` // raw: exactly as written
	fmt.Println(s)
	fmt.Println(r)
}
```

Raw literals (with backticks) are handy for regular expressions, JSON and multi-line text.

## Length is in bytes

`len(s)` counts **bytes**, not characters. Accented letters take 2 bytes in UTF-8:

```go
package main

import (
	"fmt"
	"unicode/utf8"
)

func main() {
	s := "școală"
	fmt.Println(len(s))                    // 8 bytes
	fmt.Println(utf8.RuneCountInString(s)) // 6 characters
}
```

## Indexing vs. `range`

`s[i]` returns a **byte**. `range` walks **characters** (`rune`s):

```go
package main

import "fmt"

func main() {
	s := "țară"
	fmt.Println(s[0]) // 200: the first byte of "ț", not the letter

	for i, r := range s {
		fmt.Printf("%d: %c\n", i, r) // the index jumps: 0, 2, 3, 4
	}
}
```

A `rune` is an `int32` holding a Unicode code point. A rune literal uses single quotes: `'ț'`.

## Strings are immutable

You can't change a character in a string. You create a new one:

```go,compile_fail
package main

func main() {
	s := "kitten"
	s[0] = 'K' // error: cannot assign to s[0] (neither addressable nor a map index expression)
}
```

```go
package main

import "fmt"

func main() {
	s := "kitten"
	r := []rune(s) // convert to a slice of characters
	r[0] = 'K'
	fmt.Println(string(r)) // Kitten
}
```

## Concatenation

```go
package main

import "fmt"

func main() {
	first, last := "Ana", "Pop"
	full := first + " " + last
	full += "!"
	fmt.Println(full)
	fmt.Println(fmt.Sprintf("%s is %d years old", first, 30))
}
```

## The `strings` package

```go
package main

import (
	"fmt"
	"strings"
)

func main() {
	s := "  Go is simple and fast  "

	fmt.Println(strings.TrimSpace(s))
	fmt.Println(strings.ToUpper(s))
	fmt.Println(strings.Contains(s, "simple"))          // true
	fmt.Println(strings.HasPrefix("golang", "go"))      // true
	fmt.Println(strings.Index("golang", "lang"))        // 2
	fmt.Println(strings.Replace("a-b-c", "-", "+", -1)) // a+b+c
	fmt.Println(strings.Split("a,b,c", ","))            // [a b c]
	fmt.Println(strings.Join([]string{"x", "y"}, "/"))  // x/y
	fmt.Println(strings.Fields(s))                      // the words, without spaces
	fmt.Println(strings.Repeat("ab", 3))                // ababab
}
```

## Conversions with `strconv`

`strconv` converts between text and numbers:

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	n, err := strconv.Atoi("42")
	fmt.Println(n+1, err) // 43 <nil>

	s := strconv.Itoa(99)
	fmt.Println(s + "!") // 99!

	f, _ := strconv.ParseFloat("3.14", 64)
	b, _ := strconv.ParseBool("true")
	fmt.Println(f, b)
}
```

Careful: `string(65)` does **not** give `"65"`, but `"A"` (the character with code 65). For numbers, use `strconv.Itoa`. `go vet` flags this mistake.

## Comparing

`==` compares contents, and `<` and `>` compare lexicographically, byte by byte:

```go
package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Println("abc" == "abc", "abc" < "abd") // true true
	fmt.Println(strings.EqualFold("Go", "GO")) // true: case-insensitive
}
```

## Summary

- `string` = immutable bytes, usually UTF-8.
- `len` counts bytes; `utf8.RuneCountInString` counts characters.
- `s[i]` is a byte; `range` gives characters (`rune`s).
- `strings` for searching and transforming, `strconv` for numbers.
- `string(65)` is `"A"`, not `"65"`.
