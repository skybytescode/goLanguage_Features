package main

// Fiecare exemplu ```go din conținut e compilat și rulat:
//
//   - go                → compilează, e formatat cu gofmt și se termină cu succes;
//   - go,panic          → compilează și eșuează la rulare;
//   - go,compile_fail   → nu compilează;
//   - go,ignore         → fragment; nu e verificat.
//
// go test -short sare peste acest test.

import (
	"context"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type example struct {
	where string
	block CodeBlock
}

func allExamples(t *testing.T) []example {
	t.Helper()
	var out []example
	seen := map[string]bool{}
	for _, topic := range Topics {
		for li, l := range Levels {
			for _, lang := range Langs {
				src, err := topic.Page(lang, li)
				if err != nil {
					t.Fatal(err)
				}
				suffix := ".md"
				if lang == En {
					suffix = ".en.md"
				}
				for _, b := range codeBlocks(src) {
					if b.Info.Lang != "go" || b.Info.Ignore || seen[b.Code] {
						continue
					}
					seen[b.Code] = true
					out = append(out, example{
						where: fmt.Sprintf("content/%s/%s%s:%d", topic.Slug, l.Slug, suffix, b.Line),
						block: b,
					})
				}
			}
		}
	}
	return out
}

func TestExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("-short")
	}
	gobin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go nu e în PATH")
	}
	sem := make(chan struct{}, runtime.NumCPU())
	for _, ex := range allExamples(t) {
		t.Run(ex.where, func(t *testing.T) {
			t.Parallel()
			sem <- struct{}{}
			defer func() { <-sem }()
			checkExample(t, gobin, ex)
		})
	}
}

func checkExample(t *testing.T, gobin string, ex example) {
	info, code := ex.block.Info, ex.block.Code
	if !info.CompileFail {
		formatted, err := format.Source([]byte(code))
		if err != nil {
			t.Fatalf("%s: gofmt: %v", ex.where, err)
		}
		if string(formatted) != code {
			t.Errorf("%s: codul nu e formatat cu gofmt (rulează `go run . -fmt`)", ex.where)
		}
	}

	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.26\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644))
	bin := filepath.Join(dir, "prog")
	build := exec.Command(gobin, "build", "-o", bin, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOWORK=off")
	buildOut, buildErr := build.CombinedOutput()

	if info.CompileFail {
		if buildErr == nil {
			t.Errorf("%s: marcat compile_fail, dar compilează", ex.where)
		}
		return
	}
	if buildErr != nil {
		t.Fatalf("%s: nu compilează:\n%s", ex.where, buildOut)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	run := exec.CommandContext(ctx, bin)
	run.Dir = dir
	runOut, runErr := run.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s: nu s-a terminat în 20s", ex.where)
	}
	switch {
	case info.Panic && runErr == nil:
		t.Errorf("%s: marcat panic, dar se termină cu succes:\n%s", ex.where, runOut)
	case !info.Panic && runErr != nil:
		t.Errorf("%s: eșuează la rulare (%v):\n%s", ex.where, runErr, runOut)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
