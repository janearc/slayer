package host

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/janearc/slayer/interfaces/movie"
)

// The show's numbers the package picked: the quasar slice, 2026-11-12.
const (
	// filmRate is the film's frames a second, the walk package's.
	filmRate = 24
	// wordsSize is the intro's words' size, in pixels, and wordsFoot how
	// far above the frame's foot their baseline sits.
	wordsSize = 52.0
	wordsFoot = 240
	wordsSide = 60
	wordsLead = 70
	wordsDots = 72
	full      = 255 // defined: a channel all on
	channels  = 4   // defined: red, green, blue and alpha
	alpha     = 3   // defined: alpha's place among them
	// placeholderInk is the grey a missing host is outlined in.
	placeholderInk = 150
)

// Span is a stretch of a short, in seconds from its start.
type Span struct {
	From, Until float64
}

// holds is whether a moment is in the span.
func (span Span) holds(seconds float64) bool {
	return seconds >= span.From && seconds < span.Until
}

// Intro is what comes before the short: a full-frame clip (or none, for
// a black card) and a line of words over it, for so many seconds.
type Intro struct {
	Clip    *loop
	Words   string
	Seconds float64
}

// Show is a short with its host: its frames, how many, the tune under
// it, the host's clips and box, when the host speaks, the parts whose
// kicks lift the picture and the picture's area, the intro and a seed.
type Show struct {
	Frames  func(int) (*image.RGBA, error)
	Count   int
	Tune    Tune
	Clips   Clips
	Box     Box
	Speaks  []Span
	Lifts   map[string]bool
	Picture image.Rectangle
	Intro   Intro
	Seed    uint64
	Size    image.Point
}

// introFrames is how many frames the intro is.
func (show Show) introFrames() int {
	return int(math.Round(show.Intro.Seconds * filmRate))
}

// total is how many frames the film is: the intro and the short.
func (show Show) total() int { return show.introFrames() + show.Count }

// frame is the film's frame i: the intro's, or the short's with the
// picture lifted on a kick and the host laid in its box.
func (show Show) frame(i int) (*image.RGBA, error) {
	seconds := float64(i) / filmRate
	if i < show.introFrames() {
		return show.intro(seconds)
	}
	short := float64(i-show.introFrames()) / filmRate
	frame, err := show.Frames(i - show.introFrames())
	if err != nil {
		return nil, err
	}
	frame = clone(frame)
	at := show.Tune.at(seconds, show.speaks(short))
	lift(frame, show.Picture, kickLift(at, show.Lifts))
	show.lay(frame, at, seconds)
	return frame, nil
}

// speaks is whether the host is speaking at a moment of the short.
func (show Show) speaks(seconds float64) bool {
	for _, span := range show.Speaks {
		if span.holds(seconds) {
			return true
		}
	}
	return false
}

// lay puts the host in its box: its clip for the moment, flickered, laid
// on as light on light so its black is no picture; a grey outline where
// there are no clips yet.
func (show Show) lay(frame *image.RGBA, at moment, seconds float64) {
	place := show.Box.place(at, seconds, show.Seed)
	area := image.Rectangle{Min: place, Max: place.Add(show.Box.size)}
	clip, found := show.Clips[pick(at)]
	if !found {
		outline(frame, area)
		return
	}
	screen(frame, clip.at(seconds), area, Flicker(seconds, show.Seed))
}

// intro is the intro's frame: its clip whole, or black, and its words.
func (show Show) intro(seconds float64) (*image.RGBA, error) {
	frame := image.NewRGBA(image.Rectangle{Max: show.Size})
	black(frame)
	if show.Intro.Clip != nil {
		screen(frame, show.Intro.Clip.at(seconds), frame.Bounds(), one)
	}
	return frame, write(frame, show.Intro.Words)
}

// Write writes the film to a path with the tune's own sound, replacing
// a file there, and says how many frames it wrote.
func (show Show) Write(ctx context.Context, path string) (int, error) {
	settings, err := movie.NewSettings(movie.Request{Path: path,
		Width: show.Size.X, Height: show.Size.Y, Rate: filmRate,
		Frames: show.total(), Sound: show.Tune.sound, Replace: true})
	if err != nil {
		return none, fmt.Errorf("host: %w", err)
	}
	writer, err := movie.New(ctx, settings)
	if err != nil {
		return none, fmt.Errorf("host: %w", err)
	}
	for i := range show.total() {
		frame, err := show.frame(i)
		if err == nil {
			err = writer.Frame(i, frame)
		}
		if err != nil {
			writer.Abort()
			return i, fmt.Errorf("host: frame %d: %w", i, err)
		}
	}
	return show.total(), writer.Close()
}

