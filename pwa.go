package main

// Aplicația instalabilă (PWA): manifestele, iconițele și service worker-ul.
//
// Pe iPhone, „Add to Home Screen” din Safari folosește apple-touch-icon și
// meta-tag-urile din page.html; Android și desktop-ul folosesc manifestul.
// Service worker-ul (templates/sw.js) face paginile vizitate disponibile
// offline și poate descărca toate lecțiile unei limbi.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

var swTmpl = template.Must(template.ParseFS(siteFS, "templates/sw.js"))

type manifestIcon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Purpose string `json:"purpose,omitempty"`
}

type manifest struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	ShortName       string         `json:"short_name"`
	Description     string         `json:"description"`
	Lang            string         `json:"lang"`
	StartURL        string         `json:"start_url"`
	Scope           string         `json:"scope"`
	Display         string         `json:"display"`
	BackgroundColor string         `json:"background_color"`
	ThemeColor      string         `json:"theme_color"`
	Icons           []manifestIcon `json:"icons"`
}

// iconFiles sunt iconițele generate: calea publicată, dimensiunea, maskable.
var iconFiles = []struct {
	Path     string
	Size     int
	Maskable bool
}{
	{"icons/apple-touch-icon.png", 180, false},
	{"icons/icon-32.png", 32, false},
	{"icons/icon-192.png", 192, false},
	{"icons/icon-512.png", 512, false},
	{"icons/icon-maskable-512.png", 512, true},
}

// writePWA scrie iconițele și manifestele; se apelează înaintea paginilor.
func (s *Site) writePWA(out string) error {
	for _, ic := range iconFiles {
		b, err := renderIcon(ic.Size, ic.Maskable)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(out, ic.Path), b); err != nil {
			return err
		}
	}
	for _, lang := range Langs {
		m := manifest{
			ID:              s.Base,
			Name:            "Go Language Features",
			ShortName:       "Go Features",
			Description:     strs(lang).HomeLead,
			Lang:            lang.Code(),
			StartURL:        s.homeURL(lang),
			Scope:           s.Base,
			Display:         "standalone",
			BackgroundColor: "#f8fafb",
			ThemeColor:      "#007d9c",
		}
		for _, ic := range iconFiles[2:] {
			purpose := "any"
			if ic.Maskable {
				purpose = "maskable"
			}
			m.Icons = append(m.Icons, manifestIcon{Src: s.Base + ic.Path, Sizes: sizes(ic.Size), Type: "image/png", Purpose: purpose})
		}
		b, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(out, lang.Prefix(), "manifest.webmanifest"), b); err != nil {
			return err
		}
	}
	return nil
}

func sizes(n int) string {
	v := strconv.Itoa(n)
	return v + "x" + v
}

// writeServiceWorker scrie sw.js la rădăcina site-ului; se apelează ultima,
// pentru că versiunea e hash-ul tuturor celorlalte fișiere generate.
func (s *Site) writeServiceWorker(out string) error {
	h := sha256.New()
	err := filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(out, path)
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write(b)
		return nil
	})
	if err != nil {
		return err
	}

	precache := []string{s.Base, s.Base + "en/", s.Base + "assets/style.css", s.Base + "assets/app.js"}
	for _, ic := range iconFiles {
		precache = append(precache, s.Base+ic.Path)
	}
	pages := map[string][]string{}
	for _, lang := range Langs {
		precache = append(precache, s.Base+"search/"+lang.Code()+".json")
		for ti := range Topics {
			for li := range Levels {
				pages[lang.Code()] = append(pages[lang.Code()], s.pageURL(lang, ti, li))
			}
		}
	}
	slices.Sort(precache)

	pj, err := json.Marshal(precache)
	if err != nil {
		return err
	}
	gj, err := json.Marshal(pages)
	if err != nil {
		return err
	}
	var b strings.Builder
	err = swTmpl.Execute(&b, map[string]string{
		"Version":  hex.EncodeToString(h.Sum(nil))[:12],
		"Base":     s.Base,
		"Precache": string(pj),
		"Pages":    string(gj),
	})
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(out, "sw.js"), []byte(b.String()))
}
