package explain

import (
	"fmt"
	"image/color"
	"math"
)

// Tinted is the panel's ground leaning toward a colour, given as linear
// srgb from 0 to 1, by a share from 0 (the hud theme's ground) to 1. The
// ground is then lightened or darkened until the ink stands on it at the
// theme's own contrast, so the text stays in the legibility band
// whatever the colour; a tint that would bring the ground's hue within
// hueApart of the ink's while saturated is halved until it does not.
func Tinted(tint [3]float64, share float64) (color.RGBA, error) {
	tones, err := colours()
	if err != nil {
		return color.RGBA{}, err
	}
	ink, base := tones["ink"], tones["ground"]
	target := (luminance(ink)+flare)/Contrast(ink, base) - flare
	share = clamp(share)
	for range tintTries {
		ground := atLuminance(mixLinear(linear(base), tint, share), target)
		if Legible(ground) == nil {
			return ground, nil
		}
		share /= 2
	}
	return base, nil
}

// Legible is nil when the hud theme's ink reads on a ground: its
// contrast inside the band, and the two apart in hue or the ground
// greyed. Otherwise it says which rule the ground breaks.
func Legible(ground color.RGBA) error {
	tones, err := colours()
	if err != nil {
		return err
	}
	ink := tones["ink"]
	ratio := Contrast(ink, ground)
	if ratio < bandLeast || ratio > bandMost {
		return fmt.Errorf("%w: ink on %v is %.2f to 1; the band is %v to "+
			"%v", ErrExplain, ground, ratio, bandLeast, bandMost)
	}
	apart := math.Abs(hue(ink) - hue(ground))
	apart = math.Min(apart, fullCircle-apart)
	if apart < hueApart && saturation(ground) >= groundGrey {
		return fmt.Errorf("%w: %v is %.0f degrees from the ink's hue and "+
			"%.2f saturated; it needs %v degrees or under %v", ErrExplain,
			ground, apart, saturation(ground), hueApart, groundGrey)
	}
	return nil
}

// Contrast is the contrast ratio of two colours, the lighter's over the
// darker's relative luminance, each with WCAG's flare.
func Contrast(one, other color.RGBA) float64 {
	light, dark := luminance(one), luminance(other)
	if dark > light {
		light, dark = dark, light
	}
	return (light + flare) / (dark + flare)
}

// luminance is a colour's relative luminance, 0 to 1.
func luminance(of color.RGBA) float64 {
	lin := linear(of)
	return lumaRed*lin[0] + lumaGreen*lin[1] + lumaBlue*lin[2]
}

// linear is a colour's srgb bytes as linear light, 0 to 1.
func linear(of color.RGBA) [3]float64 {
	decode := func(level uint8) float64 {
		value := float64(level) / byteMost
		if value <= srgbKnee {
			return value / srgbLinear
		}
		return math.Pow((value+srgbOffset)/srgbScale, srgbGamma)
	}
	return [3]float64{decode(of.R), decode(of.G), decode(of.B)}
}

// encoded is linear light as srgb bytes.
func encoded(light [3]float64) color.RGBA {
	encode := func(value float64) uint8 {
		value = clamp(value)
		if value <= srgbOut {
			value *= srgbLinear
		} else {
			value = srgbScale*math.Pow(value, 1/srgbGamma) - srgbOffset
		}
		return uint8(math.Round(clamp(value) * byteMost))
	}
	return color.RGBA{encode(light[0]), encode(light[1]), encode(light[2]),
		math.MaxUint8}
}

// mixLinear is one light toward another by a share.
func mixLinear(from, to [3]float64, share float64) [3]float64 {
	var made [3]float64
	for i := range made {
		made[i] = from[i] + (to[i]-from[i])*share
	}
	return made
}

// atLuminance is a light scaled to a relative luminance, its colour
// kept, as srgb bytes.
func atLuminance(light [3]float64, target float64) color.RGBA {
	now := lumaRed*light[0] + lumaGreen*light[1] + lumaBlue*light[2]
	if now <= 0 {
		return encoded([3]float64{target, target, target})
	}
	scale := target / now
	return encoded([3]float64{light[0] * scale, light[1] * scale,
		light[2] * scale})
}

// hue is a colour's hue in degrees, nought for a grey.
func hue(of color.RGBA) float64 {
	r, g, b := float64(of.R), float64(of.G), float64(of.B)
	most, least := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	span := most - least
	if span == 0 {
		return 0
	}
	var turn float64
	switch most {
	case r:
		turn = math.Mod((g-b)/span+fullCircle/sextant, fullCircle/sextant)
	case g:
		turn = (b-r)/span + greenTurn
	default:
		turn = (r-g)/span + blueTurn
	}
	return turn * sextant
}

// saturation is a colour's saturation, its spread over its brightest
// channel, 0 to 1.
func saturation(of color.RGBA) float64 {
	most := math.Max(float64(of.R), math.Max(float64(of.G), float64(of.B)))
	least := math.Min(float64(of.R), math.Min(float64(of.G), float64(of.B)))
	if most == 0 {
		return 0
	}
	return (most - least) / most
}
