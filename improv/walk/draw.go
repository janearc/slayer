package walk

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"sync"

	"github.com/janearc/slayer/hud"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// palette is the hud default theme's colours by role.
type palette map[string]color.RGBA

// themeRoles are the roles the frame draws in: words in ink, the panel
// and plates in ground, the units and small words in dim, and the
// accent of the figures and leaders in ok.
var themeRoles = []string{"ink", "ground", "dim", "ok"}

// colours is the hud's default theme, by role.
var colours = sync.OnceValues(func() (palette, error) {
	return paletteOf(hud.DefaultTheme())
})

// paletteOf is a theme's colours for the roles the frame draws in.
func paletteOf(theme hud.Theme) (palette, error) {
	made := palette{}
	for _, role := range themeRoles {
		colour, err := theme.Colour(role)
		if err != nil {
			return nil, fmt.Errorf("%w: the hud theme: %w", ErrWalk, err)
		}
		made[role] = colour
	}
	return made, nil
}

// regular is Go's proportional face, parsed once.
var regular = sync.OnceValues(func() (*opentype.Font, error) {
	return opentype.Parse(goregular.TTF)
})

// face is Go's face at a height in pixels.
func face(size float64) (font.Face, error) {
	parsed, err := regular()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWalk, err)
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{Size: size,
		DPI: typeDots, Hinting: font.HintingNone})
}

// typeDots is the dots an inch the faces are set at, so a size is in
// pixels. Defined: 72 dots an inch is a point a pixel.
const typeDots = 72

// measure is how wide words are, in pixels, at a size.
func measure(words string, size float64) (int, error) {
	drawn, err := face(size)
	if err != nil {
		return 0, err
	}
	defer drawn.Close()
	return font.MeasureString(drawn, words).Ceil(), nil
}

// write draws words with their baseline's left end at a point.
func write(frame *image.RGBA, words string, at image.Point, size float64,
	ink color.RGBA) error {
	drawn, err := face(size)
	if err != nil {
		return err
	}
	defer drawn.Close()
	pen := font.Drawer{Dst: frame, Src: image.NewUniform(ink), Face: drawn,
		Dot: fixed.P(at.X, at.Y)}
	pen.DrawString(words)
	return nil
}

// wrapped is words broken into rows no wider than width, greedily.
func wrapped(words string, size float64, width int) ([]string, error) {
	var rows []string
	row := ""
	for _, word := range strings.Fields(words) {
		trial := strings.TrimSpace(row + " " + word)
		wide, err := measure(trial, size)
		if err != nil {
			return nil, err
		}
		if wide > width && row != "" {
			rows = append(rows, row)
			trial = word
		}
		row = trial
	}
	if row != "" {
		rows = append(rows, row)
	}
	return rows, nil
}

// fill paints a rectangle one colour.
func fill(frame *image.RGBA, area image.Rectangle, colour color.RGBA) {
	draw.Draw(frame, area, image.NewUniform(colour), image.Point{}, draw.Src)
}

// mix is one colour toward another by a share from 0 to 1.
func mix(from, to color.RGBA, share float64) color.RGBA {
	share = math.Max(0, math.Min(1, share))
	blend := func(a, b uint8) uint8 {
		return uint8(math.Round(float64(a) + (float64(b)-float64(a))*share))
	}
	return color.RGBA{blend(from.R, to.R), blend(from.G, to.G),
		blend(from.B, to.B), math.MaxUint8}
}

// disc paints a filled circle.
func disc(frame *image.RGBA, centre image.Point, radius int,
	colour color.RGBA) {
	for y := -radius; y <= radius; y++ {
		reach := int(math.Sqrt(float64(radius*radius - y*y)))
		fill(frame, image.Rect(centre.X-reach, centre.Y+y,
			centre.X+reach+1, centre.Y+y+1), colour)
	}
}

// line paints a straight line of a width between two points, as discs
// along it.
func line(frame *image.RGBA, from, to image.Point, width int,
	colour color.RGBA) {
	steps := int(math.Ceil(math.Hypot(float64(to.X-from.X),
		float64(to.Y-from.Y))))
	for k := 0; k <= steps; k++ {
		share := float64(k) / float64(max(steps, 1))
		at := image.Pt(from.X+int(math.Round(share*float64(to.X-from.X))),
			from.Y+int(math.Round(share*float64(to.Y-from.Y))))
		disc(frame, at, width/2, colour)
	}
}

// Contrast is the ratio of two colours' relative luminances, the
// lighter's over the darker's, each with a twentieth added, as WCAG 2.1
// defines it: 1 to 21.
func Contrast(one, other color.RGBA) float64 {
	a, b := luminance(one)+contrastFlare, luminance(other)+contrastFlare
	return math.Max(a, b) / math.Min(a, b)
}

// contrastFlare is the twentieth WCAG 2.1 adds to each luminance.
// Defined by WCAG 2.1, contrast ratio.
const contrastFlare = 0.05

// luminance is an sRGB colour's relative luminance, WCAG 2.1's weights
// of its linear channels.
func luminance(colour color.RGBA) float64 {
	return lumaRed*linear(colour.R) + lumaGreen*linear(colour.G) +
		lumaBlue*linear(colour.B)
}

// The channels' weights in luminance. Defined by WCAG 2.1, from
// ITU-R BT.709.
const lumaRed, lumaGreen, lumaBlue = 0.2126, 0.7152, 0.0722

// linear is an sRGB channel's linear light, as WCAG 2.1 gives it.
func linear(channel uint8) float64 {
	share := float64(channel) / math.MaxUint8
	if share <= srgbKnee {
		return share / srgbToe
	}
	return math.Pow((share+srgbOffset)/(1+srgbOffset), srgbPower)
}

// The sRGB curve's knee, toe slope, offset and power. Defined by
// WCAG 2.1, relative luminance.
const (
	srgbKnee   = 0.04045
	srgbToe    = 12.92
	srgbOffset = 0.055
	srgbPower  = 2.4
)
