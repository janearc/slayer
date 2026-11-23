package explain

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

// themeRoles are the roles the panel draws in, and a show may light its
// lamps in.
var themeRoles = []string{"ink", "ground", "dim", "border", "ok", "warn"}

// ThemeColour is a hud default theme colour by role: ink, ground, dim,
// border, ok or warn.
func ThemeColour(role string) (color.RGBA, error) {
	tones, err := colours()
	if err != nil {
		return color.RGBA{}, err
	}
	colour, ok := tones[role]
	if !ok {
		return color.RGBA{}, fmt.Errorf("%w: no theme role %q; choose %v",
			errExplain, role, themeRoles)
	}
	return colour, nil
}

// colours is the hud's default theme, by role.
var colours = sync.OnceValues(func() (palette, error) {
	theme := hud.DefaultTheme()
	made := palette{}
	for _, role := range themeRoles {
		colour, err := theme.Colour(role)
		if err != nil {
			return nil, fmt.Errorf("%w: the hud theme: %w", errExplain, err)
		}
		made[role] = colour
	}
	return made, nil
})

// regular is Go's regular face, parsed once.
var regular = sync.OnceValues(func() (*opentype.Font, error) {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errExplain, err)
	}
	return parsed, nil
})

// face is Go's regular face at a height in pixels.
func face(size float64) (font.Face, error) {
	parsed, err := regular()
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingNone})
}

// Compose is the explainer frame at a moment: the film's picture,
// already scaled to the frame's width, over the panel.
func (of *Explainer) Compose(seconds float64, film *image.RGBA) (
	*image.RGBA, error) {
	if film == nil || film.Bounds().Dx() != frameSize.X ||
		film.Bounds().Dy() >= beatTop {
		return nil, fmt.Errorf("%w: the film's picture must be %d across "+
			"and under %d down; read it at that size", errExplain,
			frameSize.X, beatTop)
	}
	panel, err := of.PanelAt(seconds)
	if err != nil {
		return nil, err
	}
	theme, err := colours()
	if err != nil {
		return nil, err
	}
	tones := palette{}
	for role, colour := range theme {
		tones[role] = colour
	}
	tones["ground"] = panel.ground
	frame := image.NewRGBA(image.Rectangle{Max: frameSize})
	draw.Draw(frame, frame.Bounds(), image.NewUniform(tones["ground"]),
		image.Point{}, draw.Src)
	draw.Draw(frame, film.Bounds().Sub(film.Bounds().Min), film,
		film.Bounds().Min, draw.Src)
	for _, part := range []func(*image.RGBA, Panel, palette) error{
		drawBeat, drawLamps, drawChant, drawSentence,
	} {
		if err := part(frame, panel, tones); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

// drawBeat is a square for each beat of the bar, and the bar beside
// them.
func drawBeat(frame *image.RGBA, panel Panel, tones palette) error {
	beat := panel.Beat()
	count := beat.perBar
	side := (beatRow - (count-1)*beatGap) / count
	for k := range count {
		on := tones["ok"]
		if k == 0 {
			on = tones["ink"]
		}
		lit := 0.0
		if beat.On() && beat.InBar() == k+1 {
			lit = math.Max(beatLeast, math.Exp(-beat.Since()/beatFlash))
		}
		left := margin + k*(side+beatGap)
		square := image.Rect(left, beatTop, left+side, beatTop+beatHigh)
		shade := mix(mix(tones["ground"], on, offShare), on, lit)
		fill(frame, square, shade)
		if beat.On() && !beat.measured {
			fill(frame, square.Inset(beatHollow), tones["ground"])
		}
	}
	words := "Intro"
	if beat.On() {
		words = fmt.Sprintf("Bar %d", beat.Bar())
	}
	return write(frame, words, image.Pt(barTextLeft, barBaseline),
		barTextSize, tones["ink"])
}

// drawLamps is the lamps, each a disc as bright as its light, a ring
// when its cue fires, and its label.
func drawLamps(frame *image.RGBA, panel Panel, tones palette) error {
	column := (frameSize.X - 2*margin) / lampColumns
	for i, lamp := range panel.Lamps() {
		centre := image.Pt(margin+column*(i%lampColumns)+column/2,
			lampTop+lampRow*(i/lampColumns)+ringRadius)
		on := lamp.light.colour
		if lamp.Ring() > ringShown {
			ring := mix(tones["ground"], tones["ink"], lamp.Ring())
			disc(frame, centre, ringRadius, ring)
			disc(frame, centre, ringRadius-ringWidth, tones["ground"])
		}
		disc(frame, centre, lampRadius,
			mix(mix(tones["ground"], on, offShare), on, lamp.Level()))
		if err := centred(frame, lamp.Label(),
			image.Pt(centre.X, centre.Y+lampLabelDrop), lampLabelSize,
			tones["ink"]); err != nil {
			return err
		}
	}
	return nil
}

// drawChant is the line being sung, whole, centred, in ink.
func drawChant(frame *image.RGBA, panel Panel, tones palette) error {
	if !panel.singing {
		return nil
	}
	return centred(frame, panel.chant, image.Pt(frameSize.X/2, chantBaseline),
		chantSize, tones["ink"])
}

// drawSentence is the sentence standing, wrapped to the panel's width.
func drawSentence(frame *image.RGBA, panel Panel, tones palette) error {
	sentence, saying := panel.Says()
	if !saying {
		return nil
	}
	rows, err := wrapped(sentence.Words(), sentenceSize, sentenceWidth)
	if err != nil {
		return err
	}
	if len(rows) > sentenceMostRows {
		return fmt.Errorf("%w: %q takes %d rows; the panel has room for "+
			"%d: shorten it", errExplain, sentence.Words(), len(rows),
			sentenceMostRows)
	}
	for i, row := range rows {
		at := image.Pt(margin, sentenceTop+sentenceSize+i*sentenceLeading)
		if err := write(frame, row, at, sentenceSize,
			tones["ink"]); err != nil {
			return err
		}
	}
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

// centred draws words centred across a point, its baseline there.
func centred(frame *image.RGBA, words string, at image.Point, size float64,
	ink color.RGBA) error {
	wide, err := measure(words, size)
	if err != nil {
		return err
	}
	return write(frame, words, image.Pt(at.X-wide/2, at.Y), size, ink)
}

// mix is one colour toward another by a share from 0 to 1.
func mix(from, to color.RGBA, share float64) color.RGBA {
	share = clamp(share)
	blend := func(a, b uint8) uint8 {
		return uint8(math.Round(float64(a) + (float64(b)-float64(a))*share))
	}
	return color.RGBA{blend(from.R, to.R), blend(from.G, to.G),
		blend(from.B, to.B), math.MaxUint8}
}

// fill paints a rectangle one colour.
func fill(frame *image.RGBA, area image.Rectangle, colour color.RGBA) {
	draw.Draw(frame, area, image.NewUniform(colour), image.Point{}, draw.Src)
}

// disc paints a filled circle.
func disc(frame *image.RGBA, centre image.Point, radius int,
	colour color.RGBA) {
	for y := -radius; y <= radius; y++ {
		half := int(math.Sqrt(float64(radius*radius - y*y)))
		fill(frame, image.Rect(centre.X-half, centre.Y+y,
			centre.X+half+1, centre.Y+y+1), colour)
	}
}
