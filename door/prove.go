package door

// prove holds a raw drawn on the cpu to the exact tracer: it reads the
// job as draw does, stages its scene, makes again the pixels asked of
// the raw and the pixels the raw's own bytes pick, and checks each
// against the raw bit for bit (fermat, through the session's Prove).
//
// It writes the proof's sheet, with the raw developed and the proved
// pixels marked, and says whether the proof holds.

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/session"
)

// proveHelp is prove's page.
const proveHelp = `# prove

slayer prove JOB RAW holds RAW, a raw drawn on the cpu, to the exact
tracer. JOB is a job, a plan or a set file, read as draw reads it, with
the same flags laid over it: give the flags that drew RAW. Chosen pixels
are made again, through the same world, eye, lens, film time, size and
rays the frame is drawn with, and each is held to RAW bit for bit.

--pixel X,Y names a pixel to prove first, counted from the top left;
name several between semicolons, --pixel "3,4;10,2". --many N is how
many more pixels RAW's own bytes pick, 64 unsaid; 0 picks none, and a
proof of nothing is refused. --frame N is the frame of a film that RAW
is, 0 unsaid.

The sheet goes in --proof DIR, or beside RAW, named for it: a page for
the proof and one for each pixel, and RAW developed with the proved
pixels marked as marked.png. slayer prove says the proof and where the
sheet went, and exits 0 when the proof holds and 1 when it fails or
cannot be made.

A picture drawn on the gpu, a camera that starts in an explicit stack of
media, a frame that is the mean of a shutter's glimpses, a film through
a set's rig and a raw of another size than the job's are refused by
name. --dry-run says what would be proved and writes nothing.
`

// The names of prove's own flags; the frame is develop's.
const (
	pixelFlag = "pixel"
	manyFlag  = "many"
	proofFlag = "proof"
)

// proveFlags are prove's own flags, in the order the proof records them.
var proveFlags = []string{pixelFlag, manyFlag, frameFlag, proofFlag}

// pixelSeparator is what stands between the pixels of one --pixel, which
// takes no flag twice: its pixels are X,Y each, between semicolons.
const pixelSeparator = ";"

// proofs is the prove verb, with what it runs under.
type proofs struct {
	env  Env
	shop workshop
}

// proofAsk is what a call asks prove to do: the frame of the job the raw
// is, what is proved, and where the sheet goes, empty for beside the raw.
type proofAsk struct {
	frame   int
	proving session.Proving
	dir     string
}

// prove is the verb: the job read and planned as draw plans it, then a
// dry run says what would be proved, or the raw is proved and its sheet
// written.
func (held proofs) prove(call pzm.Call) error {
	if len(call.Args()) != jobAndRaw {
		return errors.New("prove takes a job and a raw: slayer prove " +
			"JOB RAW")
	}
	ask, err := proofAskOf(call, held.env.directory)
	if err != nil {
		return err
	}
	given := resolved(call.Args()[nothing], held.env.directory)
	run, err := drawing(held).planned(call, given, false)
	if err != nil {
		return err
	}
	if call.DryRun() {
		return wouldProve(call.Out(), run, ask)
	}
	return held.proved(call.Out(), run, ask)
}

// proved opens the job's scene, proves the raw against it, writes the
// sheet and says both; it is an error where the proof fails.
func (held proofs) proved(out io.Writer, run runner, ask proofAsk) error {
	opened, err := run.shop.Source(run.ctx, run.wanted)
	if err != nil {
		return err
	}
	proof, dir, err := proveAndSheet(opened, ask)
	if err = errors.Join(err, opened.Close()); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(out, "%s\n%s: the sheet, and the raw marked\n",
		proof, dir); err != nil {
		return err
	}
	if !proof.Holds() {
		return errors.New("the proof does not hold; the sheet says " +
			"where")
	}
	return nil
}

// proveAndSheet is the source's proof of the raw, and where its sheet
// was written.
func proveAndSheet(opened source, ask proofAsk) (fermat.Proof, string,
	error) {
	proof, err := opened.Prove(ask.frame, ask.proving)
	if err != nil {
		return fermat.Proof{}, "", err
	}
	dir, err := opened.Sheet(proof, ask.dir)
	return proof, dir, err
}

