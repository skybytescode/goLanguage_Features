package main

// Generarea site-ului static.
//
// Fiecare pagină e un fișier index.html separat, ca site-ul să funcționeze
// pe GitHub Pages fără JavaScript și fără configurare de server:
//
//	/                          pagina de start, în română
//	/<subiect>/<nivel>/        o pagină în română
//	/en/                       pagina de start, în engleză
//	/en/<subiect>/<nivel>/     o pagină în engleză
//
// JavaScript-ul (assets/app.js) adaugă doar căutarea și butoanele de copiere.

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates assets
var siteFS embed.FS

var tmpl = template.Must(template.New("").ParseFS(siteFS, "templates/*.html"))

// Site ține setările comune tuturor paginilor.
type Site struct {
	Base       string // prefixul URL-urilor, de ex. "/goLanguage_Features/"
	Playground *Playground
}

func (s *Site) homeURL(lang Lang) string { return s.Base + lang.Prefix() }

func (s *Site) pageURL(lang Lang, topic, level int) string {
	return fmt.Sprintf("%s%s%s/%s/", s.Base, lang.Prefix(), Topics[topic].Slug, Levels[level].Slug)
}

type navLink struct {
	Title  string
	URL    string
	Active bool
}

type levelLink struct {
	Num     int
	Name    string
	Tagline string
	URL     string
	Active  bool
}

// pageData e tot ce primește șablonul page.html.
type pageData struct {
	Lang        string
	OtherLang   string
	Title       string
	Description string
	Base        string
	HomeURL     string
	SwitchURL   string
	S           *Strings
	Nav         []navLink
	Levels      []levelLink

	// Doar pentru pagina de start.
	Home   bool
	Topics []navLink

	// Doar pentru paginile de subiect.
	Content template.HTML
	Prev    *navLink
	Next    *navLink
}

func (s *Site) basePage(lang Lang, currentTopic, level int) pageData {
	d := pageData{
		Lang:      lang.Code(),
		OtherLang: lang.Other().Code(),
		Base:      s.Base,
		HomeURL:   s.homeURL(lang),
		S:         strs(lang),
	}
	for i, t := range Topics {
		d.Nav = append(d.Nav, navLink{Title: t.Title(lang), URL: s.pageURL(lang, i, level), Active: i == currentTopic})
	}
	return d
}

func (s *Site) homePage(lang Lang) pageData {
	d := s.basePage(lang, -1, 0)
	d.Home = true
	d.Title = "Go Language Features"
	d.Description = strs(lang).HomeLead
	d.SwitchURL = s.homeURL(lang.Other())
	for i, l := range Levels {
		d.Levels = append(d.Levels, levelLink{Num: i + 1, Name: l.Name(lang), Tagline: l.Tagline(lang)})
	}
	for i, t := range Topics {
		d.Topics = append(d.Topics, navLink{Title: t.Title(lang), URL: s.pageURL(lang, i, 0)})
	}
	return d
}

func (s *Site) topicPage(lang Lang, topic, level int) (pageData, error) {
	t := Topics[topic]
	src, err := t.Page(lang, level)
	if err != nil {
		return pageData{}, err
	}
	body, err := renderMarkdown(src, lang, s.Playground)
	if err != nil {
		return pageData{}, err
	}
	d := s.basePage(lang, topic, level)
	d.Title = fmt.Sprintf("%s — %s · Go Language Features", t.Title(lang), Levels[level].Name(lang))
	d.Description = Levels[level].Tagline(lang)
	d.SwitchURL = s.pageURL(lang.Other(), topic, level)
	d.Content = template.HTML(body)
	for i, l := range Levels {
		d.Levels = append(d.Levels, levelLink{
			Num: i + 1, Name: l.Name(lang), Tagline: l.Tagline(lang),
			URL: s.pageURL(lang, topic, i), Active: i == level,
		})
	}
	if topic > 0 {
		d.Prev = &navLink{Title: Topics[topic-1].Title(lang), URL: s.pageURL(lang, topic-1, level)}
	}
	if topic < len(Topics)-1 {
		d.Next = &navLink{Title: Topics[topic+1].Title(lang), URL: s.pageURL(lang, topic+1, level)}
	}
	return d, nil
}

// Build scrie site-ul complet în directorul out.
func (s *Site) Build(out string) error {
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := s.writePWA(out); err != nil {
		return err
	}
	for _, lang := range Langs {
		dir := filepath.Join(out, lang.Prefix())
		if err := writePage(filepath.Join(dir, "index.html"), s.homePage(lang)); err != nil {
			return err
		}
		for ti, t := range Topics {
			for li, l := range Levels {
				d, err := s.topicPage(lang, ti, li)
				if err != nil {
					return err
				}
				if err := writePage(filepath.Join(dir, t.Slug, l.Slug, "index.html"), d); err != nil {
					return err
				}
			}
			// /<subiect>/ trimite la primul nivel, iar adresele vechi la nivelul nou
			if err := writeRedirect(filepath.Join(dir, t.Slug, "index.html"), s.pageURL(lang, ti, 0)); err != nil {
				return err
			}
			for old, li := range oldLevels {
				if err := writeRedirect(filepath.Join(dir, t.Slug, old, "index.html"), s.pageURL(lang, ti, li)); err != nil {
					return err
				}
			}
		}
		index, err := s.searchIndex(lang)
		if err != nil {
			return err
		}
		b, err := json.Marshal(index)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(out, "search", lang.Code()+".json"), b); err != nil {
			return err
		}
	}
	if err := copyAssets(out); err != nil {
		return err
	}
	// GitHub Pages: fără procesare Jekyll
	if err := writeFile(filepath.Join(out, ".nojekyll"), nil); err != nil {
		return err
	}
	return s.writeServiceWorker(out) // ultimul: versiunea depinde de toate celelalte fișiere
}

func writePage(path string, d pageData) error {
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "page.html", d); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return writeFile(path, []byte(b.String()))
}

func writeRedirect(path, url string) error {
	page := fmt.Sprintf(`<!doctype html><meta charset="utf-8"><meta http-equiv="refresh" content="0; url=%s"><link rel="canonical" href="%[1]s">`, url)
	return writeFile(path, []byte(page))
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func copyAssets(out string) error {
	return fs.WalkDir(siteFS, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := siteFS.ReadFile(path)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(out, path), b)
	})
}
