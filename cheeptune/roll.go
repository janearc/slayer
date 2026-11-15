package cheeptune

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The roll is a piece drawn so it can be checked without being heard.
//
// It is drawn from what the driver wrote on every tick, not from the score.
// An echo is where the echo actually sounded, a release is as long as it
// was, and a note the register moved is where the register moved it.
//
// Under the notes are the rendered waveform and the level the playhead
// reports, on the same time axis, so the picture cannot disagree with the
// sound.

// The palette. Text is held to the house band, 10 to 12.5 to one against the
// ground (roll_test.go measures it). The ground is near grey so the hue of a
// channel carries its edge. Channel colours are marks, not text.
var (
	rollGround = color.RGBA{0x14, 0x13, 0x1a, 0xff}
	rollInk    = color.RGBA{0xc4, 0xc1, 0xcf, 0xff}
	rollGrid   = color.RGBA{0x26, 0x24, 0x2e, 0xff}
	rollBar    = color.RGBA{0x46, 0x43, 0x52, 0xff}
	rollWave   = color.RGBA{0x8a, 0x87, 0x96, 0xff}

	roleColour = map[string]color.RGBA{
		"lead": {0xe8, 0xa8, 0x5a, 0xff},
		"pad":  {0x5c, 0xc0, 0xb4, 0xff},
		"bass": {0xa4, 0x86, 0xe6, 0xff},
		"perc": {0xe6, 0x7e, 0x96, 0xff},
	}
)

const (
	rollText   = 18 // pixels; nothing on the picture is smaller
	rollLine   = 30 // a line of text, with its leading
	rollLeft   = 76 // the margin the note names sit in
	rollRight  = 24
	rollNotes  = 420
	rollLane   = 24
	rollWaveH  = 140
	rollLevelH = 90
	rollGap    = 14
)

