package workspace

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// TestLargePictureIsShownReducedWithItsTrueSize checks that a picture over
// the limit is reduced under it, keeps its mean colour and names its size.
func TestLargePictureIsShownReducedWithItsTrueSize(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	const limit = 400
	reduced, note := reduceForViewer(source, limit)
	size := reduced.Bounds()
	if size.Dx()*size.Dy() > limit || size.Dx() == 0 || size.Dy() == 0 {
		t.Fatalf("reduced to %v; want under %d pixels", size, limit)
	}
	if !strings.Contains(note, "from 40 × 30") {
		t.Fatalf("note %q does not state the true size", note)
	}
	if got := color.NRGBAModel.Convert(reduced.At(0, 0)).(color.NRGBA); got != (color.NRGBA{R: 200, G: 100, B: 50, A: 255}) {
		t.Fatalf("a flat colour became %v", got)
	}
}

// TestPictureWithinTheLimitIsUntouched is the anti-test: a picture that fits
// is returned as it is, with no note.
func TestPictureWithinTheLimitIsUntouched(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	reduced, note := reduceForViewer(source, 200)
	if reduced != image.Image(source) || note != "" {
		t.Fatalf("a picture at the limit was changed: note %q", note)
	}
}
