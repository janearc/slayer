package door

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/internal/observability"
	"github.com/janearc/slayer/render/progress"
)

// spent is where a frame's time went.
type spent struct {
	draw, develop, titles, write, encode time.Duration
}

// plus is two frames' time, stage by stage.
func (cost spent) plus(other spent) spent {
	return spent{draw: cost.draw + other.draw,
		develop: cost.develop + other.develop,
		titles:  cost.titles + other.titles,
		write:   cost.write + other.write,
		encode:  cost.encode + other.encode}
}

// String is the stages in words, for a line.
func (cost spent) String() string {
	return fmt.Sprintf("draw %s, develop %s, titles %s, write %s, "+
		"encode %s", hundredths(cost.draw), hundredths(cost.develop),
		hundredths(cost.titles), hundredths(cost.write),
		hundredths(cost.encode))
}

// hundredths is a duration in seconds to two places: 0.41 s.
func hundredths(length time.Duration) string {
	return fmt.Sprintf("%.2f s", length.Seconds())
}

// filming is a film in flight: its frames' ledger, its movie's settings
// and the movie once opened, its sound, its meter and its totals.
type filming struct {
	run  runner
	book *ledger // nil where the job writes no frames
	// looking is a film assessed, not drawn: its refusals are made and
	// nothing is written, made, discarded or kept.
	looking bool
	// settings is the movie's, checked before any frame is drawn; nil
	// where there is no movie, or where its size is known only from the
	// first frame.
	settings *movie.Settings
	reel     reel // nil until the first picture, and where no movie
	sound    movie.Sound
	meter    *progress.Meter
	total    spent
	frames   int
	kept     int
}

// film draws a film's frames in order, writing each to the frames folder
// and the movie, skipping a frame the ledger already holds. Everything
// that can be refused is refused before the source is opened; a film
// whose frames and movie are all this job's already draws nothing. A
// cancelled context or a failed frame aborts the movie.
func (run runner) film() error {
	going, err := run.filming()
	if err != nil {
		return err
	}
	if going.done() {
		return going.said(going.nothingLeft())
	}
	tools, err := run.tools()
	if err != nil {
		return err
	}
	err = going.roll(tools)
	return errors.Join(err, tools.source.Close())
}

// roll draws, keeps and encodes every frame, then finishes the movie.
func (going *filming) roll(tools kit) error {
	for index := range going.frames {
		if err := going.run.ctx.Err(); err != nil {
			return going.abort(err)
		}
		if err := going.frame(tools, index); err != nil {
			return going.abort(err)
		}
	}
	return going.finish()
}

// filming is a film ready to draw: its sound made, its movie's settings
// checked, its frames folder looked at, then its ledger opened, the job
// copied, its meter begun. Nothing is written before the movie's
// settings are checked.
func (run runner) filming() (*filming, error) {
	going := &filming{run: run, frames: run.wanted.Clock().Frames()}
	steps := append(going.refusals(), going.ledger, run.copied,
		going.resumed, going.begin)
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	return going, nil
}

// refusals are the steps that refuse a film and write nothing: its
// sound, its movie's settings and its frames folder looked at. Draw
// takes them first; assess takes them alone.
func (going *filming) refusals() []func() error {
	return []func() error{going.soundtrack, going.check, going.peek}
}

