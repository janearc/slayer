package session

import (
	"os"
	"path/filepath"
	"testing"
)

// Places writes the table the staging kept: a header, then a line for
// each thing, the name quoted and east, north and height to a tenth of a
// metre. The stone's height, 565.5625, is worked in a hand: its tenths
// are 565.5, and the next digit is 6, so it prints 565.6. The folder of
// the file is made.
func TestPlacesWritesTheTableTheStagingKept(t *testing.T) {
	kept := &Session{given: asked{stoodKept: true, stood: []standing{
		{of: "the stone", kind: "cobble", east: 1050, north: 8990,
			height: 565.5625}}}}
	path := filepath.Join(t.TempDir(), "out", "places.csv")
	if err := kept.Places(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "of,kind,east,north,height\n" +
		"\"the stone\",cobble,1050.0,8990.0,565.6\n"
	if string(got) != want {
		t.Errorf("the table is %q, want %q", got, want)
	}
}

// the anti-test: a table with nothing stood is the header alone; a
// session whose staging stood nothing on a ground is refused and makes
// no folder or file; a closed session is refused.
func TestPlacesRefusesWhatStoodNothingOnAGround(t *testing.T) {
	empty := &Session{given: asked{stoodKept: true}}
	path := filepath.Join(t.TempDir(), "empty.csv")
	if err := empty.Places(path); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "of,kind,east,north,"+
		"height\n" {
		t.Errorf("an empty table is %q", got)
	}
	folder := filepath.Join(t.TempDir(), "never")
	unplaced := &Session{}
	if err := unplaced.Places(filepath.Join(folder, "p.csv")); err !=
		errUnplaced {
		t.Errorf("a scene that stood nothing gave %v", err)
	}
	if _, err := os.Stat(folder); err == nil {
		t.Error("a refused table made its folder")
	}
	empty.closed = true
	if err := empty.Places(path); err != errClosed {
		t.Errorf("a closed session gave %v", err)
	}
}
