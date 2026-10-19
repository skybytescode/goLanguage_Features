package main

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestEveryPageExistsInBothLanguages(t *testing.T) {
	for _, topic := range Topics {
		for li, l := range Levels {
			var blocks [2][]CodeInfo
			for _, lang := range Langs {
				src, err := topic.Page(lang, li)
				if err != nil {
					t.Errorf("%s/%s (%s): %v", topic.Slug, l.Slug, lang.Code(), err)
					continue
				}
				if !strings.HasPrefix(src, "# ") {
					t.Errorf("%s/%s (%s): pagina trebuie să înceapă cu un titlu „# ”", topic.Slug, l.Slug, lang.Code())
				}
				for _, b := range codeBlocks(src) {
					blocks[lang] = append(blocks[lang], b.Info)
				}
			}
			// traducerea are aceleași exemple, în aceeași ordine
			if !slices.Equal(blocks[Ro], blocks[En]) {
				t.Errorf("%s/%s: blocurile de cod diferă între RO și EN:\nRO %v\nEN %v", topic.Slug, l.Slug, blocks[Ro], blocks[En])
			}
		}
	}
}

func TestNoStrayContentFiles(t *testing.T) {
	known := map[string]bool{}
	for _, topic := range Topics {
		known[topic.Slug] = true
	}
	entries, err := contentFS.ReadDir("content")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !known[e.Name()] {
			t.Errorf("content/%s nu e în catalogul din content.go", e.Name())
		}
	}
}

func TestParseInfo(t *testing.T) {
	cases := map[string]CodeInfo{
		"go":              {Lang: "go"},
		"go,compile_fail": {Lang: "go", CompileFail: true},
		"go, panic":       {Lang: "go", Panic: true},
		"go,ignore":       {Lang: "go", Ignore: true},
		"bash":            {Lang: "bash"},
		"":                {},
	}
	for in, want := range cases {
		if got := parseInfo(in); got != want {
			t.Errorf("parseInfo(%q) = %+v, vreau %+v", in, got, want)
		}
	}
}

func TestPlainTextSkipsTitle(t *testing.T) {
	headings, txt := plainText("# Titlu\n\nText **gros** și `cod`.\n\n## Sub\n\n```go\nfmt.Println(1)\n```\n")
	if !slices.Equal(headings, []string{"Sub"}) {
		t.Errorf("headings = %q", headings)
	}
	if strings.Contains(txt, "Titlu") {
		t.Errorf("titlul nu trebuie să fie în text: %q", txt)
	}
	for _, want := range []string{"Text gros și cod.", "Sub", "fmt.Println(1)"} {
		if !strings.Contains(txt, want) {
			t.Errorf("lipsește %q din %q", want, txt)
		}
	}
}

// Toate linkurile interne din site duc la pagini care există.
func TestInternalLinks(t *testing.T) {
	out := t.TempDir()
	site := &Site{Base: "/base/"}
	if err := site.Build(out); err != nil {
		t.Fatal(err)
	}
	href := regexp.MustCompile(`(?:href|src)="([^"#]*)(#[^"]*)?"`)
	err := filepath.WalkDir(out, func(file string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(file, ".html") {
			return err
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(out, filepath.Dir(file))
		if err != nil {
			return err
		}
		pageDir := path.Join("/base", filepath.ToSlash(rel))
		for _, m := range href.FindAllStringSubmatch(string(b), -1) {
			link := m[1]
			if link == "" || strings.Contains(link, "://") || strings.HasPrefix(link, "data:") {
				continue
			}
			if !strings.HasPrefix(link, "/") {
				dirLink := strings.HasSuffix(link, "/")
				link = path.Join(pageDir, link)
				if dirLink {
					link += "/"
				}
			}
			if !strings.HasPrefix(link, "/base/") {
				t.Errorf("%s: link în afara site-ului: %s", file, m[1])
				continue
			}
			target := filepath.Join(out, strings.TrimPrefix(link, "/base/"))
			if strings.HasSuffix(link, "/") {
				target = filepath.Join(target, "index.html")
			}
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s: link rupt: %s", file, m[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
