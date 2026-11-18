package door

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/internal/observability"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

// runner draws a planned job and writes what it asks for.
type runner struct {
	ctx    context.Context
	out    io.Writer
	now    func() time.Time
	shop   workshop
	wanted job.Job
	given  string
	at     places
	// keys is the records the film keeps, from the job and the build.
	keys keys
	// replace is whether a still may be drawn over one another job drew.
	replace bool
	// keepFrames is whether working frames are kept once the movie is
	// written.
	keepFrames bool
	// env is what the run reads its cache home from; keeps is whether a
	// still's picture is read from the cache and kept there, which only
	// the real workshop's are; fresh draws regardless of the cache.
	env          Env
	keeps, fresh bool
}

// kit is what a job's frames are drawn with.
type kit struct {
	source    source
	developer developer
	titles    titler // nil where the job says none
}

// made is one frame, drawn and finished, and where its time went.
type made struct {
	light   *frame.Frame
	picture *image.RGBA
	spent   spent
}

// draw draws the job: every frame drawn, developed, overlaid and
// titled, and each written where the places say, or a scene of maps
// carried through its frames; then the note written. The job is copied
// beside its outputs only once the run is known to be that folder's.
func (run runner) draw() error {
	var err error
	if scene, isMaps := mapsOf(run.wanted); isMaps {
		err = run.mapped(scene)
	} else if run.wanted.Clock().Still() {
		err = run.drawStill()
	} else {
		err = run.film()
	}
	if err != nil {
		return err
	}
	if err := run.importReceipt(); err != nil {
		return err
	}
	return run.noted()
}

// drawStill draws a still: refused where it would draw over a picture
// another job drew today, then the job copied, then the frame.
func (run runner) drawStill() error {
	if err := run.unclaimed(); err != nil {
		return err
	}
	if err := run.copied(); err != nil {
		return err
	}
	if run.keeping() {
		if served, err := run.keptPicture(); served || err != nil {
			if served && err == nil {
				run.frameEvent(0, spent{}, true, true, false, false, false)
			}
			return err
		}
	}
	tools, err := run.tools()
	if err != nil {
		return err
	}
	err = errors.Join(run.still(tools), tools.source.Close())
	if err != nil || !run.keeping() {
		return err
	}
	return run.keepPicture()
}

// keeping is whether this still's picture goes through the cache: the
// real workshop's, and with no raw asked for, since the raw is not kept.
func (run runner) keeping() bool { return run.keeps && run.at.raw == "" }

// unclaimed refuses a still whose picture is already in its folder
// under the renders home and was drawn from another job file, by name,
// unless the call says to replace it. The same job file draws again.
func (run runner) unclaimed() error {
	if run.at.copied == "" || run.replace {
		return nil
	}
	if _, err := os.Stat(run.at.picture); err != nil {
		return nil
	}
	given, err := os.ReadFile(run.given)
	if err != nil {
		return err
	}
	kept, err := os.ReadFile(run.at.copied)
	if err == nil && bytes.Equal(given, kept) {
		return nil
	}
	return fmt.Errorf("%s was drawn from another job of the same name; "+
		"name this job otherwise, or give --replace to draw over it",
		run.at.picture)
}

// copied copies the file given beside the outputs, where the places say.
func (run runner) copied() error {
	if run.at.copied == "" {
		return nil
	}
	return copyFile(run.given, run.at.copied)
}

// tools opens the job's source and makes its developer and titles; the
// source is closed again when either cannot be made. What the source's
// staging took from the disk caches is said, a line each.
func (run runner) tools() (kit, error) {
	if observability.Enabled() {
		started := time.Now()
		defer func() {
			observability.Observe("slayer_scene_setup_seconds", time.Since(started).Seconds())
		}()
	}
	opened, err := run.shop.Source(run.ctx, run.wanted)
	if err != nil {
		return kit{}, err
	}
	if err := run.cached(opened); err != nil {
		return kit{}, errors.Join(err, opened.Close())
	}
	tools := kit{source: opened}
	tools.developer, err = run.shop.Developer(run.wanted, opened.Tone())
	if err != nil {
		return kit{}, errors.Join(err, opened.Close())
	}
	titles, has, err := run.shop.Titler(run.wanted)
	if err != nil {
		return kit{}, errors.Join(err, opened.Close())
	}
	if has {
		tools.titles = titles
	}
	return tools, nil
}

