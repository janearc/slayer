package door

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/learn"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/compare"
	"github.com/janearc/slayer/render/smokejudge"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
	"github.com/janearc/slayer/trace"
)

// The terminal a set is judged on, as the smokes are: 179 by 50 cells
// of sextants, two terminal pixels across a cell and three down, each
// sextant three real pixels across and four down. Defined by
// render/smokejudge's full size, which these are the same as.
const (
	textWide, textHigh = 358, 150
	sextantAcross      = 3
	sextantDown        = 4
)

// The look of the pictures a judgement draws. Defined: a sextant pixel
// is three quarters as wide as it is tall, and a real one is square.
const (
	sextantPixel = 0.75
	realPixel    = 1
	unitExposure = 1
)

// judging is what a judgement of a set is asked: where its pictures go,
// the world's time, how much finer the real picture is, the rays a
// terminal pixel, and the camera by name, the set's first when empty.
type judging struct {
	out     string
	at      time.Duration
	k       int
	samples int
	camera  string
}

// judgedSet is a set read and staged for judging: its name for the
// files, its world, the eye, the lens's width and its film.
type judgedSet struct {
	name   string
	world  *thing.World
	eye    maths.Subspace
	wide   float64
	filmed set.Film
}

// judgeSet draws a set file as the terminal shows it and as a real
// picture, from one world and one camera, judges the terminal's against
// the real, writes both, the real one again through the set's print,
// and the verdict, and says whether it was honest.
func judgeSet(out io.Writer, path string, asked judging) (bool, error) {
	if asked.k < one {
		return false, fmt.Errorf("--k is at least 1, not %d", asked.k)
	}
	staged, err := stageSet(path, asked)
	if err != nil {
		return false, err
	}
	drawn, err := drawBoth(staged, asked)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	report := compare.Cells(drawn.real, drawn.text, sextantAcross*asked.k,
		sextantDown*asked.k)
	if err := drawn.write(filepath.Join(asked.out, staged.name),
		report); err != nil {
		return false, err
	}
	_, err = fmt.Fprintf(out, "set %s at %s: %s. %d of %d terminal "+
		"pixels compared, worst %.4f at %d,%d, mean %.4f\n", staged.name,
		asked.at, verdictWord(report), report.Compared, report.Pixels,
		report.Worst, report.At.X, report.At.Y, report.Mean)
	return report.Honest(), err
}

// both is a set drawn both ways: as the terminal shows it, as a real
// picture, and the real one through the set's print.
type both struct{ text, real, look *image.RGBA }

// drawBoth draws the staged set as the terminal shows it and as a real
// picture k times finer than a sextant's third.
func drawBoth(staged judgedSet, asked judging) (both, error) {
	sextant := film.Pinhole{Wide: staged.wide, Pixel: sextantPixel}
	cells, err := trace.Sampled(staged.world, staged.eye, sextant,
		asked.at, textWide, textHigh, asked.samples)
	if err != nil {
		return both{}, err
	}
	text := learn.Chopped(terminal.Chop(colour.Develop(cells,
		unitExposure)))
	big, err := trace.Render(staged.world, staged.eye,
		film.Pinhole{Wide: staged.wide}, asked.at,
		textWide*sextantAcross*asked.k, textHigh*sextantDown*asked.k)
	if err != nil {
		return both{}, err
	}
	return both{text: text, real: colour.Develop(big, unitExposure),
		look: stage.Print(staged.filmed).Develop(big)}, nil
}

// write writes the three pictures and the verdict, each named for the
// set and what it is, in the folder named's own.
func (drawn both) write(named string, report compare.Report) error {
	if err := os.MkdirAll(filepath.Dir(named), folderMode); err != nil {
		return err
	}
	for _, each := range []struct {
		suffix  string
		picture *image.RGBA
		pixel   float64
	}{
		{"-text.png", drawn.text, sextantPixel},
		{"-real.png", drawn.real, realPixel},
		{"-look.png", drawn.look, realPixel},
	} {
		if err := file.PNG(named+each.suffix, each.picture,
			each.pixel); err != nil {
			return err
		}
	}
	return writeVerdict(named+"-verdict.json", report)
}

// stageSet reads a set file and stages its world and camera for
// judging. A set with no world, one the tracer cannot hold, a k coarser
// than its world allows, or a camera it has not, is refused by name.
func stageSet(path string, asked judging) (judgedSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return judgedSet{}, err
	}
	read, err := set.Read(path, raw)
	if err != nil {
		return judgedSet{}, err
	}
	said, has := read.World()
	if !has {
		return judgedSet{}, fmt.Errorf("%s says no world to draw", path)
	}
	world, err := stage.World(said)
	if err != nil {
		return judgedSet{}, fmt.Errorf("%s: %w", path, err)
	}
	if least := smokejudge.Fineness(world); asked.k < least {
		return judgedSet{}, fmt.Errorf("%s has a field, and a reference "+
			"is never coarser than what it judges: --k is at least %d, "+
			"not %d", path, least, asked.k)
	}
	camera := asked.camera
	if camera == "" && len(read.Cameras()) > nothing {
		camera = read.Cameras()[nothing]
	}
	shot, err := read.Shot(camera, nowhere)
	if err != nil {
		return judgedSet{}, err
	}
	filmed, _ := read.Film()
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if asked.at != nothing {
		name += "-at-" + asked.at.String()
	}
	return judgedSet{name: name, world: world, eye: shot.Eye(),
		wide: stage.Lens(filmed).Wide, filmed: filmed}, nil
}

// verdictWord is a comparison's verdict, in a word or two.
func verdictWord(report compare.Report) string {
	if report.Compared == nothing {
		return "not honest, nothing was compared"
	}
	if !report.Honest() {
		return "not honest"
	}
	return "honest"
}

// verdict is a comparison as it is kept: the numbers, and whether they
// make it honest.
type verdict struct {
	compare.Report
	Honest bool
}

// writeVerdict keeps a comparison's numbers beside its pictures.
func writeVerdict(path string, report compare.Report) error {
	raw, err := json.MarshalIndent(verdict{report, report.Honest()}, "",
		"  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), fileMode)
}

// nowhere answers a camera that stands off an asset: a set judged here
// places none of its assets, so such a camera is refused by name.
func nowhere(string) (maths.Vec, bool) { return maths.Vec{}, false }
