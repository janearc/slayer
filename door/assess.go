package door

// assess says whether a file will draw, and why not, without drawing:
// it reads the file as draw does, lays the same flags over it, works out
// the same places and makes the same refusals.
//
// Then it opens each piece draw would draw with (the scene, the
// development and the titles) and lets it go, drawing no frame.
//
// With --judge it draws a set as the terminal shows it and as a real
// picture and judges the one against the other, as slayer set did; with
// --smokes it judges the numbered smokes, as slayer smokes and
// smokecheck did (render/smokejudge).

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/render/smokejudge"
)

// assessHelp is assess's page.
const assessHelp = `# assess

slayer assess FILE says whether FILE will draw, and if not, why not, in
words. FILE is read as draw reads it, the same flags are laid over it,
and every piece draw would draw with is opened and let go: no frame is
drawn and nothing is written. It says what it would draw and each file
draw would write.

slayer assess FILE --census PATH stages FILE as draw stages it, counts
its first frame's world (a still's at --at, where given) and writes the
census to PATH as JSON, drawing nothing: each placed thing's faces, a
mesh group's instanced and stored faces and its instances, analytic
shapes and volumes by kind with no faces, and the receipt (the job's
sha256, its scene and camera, the time, the build). The faces are the
ones staged in the scene, not the ones visible or hit from the camera.

slayer assess FILE --judge draws a set file as the terminal shows it
and as a real picture, judges the one against the other, and writes
both, the real one again through the set's print, and the verdict, to
--out (out/ unsaid). --k is how much finer the real picture is (2
unsaid), --samples the rays a terminal pixel (1 unsaid), --at the
world's time and --camera the camera, the set's first unsaid.

slayer assess --smokes [GROUP|N...] judges the numbered smokes the same
way, every one when none is named, at full size or with --small at half
the size with fewer rays. --k as above, each smoke's own least unsaid;
--fresh draws again what the cache holds; --out copies the pictures and
verdicts to a folder. slayer list smokes says the smokes and groups.

A judgement that is not honest fails.
`

// The names of assess's own flags.
const (
	judgeFlag   = "judge"
	smokesFlag  = "smokes"
	kFlag       = "k"
	samplesFlag = "samples"
	smallFlag   = "small"
	freshFlag   = "fresh"
)

// What a set is judged at when the flags say nothing. Picked, as
// slayer set had them: a real picture twice as fine as a sextant's
// third, and one ray a terminal pixel.
const (
	judgeFineness = 2
	judgeSamples  = 1
)

// The flags each way of assessing takes; any other given is refused.
var (
	judgeTakes  = []string{judgeFlag, "out", "at", "camera", kFlag, samplesFlag}
	smokesTakes = []string{smokesFlag, "out", kFlag, smallFlag, freshFlag}
	checkTakes  = append([]string{"out", "width", "height", "rays", "at",
		"camera", "renderer", "quality", censusFlag}, switchNames()...)
)

// The judges assess hands a judgement to: a set's here, the smokes'
// render/smokejudge's. A test puts one that finds nothing honest in
// their place, to see what a judgement that is not honest does.
var (
	judgeSetBy    = judgeSet
	judgeSmokesBy = smokejudge.Verb
)

// assessing is the assess verb, with what it runs under.
type assessing struct {
	env  Env
	shop workshop
}

// assess is the verb: the smokes judged, a set judged, or a file
// checked.
func (held assessing) assess(call pzm.Call) error {
	if call.On(smokesFlag) {
		if err := strays(call, smokesTakes, smokesFlag); err != nil {
			return err
		}
		return held.smokes(call)
	}
	if len(call.Args()) != one {
		return errors.New("assess takes one file: slayer assess FILE, " +
			"or slayer assess --smokes [GROUP|N...]")
	}
	given := call.Args()[nothing]
	if !filepath.IsAbs(given) {
		given = filepath.Join(held.env.directory, given)
	}
	if call.On(judgeFlag) {
		if err := strays(call, judgeTakes, judgeFlag); err != nil {
			return err
		}
		return held.judge(call, given)
	}
	if err := strays(call, checkTakes, "assess FILE"); err != nil {
		return err
	}
	return held.check(call, given)
}

