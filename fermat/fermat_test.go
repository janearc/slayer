package fermat

import (
	"crypto/sha256"
	"image"
	"os"
	"path/filepath"
	"testing"
)

// the picks are the raw's to make: the same bytes pick the same pixels,
// each once and inside the picture; other bytes pick others; a picture
// no bigger than the picks gives every pixel, and none gives none.
func TestPicksAreTheRawsOwn(t *testing.T) {
	sum := sha256.Sum256([]byte("a raw"))
	first, again := Picks(sum, 40, 30, Many), Picks(sum, 40, 30, Many)
	seen := map[[2]int]bool{}
	for i, at := range first {
		if at != again[i] || seen[at] || at[0] < 0 || at[0] >= 40 ||
			at[1] < 0 || at[1] >= 30 {
			t.Fatalf("pick %d: %v, again %v, seen %v", i, at, again[i],
				seen[at])
		}
		seen[at] = true
	}
	other := Picks(sha256.Sum256([]byte("another raw")), 40, 30, Many)
	if len(first) != Many || first[0] == other[0] && first[1] == other[1] {
		t.Errorf("%d picks; other bytes picked %v, %v first", len(first),
			other[0], other[1])
	}
	if every := Picks(sum, 5, 4, Many); len(every) != 20 {
		t.Errorf("a picture of 20 gave %d picks", len(every))
	}
	if Picks(sum, 0, 4, Many) != nil || Picks(sum, 5, 4, 0) != nil {
		t.Error("nothing to pick picked something")
	}
}

// a raw that is not there, or not a raw, is an error, not a proof.
func TestAProofNeedsARaw(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone.pfm")
	if _, err := Prove(missing, "", nil, Many, nil); err == nil {
		t.Error("a missing raw was proved")
	}
	junk := filepath.Join(t.TempDir(), "junk.pfm")
	os.WriteFile(junk, []byte("not a raw\n"), 0o644)
	if _, err := Prove(junk, "", nil, Many, nil); err == nil {
		t.Error("a file that is not a raw was proved")
	}
}

// a picture is marked about each picked pixel, green where it holds and
// red where it fails, and the pixel itself is left as it was.
func TestTheMarksSayWhichHeld(t *testing.T) {
	proof := Proof{Pixels: []Pixel{{X: 4, Y: 4}, {X: 0, Y: 0,
		Err: os.ErrNotExist}}}
	picture := image.NewRGBA(image.Rect(0, 0, 10, 10))
	proof.Mark(picture)
	if green := picture.RGBAAt(7, 4); green.G != 220 || green.R != 0 {
		t.Errorf("a held pixel's mark is %v", green)
	}
	if red := picture.RGBAAt(3, 0); red.R != 255 || red.G != 0 {
		t.Errorf("a failed pixel's mark is %v", red)
	}
	if centre := picture.RGBAAt(4, 4); centre.A != 0 {
		t.Errorf("the held pixel itself was drawn on: %v", centre)
	}
}
