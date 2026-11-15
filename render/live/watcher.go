package live

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/finish"
)

// Given is what a watcher is made of.
type Given struct {
	// Path is the job file, absolute.
	Path string
	// Read is how the job file becomes a job: the door's intake with the
	// command line's flags laid over it.
	Read func(path string) (job.Job, error)
	// Stager stages a job's scene.
	Stager Stager
	// Screen is the terminal the view is shown on.
	Screen Screen
	// Now is the clock.
	Now func() time.Time
	// Ticks is told often: on each the job file is looked at, once
	// PollEvery has passed since the last look, and play moves on.
	Ticks <-chan time.Time
	// Folder is the status folder, written after each picture; nothing
	// is written where it is empty.
	Folder string
}

// Watcher is a job drawn live in a terminal: made by New, run by Run.
type Watcher struct {
	given  Given
	view   view
	foot   foot
	facts  facts
	base   job.Job // the file's last good job
	wanted job.Job // the base with the keys' changes laid over it
	read   bool    // whether any job has read yet
	stamps stamps
	looked time.Time
	shown  *held
	stage  Stage
	// the job's development, made again when the job or the tone
	// changes, and what the shown frame was last developed as
	developing finish.Developer
	titles     *finish.Titles
	version    int
	developed  developed
	// pending is what is owed once the plan in flight is done
	pending Change
	flight  *flight
	results chan message
	timings map[rung]time.Duration
	words   string // a passing word for the status row
	quit    bool
	// unpublished is whether the status folder could not be written,
	// said once; kept is what it holds of the pictures; stopped is
	// whether the watcher has quit
	unpublished bool
	kept        published
	stopped     bool
}

// held is the frame on the screen: its light and hud as drawn, the rung
// it was drawn at and its frame.
type held struct {
	drawn Drawn
	rung  rung
	index int
}

// flight is a plan in flight: how to cancel it.
type flight struct {
	cancel context.CancelFunc
	ctx    context.Context
}

// stamps are the stamp of each file followed, by path.
type stamps map[string]stamp

// stamp is what says a file changed: its time and size; the zero stamp
// for a file that is not there.
type stamp struct {
	modified time.Time
	size     int64
}

// New is a watcher of the file given, checked: a path that is absolute,
// and a reader, stager, screen, clock and ticks, each by name.
func New(given Given) (*Watcher, error) {
	var missing []error
	if !filepath.IsAbs(given.Path) {
		missing = append(missing, errors.New("live: the job file's path "+
			"is not absolute: "+given.Path))
	}
	for _, each := range []struct {
		name  string
		given bool
	}{{"reader of the job", given.Read != nil},
		{"stager", given.Stager != nil}, {"screen", given.Screen != nil},
		{"clock", given.Now != nil}, {"ticks", given.Ticks != nil}} {
		if !each.given {
			missing = append(missing, errors.New("live: no "+each.name+
				" given"))
		}
	}
	if err := errors.Join(missing...); err != nil {
		return nil, err
	}
	return &Watcher{given: given, results: make(chan message, eventRoom),
		timings: map[rung]time.Duration{}}, nil
}

// eventRoom is how many of the worker's messages wait for the loop.
// Picked by the author of this slice, 2026-11-12: a ladder's rungs and
// its staging, with room over.
const eventRoom = 16

// Run shows the job until a key quits, the input ends or the context is
// done. A job that does not read at the start is said at the foot and
// looked at again, as any change is. It returns once the worker has
// stopped and the stage is closed.
func (watching *Watcher) Run(ctx context.Context) error {
	watching.reread()
	watching.show()
	for !watching.quit {
		watching.launch()
		select {
		case event, open := <-watching.given.Screen.Events():
			watching.event(event, open)
		case now := <-watching.given.Ticks:
			watching.tick(now)
		case said := <-watching.results:
			watching.heard(said)
		case <-ctx.Done():
			watching.quit = true
		}
	}
	return watching.land()
}

// event is one thing from the terminal: a key, a resize, or the end.
func (watching *Watcher) event(event terminal.Event, open bool) {
	if !open || event.Kind() == terminal.EventEnd {
		watching.quit = true
		return
	}
	if event.Key().Name() == terminal.KeyHost {
		return
	}
	if event.Kind() == terminal.EventResize {
		watching.fold(Redraw)
		watching.show()
		return
	}
	watching.fold(watching.key(event.Key()))
	watching.show()
}

// tick looks at the job file once PollEvery has passed, and moves play
// on when the next frame is due and nothing is being drawn.
func (watching *Watcher) tick(now time.Time) {
	if now.Sub(watching.looked) >= PollEvery {
		watching.looked = now
		watching.poll()
	}
	if watching.view.playing && watching.flight == nil {
		watching.fold(watching.goTo(watching.playedTo()))
	}
}

// fold takes a change in: a redevelop is seen at the next show, which
// develops the kept light again; a redraw or restage cancels the plan in
// flight and is owed until it stops.
func (watching *Watcher) fold(change Change) {
	if change < Redraw {
		return
	}
	watching.pending = watching.pending.With(change)
	if watching.flight != nil {
		watching.flight.cancel()
	}
}

