package hillside

// The render session: a job's scene staged once, and its frames drawn
// one at a time, each at the size its caller asks.
//
// Design. Open reads what a job asks of its scene and lays it over the
// command line's defaults, as parse gives them, so the defaults live in
// one place. Then it stages the scene, once. A job that cannot be staged
// is refused at Open, where a caller expects a refusal, not at a frame.
//
// Precedence is the command line's, then the job's, then the set's, then
// the scene's own default. The record of what was said (sizing) gains
// exposure, wide and renderer, and a job's camera by name, so a set
// stands over only what the job leaves unsaid.
//
// A session draws from one of four sources, each a wrapping of a path
// that exists: a set as a still (staged, then drawMeasured), a set as a
// film (staged, then the retained scene, the job's camera path and the
// keeper, or through the set's lens rig, session_rig.go), a numbered
// smoke (its world, kept, and the same camera), and a scene built in Go
// by name (session_named.go).
//
// They are one exported Session over a small private interface, framer,
// so a caller never sees which it holds. A still has one frame; a film
// has the job's. Frames already rendered are a later slice, and Open
// refuses them by name. A film on the gpu keeps its scene on the device
// between frames, through gpu.Frames, and Close lets it go.
//
// A film's shutter draws each frame once for each of its glimpses, and
// the frame's light is their mean (session_shutter.go). On the gpu the
// eye's own move is blurred inside each glimpse, and a job's lens with an
// aperture is drawn, by the kernel (session_exposure.go).
//
// Frame hands back light, undeveloped. The caller develops it, by the
// scene's own tone (Tone) or its own print or stock, and then asks the
// Drawn for the set's hud on the developed picture.
//
// The context is looked at before each frame is drawn; a cancelled one
// gives its error and draws nothing, and the session stays open and can
// draw again. The tracers take no context, so a frame already being drawn
// runs to its end.
//
// An Ask's sink is told of rows as they finish. The tracers do not say
// rows yet, so the session tells the sink once a frame, when the frame is
// done, with the whole frame as one band. A later slice gives it rows.
//
// A session is used from one goroutine. Close lets the scene's resources
// go; closing twice is harmless, and a closed session draws nothing.

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/progress"
	"github.com/janearc/slayer/scene/thing"
)

// Session is a job's scene, staged once, that draws the job's frames.
// It is made by Open and let go by Close.
type Session struct {
	source framer
	// given is what was asked, as staged: the size, rays and lens a
	// frame takes when its Ask says none.
	given asked
	// lens is the job's film, which works the angle across a frame out
	// of a tall angle for the frame's own shape.
	lens  job.Film
	times []time.Duration
	// offsets is how far after its frame's time each glimpse is drawn: one,
	// at none, where the job says no shutter; and share how long each
	// glimpse's share of the open shutter is, none where it says none
	offsets []time.Duration
	share   time.Duration
	quality string
	tone    colour.Tone
	closed  bool
}

// Ask is what may differ from one frame to the next without staging the
// scene again: a width, a height and rays that stand over the job's, and
// where to say rows as they finish. The zero Ask is the job's own frame.
type Ask struct {
	width, height, rays int
	sink                progress.Sink
}

// Drawn is one frame's light, undeveloped, with its scene time, what the
// graphics card did for it, and what its hud is drawn from.
type Drawn struct {
	// lit is the first glimpse, and at the frame's time
	lit drawnLight
	at  time.Duration
	// glimpses are the light of each glimpse, and mean their mean, where
	// the frame has more than one
	glimpses []*frame.Frame
	mean     *frame.Frame
}

// drawnLight is a frame drawn: its light, what the device did, the
// tally of the world at its moment, and the world and what was asked as
// it was drawn, which its hud is drawn from.
type drawnLight struct {
	drawn   *frame.Frame
	profile gpu.Profile
	made    tally
	world   *thing.World
	given   asked
}

// framer is what a session draws from: a still's staged world, or a
// film's retained one. It draws one frame at a scene time, as asked.
type framer interface {
	light(at time.Duration, given asked) (drawnLight, error)
}

// closer is a source that holds something to let go when the session
// closes: a retained drawer on the gpu.
type closer interface {
	close()
}

// errClosed is the refusal of a frame from a session already closed.
var errClosed = errors.New("hillside: the session is closed; open the " +
	"job again to draw from it")

