package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/raw"
)

// small is a quick hillside to prove: two stones and a fire at night,
// drawn small, so the fire's light is in the picture.
func small() []string {
	return append([]string{"-w", "48", "-h", "30", "-n", "1", "-night"},
		only(map[string]string{"boulder": "2", "campfire": "1"})...)
}

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

// -pixel is taken once or many times, in order, and -many with it;
// both are kept as asked.
func TestThePixelsAndHowManyAreTaken(t *testing.T) {
	given := mustParse(t, "-prove", "a.pfm", "-pixel", "3,4", "-pixel",
		" 0 , 0", "-many", "5")
	if want := (named{{3, 4}, {0, 0}}); !reflect.DeepEqual(given.pixels,
		want) || given.many != 5 {
		t.Errorf("pixels %v and many %d, not %v and 5", given.pixels,
			given.many, want)
	}
	if said := given.pixels.String(); said != "3,4;0,0" {
		t.Errorf("the pixels say %q", said)
	}
	if plain := mustParse(t); plain.pixels != nil ||
		plain.many != fermat.Many {
		t.Errorf("unasked, pixels %v and many %d", plain.pixels, plain.many)
	}
}

// a pixel or a count that cannot be proved is refused by its flag: one that is
// not X,Y, one outside the picture, either without -prove, a count less than
// zero, and a proof of nothing at all.
func TestWhatCannotBeProvedIsRefusedAtTheFlags(t *testing.T) {
	refused := map[string][][]string{
		"-pixel": {{"-prove", "a.pfm", "-pixel", "3"},
			{"-prove", "a.pfm", "-pixel", "a,b"},
			{"-prove", "a.pfm", "-pixel", "96,0"},
			{"-prove", "a.pfm", "-pixel", "0,-1"},
			{"-pixel", "3,4"}},
		"-many": {{"-prove", "a.pfm", "-many", "-1"},
			{"-prove", "a.pfm", "-many", "0"},
			{"-many", "5"}},
	}
	for flag, cases := range refused {
		for _, arguments := range cases {
			_, err := parse(append([]string{"-w", "96", "-h", "60"},
				arguments...), io.Discard)
			if err == nil || !strings.Contains(err.Error(), flag[1:]) {
				t.Errorf("%v: %v, naming %s", arguments, err, flag)
			}
		}
	}
	_, err := parse([]string{"-prove", "a.pfm", "-many", "0", "-pixel",
		"1,1"}, io.Discard)
	if err != nil {
		t.Errorf("a named pixel and no picks was refused: %v", err)
	}
	if _, err := parse([]string{"-pixel", "3,4"}, io.Discard); !errors.Is(err,
		errAsked) {
		t.Errorf("-pixel without -prove: %v, not %v", err, errAsked)
	}
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
