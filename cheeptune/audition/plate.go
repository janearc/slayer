package audition

import (
	"image"
	"image/color"
	"math"
	"sort"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The plate is the still a film is made from: the tune's piano roll, the
// parts marked on it and named, and room at the foot for the part that is
// playing. Nothing on it moves; Film adds the line and the foot.

// Picture sizes. Defined by the movie package: 1280 by 720 is taken
// without padding (the height is a multiple of sixteen).
const (
	wide = 1280
	tall = 720
)

// Layout, in pixels. Picked by the writer of this package, 2026-11-12, to
// leave the roll as much of the frame as the labels allow.
const (
	margin     = 24
	titleY     = 40
	stripTop   = 64
	stripRow   = 30
	rollTop    = 140
	rollBottom = 560
	hitLane    = 12
	footY      = 610
	titleSize  = 30
	textSize   = 20
	minNote    = 2 // the narrowest a note is drawn, so a short one shows
)

// Colours. The ground and ink are the roll's, held to the house contrast
// band of 10 to 12.5 to one; channel colours are marks, not text.
var (
	ground = color.RGBA{0x14, 0x13, 0x1a, 0xff}
	ink    = color.RGBA{0xc4, 0xc1, 0xcf, 0xff}
	band   = color.RGBA{0x1d, 0x1b, 0x26, 0xff}
	marker = color.RGBA{0x8a, 0x87, 0x96, 0xff}
	cursor = color.RGBA{0xff, 0xf0, 0xc8, 0xff}
	// channel colours by the channel's order in the voicing's kind.
	pitched = []color.RGBA{
		{0xe8, 0xa8, 0x5a, 0xff}, {0x5c, 0xc0, 0xb4, 0xff},
		{0xa4, 0x86, 0xe6, 0xff},
	}
	hitColour = color.RGBA{0xe6, 0x7e, 0x96, 0xff}
)

// writer draws text in the plate's face.
type writer struct{ faces map[int]font.Face }

// newWriter parses the face once for the sizes the plate uses.
func newWriter() (*writer, error) {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	w := &writer{faces: map[int]font.Face{}}
	for _, size := range []int{titleSize, textSize} {
		w.faces[size], err = opentype.NewFace(parsed, &opentype.FaceOptions{
			Size: float64(size), DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return nil, err
		}
	}
	return w, nil
}

// text draws a line with its baseline at y.
func (w *writer) text(img *image.RGBA, x, y, size int, c color.RGBA,
	s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: w.faces[size],
		Dot: fixed.P(x, y)}
	d.DrawString(s)
}

// seconds is the tune's length in seconds, from the samples.
func (t *Tune) seconds() float64 {
	return float64(len(t.Piece.Left)) / float64(t.Piece.Rate)
}

// xAt is the column a moment falls in.
func (t *Tune) xAt(second float64) int {
	return margin + int(second/t.seconds()*float64(wide-2*margin))
}

// fill paints a rectangle.
func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(img.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// Plate draws the tune's roll with its parts marked and named. It needs
// the tune's piece: a tune that was only read is refused.
func Plate(t *Tune) (*image.RGBA, error) {
	if err := t.rendered(); err != nil {
		return nil, err
	}
	w, err := newWriter()
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, wide, tall))
	fill(img, img.Bounds(), ground)
	w.text(img, margin, titleY, titleSize, ink, t.Name+"   "+t.Sheet.Mood)
	t.drawBands(img, w)
	t.drawNotes(img)
	return img, nil
}

// drawBands shades alternate parts, marks where each starts and names it
// on one of two rows so a short part's name does not run into the next.
func (t *Tune) drawBands(img *image.RGBA, w *writer) {
	for i, span := range t.Sheet.Spans() {
		left, right := t.xAt(span.Start.Seconds()), t.xAt(span.End.Seconds())
		if i%2 == 0 {
			fill(img, image.Rect(left, rollTop, right, rollBottom), band)
		}
		fill(img, image.Rect(left, stripTop, left+2, rollBottom), marker)
		w.text(img, left+6, stripTop+stripRow*(1+i%2)-6, textSize, ink,
			span.Name)
	}
}

// drawNotes draws what the driver wrote: the pitched channels as bars on a
// pitch axis, the hits as marks in a lane under them.
func (t *Tune) drawNotes(img *image.RGBA) {
	lo, hi := t.pitchRange()
	rows := math.Max(hi-lo+1, 1)
	height := float64(rollBottom - rollTop - hitLane)
	rowH := height / rows
	pitchedSeen := 0
	for i, ticks := range t.Piece.Ticks {
		noise := t.Piece.Voicing.Percussive(i)
		colour := hitColour
		if !noise {
			colour = pitched[pitchedSeen%len(pitched)]
			pitchedSeen++
		}
		for j, tick := range ticks {
			if tick.Vol == 0 {
				continue
			}
			x0 := t.xAt(tick.At)
			x1 := x0 + minNote
			if j+1 < len(ticks) {
				x1 = max(x1, t.xAt(ticks[j+1].At))
			}
			if noise {
				fill(img, image.Rect(x0, rollBottom-hitLane, x0+minNote,
					rollBottom), colour)
				continue
			}
			y := rollTop + int((hi-tick.Pitch)*rowH)
			fill(img, image.Rect(x0, y, x1, y+max(2, int(rowH))), colour)
		}
	}
}

// pitchRange is the lowest and highest pitch any pitched channel played,
// padded by a semitone.
func (t *Tune) pitchRange() (lo, hi float64) {
	var all []float64
	for i, ticks := range t.Piece.Ticks {
		if t.Piece.Voicing.Percussive(i) {
			continue
		}
		for _, tick := range ticks {
			if tick.Vol > 0 {
				all = append(all, tick.Pitch)
			}
		}
	}
	if len(all) == 0 {
		return 0, 1
	}
	sort.Float64s(all)
	return math.Floor(all[0]) - 1, math.Ceil(all[len(all)-1]) + 1
}
