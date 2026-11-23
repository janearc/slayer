// Package progress is what a renderer reports while a frame is in flight,
// and a meter that turns the reports into something to show.
//
// Design. A Band names rows of a picture that are finished and will not be
// written again. A Sink is where a renderer sends bands. A Meter is told
// the shape of a job, then the rows and frames that finish, and answers
// with the share done, the time spent and the time left. Plain and Human
// write a meter's state as a line, and Due says whether to write one now.
//
// A Band carries only its row range, not pixels. The renderer already
// holds the frame its pixels are in, and the caller that wants them holds
// it too; a slice of them in a band would alias memory the renderer owns
// across the boundary this package exists to keep thin, and would make
// this package depend on the pixel type for nothing the meter reads. The
// price is that a viewer must be handed the frame beside the sink.
//
// The package has no terminal code and no clock of its own: a meter is
// given the time by its caller. It does not import the tracer.
package progress

import "fmt"

// Band is rows [First, First+Count) of a picture Width by Height that are
// finished and will not be written again.
type Band struct {
	width, height int
	first, count  int
}

// NewBand is the rows first to first+count-1 of a picture width across and
// height down. It refuses a picture of no size, no rows, and rows that run
// outside the picture.
func NewBand(width, height, first, count int) (Band, error) {
	switch {
	case width < fewest || height < fewest:
		return Band{}, fmt.Errorf("progress: a band is of a picture at "+
			"least %d by %d, not %d by %d", fewest, fewest, width,
			height)
	case first < none || count < fewest:
		return Band{}, fmt.Errorf("progress: a band starts at row %d "+
			"or later and holds at least %d row, not row %d for %d",
			none, fewest, first, count)
	case first > height-count:
		return Band{}, fmt.Errorf("progress: rows %d for %d run past "+
			"the last of %d rows", first, count, height)
	}
	return Band{width: width, height: height, first: first,
		count: count}, nil
}

// Width is how many pixels across the picture is.
func (band Band) Width() int { return band.width }

// Height is how many rows the picture has.
func (band Band) Height() int { return band.height }

// First is the top row of the band, counting from zero.
func (band Band) First() int { return band.first }

// Count is how many rows the band holds.
func (band Band) Count() int { return band.count }

// Sink is where a renderer sends each band as it finishes. It is called
// from the renderer's goroutine and must return quickly.
type Sink func(Band)
