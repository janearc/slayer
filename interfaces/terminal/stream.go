package terminal

// Frames written one after another to any writer.
//
// Draw puts one picture on a screen. Stream writes many. It asks the
// other end nothing. It reads no size and sets no mode.

import (
	"fmt"
	"image"
	"io"
	"math"
	"strconv"
	"strings"
)

// SextantPixel is one pixel's width over its height, for a lens.
// Each is half a cell wide and a third of one tall. On a cell twice
// as tall as wide that is 0.75. The number is defined here and not
// measured. Screen uses the same one.
const SextantPixel = 0.75

// sideMost is the most cells a stream is across or down. It is defined
// here. At 1000 a frame is 3000 pixels tall. One more digit by typo
// should be refused and not drawn for an hour.
const sideMost = 1000

// The pixels of one cell, each defined by the sextant.
const (
	pixelsAcross = 2 // defined: two squares across
	pixelsDown   = 3 // defined: three squares down
)

// fewest is the fewest cells a side holds. It is defined as one.
const fewest = 1

// none is no frames, or no start. It is defined as zero. Nought frames is a
// stream with no end.
const none = 0

// onward is how far the next frame is from this one. It is defined as one.
const onward = 1

// Size is how many cells a stream is across and down.
//
//ninja:vow practice ParseSize makes one and NewStream takes it
type Size struct {
	cols, rows int
}

// ParseSize reads a size written COLSxROWS, such as 80x24. Each side
// is a whole number from 1 to 1000. Nothing else is read.
func ParseSize(written string) (Size, error) {
	across, down, found := strings.Cut(written, "x")
	if !found {
		return Size{}, fmt.Errorf("%q is not a size: write COLSxROWS, "+
			"such as 80x24", written)
	}
	cols, err := side(across)
	if err != nil {
		return Size{}, fmt.Errorf("%q is not a size: columns %w",
			written, err)
	}
	rows, err := side(down)
	if err != nil {
		return Size{}, fmt.Errorf("%q is not a size: rows %w",
			written, err)
	}
	return Size{cols: cols, rows: rows}, nil
}

// side is one side of a size.
func side(written string) (int, error) {
	cells, err := strconv.Atoi(written)
	if err != nil || strings.HasPrefix(written, "+") {
		return none, fmt.Errorf("must be a whole number, not %q",
			written)
	}
	if cells < fewest || cells > sideMost {
		return none, fmt.Errorf("must be from %d to %d, not %d",
			fewest, sideMost, cells)
	}
	return cells, nil
}

// pixels is the picture that fills the size.
func (shape Size) pixels() (wide, high int) {
	return shape.cols * pixelsAcross, shape.rows * pixelsDown
}

// Source draws the frames of a stream.
//
//ninja:vow practice NewStream takes one, and cells.Scene is one
type Source interface {
	// Frame is frame number as a picture wide pixels across and high
	// down.
	Frame(number, wide, high int) (image.Image, error)
}

// Stream is the frames of a source, drawn as cells and written.
type Stream struct {
	size   Size
	first  int
	frames int
	source Source
}

// NewStream is a stream of size from frame first for frames frames.
// Nought frames has no end. It stops when a write or the source fails.
// It refuses a negative start or count and a missing size or source.
func NewStream(size Size, first, frames int,
	source Source) (*Stream, error) {
	switch {
	case size.cols < fewest || size.rows < fewest:
		return nil, fmt.Errorf("a stream needs a size: use ParseSize")
	case first < none || frames < none:
		return nil, fmt.Errorf("a stream from frame %d for %d frames: "+
			"neither may be negative", first, frames)
	case first > math.MaxInt-frames:
		return nil, fmt.Errorf("a stream from frame %d for %d frames "+
			"runs past the last number there is", first, frames)
	case source == nil:
		return nil, fmt.Errorf("a stream needs a source to draw from")
	}
	return &Stream{size: size, first: first, frames: frames,
		source: source}, nil
}

// Run draws each frame and writes it with Draw before it asks for the
// next. It stops at the first frame that cannot be drawn. The wrong
// size stops it too. So does a write that fails. The error names the
// frame.
func (stream *Stream) Run(out io.Writer) error {
	return stream.RunBy(out, nil)
}

// After each frame RunBy asks next, the last too, so it can be looked at.
// A stop or an error ends the stream.
func (stream *Stream) RunBy(out io.Writer,
	next func() (stop bool, err error)) error {
	for number := stream.first; stream.goes(number); number++ {
		drawn, err := stream.picture(number)
		if err != nil {
			return fmt.Errorf("frame %d: %w", number, err)
		}
		if err := Draw(out, Chop(drawn)); err != nil {
			return fmt.Errorf("frame %d: writing it: %w", number,
				err)
		}
		if next == nil {
			continue
		}
		stop, err := next()
		if err != nil {
			return fmt.Errorf("frame %d: waiting for a key: %w",
				number, err)
		}
		if stop || !stream.goes(number+onward) {
			return nil
		}
	}
	return nil
}

// goes is whether the stream draws frame number. Without an end it
// always does.
func (stream *Stream) goes(number int) bool {
	return stream.frames == none || number < stream.first+stream.frames
}

// picture is frame number from the source. It is refused if the source
// drew nothing or drew another size than the stream's.
func (stream *Stream) picture(number int) (image.Image, error) {
	wide, high := stream.size.pixels()
	drawn, err := stream.source.Frame(number, wide, high)
	if err != nil {
		return nil, err
	}
	if drawn == nil {
		return nil, fmt.Errorf("the source drew no picture")
	}
	got := drawn.Bounds()
	if got.Dx() != wide || got.Dy() != high {
		return nil, fmt.Errorf("the source drew %dx%d pixels, "+
			"not the %dx%d asked for", got.Dx(), got.Dy(),
			wide, high)
	}
	return drawn, nil
}
