package typography

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// go's own fonts come by name, each the font it names: go's bold "o"
// is wider than its regular one. a font file comes by its path from the
// directory given, or by its own when absolute; a file that is not
// there, or is not a font, is an error that says so.
func TestAFontComesByNameOrByFile(t *testing.T) {
	regular, err := Named("go", "")
	if err != nil {
		t.Fatal(err)
	}
	bold, err := Named("go-bold", "")
	if err != nil {
		t.Fatal(err)
	}
	if width(t, bold, 'o') <= width(t, regular, 'o') {
		t.Error("go-bold's o is no wider than go's")
	}
	here, _ := filepath.Abs("testdata")
	for _, each := range [][2]string{{"CFFTest.otf", "testdata"},
		{filepath.Join(here, "CFFTest.otf"), "/nowhere"}} {
		face, err := Named(each[0], each[1])
		if err != nil || face.Units() != 1000 {
			t.Errorf("%s from %s: %v", each[0], each[1], err)
		}
	}
	if _, err := Named("missing.ttf", "testdata"); !errors.Is(err,
		os.ErrNotExist) || !errors.Is(err, ErrType) {
		t.Errorf("a font not there: %v", err)
	}
	if _, err := Named("README", "testdata"); !errors.Is(err, ErrType) {
		t.Errorf("a file that is not a font: %v", err)
	}
}

// width is how far a letter moves the pen.
func width(t *testing.T, face *Font, letter rune) float64 {
	t.Helper()
	glyph, err := face.Index(letter)
	if err != nil {
		t.Fatal(err)
	}
	advance, err := face.Advance(glyph)
	if err != nil {
		t.Fatal(err)
	}
	return advance
}
