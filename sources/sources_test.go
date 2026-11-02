package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// paper is the bytes the test server sends for a source.
const paper = "a paper, as published"

// paperSum is its sha256, worked out here.
func paperSum() string {
	sum := sha256.Sum256([]byte(paper))
	return hex.EncodeToString(sum[:])
}

// served is a server sending the paper at /paper.pdf, 404 elsewhere,
// and how many times it was asked.
func served(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	asked := 0
	server := httptest.NewServer(http.HandlerFunc(func(
		answer http.ResponseWriter, request *http.Request) {
		asked++
		if request.URL.Path != "/paper.pdf" {
			http.NotFound(answer, request)
			return
		}
		answer.Write([]byte(paper))
	}))
	t.Cleanup(server.Close)
	return server, &asked
}

// listed is a list read from yaml written to a fresh file.
func listed(t *testing.T, text string) (List, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sources.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return Read(path)
}

// one is a list of one source from the server, pinned as asked.
func one(t *testing.T, server *httptest.Server, pin string) Source {
	t.Helper()
	list, err := listed(t, "sources:\n  - key: paper\n    from: "+
		server.URL+"/paper.pdf\n    licence: public domain\n"+
		"    sha256: "+pin+"\n")
	if err != nil {
		t.Fatal(err)
	}
	return list.Sources()[0]
}

// a list that cannot be is refused, by what is wrong with it; a
// tombstone needs only its key and why.
func TestAListThatCannotBeIsRefused(t *testing.T) {
	bad := map[string]string{
		"an unknown key":  "sources:\n  - key: a\n    url: x\n",
		"a bad key":       "sources:\n  - key: A b\n    from: x\n    licence: y\n",
		"a key twice":     "sources:\n  - {key: a, from: x, licence: y}\n  - {key: a, from: x, licence: y}\n",
		"no licence":      "sources:\n  - {key: a, from: x}\n",
		"no from":         "sources:\n  - {key: a, licence: y}\n",
		"a short sha256":  "sources:\n  - {key: a, from: x, licence: y, sha256: abc}\n",
		"not yaml at all": "sources: [\n",
	}
	for what, text := range bad {
		if _, err := listed(t, text); err == nil {
			t.Errorf("%s was read", what)
		}
	}
	list, err := listed(t, "sources:\n  - {key: gone, never: withdrawn}\n")
	if err != nil || list.Sources()[0].Never() != "withdrawn" {
		t.Errorf("a tombstone: %v, %v", list, err)
	}
}

// a source with no pin is fetched and kept, its hash said for pinning,
// with a sidecar saying where it came from, and nothing left part done.
func TestAnUnpinnedSourceIsKeptAndItsHashSaid(t *testing.T) {
	server, _ := served(t)
	source := one(t, server, "")
	shelf := NewShelf(t.TempDir(), server.Client())
	when := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	said, err := shelf.Fetch(context.Background(), source, when)
	if err != nil || !strings.Contains(said, "not pinned: sha256 "+
		paperSum()) {
		t.Fatalf("said %q, %v", said, err)
	}
	at := shelf.Path(source)
	if kept, _ := os.ReadFile(at); string(kept) != paper ||
		filepath.Base(at) != "paper.pdf" {
		t.Errorf("%s holds %q", at, kept)
	}
	if _, err := os.Stat(at + ".part"); err == nil {
		t.Error("a part file was left")
	}
	text, _ := os.ReadFile(at + ".source")
	var about side
	if err := yaml.Unmarshal(text, &about); err != nil ||
		about.Sha256 != paperSum() || about.From != source.From() ||
		about.Fetched != "2026-10-16T20:00:00Z" || about.Bytes != 21 {
		t.Errorf("the sidecar says %+v, %v", about, err)
	}
}

