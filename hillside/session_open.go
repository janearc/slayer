package hillside

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	milestone "github.com/janearc/slayer/scene/smoke"
)

// The scene's forms a session opens, as job.Scene's Form names them.
const (
	sceneSet   = "set"   // defined: a set file
	sceneSmoke = "smoke" // defined: a numbered smoke
)

// singleRay is defined as a single ray through the middle of a pixel.
const singleRay = 1

// opener stages a job's scene of one form into a session, from what was
// asked of it.
type opener func(wanted job.Job, given asked) (*Session, error)

// openers is the opener of each form a session draws.
var openers = map[string]opener{sceneSet: openSet, sceneSmoke: openSmoke,
	sceneNamed: openNamed}

// opened is Open: the stamp kept as Run keeps it, the shutter's glimpses
// placed, what the job asks laid over the defaults, and its scene staged
// by the opener of its form.
func opened(ctx context.Context, wanted job.Job, stamp Stamp) (*Session,
	error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	build, built, who, sessions = stamp.build, stamp.built, stamp.who,
		stamp.sessions
	form := wanted.Scene().Form()
	open, known := openers[form]
	if !known {
		return nil, fmt.Errorf("hillside: a scene given as %s is not "+
			"drawn by a session yet; a set, a smoke or a named scene is",
			form)
	}
	offsets, share, err := shutterOffsets(wanted)
	if err != nil {
		return nil, err
	}
	if err := cachingTrees(); err != nil {
		return nil, err
	}
	given, err := jobAsked(wanted)
	if err != nil {
		return nil, err
	}
	session, err := open(wanted, given)
	if err != nil {
		return nil, err
	}
	if session.offsets == nil {
		// a rig's opener has placed its own shutter's glimpses; a rig's
		// film has no job's shutter, so no share of one
		session.offsets, session.share = offsets, share
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, session.Close())
	}
	return session, nil
}

// jobAsked is what a job asks: the command line's defaults, as parse
// gives them, with what the job says laid over them, each marked said
// so the set stands over none of it, and checked as the command line is.
func jobAsked(wanted job.Job) (asked, error) {
	given, err := parse(nil, io.Discard)
	if err != nil {
		return asked{}, err
	}
	given.set, _ = wanted.Scene().Set()
	given.at, _ = wanted.Clock().At()
	given = given.filmedBy(wanted.Film())
	if name, said := wanted.Renderer(); said {
		given = given.renderedBy(name)
		given.said.renderer = true
	}
	lens, _ := wanted.Camera()
	given.camera, _ = lens.Name()
	given = given.switchedBy(wanted).resolved(noSetFlags)
	return given, given.sound()
}

// filmedBy is what was asked with what a job's film says in it: its
// size, rays, exposure and lens, each marked said. A tall angle is the
// wide one for this size; each frame works it out again for its own.
func (given asked) filmedBy(made job.Film) asked {
	for _, each := range []struct {
		value func() (int, bool)
		into  *int
		said  *bool
	}{
		{made.Width, &given.width, &given.said.width},
		{made.Height, &given.height, &given.said.height},
		{made.Rays, &given.samples, &given.said.samples},
	} {
		if value, said := each.value(); said {
			*each.into, *each.said = value, true
		}
	}
	if exposure, said := made.Exposure(); said {
		given.exposure, given.said.exposure = exposure, true
	}
	if wide, said := made.WideFor(given.width, given.height); said {
		given.wide, given.said.wide = wide, true
	}
	given.optics.aperture, _ = made.Aperture()
	given.optics.focus, _ = made.Focus()
	return given
}

// errWorldOnlyFilm is the refusal of a film of a set that is a world
// alone, with no ground, planet or things: the world half draws it, and
// draws stills.
var errWorldOnlyFilm = errors.New("a set of a world with nothing to " +
	"stand on is drawn by the world half, which draws stills; give it " +
	"ground, a planet or things to film it")

// openSet stages a set as a still or as a film, as the job's clock says.
func openSet(wanted job.Job, given asked) (*Session, error) {
	if err := setUnit(wanted); err != nil {
		return nil, err
	}
	if wanted.Clock().Still() {
		return openStill(wanted, given)
	}
	return openFilm(wanted, given)
}

