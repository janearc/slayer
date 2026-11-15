//go:build draw

package hillside

import (
	"bytes"
	"crypto/sha256"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/raw"
)

// drawnRaw runs the command on the small hillside, keeps its raw in a
// fresh directory and says where.
func drawnRaw(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hill.pfm")
	arguments := append(small(), "-out", filepath.Join(dir, "hill.png"),
		"-raw", path)
	var wrong bytes.Buffer
	if code := run(arguments, io.Discard, &wrong); code != 0 {
		t.Fatalf("drawing gave %d: %s", code, wrong.String())
	}
	return path
}

// proved runs -prove on the raw with the small hillside's flags and
// these, and gives back what it said and its code.
func proved(t *testing.T, path string, more ...string) (string, int) {
	t.Helper()
	arguments := append(append(small(), "-prove", path), more...)
	var said, wrong bytes.Buffer
	code := run(arguments, &said, &wrong)
	return said.String() + wrong.String(), code
}

// the command proves its own raw: the named pixel first, then as many as asked,
// every one made again bit for bit, with a sheet beside it.
func TestTheCommandProvesItsOwnRaw(t *testing.T) {
	path := drawnRaw(t)
	said, code := proved(t, path, "-pixel", "20,18", "-many", "6")
	if code != 0 || !strings.Contains(said, "holds: 7 of 7") {
		t.Fatalf("gave %d: %s", code, said)
	}
	sheet := strings.TrimSuffix(path, ".pfm") + "-proof"
	for _, name := range []string{"proof.md", "pixel-20-18.md",
		"marked.png"} {
		if _, err := os.Stat(filepath.Join(sheet, name)); err != nil {
			t.Error(err)
		}
	}
}

// the plan's line: a pixel of the hillside's raw changed by the least a
// float can change fails the proof where it is named. left unnamed, it
// is caught only if the raw's bytes happen to pick it, and this test
// says which they did, so the picks are never mistaken for a search.
func TestAChangedPixelOfTheHillsideFailsWhereNamed(t *testing.T) {
	path := drawnRaw(t)
	kept, err := raw.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	const x, y = 20, 18
	changed := kept.At(x, y)
	changed.R = math.Nextafter32(changed.R, float32(math.Inf(1)))
	kept.Set(x, y, changed)
	if err := raw.Write(path, kept); err != nil {
		t.Fatal(err)
	}
	said, code := proved(t, path, "-pixel", "20,18", "-many", "6")
	if code != 1 || !strings.Contains(said, "fails: 6 of 7") {
		t.Fatalf("named, gave %d: %s", code, said)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	picked := false
	for _, at := range fermat.Picks(sha256.Sum256(bytes), kept.W, kept.H,
		6) {
		picked = picked || at == [2]int{x, y}
	}
	said, code = proved(t, path, "-many", "6")
	if caught := code == 1; caught != picked {
		t.Errorf("unnamed and picked %v, the proof gave %d: %s", picked,
			code, said)
	}
}
