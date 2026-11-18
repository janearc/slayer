package door

import (
	"context"
	"fmt"
	"image"

	"github.com/janearc/slayer/hillside"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/finish"
	"github.com/janearc/slayer/render/frame"
)

// pieces is the real workshop: hillside's session, finish's developer,
// titles and plates, and movie's writer.
type pieces struct{ stamp hillside.Stamp }

// Source is a hillside session for a set, a smoke or anything else the
// session draws or refuses by name, and the plates for frames already
// rendered.
func (made pieces) Source(ctx context.Context, wanted job.Job) (source,
	error) {
	if directory, isFrames := wanted.Scene().Frames(); isFrames {
		return openPlates(wanted, directory)
	}
	session, err := hillside.Open(ctx, wanted, made.stamp)
	if err != nil {
		return nil, err
	}
	return sessionSource{session: session}, nil
}

// Developer is finish's developer for the job.
func (pieces) Developer(wanted job.Job, fallback colour.Tone) (developer,
	error) {
	return finish.NewDeveloper(wanted, fallback)
}

// Titler is finish's titles for the job, and whether it says any.
func (pieces) Titler(wanted job.Job) (titler, bool, error) {
	return finish.NewTitles(wanted)
}

// Reel is movie's writer, opened.
func (pieces) Reel(ctx context.Context, settings movie.Settings) (reel,
	error) {
	return movie.New(ctx, settings)
}

// sessionSource is a hillside session drawing each frame at the job's
// own size.
type sessionSource struct{ session *hillside.Session }

// Cached is what the session's staging took from the disk caches and
// what it worked out, a line each.
func (made sessionSource) Cached() []string {
	return made.session.Cached()
}

// OceanSpectrumLine is the staged ocean's one-line spectrum-model estimate,
// if the set authored an ocean.
func (made sessionSource) OceanSpectrumLine() (string, bool) {
	report, has := made.session.OceanSpectrum()
	if !has {
		return "", false
	}
	return report.Summary(), true
}

// Frame is the session's frame at the job's own size.
func (made sessionSource) Frame(ctx context.Context, index int) (drawn,
	error) {
	return made.session.Frame(ctx, index, hillside.Ask{})
}

// Tone is the scene's own tone.
func (made sessionSource) Tone() colour.Tone { return made.session.Tone() }

// Close lets the scene go.
func (made sessionSource) Close() error { return made.session.Close() }

// plateSource is frames already rendered, each fitted to the film.
type plateSource struct {
	plates finish.Plates
	fit    finish.Fit
}

// openPlates is the plates of a directory fitted to the job's film. A
// directory with fewer plates than the film has frames is refused.
func openPlates(wanted job.Job, directory string) (source, error) {
	plates, err := finish.OpenPlates(directory)
	if err != nil {
		return nil, err
	}
	if frames := wanted.Clock().Frames(); plates.Len() < frames {
		return nil, fmt.Errorf("%s holds %d frames and the film has %d; "+
			"give it the frames or shorten the duration", directory,
			plates.Len(), frames)
	}
	fit, err := finish.NewFit(wanted, plates)
	if err != nil {
		return nil, err
	}
	return plateSource{plates: plates, fit: fit}, nil
}

// Frame is the plate at index, fitted.
func (made plateSource) Frame(_ context.Context, index int) (drawn,
	error) {
	plate, err := made.plates.Frame(index)
	if err != nil {
		return nil, err
	}
	fitted, err := made.fit.Apply(plate)
	if err != nil {
		return nil, err
	}
	return plateDrawn{light: fitted}, nil
}

// Tone is the clamp: a plate's light comes back as it went in.
func (plateSource) Tone() colour.Tone { return colour.Tone{} }

// Close has nothing to let go: a plate is read when asked for.
func (plateSource) Close() error { return nil }

// plateDrawn is a plate's light; a plate has no hud of its own.
type plateDrawn struct{ light *frame.Frame }

// Light is the plate's light.
func (made plateDrawn) Light() *frame.Frame { return made.light }

// Overlay draws nothing: a plate has no set, so no hud.
func (plateDrawn) Overlay(*image.RGBA) error { return nil }
