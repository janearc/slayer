// Package smokejudge draws slayer's smokes twice, as the terminal sees them
// and as a real picture, and judges that the terminal's is the real one,
// coarser. It is what `slayer assess --smokes` runs.
//
// A smoke is judged at full size or at small size. Full size is the
// terminal slayer pretends to be, 179 by 50 cells. Small size is half that
// across and down, drawn with fewer rays, so a group of smokes can be
// looked at in a fraction of the time; a smoke that cannot be judged small
// says so on itself (see smoke.Smoke.NoSmall) and is skipped by name.
package smokejudge

import (
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/learn"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/compare"
	"github.com/janearc/slayer/render/frame"
	cache "github.com/janearc/slayer/render/smokecache"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// a sextant is three real pixels across and four down; the terminal a
// terminal picture pretends to be is drawn in pixels, two across and three
// down a cell
const (
	across, down = 3, 4 // defined: the shape of a sextant in real pixels
	fieldFine    = 2    // defined: the finest a field's light asks of a reference; see Fineness
	plainFine    = 1    // defined: a world without a field is judged by a reference one real pixel a sextant third
)

// The numbers of arithmetic and of files this package leans on.
const (
	none        = 0     // defined: no amount, and the first index of a list
	afterFirst  = 1     // defined: where a list is after its first
	exposure    = 1     // defined: a picture developed at unit exposure
	dirMode     = 0o755 // defined: a folder its owner writes and everyone reads
	fileMode    = 0o644 // defined: a file its owner writes and everyone reads
	flagK       = "k"   // defined: the name of the flag that sets the real picture's fineness
	dashFlag    = "-"   // defined: what a flag starts with on the command line
	unsetK      = none  // defined: no -k given, so each smoke's own Fineness
	sextantWide = 0.75  // defined: a sextant pixel is half a cell wide and a third of one tall, on a cell twice as tall as wide
	squarePixel = none  // defined: zero is a square pixel, as film.Pinhole reads it
)

// A terminal picture is drawn in terminal pixels, a cell being this many
// across and down.
const (
	cellWide = 2 // defined: a cell is two terminal pixels across
	cellHigh = 3 // defined: and three down
)

// The terminal slayer pretends to be, in cells, at full size and small.
const (
	fullCellsWide  = 179 // picked: the terminal slayer pretends to be, 179 cells across
	fullCellsHigh  = 50  // picked: and 50 down
	smallCellsWide = 90  // picked: half of full across, rounded up
	smallCellsHigh = 25  // picked: half of full down
)

// size is how big the terminal picture is, in its pixels, and the most
// rays a side it is drawn with. A real picture is across by down times the
// terminal's, times k.
type size struct {
	name         string
	textW, textH int
	rays         int // the most rays a side a pixel; zero is as many as the smoke names
}

// the two sizes. full is 179 by 50 cells. small is half that across and
// down, 90 by 25 cells, and its rays are capped at smallRays a side.
var (
	fullSize = size{name: "full", textW: fullCellsWide * cellWide,
		textH: fullCellsHigh * cellHigh}
	smallSize = size{name: "small", textW: smallCellsWide * cellWide,
		textH: smallCellsHigh * cellHigh, rays: smallRays}
)

// smallRays is how many rays a side a pixel a small picture is drawn with
// at most, picked by judging every smoke small at one, two and three: at
// one, ten smokes with a textured floor or surface were not honest; at two,
// four were; at three, one was (smoke 6, which says so in NoSmall). Only a
// smoke that names more than three, the two black holes, is drawn with
// fewer rays than at full size.
const smallRays = 3

// the verbs, and the words after them in a usage line
const (
	verbCheck = "check"
	verbSmall = "checksmall"
	Usage     = "[GROUP|N...] [-k K] [-out DIR] [-fresh]"
)

// Verbs are the two verbs the smokes answer to, check and checksmall.
func Verbs() []string { return []string{verbCheck, verbSmall} }

// Verb does the smokes asked for, as the command line gives them: args
// is the verb, check or checksmall, then groups and smoke numbers, then
// flags. It writes each verdict to out and says whether every smoke
// judged was honest.
//
//	check [GROUP|N...] [-k K] [-out DIR] [-fresh]
//	checksmall [GROUP|N...] [-k K] [-out DIR] [-fresh]
//
// With no group or number it does every smoke. -k, when given, makes the
// real picture that many times finer than a sextant third; without it each
// smoke gets Fineness of its own world. -fresh draws again what the cache
// holds. -out copies the pictures and verdicts to a folder.
func Verb(args []string, out io.Writer) (bool, error) {
	if len(args) == none || !slices.Contains(Verbs(), args[none]) {
		return false, fmt.Errorf("the verbs are %v, not %q", Verbs(),
			args)
	}
	verb, words := args[none], args[afterFirst:]
	sz := fullSize
	if verb == verbSmall {
		sz = smallSize
	}
	req, err := parse(verb, words)
	if err != nil {
		return false, err
	}
	return req.run(sz, out)
}

// request is what the command line asked for.
type request struct {
	which []smoke.Smoke
	k     int // zero is each smoke's Fineness
	out   string
	fresh bool
}

// sawFlag notes whether a flag of a name was given on the command line.
type sawFlag struct {
	name  string
	given bool
}

// see is what a flag set calls for each flag that was given.
func (saw *sawFlag) see(given *flag.Flag) {
	if given.Name == saw.name {
		saw.given = true
	}
}

// flagWords are the groups and numbers at the head of a verb's arguments,
// and the flags that follow them.
func flagWords(args []string) (picks, flags []string) {
	for len(args) > none && !strings.HasPrefix(args[none], dashFlag) {
		picks, args = append(picks, args[none]), args[afterFirst:]
	}
	return picks, args
}

// parse reads the groups and numbers, then the flags, of a verb's
// arguments. It is nil when they cannot be read.
func parse(verb string, args []string) (*request, error) {
	picks, flags := flagWords(args)
	set := flag.NewFlagSet(verb, flag.ContinueOnError)
	out := set.String("out", "", "a folder to copy the pictures to as well")
	fineness := set.Int(flagK, unsetK, "how much finer again the real picture is; "+
		"at least the smoke's Fineness, which is the default")
	fresh := set.Bool("fresh", false, "draw every smoke again")
	if err := set.Parse(flags); err != nil {
		return nil, err
	}
	saw := &sawFlag{name: flagK}
	set.Visit(saw.see)
	if saw.given && *fineness < plainFine {
		return nil, fmt.Errorf("-k is at least %d, not %d",
			plainFine, *fineness)
	}
	which, err := choose(picks)
	if err != nil {
		return nil, err
	}
	return &request{which: which, k: *fineness, out: *out, fresh: *fresh}, nil
}

// Chosen is the smokes the groups and numbers name, each once, oldest
// first, or every smoke when none is named; a group or number there is
// not is refused by name. It is what Verb judges, for a caller that
// says what would be judged before judging it.
func Chosen(picks []string) ([]smoke.Smoke, error) { return choose(picks) }

// choose is the smokes the groups and numbers name, each once, oldest
// first, or every smoke when none is named.
func choose(picks []string) ([]smoke.Smoke, error) {
	if len(picks) == none {
		return smoke.All(), nil
	}
	chosen := map[int]smoke.Smoke{}
	for _, word := range picks {
		named, err := pick(word)
		if err != nil {
			return nil, err
		}
		for _, each := range named {
			chosen[each.Number] = each
		}
	}
	var which []smoke.Smoke
	for _, each := range smoke.All() {
		if found, ok := chosen[each.Number]; ok {
			which = append(which, found)
		}
	}
	return which, nil
}

// pick is the smokes one word names: a number is that smoke, and
// anything else is a group.
func pick(word string) ([]smoke.Smoke, error) {
	number, err := strconv.Atoi(word)
	if err != nil {
		return smoke.InGroup(word)
	}
	made, ok := smoke.Get(number)
	if !ok {
		return nil, fmt.Errorf("slayer has no smoke %d", number)
	}
	return []smoke.Smoke{made}, nil
}

// build is every smoke asked for with its scene built once, and handed on:
// to the refusal here and then to the shelf, which draws from it on a miss,
// so what was refused is what is drawn.
func (req *request) build() ([]scene, error) {
	scenes := make([]scene, len(req.which))
	for i, each := range req.which {
		scenes[i].world, scenes[i].eye = each.Scene()
		if err := req.refuse(each, scenes[i].world); err != nil {
			return nil, err
		}
	}
	return scenes, nil
}

// run judges the smokes asked for at sz and says whether every one was
// honest.
func (req *request) run(sz size, out io.Writer) (bool, error) {
	scenes, err := req.build()
	if err != nil {
		return false, err
	}
	if req.out != "" {
		if err := os.MkdirAll(req.out, dirMode); err != nil {
			return false, err
		}
	}
	held := openShelf(req.fresh)
	defer held.close()
	announce(held, out)
	honest := true
	for i, each := range req.which {
		ok, err := req.one(held, each, scenes[i], sz, out)
		if err != nil {
			return false, err
		}
		honest = honest && ok
	}
	fmt.Fprintf(out, "drew %d, served %d\n", held.drew, held.served)
	if held.off == "" {
		fmt.Fprintln(out, "pictures and verdicts in",
			filepath.Join(held.root, "v1"))
	}
	if req.out != "" {
		fmt.Fprintln(out, "copies in", req.out)
	}
	return honest, nil
}

// refuse is an error when -k is below what a smoke's world allows.
func (req *request) refuse(made smoke.Smoke, stage *thing.World) error {
	if least := Fineness(stage); req.k != unsetK && req.k < least {
		return fmt.Errorf("smoke %d, %s, has a "+
			"field, and a reference is never coarser "+
			"than what it judges: -k is at least %d, "+
			"not %d", made.Number, made.Name, least, req.k)
	}
	return nil
}

// announce says what the cache is doing, when that is not the usual.
func announce(held *shelf, out io.Writer) {
	if held.off != "" {
		fmt.Fprintln(out, "cache off, so every smoke is drawn and none "+
			"kept:", held.off)
	} else if !cache.Trimmed() {
		fmt.Fprintln(out, "built without -trimpath, so what this binary "+
			"draws is shared with no other tree's")
	}
}

// howDrawn is how a verdict was got, in a few words: served or drawn, and
// why it was not kept when it was not.
func howDrawn(got *shelved) string {
	how := "served"
	if got.drawn {
		how = "drawn"
	}
	if got.unkept != "" {
		how += ", not kept: " + got.unkept
	}
	return how
}

// one judges one smoke at sz, writes its verdict line to out, copies its
// pictures when asked, and says whether it was honest. A smoke that cannot
// be judged small is named, with why, and counts as honest: it was not
// judged.
func (req *request) one(held *shelf, made smoke.Smoke, at scene, sz size,
	out io.Writer) (bool, error) {
	if sz.name == smallSize.name && made.NoSmall() != "" {
		fmt.Fprintf(out, "smoke %d, %s: skipped small, %s\n", made.Number,
			made.Name, made.NoSmall())
		return true, nil
	}
	k := req.k
	if k == unsetK {
		k = Fineness(at.world)
	}
	got, err := held.get(made, at, sz, k)
	if err != nil {
		return false, err
	}
	defer got.entry.Close()
	if req.out != "" {
		if err := copyOut(got, made, req.out); err != nil {
			return false, err
		}
	}
	report := got.report
	fmt.Fprintf(out, "smoke %d, %s: %s, %s, %s at k %d. %d of %d "+
		"terminal pixels compared, worst %.4f at %d,%d, mean %.4f\n",
		made.Number, made.Name, verdictWord(report), howDrawn(got), sz.name, k,
		report.Compared, report.Pixels, report.Worst, report.At.X,
		report.At.Y, report.Mean)
	return report.Honest(), nil
}

// scene is a smoke's world and where its eye is, as Scene built them.
type scene struct {
	world *thing.World
	eye   maths.Subspace
}

// Fineness is the least k a world may be judged at: two for a world
// with a field, since only near a hole is there light thinner than a
// pixel, which one real pixel a sextant third misses and the terminal's
// rays catch; one for the rest. A set is judged by the same rule.
func Fineness(stage *thing.World) int {
	if len(stage.Fields) > none {
		return fieldFine
	}
	return plainFine
}

// copyOut puts a smoke's pictures and verdict in out as well, named for the
// smoke, from the entry that was judged.
func copyOut(got *shelved, made smoke.Smoke, out string) error {
	name := fmt.Sprintf("smoke-%d", made.Number)
	for from, to := range map[string]string{
		cache.Text: name + "-text.png", cache.Real: name + "-real.png",
		cache.Verdict: name + "-verdict.json"} {
		raw, err := got.entry.Bytes(from)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, to), raw,
			fileMode); err != nil {
			return err
		}
	}
	return nil
}

