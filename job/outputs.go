package job

import (
	"path/filepath"
	"time"
)

// outputForms remembers whether each output was written without a directory.
// Resolution makes both bare and absolute sibling paths absolute, so this
// information must be captured before resolution.
type outputForms struct{ picture, raw, frames, movie bool }

// formsOf records, for each output, whether it was named without a directory.
func formsOf(wire *outputsWire) outputForms {
	if wire == nil {
		return outputForms{}
	}
	bare := func(path string) bool { return path != "" && filepath.Base(path) == path }
	return outputForms{picture: bare(wire.Picture), raw: bare(wire.Raw), frames: bare(wire.Frames), movie: bare(wire.Movie)}
}

// Outputs is what the job writes, by role, each an absolute path. A job
// that says none is written where its verb puts it.
type Outputs struct {
	wire  outputsWire
	forms outputForms
}

// Outputs is the job's outputs.
func (job Job) Outputs() Outputs {
	if job.wire.Outputs == nil {
		return Outputs{}
	}
	return Outputs{wire: *job.wire.Outputs, forms: job.outputForms}
}

// PictureBare reports whether the picture was named without a directory.
func (out Outputs) PictureBare() bool { return out.forms.picture }

// RawBare reports whether raw light was named without a directory.
func (out Outputs) RawBare() bool { return out.forms.raw }

// FramesBare reports whether frames were named without a directory.
func (out Outputs) FramesBare() bool { return out.forms.frames }

// MovieBare reports whether the movie was named without a directory.
func (out Outputs) MovieBare() bool { return out.forms.movie }

// path is a path and whether the job said it.
func path(said string) (string, bool) { return said, said != "" }

// Picture is a still's picture, and whether the job says one.
func (out Outputs) Picture() (string, bool) { return path(out.wire.Picture) }

// Raw is a still's light as a .pfm, and whether the job says one.
func (out Outputs) Raw() (string, bool) { return path(out.wire.Raw) }

// Frames is a film's directory of numbered pictures, and whether the job
// says one.
func (out Outputs) Frames() (string, bool) { return path(out.wire.Frames) }

// Movie is the film as an .mp4 with its sound, and whether the job says
// one.
func (out Outputs) Movie() (string, bool) { return path(out.wire.Movie) }

// Note is the job's one line for the renders page, as written, and
// whether the job says one. It is words, not a path.
func (out Outputs) Note() (string, bool) { return path(out.wire.Note) }

// Limits is the memory and time ceilings the queue holds the job to.
// They never change the picture, so they are not in the job's key.
type Limits struct {
	wire limitsWire
}

// Limits is the job's limits.
func (job Job) Limits() Limits {
	if job.wire.Limits == nil {
		return Limits{}
	}
	return Limits{*job.wire.Limits}
}

// Memory is the most memory the job may use, in GiB, and whether the job
// said.
func (ceiling Limits) Memory() (float64, bool) {
	if ceiling.wire.Memory == nil {
		return nothing, false
	}
	return ceiling.wire.Memory.Value, true
}

// Time is the longest the job may run, and whether the job said.
func (ceiling Limits) Time() (time.Duration, bool) {
	return spanOf(ceiling.wire.Time)
}
