//go:build draw

package cheeptune

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCookWritesOnceAndKeepsItsName(t *testing.T) {
	dir := t.TempDir()
	p, err := Open("@crypt", "@crypt-dmg")
	if err != nil {
		t.Fatal(err)
	}
	path, err := p.Cook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(path), "crypt-crypt-dmg-") {
		t.Errorf("cooked as %q; the name should say what it is",
			filepath.Base(path))
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() < 44 {
		t.Errorf("cooked file is %d bytes", st.Size())
	}
	// cooking again is the same file, untouched: a game may keep the
	// directory between runs.
	again, err := p.Cook(dir)
	if err != nil || again != path {
		t.Fatalf("second cook gave %q (%v), want %q", again, err, path)
	}
	st2, _ := os.Stat(path)
	if !st2.ModTime().Equal(st.ModTime()) {
		t.Error("the second cook rewrote the file")
	}
	// a voicing changed by one number is a different file, so an old
	// render cannot be served for a new one.
	vb, err := readDoc("@crypt-dmg")
	if err != nil {
		t.Fatal(err)
	}
	quieter := strings.Replace(string(vb), `"volume": 0.85`,
		`"volume": 0.8`, 1)
	vp := filepath.Join(dir, "crypt-dmg2.json")
	if err := os.WriteFile(vp, []byte(quieter), 0o644); err != nil {
		t.Fatal(err)
	}
	q, err := Open("@crypt", vp)
	if err != nil {
		t.Fatal(err)
	}
	other, err := q.Cook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if other == path {
		t.Error("two voicings cooked to one file")
	}
}

func TestACookedFileIsAWav(t *testing.T) {
	p, err := Open("@leap", "@leap-dmg")
	if err != nil {
		t.Fatal(err)
	}
	path, err := p.Cook(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		t.Errorf("header is %q, not a wav", b[0:12])
	}
	if got := int(b[22]); got != 2 {
		t.Errorf("%d channels, want 2", got)
	}
	rate := int(b[24]) | int(b[25])<<8 | int(b[26])<<16 | int(b[27])<<24
	if rate != outRate {
		t.Errorf("%d samples a second, want %d", rate, outRate)
	}
	if n := len(b) - 44; n != len(p.Left)*4 {
		t.Errorf("%d bytes of samples, want %d", n, len(p.Left)*4)
	}
}
