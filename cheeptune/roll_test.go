package cheeptune

import (
	"image/color"
	"math"
	"testing"
)

// luminance and contrast are WCAG 2.1's arithmetic. The house band is 10 to
// 12.5 to one for text, and a colour that misses it changes; the band does
// not.
func luminance(c color.RGBA) float64 {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

func contrast(a, b color.RGBA) float64 {
	la, lb := luminance(a), luminance(b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

func TestTheRollsTextIsInTheBand(t *testing.T) {
	// text is only ever drawn on the ground, so the ground is the one
	// surface to measure against.
	if r := contrast(rollInk, rollGround); r < 10 || r > 12.5 {
		t.Errorf("ink on ground is %.2f to one; the band is 10 to 12.5", r)
	}
}