// clone is a picture's copy, to draw on.
func clone(picture *image.RGBA) *image.RGBA {
	copied := image.NewRGBA(picture.Bounds())
	copy(copied.Pix, picture.Pix)
	return copied
}

// black paints a frame black.
func black(frame *image.RGBA) {
	for i := range frame.Pix {
		frame.Pix[i] = none
		if i%channels == alpha {
			frame.Pix[i] = full
		}
	}
}

// lift adds a share of white to an area of a frame.
func lift(frame *image.RGBA, area image.Rectangle, share float64) {
	if share <= none {
		return
	}
	area = area.Intersect(frame.Bounds())
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			was := frame.RGBAAt(x, y)
			frame.SetRGBA(x, y, color.RGBA{R: lifted8(was.R, share),
				G: lifted8(was.G, share), B: lifted8(was.B, share),
				A: full})
		}
	}
}

// lifted8 is a channel with a share of white added.
func lifted8(channel uint8, share float64) uint8 {
	return uint8(math.Min(full, float64(channel)+share*full))
}

// screen lays a picture on an area of a frame as light on light, each
// channel one less the product of the two's darks, at a brightness.
func screen(frame, picture *image.RGBA, area image.Rectangle,
	bright float64) {
	size := picture.Bounds().Size()
	for y := range min(size.Y, area.Dy()) {
		for x := range min(size.X, area.Dx()) {
			was := frame.RGBAAt(area.Min.X+x, area.Min.Y+y)
			on := picture.RGBAAt(picture.Bounds().Min.X+x,
				picture.Bounds().Min.Y+y)
			frame.SetRGBA(area.Min.X+x, area.Min.Y+y, color.RGBA{
				R: screened(was.R, on.R, bright),
				G: screened(was.G, on.G, bright),
				B: screened(was.B, on.B, bright), A: full})
		}
	}
}

// screened is one channel laid on another as light on light.
func screened(under, over uint8, bright float64) uint8 {
	a, b := float64(under)/full, float64(over)/full*bright
	return uint8(math.Round((one - (one-a)*(one-b)) * full))
}

// outline draws a grey rectangle where the host will be.
func outline(frame *image.RGBA, area image.Rectangle) {
	ink := color.RGBA{R: placeholderInk, G: placeholderInk,
		B: placeholderInk, A: full}
	for x := area.Min.X; x < area.Max.X; x++ {
		frame.SetRGBA(x, area.Min.Y, ink)
		frame.SetRGBA(x, area.Max.Y-one, ink)
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		frame.SetRGBA(area.Min.X, y, ink)
		frame.SetRGBA(area.Max.X-one, y, ink)
	}
}

// write sets words on a frame near its foot, wrapped to its width.
func write(frame *image.RGBA, words string) error {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return fmt.Errorf("host: %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size: wordsSize, DPI: wordsDots, Hinting: font.HintingNone})
	if err != nil {
		return fmt.Errorf("host: %w", err)
	}
	defer face.Close()
	pen := font.Drawer{Dst: frame, Src: image.NewUniform(color.White),
		Face: face}
	rows := wrap(pen, words, frame.Bounds().Dx()-two*wordsSide)
	baseline := frame.Bounds().Dy() - wordsFoot - wordsLead*(len(rows)-one)
	for i, row := range rows {
		pen.Dot = fixed.P(wordsSide, baseline+i*wordsLead)
		pen.DrawString(row)
	}
	return nil
}

// wrap breaks words into rows no wider than a width.
func wrap(pen font.Drawer, words string, width int) []string {
	var rows []string
	row := ""
	for _, word := range splitWords(words) {
		tried := word
		if row != "" {
			tried = row + " " + word
		}
		if row != "" && pen.MeasureString(tried).Ceil() > width {
			rows = append(rows, row)
			tried = word
		}
		row = tried
	}
	return append(rows, row)
}

// splitWords is words split at spaces.
func splitWords(words string) []string { return strings.Fields(words) }
