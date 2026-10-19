package main

// Indexul de căutare: câte un fișier JSON pe limbă (search/<limbă>.json),
// încărcat de assets/app.js la prima căutare. Căutarea propriu-zisă rulează
// în browser.

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type searchEntry struct {
	Topic    string   `json:"t"`
	Level    int      `json:"l"` // de la 1 la 3
	Name     string   `json:"n"` // numele nivelului
	URL      string   `json:"u"`
	Headings []string `json:"h"`
	Text     string   `json:"x"`
}

func (s *Site) searchIndex(lang Lang) ([]searchEntry, error) {
	var entries []searchEntry
	for ti, t := range Topics {
		for li, l := range Levels {
			src, err := t.Page(lang, li)
			if err != nil {
				return nil, err
			}
			headings, txt := plainText(src)
			entries = append(entries, searchEntry{
				Topic:    t.Title(lang),
				Level:    li + 1,
				Name:     l.Name(lang),
				URL:      s.pageURL(lang, ti, li),
				Headings: headings,
				Text:     txt,
			})
		}
	}
	return entries, nil
}

var plainParser = goldmark.New(goldmark.WithExtensions(extension.Table)).Parser()

// plainText extrage subtitlurile și textul simplu (inclusiv codul) dintr-o
// pagină Markdown. Titlul principal (#) nu intră în text: repetă doar numele
// subiectului și nivelul.
func plainText(markdown string) (headings []string, txt string) {
	src := []byte(markdown)
	doc := plainParser.Parse(text.NewReader(src))
	var b strings.Builder
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Heading:
			if !entering {
				return ast.WalkContinue, nil
			}
			if n.Level == 1 {
				return ast.WalkSkipChildren, nil
			}
			h := inlineText(n, src)
			headings = append(headings, h)
			b.WriteString(h + " ")
			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			if entering {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					b.Write(line.Value(src))
				}
				b.WriteString(" ")
			}
			return ast.WalkSkipChildren, nil
		case *ast.Paragraph, *ast.ListItem, *extast.TableCell:
			if !entering {
				b.WriteString(" ")
			}
		case *ast.Text:
			if entering {
				b.Write(n.Segment.Value(src))
				if n.SoftLineBreak() || n.HardLineBreak() {
					b.WriteString(" ")
				}
			}
		case *ast.String:
			if entering {
				b.Write(n.Value)
			}
		}
		return ast.WalkContinue, nil
	})
	return headings, strings.Join(strings.Fields(b.String()), " ")
}

func inlineText(n ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch c := c.(type) {
		case *ast.Text:
			b.Write(c.Segment.Value(src))
		case *ast.String:
			b.Write(c.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}
