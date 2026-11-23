package session

// A named scene, as a session's third source.
//
// Design. A scene built in Go by name (package scene/named) is built once
// at Open from the job's with block, which the scene checks and refuses by
// name. What the job's film, renderer and time say stands over the scene's
// defaults; what they leave unsaid is the scene's, as its command has it.
//
// The scene has one eye of its own. A job's camera path and zoom stand
// over it, in the scene's own unit; a camera by name is refused, since the
// scene has no names for its cameras. Its default tone is the session's
// Tone, with the job's exposure in it where the job says one.
//
// A still the job says no time for is at the scene's default time, and a
// film that says no start begins there: the orbital's first frame is half
// a second in, as its command draws it.
//
// Each frame tells the scene the frame's time first, where it keeps its
// own (the lantern's beams), and, where the job plays the scene by a cue
// sheet, holds the channels the sheet gives at that moment of the music,
// which starts with the film's first frame; then draws.
//
// On the gpu a
// film's frames go through one gpu.Frames: a world with nothing timed
// keeps its generation while only the camera moves, and a timed one takes
// a new one for each new time (see worldChanged).

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/janearc/slayer/cuesheet"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	scenes "github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// sceneNamed is defined as the form job.Scene's Form names a scene built
// in Go by name.
const sceneNamed = "named"

// metres is defined as the unit of a set's and a smoke's lengths, as a
// job's camera path writes it.
const metres = "m"

// notesOf is where a named scene says what it built: standard error for
// a scene whose notes are for the reader, and nowhere for the rest, as a
// job always had them.
func notesOf(scene *scenes.Scene) io.Writer {
	if scene.Noted() {
		return os.Stderr
	}
	return io.Discard
}

// namedFramer draws a named scene: its world, built once, told each
// frame's time where it keeps its own, from its eye or the job's path.
type namedFramer struct {
	built *scenes.Built
	plan  sequencePlan
	// sheet plays the scene's channels where the job gives one, from
	// music whose first sample is at the scene time music
	sheet *cuesheet.Sheet
	music time.Duration
	clock job.Clock
	// frames is the retained drawer where the scene is drawn on the gpu,
	// generation the last generation it was given, and drawnAt the time
	// of the frame it was given for
	frames     *gpu.Frames
	generation uint64
	drawnAt    time.Duration
}

// worldChanged is whether a frame at a time can show another world than the
// last frame drawn, so takes a new generation: the first frame, and a
// frame at another time of a scene whose world changes with time. A
// moved camera changes nothing in the world.
//
// A timed scene drawn again at the same time is the same world, and its
// generation lets the drawer draw only the camera's move. Things that
// only move to a new time are a new generation; the drawer keeps their
// meshes by their bytes (see gpu.Frames).
func worldChanged(first, timed bool, last, at time.Duration) bool {
	return first || (timed && at != last)
}

// openNamed builds a named scene as a job asks: its with block checked
// by the scene, its renderer one the scene draws, its camera its own or
// the job's path in its unit, and its defaults under what the job says.
func openNamed(wanted job.Job, given asked) (*Session, error) {
	name, _ := wanted.Scene().Named()
	scene, err := scenes.Get(name)
	if err != nil {
		return nil, placed(wanted, err)
	}
	if (given.fastLighting || given.restir) && !scene.AllowsFastLighting() {
		return nil, placed(wanted, errFastNamed)
	}
	with := wanted.Scene().Parameters()
	if err := namedReady(wanted, scene, with); err != nil {
		return nil, err
	}
	if err := sceneUnit(wanted, scene.Unit()); err != nil {
		return nil, err
	}
	built, err := scene.Build(with, notesOf(scene))
	if err != nil {
		return nil, placed(wanted, err)
	}
	defaults := scene.Defaults()
	given = given.namedBy(defaults, built)
	source := &namedFramer{built: built, plan: cameraPlan(wanted)}
	if err := given.withNamedMedia(built, wanted); err != nil {
		return nil, placed(wanted, err)
	}
	if given.onGPU {
		if source.frames, err = gpu.NewFrames(framesMode(given)); err != nil {
			return nil, err
		}
	}
	tone := defaults.Tone()
	if given.said.exposure {
		tone.Exposure = given.exposure
	}
	session := newSession(source, given, wanted, tone)
	session.times = startedAt(wanted, defaults.At())
	if source.sheet, err = sheetOf(wanted, built); err != nil {
		return nil, err
	}
	source.music = session.times[nothing]
	source.clock = wanted.Clock()
	// the path is sampled from the film's first frame, wherever it begins
	source.plan.start += session.times[nothing] - wanted.Times()[nothing]
	return session, nil
}

