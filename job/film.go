package job

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Film is the picture's size, rays and lens, as the job says them, its
// preset's size counted as said. What it leaves unsaid comes from the
// set, then the scene's own default.
type Film struct {
	wire filmWire
}

// Film is the job's film.
func (job Job) Film() Film {
	if job.wire.Film == nil {
		return Film{}
	}
	return Film{*job.wire.Film}
}

// said is a whole number and whether the job said it.
func said(number int) (int, bool) { return number, number != nothing }

// saidNumber is a number and whether the job said it.
func saidNumber(number float64) (float64, bool) {
	return number, number != nothing
}

// Preset is the film's preset, and whether it names one.
func (made Film) Preset() (string, bool) {
	return made.wire.Preset, made.wire.Preset != ""
}

// Width is the picture's width in pixels, and whether the job or its
// preset said it.
func (made Film) Width() (int, bool) { return said(made.wire.Width) }

// Height is the picture's height in pixels, and whether the job or its
// preset said it.
func (made Film) Height() (int, bool) { return said(made.wire.Height) }

// Rays is the rays along each side of a pixel, and whether the job said.
func (made Film) Rays() (int, bool) { return said(made.wire.Rays) }

// Wide is the angle across the picture in degrees, and whether the job
// said it.
func (made Film) Wide() (float64, bool) { return saidNumber(made.wire.Wide) }

// Tall is the angle down the picture in degrees, and whether the job
// said it.
func (made Film) Tall() (float64, bool) { return saidNumber(made.wire.Tall) }

// Exposure is what one watt a steradian is in the picture's light, and
// whether the job said it.
func (made Film) Exposure() (float64, bool) {
	return saidNumber(made.wire.Exposure)
}

// Aperture is how wide the lens is open, across, in the scene's own
// unit, and whether the job said it. A lens that says none is a pinhole:
// everything is sharp.
func (made Film) Aperture() (float64, bool) {
	return saidNumber(made.wire.Aperture)
}

// Focus is how far off the lens is focused, in the scene's own unit, and
// whether the job said it. A job says it beside an aperture.
func (made Film) Focus() (float64, bool) {
	return saidNumber(made.wire.Focus)
}

// WideFor is the angle across a picture width by height pixels, in
// degrees: wide as said, or worked out from tall as a pinhole sees, and
// whether the job said either.
func (made Film) WideFor(width, height int) (float64, bool) {
	if made.wire.Wide != nothing {
		return made.wire.Wide, true
	}
	if made.wire.Tall == nothing {
		return nothing, false
	}
	half := math.Tan(made.wire.Tall * maths.Degree / halves)
	across := half * float64(width) / float64(height)
	return halves * math.Atan(across) / maths.Degree, true
}