// wouldProve says what prove would do, and does none of it.
func wouldProve(out io.Writer, run runner, ask proofAsk) error {
	where := "beside the raw, named for it"
	if ask.dir != "" {
		where = "in " + ask.dir
	}
	_, err := fmt.Fprintf(out, "would prove %s\nas frame %d of %s, with "+
		"%d named pixels and %d more picked\nwould write the sheet %s\n",
		ask.proving.Raw, ask.frame, filepath.Base(run.given),
		len(ask.proving.Pixels), ask.proving.Many, where)
	return err
}

// proofAskOf is what the call's flags ask of prove, each refused by its
// name where it cannot be read. The pixels are numbers, and a pixel in
// no picture is refused where the raw is read.
func proofAskOf(call pzm.Call, directory string) (proofAsk, error) {
	pixels, err := pixelsOf(call.Value(pixelFlag))
	if err != nil {
		return proofAsk{}, err
	}
	many, err := wholeFlag(call, manyFlag, fermat.Many)
	if err == nil && many < nothing {
		err = fmt.Errorf("--%s takes a whole number, nought or more, "+
			"not %q", manyFlag, call.Value(manyFlag))
	}
	if err != nil {
		return proofAsk{}, err
	}
	frame, err := wholeFlag(call, frameFlag, nothing)
	if err != nil {
		return proofAsk{}, err
	}
	ask := proofAsk{frame: frame, proving: session.Proving{
		Raw:    resolved(call.Args()[one], directory),
		Pixels: pixels, Many: many, Asked: askedOf(call)}}
	if call.Given(proofFlag) {
		ask.dir = resolved(call.Value(proofFlag), directory)
	}
	return ask, nil
}

// pixelsOf is the pixels --pixel names, X,Y each between semicolons, in
// the order named; none where the flag says none.
func pixelsOf(text string) ([][2]int, error) {
	if text == "" {
		return nil, nil
	}
	var pixels [][2]int
	for _, each := range strings.Split(text, pixelSeparator) {
		across, down, found := strings.Cut(each, ",")
		x, wrongX := strconv.Atoi(strings.TrimSpace(across))
		y, wrongY := strconv.Atoi(strings.TrimSpace(down))
		if !found || wrongX != nil || wrongY != nil {
			return nil, fmt.Errorf("--%s takes pixels as X,Y between "+
				"semicolons, not %q", pixelFlag, each)
		}
		pixels = append(pixels, [2]int{x, y})
	}
	return pixels, nil
}

// askedOf is the call as the proof records it: the verb, its words and
// each flag given, in the order declared, the overrides first.
func askedOf(call pzm.Call) string {
	words := append([]string{"slayer", "prove"}, call.Args()...)
	var names []string
	for _, each := range overrides {
		names = append(names, string(each.name))
	}
	names = append(names, proveFlags...)
	for _, name := range names {
		if call.Given(name) {
			words = append(words, "--"+name, call.Value(name))
		}
	}
	return strings.Join(words, " ")
}

// declareProve adds prove to the tool: it writes the proof's sheet.
func declareProve(made *pzm.Tool, env Env, shop workshop) {
	writes := verb.WritesTo("the proof's sheet, at --proof or beside " +
		"the raw")
	proves := verb.NewFacts(writes, verb.Changes, verb.ThisMachine,
		verb.NoOne, verb.FullCPU)
	held := proofs{env: env, shop: shop}
	flags := append(jobFlags(),
		pzm.Value(pixelFlag, verb.NoShort, "pixels to prove first, X,Y "+
			"each between semicolons"),
		pzm.Value(manyFlag, verb.NoShort, "how many more pixels the "+
			"raw's bytes pick"),
		pzm.Value(proofFlag, verb.NoShort, "the folder for the sheet"),
		pzm.Value(frameFlag, verb.NoShort, "which frame of a film the "+
			"raw is"))
	made.Verb("prove", verb.NoShort, "hold a raw drawn on the cpu to the "+
		"exact tracer, bit for bit", proveHelp, "Example_prove", proves,
		held.prove, flags...)
}
