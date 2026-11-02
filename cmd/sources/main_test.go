package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/sources"
)

// demo is a place for the examples: a server sending one paper, a list
// naming it unpinned, a shelf in a fresh directory, and a fixed clock.
// done takes it all away.
func demo(pin string) (place, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(
		answer http.ResponseWriter, request *http.Request) {
		answer.Write([]byte("a paper, as published"))
	}))
	dir, _ := os.MkdirTemp("", "sources-example")
	list := filepath.Join(dir, "sources.yaml")
	os.WriteFile(list, []byte("sources:\n  - key: candle\n    from: "+
		server.URL+"/candle.pdf\n    licence: public domain\n"+
		"    sha256: "+pin+"\n"), 0o644)
	clock := func() time.Time {
		return time.Date(2026, 10, 16, 0, 0, 0, 0,
			time.UTC)
	}
	at := place{list: list, now: clock,
		shelf: sources.NewShelf(filepath.Join(dir, "shelf"),
			server.Client())}
	return at, func() { server.Close(); os.RemoveAll(dir) }
}

// Fetch with --dry-run says what it would bring onto the shelf, and the
// shelf stays empty.
func Example_fetch() {
	at, done := demo("")
	defer done()
	var out bytes.Buffer
	code := tool(at).Run([]string{"fetch", "--dry-run"}, &out, &out)
	said, _, _ := strings.Cut(out.String(), "http")
	list, _ := sources.Read(at.list)
	candle, _ := list.Find("candle")
	_, err := os.Stat(at.shelf.Path(candle))
	fmt.Println(code, said, os.IsNotExist(err))
	// Output: 0 candle: would fetch  true
}

// Fetch brings an unpinned source onto the shelf and says its hash, to
// be pinned in the list.
func TestFetchBringsAnUnpinnedSourceAndSaysItsHash(t *testing.T) {
	at, done := demo("")
	defer done()
	var out bytes.Buffer
	tool(at).Run([]string{"fetch"}, &out, &out)
	const want = "candle: fetched, not pinned: sha256 " +
		"38539f799bc2524c8c517f79aa1bd9b2e414601bf578caa4ed775c6aef670fbb, " +
		"21 bytes\n"
	if out.String() != want {
		t.Errorf("fetch said %q, want %q", out.String(), want)
	}
}

// Check holds a pinned source to its hash; one not yet fetched is said
// to be missing, and the verb fails.
func Example_check() {
	at, done := demo(strings.Repeat("a", 64))
	defer done()
	var out bytes.Buffer
	code := tool(at).Run([]string{"check"}, &out, &out)
	fmt.Println(code, strings.Contains(out.String(), "not on the shelf"))
	// Output: 1 true
}

// Path says where a source's bytes are, under the shelf, by its key.
func Example_path() {
	at, done := demo("")
	defer done()
	var out bytes.Buffer
	tool(at).Run([]string{"path", "candle"}, &out, &out)
	fmt.Println(filepath.Base(strings.TrimSpace(out.String())))
	// Output: candle.pdf
}

// every verb is declared soundly and its example holds its output.
func TestTheExamplesHere(t *testing.T) {
	at, done := demo("")
	defer done()
	if err := tool(at).Examples(os.DirFS(".")); err != nil {
		t.Error(err)
	}
}

// fetch takes the keys it is given, refuses one not in the list, and
// path wants exactly one.
func TestTheVerbsTakeKeys(t *testing.T) {
	at, done := demo("")
	defer done()
	var out bytes.Buffer
	if code := tool(at).Run([]string{"fetch", "nothing"}, &out,
		&out); code != 1 || !strings.Contains(out.String(), "not in the list") {
		t.Errorf("fetch of an unknown key gave %d: %s", code, out.String())
	}
	out.Reset()
	if code := tool(at).Run([]string{"path"}, &out, &out); code != 1 {
		t.Errorf("path of nothing gave %d: %s", code, out.String())
	}
}