// raysFor is how many rays a side a pixel smoke made is drawn with at sz: the
// smoke's own, never fewer than one, and no more than sz's cap when it has
// one.
func raysFor(made smoke.Smoke, sz size) int {
	rays := max(plainFine, made.Samples)
	if sz.rays > none {
		rays = min(rays, sz.rays)
	}
	return rays
}

// pinholeOf is the lens a smoke is seen through, with a pixel's shape: the
// terminal's sextants, or square ones. It is nil when the smoke has no valid
// field of view.
func pinholeOf(made smoke.Smoke, pixel float64) (*film.Pinhole, error) {
	pinhole, err := film.PinholeOf(made.Wide)
	if err != nil {
		return nil, fmt.Errorf("smoke %d: %w", made.Number, err)
	}
	pinhole.Pixel = pixel
	return pinhole, nil
}

// view is one way a smoke's scene is looked at: through a lens, at a size
// in pixels.
type view struct {
	at     scene
	lens   *film.Pinhole
	width  int
	height int
}

// in is the view in one band outside the visible, one ray a pixel. The
// terminal's visible picture has more rays a pixel than that; drawing its
// bands with as many was tried on 2026-11-12 and changed no smoke's
// verdict, so it was left out.
func (seen view) in(band maths.Band) (*frame.Frame, error) {
	return trace.RenderIn(seen.at.world, seen.at.eye, seen.lens, none,
		seen.width, seen.height, band)
}

