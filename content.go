package main

// Catalogul de conținut: limbile, nivelurile și subiectele site-ului.
//
// Fiecare subiect are câte un fișier Markdown pentru fiecare nivel și limbă:
// content/<slug>/<nivel>.md (română) și content/<slug>/<nivel>.en.md
// (engleză), unde <nivel> e junior, middle sau senior. Fișierele sunt
// incluse în binar cu go:embed; un fișier lipsă oprește generarea site-ului.

import (
	"embed"
	"fmt"
)

//go:embed content
var contentFS embed.FS

type Lang int

const (
	Ro Lang = iota
	En
)

var Langs = []Lang{Ro, En}

func (l Lang) Code() string {
	if l == En {
		return "en"
	}
	return "ro"
}

func (l Lang) Other() Lang {
	if l == En {
		return Ro
	}
	return En
}

// Prefix e prefixul URL-urilor în limba dată, relativ la rădăcina site-ului.
func (l Lang) Prefix() string {
	if l == En {
		return "en/"
	}
	return ""
}

type Level struct {
	Slug    string
	name    [2]string
	tagline [2]string
}

func (l Level) Name(lang Lang) string    { return l.name[lang] }
func (l Level) Tagline(lang Lang) string { return l.tagline[lang] }

var Levels = []Level{
	{
		Slug:    "junior",
		name:    [2]string{"Junior", "Junior"},
		tagline: [2]string{"Ce este și cum îl folosesc", "What it is and how to use it"},
	},
	{
		Slug:    "middle",
		name:    [2]string{"Middle", "Middle"},
		tagline: [2]string{"Cum îl folosesc corect: idiomuri, pattern-uri, capcane", "How to use it well: idioms, patterns, pitfalls"},
	},
	{
		Slug:    "senior",
		name:    [2]string{"Senior", "Senior"},
		tagline: [2]string{"Cum funcționează pe dinăuntru și de ce", "How it works under the hood, and why"},
	},
}

type Topic struct {
	Slug  string
	title [2]string
}

func (t Topic) Title(lang Lang) string { return t.title[lang] }

// Page întoarce Markdown-ul paginii pentru nivelul dat (de la 0 la 2).
func (t Topic) Page(lang Lang, level int) (string, error) {
	suffix := ".md"
	if lang == En {
		suffix = ".en.md"
	}
	b, err := contentFS.ReadFile(fmt.Sprintf("content/%s/%s%s", t.Slug, Levels[level].Slug, suffix))
	return string(b), err
}

func topic(slug, ro, en string) Topic {
	return Topic{Slug: slug, title: [2]string{ro, en}}
}

var Topics = []Topic{
	topic("variabile-tipuri", "Variabile, tipuri și constante", "Variables, types and constants"),
	topic("control-flow", "Control flow", "Control flow"),
	topic("functii", "Funcții și closures", "Functions and closures"),
	topic("pointeri", "Pointeri", "Pointers"),
	topic("structs-metode", "Structs și metode", "Structs and methods"),
	topic("interfete", "Interfețe", "Interfaces"),
	topic("erori", "Erori, defer, panic și recover", "Errors, defer, panic and recover"),
	topic("slices-arrays", "Arrays și slices", "Arrays and slices"),
	topic("maps", "Maps", "Maps"),
	topic("stringuri", "Stringuri, bytes și runes", "Strings, bytes and runes"),
	topic("generics", "Generics", "Generics"),
	topic("iteratori", "Range și iteratori", "Range and iterators"),
	topic("goroutines", "Goroutines", "Goroutines"),
	topic("channels", "Channels", "Channels"),
	topic("select", "Select", "Select"),
	topic("sync", "Sync și atomic", "Sync and atomic"),
	topic("context", "Context", "Context"),
	topic("pachete-module", "Pachete și module", "Packages and modules"),
	topic("io", "I/O: Reader și Writer", "I/O: Reader and Writer"),
	topic("json-encoding", "JSON și encoding", "JSON and encoding"),
	topic("http", "net/http", "net/http"),
	topic("testare", "Testare, benchmarks și fuzzing", "Testing, benchmarks and fuzzing"),
	topic("reflect-unsafe", "Reflection și unsafe", "Reflection and unsafe"),
	topic("runtime", "Runtime: scheduler, memorie, GC", "Runtime: scheduler, memory, GC"),
	topic("tooling", "Tooling și profiling", "Tooling and profiling"),
}