// openStill stages a set as the still path does, all it can draw: sets
// with no ground, and lens rigs among it.
func openStill(wanted job.Job, given asked) (*Session, error) {
	world, made, given, err := staged(given)
	if err == nil {
		err = rigZoom(wanted, given)
	}
	if err != nil {
		letGo(given)
		return nil, err
	}
	still := stillFramer{world: world, made: made,
		plan: cameraPlan(wanted)}
	return newSession(still, given, wanted, given.tone()), nil
}

// openFilm stages a set as the sequence path does, with its refusals:
// a world with nothing to stand on, which only the world half draws, and
// lighting it cannot retain. A planet alone, or a set in space, needs no
// ground: nothing a film keeps from frame to frame is the ground's.
func openFilm(wanted job.Job, given asked) (*Session, error) {
	read, err := readDrawable(given.set)
	if err != nil {
		return nil, err
	}
	if worldOnly(read) {
		return nil, fmt.Errorf("%s: %w", given.set, errWorldOnlyFilm)
	}
	_, _, given, err = staged(given)
	if err != nil {
		letGo(given)
		return nil, err
	}
	if given.glass != nil {
		session, err := openRigFilm(wanted, given)
		if err != nil {
			letGo(given)
		}
		return session, err
	}
	if err = retainable(given); err != nil {
		letGo(given)
		return nil, fmt.Errorf("%s: %w", given.set, err)
	}
	if given.kept, err = filmKeeper(given); err != nil {
		letGo(given)
		return nil, err
	}
	film := timedFramer{plan: cameraPlan(wanted)}
	return newSession(film, given, wanted, given.tone()), nil
}

// openSmoke stages a numbered smoke as a smoke sequence does, drawn by
// the job's renderer. A smoke the session would draw without something
// of its own is refused, and so is a camera by name: a smoke has one.
func openSmoke(wanted job.Job, given asked) (*Session, error) {
	number, _ := wanted.Scene().Smoke()
	scene, exists := milestone.Get(number)
	if !exists {
		return nil, fmt.Errorf("hillside: no smoke %d", number)
	}
	if err := plainSmoke(scene); err != nil {
		return nil, err
	}
	if err := setUnit(wanted); err != nil {
		return nil, err
	}
	if given.camera != "" {
		return nil, fmt.Errorf("hillside: smoke %d has one camera, its "+
			"own, and the job names camera %q", number, given.camera)
	}
	if _, err := scene.RequireInputs(); err != nil {
		return nil, err
	}
	tone := scene.Tone
	if given.said.exposure {
		tone.Exposure = given.exposure
	}
	given = smokeSeen(scene, given)
	smoke := timedFramer{plan: cameraPlan(wanted)}
	return newSession(smoke, given, wanted, tone), nil
}

// plainSmoke refuses, by name, a smoke with bands, a gauge or a palette
// of its own, which a session would draw without them.
func plainSmoke(scene milestone.Smoke) error {
	for _, own := range []struct {
		has  bool
		what string
	}{
		{len(scene.Bands) > nothing, "bands"},
		{scene.Gauge != (film.Gauge{}), "a gauge"},
		{len(scene.Palette) > nothing, "a palette"},
	} {
		if own.has {
			return fmt.Errorf("hillside: smoke %d has %s of its own, "+
				"which a session would draw without; slayer assess "+
				"--smokes judges it with them", scene.Number, own.what)
		}
	}
	return nil
}

// cameraPlan is the part of a sequence plan withCamera reads, from the
// job: its camera path and zoom, each sampled from the film's start, or
// for a still at its own moment, so a still sees a path's first place.
func cameraPlan(wanted job.Job) sequencePlan {
	lens, _ := wanted.Camera()
	path, _ := lens.Path()
	zoom, _ := lens.Zoom()
	clock := wanted.Clock()
	start, _ := clock.Start()
	if clock.Still() {
		start, _ = clock.At()
	}
	return sequencePlan{camera: path, zoom: zoom, start: start}
}

// letGo lets go what a staging that failed kept, as Run does.
func letGo(given asked) {
	if given.live != nil {
		given.live.closeBeds()
	}
}

// newSession is a session of a staged source and what was asked of it,
// the job's times, quality and film, and the scene's own tone.
func newSession(source framer, given asked, wanted job.Job,
	tone colour.Tone) *Session {
	quality, _ := wanted.Quality()
	return &Session{source: source, given: given, lens: wanted.Film(),
		times: wanted.Times(), quality: quality, tone: tone}
}
