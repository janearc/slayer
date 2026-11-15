package hud

import (
	"bytes"
	"image"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// a face from the font parsed once draws the same pixels as a face from a
// font parsed afresh, at several sizes.
func TestTheParsedFontDrawsAsAFreshOneDoes(t *testing.T) {
	fresh, err := opentype.Parse(gomono.TTF)
	if err != nil {
		t.Fatal(err)
	}
	for _, high := range []int{4, 10, 17, 32} {
		want, err := opentype.NewFace(fresh, &opentype.FaceOptions{
			Size: float64(high) * 0.8, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			t.Fatal(err)
		}
		got, err := mono(high)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range "aZ09#%@gj|_" {
			a, ma, ga, aa, oka := want.Glyph(fixedPt(), r)
			b, mb, gb, ab, okb := got.Glyph(fixedPt(), r)
			if oka != okb || a != b || aa != ab ||
				!bytes.Equal(pix(ma), pix(mb)) || ga != gb {
				t.Errorf("high %d rune %q: glyphs differ", high, r)
			}
		}
	}
}

func pix(m image.Image) []byte {
	if a, ok := m.(*image.Alpha); ok {
		return a.Pix
	}
	return nil
}

func fixedPt() fixed.Point26_6 { return fixed.Point26_6{} }
