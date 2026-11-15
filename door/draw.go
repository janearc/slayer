package door

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/job"
	yaml "gopkg.in/yaml.v3"
)

// drawHelp is draw's page.
const drawHelp = `# draw

slayer draw JOB draws a job here and writes what it asks for. JOB is a
job file (job-v1), a sequence plan (sequence-v1), or a set file, which is
drawn as a still with the set's own defaults.

The flags stand over the job: --out, --width, --height, --rays, --at,
--camera, --renderer and --quality, and the job's flags and shortcuts,
each on or off: --vibes, --fastlight, --lamp-store and --sky-looks.
--dry-run prints the job as it will
be drawn and each file it would write, and writes nothing.

With no outputs said, a still goes to out/NAME.png and a film to
out/NAME/ and out/NAME.mp4. When SLAYER_RENDERS names a renders home,
outputs said with no directory, or not said, go under it in a dated
folder, with the job file copied beside them; a note the job says is
written to the team's notes file (SLAYER_TEAM), one line a folder.

A film that writes frames resumes: a second run keeps every frame whose
file is unchanged and draws the rest, when the job, the files it reads
and the build are the same; frames of an earlier version are removed
and drawn again. A film with a movie and no frames keeps its frames in
a working folder while it draws, resumes from them, and removes them
once the movie is written; --keep-frames keeps them there.

A still drawn under the renders home is not drawn over one another job
drew in the same folder unless --replace is given.

Control-c stops between frames, throws the movie away and leaves the
frames to resume.
`

// drawing is the draw verb, with what it runs under.
type drawing struct {
	env  Env
	shop workshop
}

// draw is the verb: the file read as a job, the flags laid over it, the
// places worked out; then a dry run says it all, or the runner draws.
func (drawer drawing) draw(call pzm.Call) error {
	if len(call.Args()) != one {
		return errors.New("draw takes one file: slayer draw JOB")
	}
	given := call.Args()[nothing]
	if !filepath.IsAbs(given) {
		given = filepath.Join(drawer.env.directory, given)
	}
	run, err := drawer.planned(call, given)
	if err != nil {
		return err
	}
	if call.DryRun() {
		return dryRun(call.Out(), run.wanted, run.at)
	}
	run.replace, run.keepFrames = call.On(replaceFlag), call.On(keepFlag)
	run.fresh = call.On(freshFlag)
	_, run.keeps = drawer.shop.(pieces)
	return run.draw()
}

// planned is the runner for the file given: its job with the call's
// flags laid over it, its keys and places, and the refusals they make;
// draw's own switches are the caller's to set. draw runs it; assess
// makes its refusals and draws nothing.
func (drawer drawing) planned(call pzm.Call, given string) (runner,
	error) {
	wanted, err := drawer.job(call, given)
	if err != nil {
		return runner{}, err
	}
	drawn := keysOf(wanted, drawer.env.build.drawnBy())
	at := placesOf(placing{wanted: wanted, given: given,
		env: drawer.env, key: drawn.frames})
	if err := refused(wanted, at); err != nil {
		return runner{}, err
	}
	return runner{ctx: drawer.env.ctx, out: call.Out(),
		now: drawer.env.now, shop: drawer.shop, wanted: wanted,
		given: given, at: at, keys: drawn, env: drawer.env}, nil
}

// draw's own switches: to draw a still over one another job drew in the
// same folder, and to keep the working frames of a film that writes a
// movie and no frames.
const (
	replaceFlag = "replace"
	keepFlag    = "keep-frames"
)

// job is the file's job with the call's flags laid over it.
func (drawer drawing) job(call pzm.Call, given string) (job.Job,
	error) {
	wanted, err := jobOf(given)
	if err != nil {
		return job.Job{}, err
	}
	laid, err := overridesOf(call, drawer.env.directory)
	if err != nil || len(laid) == nothing {
		return wanted, err
	}
	return wanted.With(laid...)
}

// refused is why a job cannot be drawn as its places say, by name: sound
// with no movie to carry it, or a movie with no rate.
func refused(wanted job.Job, at places) error {
	_, hasSound := wanted.Sound()
	if hasSound && at.movie == "" {
		return errors.New("the job says sound and writes no movie to " +
			"carry it; say outputs.movie, or take the sound out")
	}
	if _, hasRate := wanted.Clock().Rate(); at.movie != "" && !hasRate {
		return errors.New("the job writes a movie and says no rate; " +
			"say fps, or a film preset that gives one")
	}
	return nil
}

// dryRun says the job as it will be drawn, as YAML, and each file it
// would write, and writes nothing.
func dryRun(out io.Writer, wanted job.Job, at places) error {
	var canonical any
	// the canonical form is JSON the job package wrote, so it reads
	if err := json.Unmarshal(wanted.Canonical(), &canonical); err != nil {
		return err
	}
	text, err := yaml.Marshal(canonical)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "job %s\n%s", wanted.Key(),
		text); err != nil {
		return err
	}
	for _, line := range at.written() {
		if _, err := fmt.Fprintln(out, "would write "+line); err != nil {
			return err
		}
	}
	if at.why != "" {
		_, err = fmt.Fprintf(out, "would not write the note: %s\n", at.why)
	}
	return err
}
