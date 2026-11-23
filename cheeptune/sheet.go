package cheeptune

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// The instrument sheet: every instrument of the built-in studio voicings
// as a picture of one note, time left to right, pitch low to high, bright
// where the sound is strong. It is how an instrument is looked at by
// somebody who cannot hear it: a bell shows partials that are not evenly
// spaced, the high ones gone first; a string shows an even ladder; a
// filter that opens shows as a wedge.

// The sheet's layout, in pixels, and its ranges. Picked by the writer of
// the studio, 2026-11-12.
const (
	sheetPanelWide = 300
	sheetPanelTall = 190
	sheetGap       = 16
	sheetLabel     = 26
	sheetMargin    = 24
	sheetTitle     = 64
	// sheetSeconds is how much of a note a panel shows, and sheetHeld
	// how long the note is held, in beats at sixty a minute.
	sheetSeconds = 4
	sheetHeld    = 1
	// sheetLowest and sheetHighest are the bottom and top of a panel,
	// in hertz.
	sheetLowest  = 40
	sheetHighest = 18000
	// sheetFloor is the quietest thing a panel draws, in decibels under
	// its loudest.
	sheetFloor = -70
	// sheetWindow is the samples each column of a panel is measured
	// over.
	sheetWindow = 2048
)

// sheetNotes is the note each role's instrument is shown playing.
var sheetNotes = map[string]string{"lead": "a4", "pad": "a3", "bass": "a2"}

// panel is one instrument's picture: what it is called and its samples.
type panel struct {
	label string
	role  string
	sound []float32
}

// InstrumentSheet draws every instrument of the named studio voicings,
// a row a voicing.
func InstrumentSheet(voicings []string) (*image.RGBA, error) {
	var rows [][]panel
	most := 0
	for _, name := range voicings {
		row, err := panels(name)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
		most = max(most, len(row))
	}
	if most == 0 {
		return nil, fmt.Errorf("no studio voicings to draw; name some")
	}
	face, err := sheetFace()
	if err != nil {
		return nil, err
	}
	defer face.Close()
	wide := 2*sheetMargin + most*(sheetPanelWide+sheetGap)
	tall := sheetTitle + sheetMargin +
		len(rows)*(sheetPanelTall+sheetLabel+sheetGap)
	img := image.NewRGBA(image.Rect(0, 0, wide, tall))
	fill(img, img.Bounds(), rollGround, 1)
	text(img, face, sheetMargin, sheetTitle/2, rollInk, fmt.Sprintf(
		"one note of each instrument: %d seconds left to right, %d Hz "+
			"to %d kHz bottom to top, held for the first second",
		sheetSeconds, sheetLowest, sheetHighest/1000))
	for r, row := range rows {
		y := sheetTitle + r*(sheetPanelTall+sheetLabel+sheetGap)
		for c, p := range row {
			x := sheetMargin + c*(sheetPanelWide+sheetGap)
			text(img, face, x, y+sheetLabel-8, rollInk, p.label)
			at := image.Rect(x, y+sheetLabel, x+sheetPanelWide,
				y+sheetLabel+sheetPanelTall)
			drawPanel(img, at, p)
		}
	}
	return img, nil
}

// sheetFace is the face the sheet's labels are set in.
func sheetFace() (font.Face, error) {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size: rollText, DPI: 72, Hinting: font.HintingFull})
}

// panels is one voicing's instruments, each rendered alone and dry: every
// channel that plays a role, in order.
func panels(name string) ([]panel, error) {
	b, err := readDoc(name)
	if err != nil {
		return nil, err
	}
	v, err := ParseVoicing(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if v.Machine != "studio" {
		return nil, fmt.Errorf("%s is on the %s machine; the sheet "+
			"draws studio voicings", name, v.Machine)
	}
	var out []panel
	for i := range v.Channels {
		c := v.Channels[i]
		if c.Role == "" {
			continue
		}
		sound, err := alone(c)
		if err != nil {
			return nil, fmt.Errorf("%s channel %d: %w", name, i+1, err)
		}
		out = append(out, panel{role: c.Role, sound: sound,
			label: fmt.Sprintf("%s  %s (%s)", name, c.Role, c.Kind)})
	}
	return out, nil
}

// alone is one channel playing its role's note, or its hits one a
// second, with no room, no repeats and nothing beside it.
func alone(c channel) ([]float32, error) {
	c.Send, c.Place, c.Volume = nil, nil, nil
	bar := sheetNotes[c.Role] + ":1 -:3"
	if engines[c.Kind].percussive {
		bar = strings.Join(drumNames(c.Drums), " ")
		for n := len(c.Drums); n < sheetSeconds; n++ {
			bar += " -"
		}
		if len(c.Drums) > sheetSeconds {
			bar = strings.Join(drumNames(c.Drums)[:sheetSeconds], " ")
		}
	}
	quiet := masterLeast * 10
	v := &Voicing{Name: "alone", Machine: "studio", Master: &quiet,
		Channels: []channel{c}}
	s := &Score{Name: "one note", Tempo: 60, Meter: sheetSeconds,
		Parts: []Part{{Role: c.Role, Bars: []string{bar}}}}
	if err := s.check(); err != nil {
		return nil, err
	}
	p, err := Render(s, v)
	if err != nil {
		return nil, err
	}
	out := make([]float32, sheetSeconds*p.Rate)
	copy(out, p.Left)
	return out, nil
}

// drawPanel draws one instrument's note: a column a pixel, each measured
// over a short window, on a pitch axis that gives every octave the same
// height.
func drawPanel(img *image.RGBA, at image.Rectangle, p panel) {
	fill(img, at, rollGrid, 1)
	columns := make([]*spectrum, at.Dx())
	loudest := 0.0
	for x := range columns {
		start := x * (len(p.sound) - sheetWindow) / at.Dx()
		columns[x] = newSpectrum(p.sound[start:start+sheetWindow],
			studioRate)
		loudest = math.Max(loudest,
			columns[x].most(sheetLowest, sheetHighest))
	}
	if loudest == 0 {
		return
	}
	octaves := math.Log2(float64(sheetHighest) / sheetLowest)
	hue := roleColour[p.role]
	for x, s := range columns {
		for y := range at.Dy() {
			top := sheetLowest * math.Exp2(octaves*
				float64(at.Dy()-y)/float64(at.Dy()))
			bottom := sheetLowest * math.Exp2(octaves*
				float64(at.Dy()-y-1)/float64(at.Dy()))
			level := decibels(s.most(bottom, top) / loudest)
			strength := 1 - level/sheetFloor
			if strength <= 0 {
				continue
			}
			dot := image.Rect(at.Min.X+x, at.Min.Y+y, at.Min.X+x+1,
				at.Min.Y+y+1)
			fill(img, dot, sheetColour(hue, strength), 1)
		}
	}
}

// sheetColour is a strength from nought to one as a colour: from the
// panel's ground through the role's colour to the ink.
func sheetColour(hue color.RGBA, strength float64) color.RGBA {
	mix := func(a, b color.RGBA, t float64) color.RGBA {
		at := func(x, y uint8) uint8 {
			return uint8(float64(x) + (float64(y)-float64(x))*t)
		}
		return color.RGBA{at(a.R, b.R), at(a.G, b.G), at(a.B, b.B), 0xff}
	}
	if strength < sheetKnee {
		return mix(rollGrid, hue, strength/sheetKnee)
	}
	return mix(hue, rollInk, (strength-sheetKnee)/(1-sheetKnee))
}

// sheetKnee is the strength at which a panel's colour is the role's own.
// Picked.
const sheetKnee = 0.7