// pictured is a smoke's picture from its visible frame and its view. A
// smoke seen in the eye's light alone is developed at unit exposure, as it
// always was.
//
// A smoke that is also seen in bands has each band drawn and
// laid over the visible in false colour, in its palette's colour where it
// has one, through the smoke's own tone: what the Smoke type says its
// Bands, Palette and Tone are for. Until 2026-11-12 the bands were not
// drawn at all, and a smoke lit only in a band was a black picture.
func pictured(made smoke.Smoke, visible *frame.Frame,
	seen view) (*image.RGBA, error) {
	if len(made.Bands) == none {
		return colour.Develop(visible, exposure), nil
	}
	layers := make([]colour.Layer, plainFine, plainFine+len(made.Bands))
	layers[none] = colour.Layer{Frame: visible}
	for index, band := range made.Bands {
		drawn, err := seen.in(band)
		if err != nil {
			return nil, fmt.Errorf("smoke %d: %w", made.Number, err)
		}
		layer := colour.Layer{Band: band, Frame: drawn}
		if index < len(made.Palette) {
			layer.Tint = made.Palette[index]
		}
		layers = append(layers, layer)
	}
	return made.Tone.False(layers), nil
}

// both is one smoke as the terminal shows it, one pixel a sextant, and as a
// real picture of square pixels, from the one scene it is given and the
// same camera: the world is not built again for the second picture. Each
// is the smoke's whole picture, its bands laid over the visible when it
// has them. It returns the tracer's error when either picture cannot be
// drawn.
func both(made smoke.Smoke, at scene, sz size, k int) (terminalPicture,
	realPicture image.Image, err error) {
	sextant, err := pinholeOf(made, sextantWide)
	if err != nil {
		return nil, nil, err
	}
	shot, err := trace.Sampled(at.world, at.eye, sextant, none, sz.textW,
		sz.textH, raysFor(made, sz))
	if err != nil {
		return nil, nil, fmt.Errorf("smoke %d: %w", made.Number, err)
	}
	developed, err := pictured(made, shot,
		view{at: at, lens: sextant, width: sz.textW, height: sz.textH})
	if err != nil {
		return nil, nil, err
	}
	chopped := terminal.Chop(developed)
	terminalPicture = learn.Chopped(chopped)

	square, err := pinholeOf(made, squarePixel)
	if err != nil {
		return nil, nil, err
	}
	wide, high := sz.textW*across*k, sz.textH*down*k
	big, err := trace.Render(at.world, at.eye, square, none, wide, high)
	if err != nil {
		return nil, nil, fmt.Errorf("smoke %d: %w", made.Number, err)
	}
	realPicture, err = pictured(made, big,
		view{at: at, lens: square, width: wide, height: high})
	if err != nil {
		return nil, nil, err
	}
	return terminalPicture, realPicture, nil
}

// verdictWord is a comparison's verdict in a few words. A comparison that
// compared no pixel says so, rather than that the picture differs.
func verdictWord(report compare.Report) string {
	switch {
	case report.Compared == none:
		return "not honest, nothing was compared"
	case !report.Honest():
		return "not honest"
	}
	return "honest"
}
