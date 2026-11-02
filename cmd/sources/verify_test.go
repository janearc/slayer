package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/sources"
)

// sheet is the one paper verify's examples read: a text source, so no
// page of it needs poppler.
const sheet = "the candle burns down 6.84 millimetres an hour"

// paper is a server's one paper, sent whatever is asked.
type paper string

// ServeHTTP sends the paper.
func (body paper) ServeHTTP(answer http.ResponseWriter, _ *http.Request) {
	answer.Write([]byte(body))
}

// held is research data already read, handed to a place.
type held numbers.Numbers

// read is the data, as a place asks for it.
func (data held) read() (numbers.Numbers, error) {
	return numbers.Numbers(data), nil
}

// rig is what a verify example stands on: a server and a directory,
// taken away when it is done.
type rig struct {
	server *httptest.Server
	dir    string
}

// done takes the rig away.
func (made rig) done() {
	made.server.Close()
	os.RemoveAll(made.dir)
}

// verifying is a place for verify: the sheet served and pinned in the
// list, the shelf in a fresh directory, and research data quoting the
// sheet's text as quoted and holding value for it.
func verifying(quoted, value string) (place, rig) {
	made := rig{server: httptest.NewServer(paper(sheet))}
	made.dir, _ = os.MkdirTemp("", "sources-verify")
	sum := sha256.Sum256([]byte(sheet))
	list := filepath.Join(made.dir, "sources.yaml")
	os.WriteFile(list, []byte("sources:\n  - key: candle\n    from: "+
		made.server.URL+"/candle.txt\n    licence: public domain\n"+
		"    sha256: "+hex.EncodeToString(sum[:])+"\n"), openFile)
	data, _ := numbers.Read("research.yml", []byte("numbers:\n"+
		"  - name: burn\n    value: "+value+"\n"+
		"    says: how fast the candle burns down\n    kind: measured\n"+
		"    source:\n      key: candle\n      where: its one line\n"+
		"      text: \""+quoted+"\"\n"))
	shelf := sources.NewShelf(filepath.Join(made.dir, "shelf"),
		made.server.Client())
	return place{list: list, shelf: shelf, now: time.Now,
		research: held(data).read}, made
}

// openFile is the mode the list is written with: the owner writes and
// anyone reads.
const openFile = 0o644

// Verify holds each measured number to its source: on the shelf as
// pinned, its text there, and saying the number.
func Example_verify() {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	var out bytes.Buffer
	tool(at).Run([]string{"fetch"}, &out, &out)
	out.Reset()
	code := tool(at).Run([]string{"verify"}, &out, &out)
	fmt.Print(out.String())
	fmt.Println(code)
	// Output:
	// burn: 6.84, in candle
	// 0
}

// verified is what verify says of the rig's number, quoted and holding
// value, its source fetched first or not.
func verified(quoted, value string, fetched bool) string {
	at, made := verifying(quoted, value)
	defer made.done()
	var out bytes.Buffer
	if fetched {
		tool(at).Run([]string{"fetch"}, &out, &out)
	}
	out.Reset()
	tool(at).Run([]string{"verify"}, &out, &out)
	return out.String()
}

func TestANumberNotOnItsPageFails(t *testing.T) {
	said := verified("7.84", "7.84", true)
	if !strings.Contains(said, `"7.84" is not in candle`) {
		t.Errorf("text not on the page: %s", said)
	}
}

func TestANumberItsTextDoesNotSayFails(t *testing.T) {
	said := verified("6.84", "6.8", true)
	if !strings.Contains(said, "the page says 6.84, the data holds 6.8") {
		t.Errorf("a value the text does not say: %s", said)
	}
}

func TestANumberWhoseSourceIsNotOnTheShelfFails(t *testing.T) {
	said := verified("6.84", "6.84", false)
	if !strings.Contains(said, "not on the shelf") {
		t.Errorf("a source never fetched: %s", said)
	}
}
