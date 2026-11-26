package door

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/live"
)

// watchHelp is watch's page.
const watchHelp = `# watch

slayer watch JOB draws a job live in this terminal: small at once, then
finer, up to the pane's size and then more rays, and again whenever the
file is saved. JOB is read as draw reads it, and the same flags stand
over it.

It draws at preview quality unless the job or --quality exact says
otherwise; the status row says which. A preview is never evidence of
what the final render looks like to the bit: slayer draw draws that.

A file that does not read is said at the foot and the last good picture
stays up; fixing the file clears it.

Keys: left and right a frame; [ and ] a second; space plays a film; c
the next camera; e and E exposure by a third of a stop; f the frame at
full size; s writes the moment, camera and exposure into the job; ?
help; q or control-c quits. Control-right-bracket is never bound.

Beside the job, in .NAME.watch/, it keeps frame.png, screen.png and
status.json, each written whole, for a graphical front end.

It needs a terminal; with output that is not one, use slayer draw.
--dry-run prints the job as it would be watched and the status folder.
`

// tick is how often the view is told the time: the job file is looked
// at once live.PollEvery has passed, and play moves on. Picked by the
// author of this slice, 2026-11-12: twenty-five times a second, faster
// than any film's rate here.
const tick = time.Second / 25

// watching is the watch verb, with what it runs under.
type watching struct {
	env    Env
	stager live.Stager
}

// watch is the verb: the file and flags read as a job is, then drawn
// live in the call's terminal until a key quits.
func (watcher watching) watch(call pzm.Call) error {
	if len(call.Args()) != one {
		return errors.New("watch takes one file: slayer watch JOB")
	}
	given := absolute(call.Args()[nothing], watcher.env.directory)
	read, err := readerOf(call, watcher.env.directory)
	if err != nil {
		return err
	}
	if _, err := os.Stat(given); err != nil {
		return fmt.Errorf("no job to watch at %s; slayer edit %s starts "+
			"one", given, given)
	}
	if call.DryRun() {
		return watchDry(call, given, read)
	}
	return watcher.live(call, given, read)
}

// live takes the call's terminal and runs the view on it until it is
// done, and gives the terminal back.
func (watcher watching) live(call pzm.Call, given string,
	read func(string) (job.Job, error)) error {
	in, isIn := call.In().(*os.File)
	out, isOut := call.Out().(*os.File)
	if !isIn || !isOut || terminal.Check(in, out) != nil {
		return errors.New("watch draws in a terminal, and this is not " +
			"one; slayer draw JOB draws the job to files")
	}
	screen, err := terminal.OpenLive(in, out)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	view, err := live.New(live.Given{Path: given, Read: read,
		Stager: watcher.stager, Screen: screen, Now: watcher.env.now,
		Ticks: ticker.C, Folder: statusFolder(given)})
	if err != nil {
		return errors.Join(err, screen.Close())
	}
	ran := false
	defer func() {
		if !ran {
			_ = screen.Close() // a panic in the view: the terminal back first
		}
	}()
	err = view.Run(watcher.env.ctx)
	ran = true
	return errors.Join(err, screen.Close())
}

// readerOf is how watch reads the job file each time it changes, with
// the call's flags laid over it.
func readerOf(call pzm.Call, directory string) (func(string) (job.Job,
	error), error) {
	laid, err := overridesOf(call, directory)
	if err != nil {
		return nil, err
	}
	return reading(laid, call.Given("quality")), nil
}

// reading reads a job file as draw takes it in, the overrides laid over
// it, at preview quality where neither the job nor the command line
// says one.
func reading(laid []job.Override, qualitySaid bool) func(string) (
	job.Job, error) {
	return func(path string) (job.Job, error) {
		wanted, err := jobOf(path)
		if err == nil && len(laid) > nothing {
			wanted, err = wanted.With(laid...)
		}
		if err != nil {
			return job.Job{}, err
		}
		if err := refuseRegionOutput(wanted); err != nil {
			return job.Job{}, err
		}
		if _, said := wanted.Quality(); said || qualitySaid {
			return wanted, nil
		}
		return wanted.With(job.Quality(previewQuality))
	}
}

// previewQuality is the quality watch draws at unless told otherwise.
const previewQuality = "preview"

// statusFolder is the folder beside a job file that watch keeps its
// latest frame and status in: .NAME.watch, NAME the file's name without
// its extension.
func statusFolder(given string) string {
	base := filepath.Base(given)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(filepath.Dir(given), "."+name+".watch")
}

// absolute is a path given on the command line, against the working
// directory where it is relative.
func absolute(given, directory string) string {
	if filepath.IsAbs(given) {
		return given
	}
	return filepath.Join(directory, given)
}

// watchDry says the job as it would be watched, or why it does not
// read, and where the status folder would be.
func watchDry(call pzm.Call, given string,
	read func(string) (job.Job, error)) error {
	wanted, err := read(given)
	if err != nil {
		_, err = fmt.Fprintf(call.Out(), "the job does not read yet: "+
			"%v\n", err)
	} else {
		quality, _ := wanted.Quality()
		_, err = fmt.Fprintf(call.Out(), "would watch %s at %s "+
			"quality\n", given, quality)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(call.Out(), "would write %s\n",
		statusFolder(given))
	return err
}