// NewAsk is an Ask for a picture width by height pixels with rays along
// each side of a pixel, telling sink of rows as they finish. A zero
// stands for the job's own; a nil sink is told nothing. It refuses a
// number below zero, by name.
func NewAsk(width, height, rays int, sink progress.Sink) (Ask, error) {
	for _, each := range []struct {
		name  string
		value int
	}{{"width", width}, {"height", height}, {"rays", rays}} {
		if each.value < nothing {
			return Ask{}, fmt.Errorf("hillside: an ask's %s is %d; it "+
				"is zero for the job's own, or more", each.name,
				each.value)
		}
	}
	return Ask{width: width, height: height, rays: rays, sink: sink}, nil
}

// Open stages the job's scene once and gives a session that draws its
// frames. Nothing is drawn yet. A job the session cannot draw is refused
// here, by name; a cancelled context gives its error and stages nothing.
func Open(ctx context.Context, wanted job.Job, stamp Stamp) (*Session,
	error) {
	return opened(ctx, wanted, stamp)
}

// Times is the scene time of each frame, in order; a still has one.
func (session *Session) Times() []time.Duration {
	return append([]time.Duration(nil), session.times...)
}

// Quality is what the job says of the frames' quality: exact, or
// preview. It does not yet change what is drawn.
func (session *Session) Quality() string { return session.quality }

// Tone is the scene's own tone, which develops the light where the job
// says no print and no stock: a set's print or hillside's own, or a
// smoke's. A job's exposure is in the light of a set with ground, and in
// this tone for a world of a set's pieces and for a smoke.
func (session *Session) Tone() colour.Tone { return session.tone }

// Size is the size a frame takes when its Ask says none: the job's
// film, else the set's, else the command line's default.
func (session *Session) Size() (width, height int) {
	return session.given.width, session.given.height
}

// Exposure is the exposure the scene was staged at: the job's, else the
// set's, else the command line's default. A scene whose light is not
// exposed as it is staged, a world of a set's pieces, is exposed by its
// tone, and this is the tone's.
func (session *Session) Exposure() float64 {
	if session.given.exposure > nothing {
		return session.given.exposure
	}
	if session.tone.Exposure > nothing {
		return session.tone.Exposure
	}
	return toneUnit
}

// toneUnit is the exposure of a tone that says none. Defined: colour.Tone
// reads a zero exposure as one.
const toneUnit = 1.0

// Frame draws frame number index, counted from zero, at the size the
// ask says, and gives its light. A cancelled context gives its error and
// draws nothing; the session stays open.
func (session *Session) Frame(ctx context.Context, index int,
	ask Ask) (Drawn, error) {
	return session.frame(ctx, index, ask)
}

// Cached is what staging took from the disk caches and what it worked
// out and kept, a line each, as a progress line says them.
func (session *Session) Cached() []string {
	lines := make([]string, nothing, len(session.given.cached))
	for _, note := range session.given.cached {
		if note.what != "" {
			lines = append(lines, note.String())
		}
	}
	return lines
}

// Close lets the scene's resources go. Closing twice is harmless.
func (session *Session) Close() error {
	if session.closed {
		return nil
	}
	session.closed = true
	reportTrees(os.Stderr)
	if session.given.live != nil {
		session.given.live.closeBeds()
	}
	session.given.kept.close()
	if source, holds := session.source.(closer); holds {
		source.close()
	}
	return nil
}

// Light is the frame's light, undeveloped. The session may keep it to
// draw the next frame from, so it is read and not written.
//
// Through a shutter of more than one glimpse it is their mean, as a plain
// tone develops it; Glimpses gives each, for a film stock that prints
// them together.
func (drawn Drawn) Light() *frame.Frame {
	if drawn.mean != nil {
		return drawn.mean
	}
	return drawn.lit.drawn
}

// Glimpses is the light of each glimpse the shutter took, in order of
// time: one, the frame's light, where the job says no shutter.
func (drawn Drawn) Glimpses() []*frame.Frame {
	return append([]*frame.Frame(nil), drawn.glimpses...)
}

// At is the frame's scene time: where the shutter opens.
func (drawn Drawn) At() time.Duration { return drawn.at }

// Profile is what the graphics card did for the frame's first glimpse;
// it is empty for a frame drawn on the cpu.
func (drawn Drawn) Profile() gpu.Profile { return drawn.lit.profile }

// Overlay draws the set's own hud, where it has one, on a developed
// picture of the frame: seen through the frame's eye and lens, at its
// moment; through a shutter, as its first glimpse sees it. It refuses a
// picture of another size than the frame, by name.
func (drawn Drawn) Overlay(picture *image.RGBA) error {
	return drawn.lit.overlay(picture)
}