// strays refuses a flag given that the way of assessing does not take,
// by its name and the way's.
func strays(call pzm.Call, takes []string, way string) error {
	all := slices.Concat(judgeTakes, smokesTakes, checkTakes)
	for _, name := range all {
		if call.Given(name) && !slices.Contains(takes, name) {
			return fmt.Errorf("--%s does nothing with %s", name, way)
		}
	}
	return nil
}

// check is whether the file will draw: everything draw does before its
// first frame, and no frame.
func (held assessing) check(call pzm.Call, given string) error {
	drawer := drawing{env: held.env, shop: held.shop}
	name := filepath.Base(given)
	run, err := drawer.planned(call, given)
	if err != nil {
		return fmt.Errorf("%s will not draw: %w", name, err)
	}
	lines, err := held.opened(call, run)
	if err != nil {
		return fmt.Errorf("%s will not draw: %w", name, err)
	}
	wanted, at := run.wanted, run.at
	what := "a still"
	if frames := wanted.Clock().Frames(); !wanted.Clock().Still() {
		what = fmt.Sprintf("a film of %d frames", frames)
	}
	lines = append([]string{fmt.Sprintf("%s will draw, as %s; nothing "+
		"was drawn", name, what)}, lines...)
	for _, line := range at.written() {
		lines = append(lines, "draw would write "+line)
	}
	if at.why != "" {
		lines = append(lines, "draw would not write the note: "+at.why)
	}
	_, err = fmt.Fprintln(call.Out(), strings.Join(lines, "\n"))
	return err
}

// opened opens and lets go every piece draw would draw with, and where
// the call asks for a census, writes it from the scene so opened; it
// says the census written, a line, or nothing.
func (held assessing) opened(call pzm.Call, run runner) ([]string,
	error) {
	if !call.Given(censusFlag) {
		return nil, opens(run)
	}
	path := resolved(call.Value(censusFlag), held.env.directory)
	if err := censused(run, path); err != nil {
		return nil, err
	}
	return []string{"wrote the census " + path}, nil
}

// opens makes every refusal draw makes before its first frame, opens the
// pieces draw would draw with and lets them go. It writes nothing.
func opens(run runner) error {
	tools, err := opening(run)
	if err != nil {
		return err
	}
	return tools.source.Close()
}

// opening makes, by draw's own runner and in draw's order, every refusal
// draw makes before its first frame: for a still, a picture another job
// drew; for a film, its sound, movie settings and frames folder; then
// the source, developer and titles opened, the source left open for the
// caller to close.
func opening(run runner) (kit, error) {
	refusal := run.unclaimed
	if !run.wanted.Clock().Still() {
		refusal = run.looked
	}
	if err := refusal(); err != nil {
		return kit{}, err
	}
	return run.tools()
}

// judge draws a set both ways and judges it, or in a dry run says the
// files it would write.
func (held assessing) judge(call pzm.Call, given string) error {
	asked, err := held.judging(call)
	if err != nil {
		return err
	}
	if call.DryRun() {
		staged, err := stageSet(given, asked)
		if err != nil {
			return err
		}
		named := filepath.Join(asked.out, staged.name)
		_, err = fmt.Fprintf(call.Out(), "would write %s-text.png, "+
			"-real.png, -look.png and -verdict.json\n", named)
		return err
	}
	honest, err := judgeSetBy(call.Out(), given, asked)
	if err != nil || honest {
		return err
	}
	return fmt.Errorf("%s is not honest: the terminal's picture is not "+
		"the real one, coarser; the pictures are in %s",
		filepath.Base(given), asked.out)
}

// judging is what the flags ask of a judgement, each unsaid one its
// default; a value a flag cannot take is refused by the flag's name.
func (held assessing) judging(call pzm.Call) (judging, error) {
	asked := judging{out: filepath.Join(held.env.directory, outFolder),
		k: judgeFineness, samples: judgeSamples,
		camera: call.Value("camera")}
	if call.Given("out") {
		asked.out = resolved(call.Value("out"), held.env.directory)
	}
	var err error
	if asked.k, err = wholeFlag(call, kFlag, asked.k); err != nil {
		return judging{}, err
	}
	if asked.samples, err = wholeFlag(call, samplesFlag,
		asked.samples); err != nil {
		return judging{}, err
	}
	if call.Given("at") {
		asked.at, err = time.ParseDuration(call.Value("at"))
		if err != nil {
			return judging{}, fmt.Errorf("--at takes a time such as "+
				"20s or 1m30s, not %q", call.Value("at"))
		}
	}
	return asked, nil
}