// noting is a source that says what it took from the disk caches.
type noting interface{ Cached() []string }

// cached says, a line each, what a source's staging took from the disk
// caches or worked out; a source that keeps nothing says nothing.
func (run runner) cached(opened source) error {
	noted, notes := opened.(noting)
	if notes {
		for _, line := range noted.Cached() {
			if _, err := fmt.Fprintln(run.out, line); err != nil {
				return err
			}
		}
	}
	if spectrum, has := opened.(oceanSpectrumNoting); has {
		line, present := spectrum.OceanSpectrumLine()
		if present {
			_, err := fmt.Fprintln(run.out, line)
			return err
		}
	}
	return nil
}

type oceanSpectrumNoting interface {
	OceanSpectrumLine() (string, bool)
}

// frame draws frame index and finishes it: developed, the set's hud and
// the titles at film time at drawn on it.
func (run runner) frame(tools kit, index int, at time.Duration) (made,
	error) {
	start := run.now()
	light, err := tools.source.Frame(run.ctx, index)
	if err != nil {
		return made{}, err
	}
	drew := run.now()
	picture, err := tools.developer.Develop(index,
		[]*frame.Frame{light.Light()})
	if err != nil {
		return made{}, err
	}
	developed := run.now()
	if err := light.Overlay(picture); err != nil {
		return made{}, err
	}
	if tools.titles != nil {
		if err := tools.titles.Draw(picture, at); err != nil {
			return made{}, err
		}
	}
	titled := run.now()
	if observability.Enabled() {
		observability.Add("slayer_frames_drawn_total", 1)
		observability.Observe("slayer_frame_draw_seconds", drew.Sub(start).Seconds())
		observability.Observe("slayer_frame_develop_seconds", developed.Sub(drew).Seconds())
		observability.Observe("slayer_frame_titles_seconds", titled.Sub(developed).Seconds())
	}
	return made{light: light.Light(), picture: picture,
		spent: spent{draw: drew.Sub(start), develop: developed.Sub(drew),
			titles: titled.Sub(developed)}}, nil
}

// still draws the one frame of a still and writes its picture, and its
// light where the job asks for it.
func (run runner) still(tools kit) error {
	finished, err := run.frame(tools, nothing, nothing)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(run.at.picture),
		folderMode); err != nil {
		return err
	}
	if err := file.PNG(run.at.picture, finished.picture,
		square); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(run.out, "wrote %s\n",
		run.at.picture); err != nil {
		return err
	}
	if run.at.raw == "" {
		run.frameEvent(0, finished.spent, false, true, false, false, false)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(run.at.raw),
		folderMode); err != nil {
		return err
	}
	if err := raw.Write(run.at.raw, finished.light); err != nil {
		return err
	}
	_, err = fmt.Fprintf(run.out, "wrote %s\n", run.at.raw)
	if err == nil {
		run.frameEvent(0, finished.spent, false, true, false, false, false)
	}
	return err
}

// noted writes the job's note where it can be written, and says why not
// where it cannot.
func (run runner) noted() error {
	if run.at.why != "" {
		_, err := fmt.Fprintf(run.out, "the note is not written: %s\n",
			run.at.why)
		return err
	}
	if run.at.notes == "" {
		return nil
	}
	note, err := newEntry(run.at.folder, run.at.note)
	if err != nil {
		return err
	}
	if err := note.into(run.at.notes); err != nil {
		return err
	}
	_, err = fmt.Fprintf(run.out, "noted %s in %s\n", run.at.folder,
		run.at.notes)
	return err
}