// a pinned source is fetched once; asked again, the bytes on the shelf
// are kept without asking the server, and check finds them as pinned.
func TestAPinnedSourceIsFetchedOnce(t *testing.T) {
	server, asked := served(t)
	source := one(t, server, paperSum())
	shelf := NewShelf(t.TempDir(), server.Client())
	for range 2 {
		if _, err := shelf.Fetch(context.Background(), source,
			time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	said, err := shelf.Check(source)
	if *asked != 1 || err != nil || !strings.Contains(said, "as pinned") {
		t.Errorf("asked %d times; check said %q, %v", *asked, said, err)
	}
}

// bytes that are not what the list pins are refused, and nothing of
// them is kept on the shelf.
func TestBytesThatDoNotMatchThePinAreRefused(t *testing.T) {
	server, _ := served(t)
	source := one(t, server, strings.Repeat("0", 64))
	shelf := NewShelf(t.TempDir(), server.Client())
	_, err := shelf.Fetch(context.Background(), source, time.Now())
	if err == nil || !strings.Contains(err.Error(), "the list pins") {
		t.Fatalf("fetched against a wrong pin: %v", err)
	}
	left, _ := os.ReadDir(filepath.Dir(shelf.Path(source)))
	if len(left) != 0 {
		t.Errorf("left on the shelf: %v", left)
	}
}

// a source the server will not send, or one whose size is not the
// list's, is refused; a tombstone is never asked for.
func TestWhatCannotBeFetchedIsRefused(t *testing.T) {
	server, asked := served(t)
	shelf := NewShelf(t.TempDir(), server.Client())
	list, err := listed(t, "sources:\n"+
		"  - {key: lost, from: "+server.URL+"/lost.pdf, licence: y}\n"+
		"  - {key: short, from: "+server.URL+"/paper.pdf, licence: y,"+
		" bytes: 5}\n"+
		"  - {key: gone, from: "+server.URL+"/paper.pdf, never: withdrawn}\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range list.Sources()[:2] {
		if _, err := shelf.Fetch(context.Background(), source,
			time.Now()); err == nil {
			t.Errorf("%s was fetched", source.Key())
		}
	}
	before := *asked
	said, err := shelf.Fetch(context.Background(), list.Sources()[2],
		time.Now())
	if err != nil || *asked != before || !strings.Contains(said, "never") {
		t.Errorf("a tombstone: %q, %v, asked %d", said, err, *asked-before)
	}
}

// check says what is wrong: a source not on the shelf, one changed
// since it was fetched, and one not pinned.
func TestCheckSaysWhatIsWrong(t *testing.T) {
	server, _ := served(t)
	pinned := one(t, server, paperSum())
	shelf := NewShelf(t.TempDir(), server.Client())
	if _, err := shelf.Check(pinned); err == nil ||
		!strings.Contains(err.Error(), "not on the shelf") {
		t.Errorf("missing: %v", err)
	}
	shelf.Fetch(context.Background(), pinned, time.Now())
	os.WriteFile(shelf.Path(pinned), []byte("changed"), 0o644)
	if _, err := shelf.Check(pinned); err == nil ||
		!strings.Contains(err.Error(), "the list pins") {
		t.Errorf("changed: %v", err)
	}
	if _, err := shelf.Check(one(t, server, "")); err == nil {
		t.Error("an unpinned source checked")
	}
}

// the shelf is where the environment says, a whole path, and nowhere
// is guessed: unset or relative, it is refused, saying how to set it.
func TestTheShelfIsNeverGuessed(t *testing.T) {
	for _, given := range []string{"", "scratch/sources"} {
		t.Setenv(Placed, given)
		if _, err := Here(); err == nil ||
			!strings.Contains(err.Error(), Placed) {
			t.Errorf("%q: %v", given, err)
		}
	}
	t.Setenv(Placed, "/somewhere/scratch/sources")
	if dir, err := Here(); err != nil || dir != "/somewhere/scratch/sources" {
		t.Errorf("a whole path: %q, %v", dir, err)
	}
}

// a source's file keeps the ending its url gives when it is a kind of
// file, and none from a script's page.
func TestAPathKeepsAFilesEnding(t *testing.T) {
	shelf := NewShelf("/shelf", nil)
	for from, want := range map[string]string{
		"https://x.gov/a/paper.pdf":          "/shelf/k.pdf",
		"https://x.gov/get_pdf.cfm?pub_id=1": "/shelf/k",
		"https://x.gov/dem/kilauea-2019.tif": "/shelf/k.tif",
		"https://x.gov/noending":             "/shelf/k",
	} {
		got := shelf.Path(Source{row: row{Key: "k", From: from}})
		if got != want {
			t.Errorf("%s: %s, not %s", from, got, want)
		}
	}
}