// looked is a film assessed: each refusal made, and nothing written.
func (run runner) looked() error {
	going := &filming{run: run, frames: run.wanted.Clock().Frames(),
		looking: true}
	for _, step := range going.refusals() {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// peek is the frames folder as it stands, read and not changed: one
// draw would refuse is refused, and one of an earlier version of this
// job is said, when the film is only looked at, as frames draw redraws.
func (going *filming) peek() error {
	if going.run.at.frames == "" {
		return nil
	}
	stale, err := peekLedger(going.run.at.frames, going.run.keys.frames)
	if err != nil || !stale || !going.looking {
		return err
	}
	return going.said("the frames there are from an earlier version of " +
		"this job; draw would redraw them")
}

// soundtrack makes the film's sound, where the job says one, and says
// how it fits.
func (going *filming) soundtrack() error {
	said, has := going.run.wanted.Sound()
	if !has {
		return nil
	}
	made, err := soundtrack(said, going.run.wanted.Clock().Duration())
	if err != nil {
		return err
	}
	going.sound = made.sound
	return going.said(made.said)
}

// ledger opens the frames folder's ledger, where the job writes frames.
func (going *filming) ledger() error {
	if going.run.at.frames == "" {
		return nil
	}
	at := going.run.at
	book, stale, err := openLedger(at.frames, going.run.keys.frames)
	going.book = book
	if err != nil {
		return err
	}
	if stale {
		err = going.said("the frames here are from an earlier version " +
			"of this job; drawing again")
	}
	if err == nil && at.work {
		err = book.plan(at.movie)
	}
	return err
}

// resumed says so when the film resumes a folder of an earlier day.
func (going *filming) resumed() error {
	if going.run.at.resumed == "" {
		return nil
	}
	return going.said("resuming " + going.run.at.resumed +
		", begun on an earlier day from this job")
}

// begin starts the meter.
func (going *filming) begin() error {
	meter, err := progress.NewMeter(going.run.now)
	if err != nil {
		return err
	}
	going.meter = meter
	return meter.Begin(going.frames, rowsPerFrame)
}

// done is whether every frame is kept and the movie, where there is one,
// was made from them for this job and build.
func (going *filming) done() bool {
	run := going.run
	return going.book != nil && going.book.complete(going.frames) &&
		(run.at.movie == "" || going.book.screened(run.at.movie,
			run.keys.movie))
}

// nothingLeft is the line of a film with nothing left to draw.
func (going *filming) nothingLeft() string {
	if going.run.at.movie == "" {
		return "every frame is this job's already; nothing to draw"
	}
	return "every frame and the movie are this job's already; nothing " +
		"to draw"
}

// said writes a line on the verb's output.
func (going *filming) said(line string) error {
	_, err := fmt.Fprintln(going.run.out, line)
	return err
}

// frame is frame index, kept from the last run where the ledger holds
// it and drawn otherwise, written and handed to the movie, and its line.
func (going *filming) frame(tools kit, index int) error {
	name := fmt.Sprintf(frameName, index)
	picture, cost, kept, err := going.picture(tools, index, name)
	if err != nil {
		return err
	}
	encoding := going.run.now()
	if err := going.encode(index, picture); err != nil {
		return err
	}
	cost.encode = going.run.now().Sub(encoding)
	if observability.Enabled() {
		if kept {
			observability.Add("slayer_frames_kept_total", 1)
			if going.run.at.movie != "" {
				observability.Observe("slayer_frame_kept_read_seconds", cost.write.Seconds())
			}
		} else if going.book != nil {
			observability.Observe("slayer_frame_write_seconds", cost.write.Seconds())
		}
		if going.run.at.movie != "" {
			observability.Observe("slayer_frame_movie_encode_seconds", cost.encode.Seconds())
		}
	}
	going.total = going.total.plus(cost)
	if err := going.meter.FrameDone(); err != nil {
		return err
	}
	if err := going.progress(index, cost, kept); err != nil {
		return err
	}
	going.run.frameEvent(index, cost, kept, false, !kept && going.book != nil, kept && going.run.at.movie != "", going.run.at.movie != "")
	return nil
}

// picture is frame index's picture: read back where the ledger keeps it
// and a movie wants it, else drawn and written to the frames folder.
func (going *filming) picture(tools kit, index int, name string) (
	*image.RGBA, spent, bool, error) {
	run := going.run
	if going.book != nil && going.book.kept(name) {
		going.kept++
		if run.at.movie == "" {
			return nil, spent{}, true, nil
		}
		reading := run.now()
		picture, err := going.book.picture(name)
		return picture, spent{write: run.now().Sub(reading)}, true, err
	}
	frameRate := going.rate()
	finished, err := run.frame(tools, index, filmTime(index,
		frameRate))
	if err != nil || going.book == nil {
		return finished.picture, finished.spent, false, err
	}
	writing := run.now()
	sum, err := file.SequencePNGHash(filepath.Join(going.book.folder,
		name), finished.picture)
	if err != nil {
		return nil, spent{}, false, err
	}
	if err := going.book.note(name, sum); err != nil {
		return nil, spent{}, false, err
	}
	finished.spent.write = run.now().Sub(writing)
	return finished.picture, finished.spent, false, nil
}

// rate is the film's frames a second; a film always has one, as the job
// refuses a film without.
func (going *filming) rate() int {
	rate, _ := going.run.wanted.Clock().Rate()
	return rate
}

// filmTime is when frame index is shown, from the film's start.
func filmTime(index, rate int) time.Duration {
	return time.Duration(index) * time.Second / time.Duration(rate)
}

// encode hands a picture to the movie, opening it at the first.
func (going *filming) encode(index int, picture *image.RGBA) error {
	if going.run.at.movie == "" {
		return nil
	}
	if going.reel == nil {
		if err := going.open(picture.Bounds().Size()); err != nil {
			return err
		}
	}
	return going.reel.Frame(index, picture)
}

// progress is a finished frame's line: its number, where its time went,
// and the time left once the meter offers it.
func (going *filming) progress(index int, cost spent, kept bool) error {
	line := fmt.Sprintf("frame %d of %d: %s", index+one, going.frames,
		cost)
	if kept {
		line = fmt.Sprintf("frame %d of %d: kept from the last run, "+
			"read %s, encode %s", index+one, going.frames,
			hundredths(cost.write), hundredths(cost.encode))
	}
	left, offered := going.meter.Remaining()
	if offered && going.meter.FramesDone() < going.frames {
		line += "; about " + hundredths(left) + " left"
	}
	_, err := fmt.Fprintln(going.run.out, line)
	return err
}

// framesLine says where the frames are: written where the job says them,
// kept as working frames where the call asked, or removed, the movie
// written and closed well, where they were working files.
func (going *filming) framesLine() (string, error) {
	run := going.run
	switch {
	case going.book == nil:
		return "no frames written", nil
	case !run.at.work:
		return fmt.Sprintf("wrote %d frames in %s", going.frames,
			going.book.folder), nil
	case run.keepFrames:
		return fmt.Sprintf("kept the working frames in %s, as asked",
			going.book.folder), nil
	}
	if err := going.book.removeAll(); err != nil {
		return "", err
	}
	if err := os.Remove(going.book.folder); err != nil {
		return "", fmt.Errorf("%s: the working frames are removed and "+
			"the folder is not: %w", going.book.folder, err)
	}
	return "removed the working frames, the movie being written", nil
}

// abort throws the movie away and gives the reason it stopped.
func (going *filming) abort(reason error) error {
	if going.reel == nil {
		return reason
	}
	return errors.Join(reason, going.reel.Abort())
}

// finish closes the movie and says what was written and where the time
// went.
func (going *filming) finish() error {
	run := going.run
	if going.reel != nil {
		if err := going.reel.Close(); err != nil {
			return err
		}
	}
	if going.reel != nil && going.book != nil {
		if err := going.book.filmed(run.at.movie,
			run.keys.movie); err != nil {
			return err
		}
	}
	drawn := going.frames - going.kept
	drawTime := going.meter.Elapsed()
	lines := []string{fmt.Sprintf("drew %d frames and kept %d in %s: %s",
		drawn, going.kept, hundredths(drawTime),
		going.total)}
	framed, err := going.framesLine()
	if err != nil {
		return err
	}
	lines = append(lines, framed)
	if going.reel != nil {
		lines = append(lines, "wrote "+run.at.movie)
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(run.out, line); err != nil {
			return err
		}
	}
	return nil
}
