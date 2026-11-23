//go:build draw

package cheeptune

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// dusk is one rendered piece the tests below share, so that a render is
// paid for once.
var dusk = sync.OnceValues(func() (*Piece, error) {
	return Open("@dusk", "@dmg-pad")
})

func TestACookedPieceThatWasRenderedNotOpenedIsNamedForItsSamples(t *testing.T) {
	s, err := ReadScore("@dusk")
	if err != nil {
		t.Fatal(err)
	}
	v, err := ReadVoicing("@dmg-pad")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Render(s, v)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	first, err := p.Cook(dir)
	if err != nil {
		t.Fatal(err)
	}
	again, err := p.Cook(dir)
	if err != nil || again != first {
		t.Errorf("a second cook moved the file: %v %s %s", err, first, again)
	}
	if !strings.Contains(filepath.Base(first), "dusk-dmg-pad-") {
		t.Errorf("the name %s does not hold the score and voicing", first)
	}
	if _, err := p.Cook(filepath.Join(first, "inside")); err == nil {
		t.Error("a cook into a file was accepted")
	}
}

func TestAPieceCannotBeWrittenWhereNothingCanBeCreated(t *testing.T) {
	p, err := dusk()
	if err != nil {
		t.Fatal(err)
	}
	nowhere := filepath.Join(t.TempDir(), "no", "such", "dir", "x")
	if err := p.WriteWAV(nowhere + ".wav"); err == nil {
		t.Error("a wav was written into a directory that is not there")
	}
	if err := p.WriteRoll(nowhere + ".png"); err == nil {
		t.Error("a roll was written into a directory that is not there")
	}
}

func TestARollIsWrittenAsAPNG(t *testing.T) {
	p, err := dusk()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dusk.png")
	if err := p.WriteRoll(path); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("\x89PNG")) {
		t.Errorf("the file starts %q", b[:4])
	}
}