// WriteRoll draws the piece to a PNG.
func (p *Piece) WriteRoll(path string) error {
	img, err := p.Roll()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Roll draws the piece.
func (p *Piece) Roll() (*image.RGBA, error) {
	face, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	fc, err := opentype.NewFace(face, &opentype.FaceOptions{
		Size: rollText, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	defer fc.Close()

	secs := float64(len(p.Left)) / float64(p.Rate)
	// sixty pixels a second reads a sixteenth at 92; a long piece is
	// narrowed so the picture stays one a person can open.
	pps := 60.0
	if secs*pps > 4000 {
		pps = 4000 / secs
	}

	// what is pitched and what is hit, and the range the pitched span.
	lo, hi := math.Inf(1), math.Inf(-1)
	hitSet := map[string]bool{}
	for i, ts := range p.Ticks {
		noise := p.Voicing.Percussive(i)
		for _, t := range ts {
			switch {
			case t.Vol == 0:
			case noise:
				hitSet[t.Hit] = true
			default:
				lo = math.Min(lo, t.Pitch)
				hi = math.Max(hi, t.Pitch)
			}
		}
	}
	var hits []string
	for h := range hitSet {
		hits = append(hits, h)
	}
	sort.Strings(hits)
	pitched := !math.IsInf(lo, 1)
	if pitched {
		lo, hi = math.Floor(lo)-2, math.Ceil(hi)+2
	}
	notesH := 0
	rowH := 0.0
	if pitched {
		rowH = math.Min(14, float64(rollNotes)/(hi-lo+1))
		notesH = int(math.Ceil(rowH * (hi - lo + 1)))
	}
	lanesH := len(hits) * rollLane

	// the picture is as wide as the piece, or as its title and legend if
	// those are wider: a short piece otherwise cuts its own title off.
	s := p.Score
	loop := "plays once"
	if s.Loop {
		loop = "loops"
	}
	title := fmt.Sprintf("%s through %s   %v bpm, %d beats a bar, "+
		"%d bars, %.2f s, %s, peak %.3f", s.Name, p.Voicing.Name,
		s.Tempo, s.Meter, s.Bars(), secs, loop, p.Peak)
	names := make([]string, len(p.Labels))
	legend := 0
	for i, label := range p.Labels {
		names[i] = label + " (" + p.Voicing.Channels[i].Kind + ")"
		legend += 30 + measure(fc, names[i]) + 28
	}
	width := rollLeft + int(math.Ceil(secs*pps)) + rollRight
	width = max(width, rollLeft+measure(fc, title)+rollRight,
		rollLeft+legend)

	top := rollLine*2 + rollGap
	notesY := top + rollLine
	lanesY := notesY + notesH + rollGap
	waveY := lanesY + lanesH + rollGap + rollLine
	levelY := waveY + rollWaveH + rollGap + rollLine
	height := levelY + rollLevelH + rollGap + rollLine

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(img, img.Bounds(), rollGround, 1)
	x := func(s float64) int { return rollLeft + int(math.Round(s*pps)) }
	right := x(secs)

	// the title and the legend.
	text(img, fc, rollLeft, rollLine-8, rollInk, title)
	lx := rollLeft
	for i, label := range p.Labels {
		c, a := channelColour(label)
		y := 2 * rollLine
		fill(img, image.Rect(lx, y-20, lx+22, y-6), c, a)
		text(img, fc, lx+30, y-6, rollInk, names[i])
		lx += 30 + measure(fc, names[i]) + 28
	}

	// the grid: a line a beat, a heavier line a bar, through every panel.
	for b := 0; s.SecondsAt(float64(b)) <= secs+1e-9; b++ {
		bx := x(s.SecondsAt(float64(b)))
		c := rollGrid
		if b%s.Meter == 0 {
			c = rollBar
		}
		for _, span := range [][2]int{
			{notesY, notesY + notesH}, {lanesY, lanesY + lanesH},
			{waveY, waveY + rollWaveH},
			{levelY, levelY + rollLevelH},
		} {
			fill(img, image.Rect(bx, span[0], bx+1, span[1]), c, 1)
		}
		if b%s.Meter == 0 && b/s.Meter < s.Bars() {
			text(img, fc, bx+4, notesY-8, rollInk,
				fmt.Sprint(b/s.Meter+1))
		}
	}

	// the notes. a row a semitone, the octave's c named at the left, and
	// every tick a mark as bright as the volume the driver wrote.
	if pitched {
		rowY := func(m float64) int {
			return notesY + int(math.Round((hi-m)*rowH))
		}
		for m := int(lo); m <= int(hi); m++ {
			if ((m%12)+12)%12 == 0 {
				y := rowY(float64(m))
				line := image.Rect(rollLeft, y, right, y+1)
				fill(img, line, rollGrid, 1)
				text(img, fc, 12, y+6, rollInk, NoteName(m))
			}
		}
		tw := 1 / tickRate
		for i, ts := range p.Ticks {
			if p.Voicing.Percussive(i) {
				continue
			}
			c, a := channelColour(p.Labels[i])
			for _, t := range ts {
				if t.Vol == 0 {
					continue
				}
				y := rowY(t.Pitch) - int(rowH/2)
				r := image.Rect(x(t.At), y, x(t.At+tw)+1,
					y+int(math.Max(3, rowH-2)))
				fill(img, r, c, a*(0.25+0.75*t.Vol))
			}
		}
	}
	// the hits, a lane each.
	for li, h := range hits {
		y := lanesY + li*rollLane
		text(img, fc, 12, y+rollLane-5, rollInk, h)
		fill(img, image.Rect(rollLeft, y, right, y+1), rollGrid, 1)
	}
	for i, ts := range p.Ticks {
		if !p.Voicing.Percussive(i) {
			continue
		}
		c, a := channelColour(p.Labels[i])
		for _, t := range ts {
			if t.Vol == 0 {
				continue
			}
			y := lanesY + sort.SearchStrings(hits, t.Hit)*rollLane
			r := image.Rect(x(t.At), y+3, x(t.At+1/tickRate)+1,
				y+rollLane-3)
			fill(img, r, c, a*(0.25+0.75*t.Vol))
		}
	}

	// the waveform: for each column, the least and most of both
	// channels. what plays, at the scale it plays at.
	text(img, fc, rollLeft, waveY-8, rollInk,
		"waveform, left and right together, full scale")
	mid := waveY + rollWaveH/2
	fill(img, image.Rect(rollLeft, mid, right, mid+1), rollGrid, 1)
	for col := rollLeft; col < right; col++ {
		a := int(float64(col-rollLeft) / pps * float64(p.Rate))
		b := int(float64(col-rollLeft+1) / pps * float64(p.Rate))
		b = min(b, len(p.Left))
		mn, mx := 0.0, 0.0
		for j := a; j < b; j++ {
			for _, v := range []float32{p.Left[j], p.Right[j]} {
				mn = math.Min(mn, float64(v))
				mx = math.Max(mx, float64(v))
			}
		}
		y0 := mid - int(mx*float64(rollWaveH/2))
		y1 := mid - int(mn*float64(rollWaveH/2))
		fill(img, image.Rect(col, y0, col+1, y1+1), rollWave, 1)
	}

	// the level, exactly as the playhead reports it.
	text(img, fc, rollLeft, levelY-8, rollInk,
		"level, as the playhead reports it, 50 ms centred")
	base := levelY + rollLevelH
	for col := rollLeft; col < right; col++ {
		at := time.Duration(float64(col-rollLeft) / pps *
			float64(time.Second))
		_, _, _, l, _ := p.At(at)
		y := base - int(math.Round(l*float64(rollLevelH)))
		fill(img, image.Rect(col, y, col+1, base), rollWave, 0.55)
		fill(img, image.Rect(col, y, col+1, y+2), rollInk, 1)
	}
	return img, nil
}

// channelColour is a label's colour and how strongly it is laid down. An
// echo is its role's colour, faint.
func channelColour(label string) (color.RGBA, float64) {
	if role, ok := strings.CutPrefix(label, "echo of "); ok {
		return roleColour[role], 0.5
	}
	return roleColour[label], 1
}

// fill lays a colour over a rectangle at an opacity from nought to one.
func fill(img *image.RGBA, r image.Rectangle, c color.RGBA, alpha float64) {
	r = r.Intersect(img.Bounds())
	a := math.Max(0, math.Min(1, alpha))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := img.PixOffset(x, y)
			px := img.Pix[i : i+4 : i+4]
			px[0] = uint8(float64(px[0])*(1-a) + float64(c.R)*a)
			px[1] = uint8(float64(px[1])*(1-a) + float64(c.G)*a)
			px[2] = uint8(float64(px[2])*(1-a) + float64(c.B)*a)
			px[3] = 0xff
		}
	}
}

// text writes a line of type with its baseline at y.
func text(img *image.RGBA, fc font.Face, x, y int, c color.RGBA, s string) {
	d := font.Drawer{
		Dst: img, Src: image.NewUniform(c), Face: fc,
		Dot: fixed.P(x, y),
	}
	d.DrawString(s)
}

// measure is how wide a line of type is, in pixels.
func measure(fc font.Face, s string) int {
	return font.MeasureString(fc, s).Round()
}
