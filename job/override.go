package job

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

// Override is one value the command line lays over a job. It is made by
// Output, Width, Height, Rays, At, CameraName, Renderer or Quality.
type Override struct {
	key    string
	empty  bool
	output bool
	movie  bool
	apply  func(wire *document) error
}

// With is the job with the command line's values laid over it, checked
// again as a whole, so an override the job cannot take is refused as a
// key written in the file would be. The job itself is left as it was.
func (job Job) With(overrides ...Override) (Job, error) {
	name := "the command line over " + job.name
	wire := job.wire
	forms := job.outputForms
	for _, each := range overrides {
		if each.empty {
			return Job{}, refusal{fmt.Errorf("%s: %s is empty; give "+
				"a value or leave it out", name, each.key)}
		}
		if err := each.apply(&wire); err != nil {
			return Job{}, err
		}
		if each.output {
			switch {
			case !wire.film():
				forms.picture = false
			case each.movie:
				forms.movie = false
			default:
				forms.frames = false
			}
		}
	}
	// the marshalling cannot fail: the wire was JSON when it was read
	raw, _ := json.Marshal(wire)
	from := source{name: name, base: filepath.Dir(job.file),
		file: job.file, inputs: job.inputs, imported: job.imported,
		outputForms: &forms}
	return admitted(from, raw)
}

// Output writes the job's main output to path, resolved against the
// working directory: a still's picture, or a film's movie where path
// ends .mp4 and its frames where it does not. Other roles are kept.
func Output(path string) Override {
	return Override{key: "output", empty: path == "", output: true,
		movie: filepath.Ext(path) == movieExtension,
		apply: func(wire *document) error {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			outputs := outputsWire{}
			if wire.Outputs != nil {
				outputs = *wire.Outputs
			}
			*outputs.role(wire.film(), absolute) = absolute
			wire.Outputs = &outputs
			return nil
		}}
}

// role is where a command line's output goes: a still's picture, or a
// film's movie or frames by the path's extension.
func (outputs *outputsWire) role(film bool, path string) *string {
	if !film {
		return &outputs.Picture
	}
	if filepath.Ext(path) == movieExtension {
		return &outputs.Movie
	}
	return &outputs.Frames
}

// filmOverride lays one number over the film, on a copy of it.
func filmOverride(key string, number int,
	into func(film *filmWire)) Override {
	return Override{key: key, empty: number == nothing,
		apply: func(wire *document) error {
			film := filmWire{}
			if wire.Film != nil {
				film = *wire.Film
			}
			into(&film)
			wire.Film = &film
			return nil
		}}
}

// Width is the picture's width in pixels, over the job's and its
// preset's.
func Width(pixels int) Override {
	return filmOverride("width", pixels, func(film *filmWire) {
		film.Width = pixels
	})
}

// Height is the picture's height in pixels, over the job's and its
// preset's.
func Height(pixels int) Override {
	return filmOverride("height", pixels, func(film *filmWire) {
		film.Height = pixels
	})
}

// Rays is the rays along each side of a pixel, over the job's.
func Rays(count int) Override {
	return filmOverride("rays", count, func(film *filmWire) {
		film.Rays = count
	})
}

// At is a still's time, over the job's. On a film it is refused, as at
// beside fps is in a file.
func At(when time.Duration) Override {
	return Override{key: "at", apply: func(wire *document) error {
		wire.At = secondsOf(when)
		return nil
	}}
}

// CameraName is a camera the scene defines, by name, in place of the
// job's camera; a zoom the job says is kept beside it.
func CameraName(name string) Override {
	return Override{key: "camera", empty: name == "",
		apply: func(wire *document) error {
			lens := cameraWire{Name: name}
			if wire.Camera != nil {
				lens.Zoom = wire.Camera.Zoom
			}
			wire.Camera = &lens
			return nil
		}}
}

// Renderer is the renderer, over the job's.
func Renderer(name string) Override {
	return Override{key: "renderer", empty: name == "",
		apply: func(wire *document) error {
			wire.Renderer = name
			return nil
		}}
}

// Quality is exact or preview, over the job's.
func Quality(name string) Override {
	return Override{key: "quality", empty: name == "",
		apply: func(wire *document) error {
			wire.Quality = name
			return nil
		}}
}
