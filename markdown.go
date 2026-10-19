package main

// Transformă Markdown în HTML.
//
// Blocurile ```go primesc un buton care deschide exemplul în Go Playground
// și un buton de copiere. Atributele de după virgulă schimbă afișarea:
// go,compile_fail marchează un exemplu care nu compilează intenționat,
// go,panic unul care eșuează la rulare intenționat (panic sau deadlock), iar
// go,ignore un fragment care nu e program complet: nu are buton de rulare.
//
// Toate blocurile de cod sunt colorate cu chroma, prin clase CSS.

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// CodeInfo descrie un bloc de cod după linia de deschidere (de ex. "go,panic").
type CodeInfo struct {
	Lang        string
	CompileFail bool
	Panic       bool
	Ignore      bool
}

func parseInfo(info string) CodeInfo {
	parts := strings.Split(strings.TrimSpace(info), ",")
	ci := CodeInfo{Lang: strings.TrimSpace(parts[0])}
	for _, p := range parts[1:] {
		switch strings.TrimSpace(p) {
		case "compile_fail":
			ci.CompileFail = true
		case "panic":
			ci.Panic = true
		case "ignore":
			ci.Ignore = true
		}
	}
	return ci
}

// Runnable spune dacă blocul e un program Go complet, care poate fi rulat.
func (ci CodeInfo) Runnable() bool {
	return ci.Lang == "go" && !ci.Ignore && !ci.CompileFail
}

var formatter = chromahtml.New(chromahtml.WithClasses(true), chromahtml.PreventSurroundingPre(true))

func highlight(code, lang string) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		return html.EscapeString(code)
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, code)
	if err != nil {
		return html.EscapeString(code)
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, chroma.MustNewStyle("none", chroma.StyleEntries{}), it); err != nil {
		return html.EscapeString(code)
	}
	return buf.String()
}

// codeRenderer înlocuiește randarea implicită a blocurilor de cod delimitate.
type codeRenderer struct {
	lang       Lang
	playground *Playground
}

func (r *codeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.render)
}

func (r *codeRenderer) render(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	block := n.(*ast.FencedCodeBlock)
	var info CodeInfo
	if block.Info != nil {
		info = parseInfo(string(block.Info.Value(src)))
	}
	var code strings.Builder
	for i := 0; i < block.Lines().Len(); i++ {
		line := block.Lines().At(i)
		code.Write(line.Value(src))
	}
	w.WriteString(codeBlock(info, code.String(), r.lang, r.playground))
	return ast.WalkSkipChildren, nil
}

func codeBlock(info CodeInfo, code string, lang Lang, pg *Playground) string {
	s := strs(lang)
	var bar strings.Builder
	if info.Lang != "" {
		fmt.Fprintf(&bar, `<span class="code-lang">%s</span>`, html.EscapeString(info.Lang))
	}
	bar.WriteString(`<span class="spacer"></span>`)
	switch {
	case info.CompileFail:
		fmt.Fprintf(&bar, `<span class="badge fail">%s</span>`, s.BadgeCompileFail)
	case info.Panic:
		fmt.Fprintf(&bar, `<span class="badge fail">%s</span>`, s.BadgePanic)
	}
	fmt.Fprintf(&bar, `<button type="button" class="copy" data-copied="%s">%s</button>`, s.Copied, s.Copy)
	if info.Lang == "go" && !info.Ignore {
		if id, ok := pg.ID(code); ok {
			fmt.Fprintf(&bar, `<a class="run" href="https://go.dev/play/p/%s" target="_blank" rel="noopener">%s</a>`, id, s.Run)
		}
	}
	class := "code"
	if info.CompileFail || info.Panic {
		class += " fails"
	}
	return fmt.Sprintf(`<div class="%s"><div class="code-bar">%s</div><pre><code>%s</code></pre></div>`,
		class, bar.String(), highlight(code, info.Lang))
}

func newMarkdown(lang Lang, pg *Playground) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(), // conținutul e al nostru; permite HTML în Markdown
			renderer.WithNodeRenderers(util.Prioritized(&codeRenderer{lang: lang, playground: pg}, 100)),
		),
	)
}

func renderMarkdown(src string, lang Lang, pg *Playground) (string, error) {
	var buf bytes.Buffer
	if err := newMarkdown(lang, pg).Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CodeBlock e un bloc de cod extras dintr-o pagină, pentru teste și Playground.
type CodeBlock struct {
	Info CodeInfo
	Code string
	Line int // linia din fișierul Markdown, pentru mesaje de eroare

	// Start și End delimitează codul în fișierul Markdown.
	Start, End int
}

// codeBlocks extrage blocurile de cod delimitate dintr-o pagină Markdown.
func codeBlocks(src string) []CodeBlock {
	var out []CodeBlock
	source := []byte(src)
	doc := goldmark.New().Parser().Parse(text.NewReader(source))
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		block, ok := n.(*ast.FencedCodeBlock)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		var info CodeInfo
		if block.Info != nil {
			info = parseInfo(string(block.Info.Value(source)))
		}
		var code strings.Builder
		for i := 0; i < block.Lines().Len(); i++ {
			line := block.Lines().At(i)
			code.Write(line.Value(source))
		}
		cb := CodeBlock{Info: info, Code: code.String(), Line: 1}
		if n := block.Lines().Len(); n > 0 {
			cb.Start, cb.End = block.Lines().At(0).Start, block.Lines().At(n-1).Stop
			cb.Line = bytes.Count(source[:cb.Start], []byte("\n")) // linia cu ```
		}
		out = append(out, cb)
		return ast.WalkSkipChildren, nil
	})
	return out
}
