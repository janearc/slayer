package door

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/job"
	yaml "gopkg.in/yaml.v3"
)

// drawHelp is draw's page.
const drawHelp = `# draw

slayer draw JOB draws a job here and writes what it asks for. JOB is a
job file (job-v1), a sequence plan (sequence-v1), or a set file, which is
drawn as a still with the set's own defaults.

The flags stand over the job: --out, --width, --height, --region, --rays, --spp, --at,
--camera, --renderer and --quality, and the job's flags and shortcuts,
each on or off: --vibes, --fastlight, --lamp-store and --sky-looks.
--dry-run prints the job as it will
be drawn and each file it would write, and writes nothing.

--spp is an exact total per pixel and stands over a job's rays; --rays
counts per side and stands over samples. Give only one of the two flags.
Exact samples support CPU and software-GPU stills without a rig or region.

--fetch-terrain fetches the datasets the set names (its terrain, and
its moon's face) that are not on the shelf, SLAYER_SHELF, checks them
against the sources.yaml nearest the set, and then draws. --refresh-terrain
fetches them all again, replaces a file only with a good one, and then
draws. One at a time. Nothing is downloaded unless one is given.

--places PATH writes a table of where each thing of the set was stood
(of, kind, east, north and height, in metres), as a table of commas, and
draws as well; the picture is drawn, not taken from the cache.

With no outputs said, a still goes to out/NAME.png and a film to
out/NAME/ and out/NAME.mp4. When SLAYER_RENDERS names a renders home,
outputs said with no directory, or not said, go under it in a dated
folder, with the job file copied beside them; a note the job says is
written to the team's notes file (SLAYER_TEAM), one line a folder.

A film that writes frames resumes: a second run keeps every frame whose
file is unchanged and draws the rest, when the job, the files it reads
and the build are the same; frames of an earlier version are removed
and drawn again. A film with a movie and no frames keeps its frames in
a working folder, frames/TEAM under the cache home (SLAYER_CACHE,
refused by name where not set; frames/ alone where SLAYER_TEAM is not
set), while it draws, resumes from them, and removes them once the movie is written;
--keep-frames keeps them there.

A still drawn under the renders home is not drawn over one another job
drew in the same folder unless --replace is given.

A still drawn on the processor says how far it is on standard error
every fifteen seconds: rows finished of all, rows taken up, the time
taken and the time left. Each row is kept under the cache home as it
is finished, so a still that is stopped, or whose process dies, is
picked up from its rows when the same job is drawn again by the same
build, and is the same picture to the byte. The rows are removed once
the still is written. --fresh takes up no rows. A still through a lens
rig, of several shutter glimpses, or told ray by ray neither says how
far it is nor keeps its rows.

Control-c or a terminate stops a still within a ray of each row being
drawn, keeping the rows finished. A film's frame drawn whole on the
processor stops the same way, keeping nothing of that frame; a film
drawn on the graphics card stops between frames. Either way the movie
is thrown away and the frames written are left to resume.
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
	way, tends, err := tendAsked(call, drawTends)
	if err != nil {
		return err
	}
	given := call.Args()[nothing]
	if !filepath.IsAbs(given) {
		given = filepath.Join(drawer.env.directory, given)
	}
	run, err := drawer.planned(call, given, true)
	if err == nil {
		run, err = run.withPlaces(call, drawer.env.directory)
	}
	if err != nil {
		return err
	}
	if call.DryRun() {
		return dryDraw(call.Out(), run, way, tends)
	}
	if _, region := run.wanted.Film().Region(); region && tends {
		return errors.New("region: terrain tending is not supported with a region draw")
	}
	if tends {
		if err := tended(call.Out(), way, run.wanted); err != nil {
			return err
		}
	}
	run.replace, run.keepFrames = call.On(replaceFlag), call.On(keepFlag)
	run.fresh = call.On(freshFlag)
	_, run.keeps = drawer.shop.(pieces)
	return run.draw()
}

// dryDraw is draw's dry run: the job and the files it would write, and
// the tending of the terrain it would do, which it does not.
func dryDraw(out io.Writer, run runner, way tendWay, tends bool) error {
	if err := dryRun(out, run.wanted, run.at); err != nil {
		return err
	}
	if !tends {
		return nil
	}
	return wouldTend(out, way, run.wanted)
}

// declareDraw adds draw to the tool: it writes the job's outputs, and
// the places table and the shelf where it is asked to; the shelf is
// fetched to from outside the machine.
func declareDraw(made *pzm.Tool, drawer drawing) {
	writes := verb.WritesTo("the job's outputs, and its folder and " +
		"note under the renders home, with --places the places " +
		"table, and with --fetch-terrain or --refresh-terrain the shelf")
	draws := verb.NewFacts(writes, verb.Changes, verb.Outside,
		verb.NoOne, verb.FullGPU)
	flags := append(jobFlags(),
		pzm.Value("source-path", verb.NoShort, "original document path for relative inputs in a frozen copy"),
		pzm.List("region", verb.NoShort,
			"a rectangle of the full picture, over the job's"),
		pzm.Switch(replaceFlag, verb.NoShort,
			"draw a still over one another job drew"),
		pzm.Switch(keepFlag, verb.NoShort,
			"keep a movie's working frames once it is written"),
		pzm.Switch(freshFlag, verb.NoShort, "draw a still even where "+
			"the cache holds its picture or its rows"),
		pzm.Value(placesFlag, verb.NoShort, "write where each thing "+
			"was stood to this file, as commas"),
		pzm.Switch(fetchTerrainFlag, verb.NoShort, "fetch the datasets "+
			"the shelf lacks for the set, then draw"),
		pzm.Switch(refreshTerrainFlag, verb.NoShort, "fetch the set's "+
			"datasets again, replacing the shelf's, then draw"))
	made.Verb("draw", verb.NoShort, "carry out a job here and write "+
		"what it asks", drawHelp, "Example_draw", draws, drawer.draw,
		flags...)
}

// planned is the runner for the file given: its job with the call's
// flags laid over it, its keys and places, and the refusals they make;
// draw's own switches are the caller's to set. draw runs it; assess
// makes its refusals and draws nothing.
func (drawer drawing) planned(call pzm.Call, given string, regionFlag bool) (runner,
	error) {
	wanted, err := drawer.job(call, given, regionFlag)
	if err != nil {
		return runner{}, err
	}
	if !regionFlag {
		if err := refuseRegionOutput(wanted); err != nil {
			return runner{}, err
		}
	}
	drawn := keysOf(wanted, drawer.env.build.drawnBy())
	at, err := placesOf(placing{wanted: wanted, given: given,
		env: drawer.env, key: drawn.frames})
	if err != nil {
		return runner{}, err
	}
	if err := refused(wanted, at); err != nil {
		return runner{}, err
	}
	return runner{ctx: drawer.env.ctx, out: call.Out(),
		now: drawer.env.now, shop: drawer.shop, wanted: wanted,
		given: given, at: at, keys: drawn, env: drawer.env}, nil
}

// refuseRegionOutput keeps unverified development and overlays from being written.
func refuseRegionOutput(wanted job.Job) error {
	if _, said := wanted.Film().Region(); said {
		return errors.New("region: output is not yet supported while development and overlays are being checked")
	}
	return nil
}

// draw's own switches: to draw a still over one another job drew in the
// same folder, and to keep the working frames of a film that writes a
// movie and no frames.
const (
	replaceFlag = "replace"
	keepFlag    = "keep-frames"
)

// job is the file's job with the call's flags laid over it.
func (drawer drawing) job(call pzm.Call, given string, regionFlag bool) (job.Job,
	error) {
	var wanted job.Job
	var err error
	if regionFlag && call.Given("source-path") {
		original := call.Value("source-path")
		if original == "" {
			return job.Job{}, errors.New("--source-path needs the original document path")
		}
		if !filepath.IsAbs(original) {
			original = filepath.Join(drawer.env.directory, original)
		}
		wanted, err = jobOfSource(given, original)
	} else {
		wanted, err = jobOf(given)
	}
	if err != nil {
		return job.Job{}, err
	}
	laid, err := overridesOf(call, drawer.env.directory)
	if err == nil && regionFlag && call.Given("region") {
		value := call.List("region")
		made, ok := region(value)
		if !ok {
			return job.Job{}, fmt.Errorf("--region takes left,top,width,height in pixels, not %v", value)
		}
		laid = append(laid, made)
	}
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