// withNamedMedia gives the session the media a named scene starts in. It
// refuses a camera path, which a fixed start medium cannot serve, and what
// the GPU refuses of initial media.
func (given *asked) withNamedMedia(built *scenes.Built, wanted job.Job) error {
	media := built.InitialMedia()
	if len(media) == 0 {
		return nil
	}
	camera, _ := wanted.Camera()
	if _, hasPath := camera.Path(); hasPath {
		return fmt.Errorf("a camera path cannot use a named scene's fixed start medium")
	}
	for _, material := range media {
		given.initialMedia = append(given.initialMedia, trace.InitialMedium{
			Index: material.Dense(), Rank: material.Rank,
			ClearMedium: material.ClearMedium,
		})
	}
	return refuseGPUInitialMediaAperture(*given)
}

// sheetOf is the cue sheet the job plays the scene by, read against the
// job's tune and the scene's channels, or none where the job gives none.
// The sheet's music starts with the film's first frame, as its sound
// does.
func sheetOf(wanted job.Job, built *scenes.Built) (*cuesheet.Sheet,
	error) {
	path, played := wanted.Scene().Play()
	if !played {
		return nil, nil
	}
	track, _ := wanted.Sound()
	clock, err := soundtrackClock(track)
	if err != nil {
		return nil, placed(wanted, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, placed(wanted, err)
	}
	var channels []cuesheet.Channel
	for _, one := range built.Channels() {
		channels = append(channels, cuesheet.Channel{Name: one.Name,
			Least: one.Least, Most: one.Most, Rest: one.Rest})
	}
	sheet, err := cuesheet.Read(raw, channels, clock)
	if err != nil {
		return nil, placed(wanted, fmt.Errorf("%s: %w", path, err))
	}
	return sheet, nil
}

// namedReady refuses, by name, what a named scene cannot be drawn with:
// a duration written as a quantity, values the scene's Check refuses, a
// renderer it does not draw, and a camera by name.
func namedReady(wanted job.Job, scene *scenes.Scene,
	with map[string]any) error {
	for _, each := range scene.Parameters() {
		if _, quantity := with[each.Name()].(map[string]any); quantity &&
			each.Kind() == scenes.Duration {
			return placed(wanted, fmt.Errorf("%s is a duration, written "+
				"as Go writes one, such as %s: 12h, not as {value, unit}",
				each.Name(), each.Name()))
		}
	}
	if err := scene.Check(with); err != nil {
		return placed(wanted, err)
	}
	renderer, said := wanted.Renderer()
	if said && !scene.Draws(scenes.Renderer(renderer)) {
		return placed(wanted, fmt.Errorf("the %s scene is not drawn by "+
			"%s; it is drawn by %s", scene.Name(), renderer,
			rendererNames(scene.Renderers())))
	}
	if lens, _ := wanted.Camera(); hasName(lens) {
		return placed(wanted, fmt.Errorf("the %s scene has one camera, "+
			"its own; a job gives it a path, not a name", scene.Name()))
	}
	return nil
}

// hasName is whether a job's camera is one by name.
func hasName(lens job.Camera) bool {
	_, said := lens.Name()
	return said
}

// rendererNames are a scene's renderers, written as a list.
func rendererNames(renderers []scenes.Renderer) string {
	words := make([]string, len(renderers))
	for i, each := range renderers {
		words[i] = string(each)
	}
	return strings.Join(words, ", ")
}

// placed is an error with the job file it is about, so a refusal says
// where to look.
func placed(wanted job.Job, err error) error {
	return fmt.Errorf("%s: scene: %w", wanted.File(), err)
}

// sceneUnit refuses a job's camera path written in a unit that is not
// the scene's own, by name.
func sceneUnit(wanted job.Job, own string) error {
	lens, _ := wanted.Camera()
	unit, said := lens.Unit()
	if !said || unit == own {
		return nil
	}
	return placed(wanted, fmt.Errorf("the camera path is in %q, and the "+
		"scene's unit is %q; write the path in the scene's", unit, own))
}

// namedBy is what was asked with a named scene's defaults where the job
// said nothing, and the scene's eye.
func (given asked) namedBy(defaults scenes.Defaults,
	built *scenes.Built) asked {
	for _, each := range []struct {
		said  bool
		value int
		into  *int
	}{
		{given.said.width, defaults.Width(), &given.width},
		{given.said.height, defaults.Height(), &given.height},
		{given.said.samples, defaults.Rays(), &given.samples},
	} {
		if !each.said {
			*each.into = each.value
		}
	}
	if !given.said.wide {
		given.wide = defaults.Wide() / maths.Degree
	}
	eye := built.Eye()
	given.eye = &eye
	return given
}

// startedAt is the job's times, begun at the scene's own time where the
// job says no time for a still or no start for a film.
func startedAt(wanted job.Job, own time.Duration) []time.Duration {
	clock := wanted.Clock()
	_, said := clock.Start()
	if clock.Still() {
		_, said = clock.At()
	}
	times := clock.Times()
	if said {
		return times
	}
	for i := range times {
		times[i] += own
	}
	return times
}

// eyed is the scene advanced to a time and what it is asked there, seen
// from its eye or the job's path: the steps light takes before drawing.
func (source *namedFramer) eyed(at time.Duration, given asked) (asked,
	error) {
	eye, err := source.advance(at)
	if err != nil {
		return given, err
	}
	given.eye = &eye
	given.at = at
	if !given.said.wide {
		if wide, varies := source.built.WideAt(at); varies {
			given.wide = wide / maths.Degree
		}
	}
	err = source.plan.exposing(&given, at)
	return given, err
}

// staged is the scene at a time, told the time where it keeps its own,
// and what is asked of it there, seen from its eye or the job's path,
// before it is drawn.
func (source *namedFramer) staged(at time.Duration, given asked) (
	drawnLight, error) {
	given, err := source.eyed(at, given)
	if err != nil {
		return drawnLight{}, err
	}
	world := source.built.World()
	given.footprint(world)
	return drawnLight{world: world, given: given}, nil
}

// light is the scene at a time, told the time where it keeps its own,
// seen from its eye or the job's path, drawn on the cpu, or on the gpu by
// the session's retained drawer.
func (source *namedFramer) light(at time.Duration, given asked) (
	drawnLight, error) {
	made, err := source.staged(at, given)
	if err != nil {
		return drawnLight{}, err
	}
	world, given := made.world, made.given
	if source.frames == nil {
		made.drawn, made.profile, err = drawMeasured(world, given, nothing)
		return made, err
	}
	timed := source.built.Timed() || source.sheet != nil
	if worldChanged(source.generation == nothing, timed, source.drawnAt,
		at) {
		source.generation++
	}
	source.drawnAt = at
	made.drawn, made.profile, err = given.drawnBy(source.frames, world,
		source.generation, *given.eye)
	return made, err
}

// inspect advances the named source to a time, for inspection.
func (source *namedFramer) inspect(at time.Duration) (*thing.World, error) {
	if _, err := source.advance(at); err != nil {
		return nil, err
	}
	return source.built.World(), nil
}

// advance updates the named scene at absolute time, captures its camera, then
// applies cue-sheet values, preserving Frame's historical update order.
func (source *namedFramer) advance(at time.Duration) (maths.Subspace, error) {
	if err := source.built.Advance(at); err != nil {
		return maths.Subspace{}, err
	}
	eye := source.built.Eye()
	if source.sheet != nil {
		values := source.sheet.At(source.clock.Playback(at - source.music))
		if err := source.built.Play(values); err != nil {
			return maths.Subspace{}, err
		}
	}
	return eye, nil
}

// close lets the retained drawer go, where there is one.
func (source *namedFramer) close() {
	if source.frames != nil {
		source.frames.Close()
	}
}

// errUnitOfSet is the refusal of a camera path on a set or a smoke in a
// unit that is not metres.
var errUnitOfSet = errors.New("a set's and a smoke's camera path is in " +
	"metres, unit: m")

// setUnit refuses a camera path on a set or a smoke in another unit than
// metres.
func setUnit(wanted job.Job) error {
	lens, _ := wanted.Camera()
	if unit, said := lens.Unit(); said && unit != metres {
		return fmt.Errorf("%s: camera unit %q: %w", wanted.File(), unit,
			errUnitOfSet)
	}
	return nil
}
