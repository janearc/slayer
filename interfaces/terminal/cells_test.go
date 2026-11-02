package terminal

import (
	"image"
	"image/color"
	"testing"
)

func TestGlyphsRunInPatternOrder(t *testing.T) {
	for _, c := range []struct {
		p    uint8
		want rune
	}{
		{0, ' '}, {63, '█'}, {21, '▌'}, {42, '▐'},
		{1, 0x1FB00}, {20, 0x1FB13}, {22, 0x1FB14},
		{41, 0x1FB27}, {43, 0x1FB28}, {62, 0x1FB3B},
	} {
		if got := Glyph(c.p); got != c.want {
			t.Errorf("Glyph(%d) = %U, want %U", c.p, got, c.want)
		}
	}
}

// every pattern gets its own character: 64 patterns, 64 glyphs.
func TestNoTwoPatternsShareAGlyph(t *testing.T) {
	seen := map[rune]uint8{}
	for p := uint8(0); p < 64; p++ {
		g := Glyph(p)
		if q, dup := seen[g]; dup {
			t.Fatalf("patterns %d and %d both draw %U", q, p, g)
		}
		seen[g] = p
	}
}

func TestTwoColoursComeBackExactly(t *testing.T) {
	red, blue := color.RGBA{200, 30, 30, 255}, color.RGBA{20, 40, 220, 255}
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	// the right column and the bottom left
	for i := 0; i < 6; i++ {
		c := blue
		if i%2 == 1 || i == 4 {
			c = red
		}
		img.SetRGBA(i%2, i/2, c)
	}
	cell := Chop(img).At(0, 0)
	want := uint8(1<<1 | 1<<3 | 1<<4 | 1<<5)
	if cell.Pattern == want {
		if cell.Fg != red || cell.Bg != blue {
			t.Fatalf("fg %v bg %v", cell.Fg, cell.Bg)
		}
	} else if cell.Pattern == ^want&63 {
		if cell.Fg != blue || cell.Bg != red {
			t.Fatalf("fg %v bg %v", cell.Fg, cell.Bg)
		}
	} else {
		t.Fatalf("pattern %06b, want %06b or its opposite", cell.Pattern, want)
	}
}

func TestOneColourIsASpace(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	for i := 0; i < 6; i++ {
		img.SetRGBA(i%2, i/2, color.RGBA{90, 90, 90, 255})
	}
	if c := Chop(img).At(0, 0); c.Rune != ' ' || c.Bg != (color.RGBA{90, 90, 90, 255}) {
		t.Fatalf("got %q on %v", c.Rune, c.Bg)
	}
}
