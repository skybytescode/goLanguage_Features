// Command goLanguage_Features generează site-ul static despre funcționalitățile
// limbajului Go.
//
//	go run .                    generează site-ul în dist/
//	go run . -serve             îl generează și îl servește pe http://localhost:8080
//	go run . -share             trimite exemplele noi în Go Playground (playground.json)
//	go run . -fmt               formatează cu gofmt exemplele din content/
//	go run . -base /repo/       prefixul URL-urilor, pentru GitHub Pages
package main

import (
	"flag"
	"fmt"
	"go/format"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	out := flag.String("out", "dist", "directorul în care se scrie site-ul")
	base := flag.String("base", "/", "prefixul URL-urilor site-ului")
	share := flag.Bool("share", false, "trimite exemplele fără ID în Go Playground și actualizează playground.json")
	serve := flag.String("serve", "", "după generare, servește site-ul la adresa dată (de ex. :8080)")
	gofmt := flag.Bool("fmt", false, "formatează cu gofmt exemplele Go din content/ și iese")
	flag.Parse()

	if *gofmt {
		if err := formatContent("content"); err != nil {
			log.Fatal(err)
		}
		return
	}

	if !strings.HasPrefix(*base, "/") {
		*base = "/" + *base
	}
	if !strings.HasSuffix(*base, "/") {
		*base += "/"
	}

	pg, err := loadPlayground("playground.json")
	if err != nil {
		log.Fatal(err)
	}
	if *share {
		codes, err := shareableCode()
		if err != nil {
			log.Fatal(err)
		}
		if err := pg.Share(codes); err != nil {
			log.Fatal(err)
		}
		log.Printf("playground.json: %d exemple", len(codes))
	}

	site := &Site{Base: *base, Playground: pg}
	if err := site.Build(*out); err != nil {
		log.Fatal(err)
	}
	log.Printf("site generat în %s/", *out)

	if *serve != "" {
		mux := http.NewServeMux()
		mux.Handle(*base, http.StripPrefix(strings.TrimSuffix(*base, "/"), http.FileServer(http.Dir(*out))))
		log.Printf("http://localhost%s%s", *serve, *base)
		log.Fatal(http.ListenAndServe(*serve, mux))
	}
}

// shareableCode întoarce codul tuturor exemplelor care primesc buton de rulare.
func shareableCode() ([]string, error) {
	var codes []string
	for _, t := range Topics {
		for _, lang := range Langs {
			for li := range Levels {
				src, err := t.Page(lang, li)
				if err != nil {
					return nil, err
				}
				for _, b := range codeBlocks(src) {
					if b.Info.Lang == "go" && !b.Info.Ignore {
						codes = append(codes, b.Code)
					}
				}
			}
		}
	}
	return codes, nil
}

// formatContent rescrie, cu gofmt, exemplele Go din fișierele Markdown din dir.
// Exemplele care nu compilează intenționat sunt lăsate cum sunt.
func formatContent(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(b)
		blocks := codeBlocks(src)
		// de la coadă spre început, ca pozițiile blocurilor rămase să fie valide
		for i := len(blocks) - 1; i >= 0; i-- {
			blk := blocks[i]
			if blk.Info.Lang != "go" || blk.Info.CompileFail || blk.Info.Ignore {
				continue
			}
			formatted, err := format.Source([]byte(blk.Code))
			if err != nil {
				return fmt.Errorf("%s:%d: %w", path, blk.Line, err)
			}
			src = src[:blk.Start] + string(formatted) + src[blk.End:]
		}
		if src == string(b) {
			return nil
		}
		log.Printf("formatat: %s", path)
		return os.WriteFile(path, []byte(src), 0o644)
	})
}
