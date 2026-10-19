package main

// Legăturile către Go Playground.
//
// Spre deosebire de Rust Playground, Go Playground nu primește codul în URL:
// exemplul trebuie întâi trimis la /share, care întoarce un ID. ID-urile sunt
// păstrate în playground.json (cheia e SHA-256 al codului), ca generarea
// obișnuită a site-ului să nu depindă de rețea. `go run . -share` trimite
// exemplele noi sau modificate și actualizează fișierul.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

const shareURL = "https://play.golang.org/share"

type Playground struct {
	path string
	ids  map[string]string
}

func loadPlayground(path string) (*Playground, error) {
	pg := &Playground{path: path, ids: map[string]string{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return pg, nil
	}
	if err != nil {
		return nil, err
	}
	return pg, json.Unmarshal(b, &pg.ids)
}

func codeKey(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// ID întoarce ID-ul din Playground al exemplului, dacă a fost deja trimis.
func (pg *Playground) ID(code string) (string, bool) {
	if pg == nil {
		return "", false
	}
	id, ok := pg.ids[codeKey(code)]
	return id, ok
}

// Share trimite exemplele care nu au încă ID și păstrează doar ID-urile
// exemplelor care mai există în conținut.
func (pg *Playground) Share(codes []string) error {
	client := &http.Client{Timeout: 20 * time.Second}
	used := map[string]bool{}
	for _, code := range codes {
		key := codeKey(code)
		used[key] = true
		if _, ok := pg.ids[key]; ok {
			continue
		}
		resp, err := client.Post(shareURL, "text/plain; charset=utf-8", strings.NewReader(code))
		if err != nil {
			return err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("playground: %s: %s", resp.Status, body)
		}
		pg.ids[key] = strings.TrimSpace(string(body))
	}
	for key := range pg.ids {
		if !used[key] {
			delete(pg.ids, key)
		}
	}
	return pg.save()
}

func (pg *Playground) save() error {
	keys := make([]string, 0, len(pg.ids))
	for k := range pg.ids {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	// o cheie pe linie, în ordine, pentru diff-uri curate
	var b strings.Builder
	b.WriteString("{\n")
	for i, k := range keys {
		fmt.Fprintf(&b, "  %q: %q", k, pg.ids[k])
		if i < len(keys)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return os.WriteFile(pg.path, []byte(b.String()), 0o644)
}
