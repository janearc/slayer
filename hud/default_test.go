package hud

import (
	"image/color"
	"math"
	"testing"
)

// the band the house holds body text to, ink on ground, as a ratio of
// luminances. It is picked: 10 to 1 up to 12.5 to 1.
const (
	bandLow  = 10.0
	bandHigh = 12.5
)

// linear is an srgb byte as linear light, by the srgb transfer curve.
func linear(code uint8) float64 {
	share := float64(code) / 255
	if share <= 0.04045 {
		return share / 12.92
	}
	return math.Pow((share+0.055)/1.055, 2.4)
}

// luminance is the relative luminance of a colour, as the web's
// contrast guidance defines it.
func luminance(c color.RGBA) float64 {
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}

// ratio is the contrast of two colours: the lighter's luminance and a
// flare of 0.05 over the darker's and the same.
func ratio(a, b color.RGBA) float64 {
	high, low := luminance(a), luminance(b)
	if low > high {
		high, low = low, high
	}
	return (high + 0.05) / (low + 0.05)
}

// the default theme's body text, ink on ground, is inside the house
// band, and the ink is the lighter of the two.
func TestTheDefaultThemeHoldsTheHouseBand(t *testing.T) {
	theme := DefaultTheme()
	ink, err := theme.Colour("ink")
	if err != nil {
		t.Fatal(err)
	}
	ground, err := theme.Colour("ground")
	if err != nil {
		t.Fatal(err)
	}
	got := ratio(ink, ground)
	if got < bandLow || got > bandHigh {
		t.Errorf("ink on ground is %.2f to 1, want %v to %v", got, bandLow,
			bandHigh)
	}
	if luminance(ink) <= luminance(ground) {
		t.Error("the default theme is dark ink on a light ground")
	}
}

// the anti-test: the ratio computed here sees a pair outside the band,
// twilight's own 8.9 to 1 and a pair too stark.
func TestTheBandRefusesPairsOutsideIt(t *testing.T) {
	navy, amber := rgba("#051759"), rgba("#ffac4e")
	if got := ratio(navy, amber); got >= bandLow {
		t.Errorf("twilight is %.2f to 1; it is below the band", got)
	}
	if got := ratio(rgba("#ffffff"), rgba("#000000")); got <= bandHigh {
		t.Errorf("white on black is %.2f to 1; it is above the band", got)
	}
}

// every role the spec names is there, and each is a colour other than
// the ground it sits on, so it is seen.
func TestTheDefaultThemeHasItsRoles(t *testing.T) {
	theme := DefaultTheme()
	ground, _ := theme.Colour("ground")
	for _, role := range []string{"ink", "dim", "border", "ok", "warn",
		"err"} {
		got, err := theme.Colour(role)
		if err != nil {
			t.Errorf("role %s: %v", role, err)
			continue
		}
		if ratio(got, ground) < 3 {
			t.Errorf("role %s is %.2f to 1 on the ground", role,
				ratio(got, ground))
		}
	}
	if _, err := theme.Colour("no-such-role"); err == nil {
		t.Error("a role the theme has not was found")
	}
}