// launch starts what is owed, when nothing is in flight.
func (watching *Watcher) launch() {
	if watching.flight != nil || watching.pending == Unchanged ||
		!watching.read || watching.quit {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	watching.flight = &flight{cancel: cancel, ctx: ctx}
	restage := watching.pending == Restage || watching.stage == nil
	asked := plan{stage: watching.stage, wanted: watching.wanted,
		restage: restage, index: watching.view.index,
		screen: watching.given.Screen.Screen(), mode: watching.view.mode,
		playing: watching.view.playing, fastest: watching.fastest()}
	watching.stage, watching.pending = nil, Unchanged
	if restage {
		watching.words = stagingWords
	}
	working := worker{stager: watching.given.Stager,
		now: watching.given.Now, out: watching.results}
	go working.run(ctx, asked)
}

// heard is a message from the worker.
func (watching *Watcher) heard(said message) {
	cancelled := watching.flight == nil ||
		watching.flight.ctx.Err() != nil
	switch {
	case said.done:
		watching.landed(said)
	case said.staged:
		watching.opened(said.facts)
	case said.err != nil:
		watching.foot.troubled(said.err.Error())
	case !cancelled:
		watching.timings[said.rung] = said.took
		watching.shown = &held{drawn: said.drawn, rung: said.rung,
			index: said.index}
	}
	watching.show()
}

// opened keeps a fresh stage's facts, clears a refusal, and holds the
// frame shown to the film's end, drawing again where it moved.
func (watching *Watcher) opened(made facts) {
	watching.facts, watching.words = made, ""
	watching.foot.cleared()
	watching.developer()
	if last := watching.frames() - one; watching.view.index > last {
		watching.view.index = last
		watching.fold(Redraw)
	}
}

// landed is the plan done: its stage kept, its refusal said, and the
// loop free for what is owed.
func (watching *Watcher) landed(said message) {
	watching.flight.cancel()
	watching.flight, watching.stage = nil, said.stage
	if said.err != nil {
		watching.foot.troubled(said.err.Error())
		watching.words = ""
	}
}

// land stops the plan in flight, waits for it, and closes the stage.
func (watching *Watcher) land() error {
	for watching.flight != nil {
		watching.flight.cancel()
		said := <-watching.results
		if said.done {
			watching.landed(said)
		}
	}
	watching.stopped = true
	watching.publish(nil, nil, none, false)
	if watching.stage == nil {
		return nil
	}
	return watching.stage.Close()
}

// stagingWords is the status row while the job is being opened.
const stagingWords = "staging..."

// poll reads the job file again when its time or size changed, or that
// of a file it names that is followed: its set, or its titles' theme.
func (watching *Watcher) poll() {
	if _, err := os.Stat(watching.given.Path); err != nil {
		watching.foot.troubled(err.Error())
		watching.show()
		return
	}
	now := stampsOf(watching.followed())
	if maps.Equal(now, watching.stamps) {
		return
	}
	watching.stamps = now
	trouble := watching.foot.trouble
	if watching.reread() != Unchanged || watching.foot.trouble != trouble {
		watching.show()
	}
}

// followed is the files a change to which is drawn again: the job file,
// and the set and the titles' theme the last good job names. What a set
// reads in its turn, a shelf's sources or terrain, is not followed.
func (watching *Watcher) followed() []string {
	paths := []string{watching.given.Path}
	if !watching.read {
		return paths
	}
	if set, isSet := watching.base.Scene().Set(); isSet {
		paths = append(paths, set)
	}
	if titles, has := watching.base.Titles(); has {
		if theme, said := titles.Theme(); said {
			paths = append(paths, theme)
		}
	}
	return paths
}

// stampsOf is the stamp of each path, the zero stamp for one that is not
// there, so its coming back is a change too.
func stampsOf(paths []string) stamps {
	made := stamps{}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			made[path] = stamp{modified: info.ModTime(), size: info.Size()}
			continue
		}
		made[path] = stamp{}
	}
	return made
}

// reread reads the job file: a refusal is said at the foot and the last
// good job and picture kept; a job that reads clears it and is folded
// in at what it changed, which it gives.
func (watching *Watcher) reread() Change {
	read, err := watching.given.Read(watching.given.Path)
	if err != nil {
		watching.foot.troubled(err.Error())
		return Unchanged
	}
	// the digests of the files the job reads are taken when first
	// asked for: take them now, so a later change on disk is seen
	read.Key()
	if watching.foot.cleared() {
		watching.foot.note("the job reads again")
	}
	was, first := watching.wanted, !watching.read
	watching.base, watching.read = read, true
	watching.relaid()
	change := Restage
	if !first {
		change = Compare(was, watching.wanted)
	}
	watching.fold(change)
	return change
}

// relaid is the job shown made again from the file's with the keys'
// camera and moment laid over it. One the job cannot take is said and
// let go. It restages: the session fixes both when it opens.
func (watching *Watcher) relaid() Change {
	var laid []job.Override
	if watching.view.camera != "" {
		laid = append(laid, job.CameraName(watching.view.camera))
	}
	if watching.view.said && watching.base.Clock().Still() {
		laid = append(laid, job.At(watching.view.moment))
	}
	wanted, err := watching.base.With(laid...)
	if len(laid) == none {
		wanted, err = watching.base, nil
	}
	if err != nil {
		watching.foot.note(err.Error())
		watching.view.camera, watching.view.said = "", false
		wanted = watching.base
	}
	watching.wanted = wanted
	watching.developer()
	return Restage
}
