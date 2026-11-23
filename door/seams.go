package door

import (
	"context"
	"image"
	"time"

	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/session"
)

// The seams the runner draws through. Each is the least a caller needs
// of a piece that already exists, so the runner's bookkeeping can be
// tested with mocks (seams_mock_test.go, made by mockgen) and its
// pixels with the real pieces.

// drawn is one frame's light and its own hud, as session.Drawn gives
// them; a plate has no hud.
type drawn interface {
	Light() *frame.Frame
	Overlay(picture *image.RGBA) error
}

// source gives a job's frames of light, in order from nought, and the
// tone that develops them where the job says no print and no stock.
type source interface {
	Frame(ctx context.Context, index int) (drawn, error)
	// Prove replays pixels of a raw of frame index on the exact tracer
	// and holds each to the raw bit for bit, as session.Prove does.
	Prove(index int, proving session.Proving) (fermat.Proof, error)
	// Sheet writes a proof's sheet in dir, or beside its raw where dir
	// is empty, and says where.
	Sheet(proof fermat.Proof, dir string) (string, error)
	Tone() colour.Tone
	Close() error
}

// developer turns a frame's glimpses of light into its picture, as
// finish.Developer does.
type developer interface {
	Develop(number int, glimpses []*frame.Frame) (*image.RGBA, error)
}

// titler draws a job's titles on a picture at a film time, as
// finish.Titles does.
type titler interface {
	Draw(picture *image.RGBA, at time.Duration) error
}

// reel is the movie being written, as movie.Writer is: pictures in
// order, then Close to publish or Abort to throw it away.
type reel interface {
	Frame(index int, picture *image.RGBA) error
	Close() error
	Abort() error
}

// workshop makes the pieces a job is drawn with.
type workshop interface {
	// Source opens the frames the job's scene gives.
	Source(ctx context.Context, wanted job.Job) (source, error)
	// Developer is the job's development, fallback where it says none.
	Developer(wanted job.Job, fallback colour.Tone) (developer, error)
	// Titler is the job's titles, and whether it says any.
	Titler(wanted job.Job) (titler, bool, error)
	// Reel opens a movie for the settings.
	Reel(ctx context.Context, settings movie.Settings) (reel, error)
}

// programs is what edit needs of the machine: where a program is on
// PATH, where slayer itself is, and running one to its end.
type programs interface {
	Find(name string) (string, error)
	Self() (string, error)
	Run(ctx context.Context, path string, args ...string) (string, error)
}