// wholeFlag is a flag's whole number, or fallback when it is not given.
func wholeFlag(call pzm.Call, name string, fallback int) (int, error) {
	if !call.Given(name) {
		return fallback, nil
	}
	number, err := strconv.Atoi(call.Value(name))
	if err != nil {
		return nothing, fmt.Errorf("--%s takes a whole number, not %q",
			name, call.Value(name))
	}
	return number, nil
}

// resolved is a path given on the command line, under directory when it
// is relative.
func resolved(path, directory string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(directory, path)
}

// smokes judges the smokes named, or in a dry run says which it would.
func (held assessing) smokes(call pzm.Call) error {
	fine, err := wholeFlag(call, kFlag, one)
	if err != nil {
		return err
	}
	if fine < one {
		return fmt.Errorf("--k is at least 1, not %d", fine)
	}
	if call.DryRun() {
		chosen, err := smokejudge.Chosen(call.Args())
		if err != nil {
			return err
		}
		var numbers []string
		for _, each := range chosen {
			numbers = append(numbers, strconv.Itoa(each.Number))
		}
		_, err = fmt.Fprintf(call.Out(), "would judge smokes %s\n",
			strings.Join(numbers, " "))
		return err
	}
	honest, err := judgeSmokesBy(held.smokeWords(call), call.Out())
	if err != nil || honest {
		return err
	}
	return errors.New("not every smoke judged was honest; each verdict " +
		"is in the lines above, and --out DIR keeps the pictures")
}

// smokeWords is the call as render/smokejudge's Verb reads it: check or
// checksmall, the groups and numbers, then its flags.
func (held assessing) smokeWords(call pzm.Call) []string {
	words := []string{smokejudge.Verbs()[nothing]}
	if call.On(smallFlag) {
		words = []string{smokejudge.Verbs()[one]}
	}
	words = append(words, call.Args()...)
	if call.Given(kFlag) {
		words = append(words, "-"+kFlag, call.Value(kFlag))
	}
	if call.Given("out") {
		words = append(words, "-out", resolved(call.Value("out"),
			held.env.directory))
	}
	if call.On(freshFlag) {
		words = append(words, "-"+freshFlag)
	}
	return words
}

// declareAssess adds assess to the tool: it writes only when it judges
// or counts.
func declareAssess(made *pzm.Tool, env Env, shop workshop) {
	writes := verb.WritesTo("with --judge or --smokes, the pictures " +
		"and verdicts, in --out and the smoke cache, and with --census " +
		"the census")
	judges := verb.NewFacts(writes, verb.Changes, verb.ThisMachine,
		verb.NoOne, verb.FullCPU)
	held := assessing{env: env, shop: shop}
	flags := append(jobFlags(),
		pzm.Switch(judgeFlag, verb.NoShort, "draw a set both ways and "+
			"judge it"),
		pzm.Switch(smokesFlag, verb.NoShort, "judge the smokes named, "+
			"or all"),
		pzm.Value(kFlag, verb.NoShort, "how much finer the real "+
			"picture is"),
		pzm.Value(samplesFlag, verb.NoShort, "rays a terminal pixel, "+
			"for --judge"),
		pzm.Switch(smallFlag, verb.NoShort, "judge the smokes at half "+
			"the size"),
		pzm.Switch(freshFlag, verb.NoShort, "draw again what the smoke "+
			"cache holds"),
		pzm.Value(censusFlag, verb.NoShort, "write the geometry staged "+
			"for the first frame to this file, as JSON"))
	made.Verb("assess", verb.NoShort, "say whether a file will draw, "+
		"or judge a set or the smokes", assessHelp, "Example_assess",
		judges, held.assess, flags...)
}
