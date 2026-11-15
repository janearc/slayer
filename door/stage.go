package door

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/janearc/slayer/hillside"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/live"
	"github.com/janearc/slayer/set"
)

// sessions is watch's stager: a hillside session for the job, and the
// set's cameras by name where the scene is a set.
type sessions struct{ stamp hillside.Stamp }

// Stage opens a session for the job. Frames already rendered are not a
// scene to watch, and are refused by name.
func (made sessions) Stage(ctx context.Context, wanted job.Job) (
	live.Stage, error) {
	if _, isFrames := wanted.Scene().Frames(); isFrames {
		return nil, errors.New("watch draws a scene, and this job's " +
			"scene is frames already rendered; slayer draw develops them")
	}
	session, err := hillside.Open(ctx, wanted, made.stamp)
	if err != nil {
		return nil, err
	}
	return staged{session: session, cameras: camerasOf(wanted)}, nil
}

// camerasOf is a set's cameras by name, in its order; none for a scene
// that is not a set, or a set that does not read, which the session has
// already refused.
func camerasOf(wanted job.Job) []string {
	path, isSet := wanted.Scene().Set()
	if !isSet {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	read, err := set.Read(path, raw)
	if err != nil {
		return nil
	}
	return read.Cameras()
}

// staged is a hillside session as watch draws from it.
type staged struct {
	session *hillside.Session
	cameras []string
}

// Frame is the session's frame at the size asked.
func (made staged) Frame(ctx context.Context, index int,
	size live.Size) (live.Drawn, error) {
	ask, err := hillside.NewAsk(size.Width(), size.Height(), size.Rays(),
		nil)
	if err != nil {
		return nil, err
	}
	drawn, err := made.session.Frame(ctx, index, ask)
	if err != nil {
		return nil, err
	}
	return drawn, nil
}

// Times is each frame's scene time.
func (made staged) Times() []time.Duration { return made.session.Times() }

// Tone is the scene's own tone.
func (made staged) Tone() colour.Tone { return made.session.Tone() }

// Shape is the size a frame takes when asked none.
func (made staged) Shape() (width, height int) { return made.session.Size() }

// Exposure is the exposure the scene was staged at.
func (made staged) Exposure() float64 { return made.session.Exposure() }

// Cameras is the set's cameras by name.
func (made staged) Cameras() []string { return made.cameras }

// Close lets the scene go.
func (made staged) Close() error { return made.session.Close() }
