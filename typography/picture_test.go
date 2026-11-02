package typography_test

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	skies "github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"github.com/janearc/slayer/typography"
	"golang.org/x/image/font/gofont/goregular"
)

// update rewrites the golden picture from what is drawn now, as
// trace's -update does: by hand, by a change meant to move it.
var update = flag.Bool("update", false,
	"rewrite typography/testdata/golden from the picture drawn now")

// picture, given a directory, writes the picture there as a png at
// eight times the golden's size, for an eye to look at.
var picture = flag.String("picture", "",
	"write carrie.png, eight times the golden's size, into this directory")

// the golden's size, and how far a channel may be from its golden value:
// trace's own tolerances, a part in a hundred thousand or a ten
// millionth outright, for arm64's fused multiply and add.
const (
	goldenWide, goldenHigh = 96, 32
	goldenRelative         = 1e-5
	goldenAbsolute         = 1e-7
)

// carrie is the word in go's regular, an em high, a quarter em deep,
// bevelled by a hundredth, turned twenty degrees about up so its walls
// and bevels show, on a chalk floor under a sun and a sky.
func carrie(t *testing.T) *thing.World {
	t.Helper()
	face, err := typography.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	letters, err := typography.Make(face, typography.Text{Words: "carrie",
		Size: 1, Depth: 0.25, Bevel: 0.01, Smooth: 30 * maths.Degree})
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.6)}})
	clay := &thing.Material{Colour: maths.RGB(0.75, 0.35, 0.2)}
	turned := thing.Spin{In: maths.Subspace{Origin: maths.V(0, 0.05, 0)},
		Axis: maths.Up, Phase: 20 * maths.Degree}
	for _, each := range typography.Things(letters, clay, turned) {
		world.Add(each)
	}
	sunwards := maths.V(-0.4, 0.8, 0.6).Unit()
	world.Lamp(thing.Sun(sunwards, maths.Grey(math.Pi)))
	world.Sky = skies.Gradient{Horizon: maths.Grey(0.5),
		Zenith: maths.RGB(0.15, 0.25, 0.5)}
	return world
}

// drawn is the world seen from in front and a little above, so wide.
func drawn(t *testing.T, world *thing.World, wide, high int) *frame.Frame {
	t.Helper()
	from, to := maths.V(0, 1.1, 3.6), maths.V(0, 0.35, 0)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, to)}
	picture, err := trace.Render(world, eye,
		film.Pinhole{Wide: 50 * maths.Degree}, time.Duration(0), wide,
		high)
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

// "carrie" in go's regular, extruded, bevelled and lit, is drawn as it
// was when it was frozen: a change to the letters or the tracer that
// moves the picture fails here, with the pixel that moved most.
func TestCarrieIsDrawnAsFrozen(t *testing.T) {
	world := carrie(t)
	got := drawn(t, world, goldenWide, goldenHigh)
	path := filepath.Join("testdata", "golden", "carrie.txt")
	if *picture != "" {
		big := drawn(t, world, 8*goldenWide, 8*goldenHigh)
		out := filepath.Join(*picture, "carrie.png")
		if err := file.PNG(out, colour.Develop(big, 1), 1); err != nil {
			t.Fatal(err)
		}
	}
	if *update {
		err := os.WriteFile(path, []byte(written(got)), 0o644)
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := read(path)
	if err != nil {
		t.Fatalf("%v; draw it once with -update", err)
	}
	if said, off := worst(got, want); off {
		t.Errorf("the picture moved; furthest: %s", said)
	}
}

// the comparison can fail: the word with its last letter gone, or the
// letters smoothed with no angle, is not the frozen picture.
func TestCarrieChangedIsNotTheFrozenPicture(t *testing.T) {
	want, err := read(filepath.Join("testdata", "golden", "carrie.txt"))
	if err != nil {
		t.Fatal(err)
	}
	world := carrie(t)
	world.Things = world.Things[:len(world.Things)-1]
	less := drawn(t, world, goldenWide, goldenHigh)
	if _, off := worst(less, want); !off {
		t.Error("carrie without her e is the frozen picture")
	}
}

// the letters are in the picture where the word is: the middle row
// through them meets letters, nearer than the floor behind, across a
// run of the picture's middle; and the corners' rays go as far as they
// would with no letters, to the floor or the sky, though shadows reach
// them.
func TestCarrieIsWhereTheWordIs(t *testing.T) {
	world := carrie(t)
	floorOnly := thing.NewWorld()
	floorOnly.Add(*world.Things[0])
	floorOnly.Lamps, floorOnly.Sky = world.Lamps, world.Sky
	with := drawn(t, world, goldenWide, goldenHigh)
	without := drawn(t, floorOnly, goldenWide, goldenHigh)
	nearer := 0
	for x := range goldenWide {
		y := goldenHigh / 2
		if with.At(x, y).Depth < without.At(x, y).Depth-1e-9 {
			nearer++
		}
	}
	if nearer < goldenWide/3 || nearer > goldenWide-8 {
		t.Errorf("the letters are nearer than the floor in %d of %d "+
			"columns of the middle row", nearer, goldenWide)
	}
	for _, at := range [][2]int{{0, 0}, {goldenWide - 1, 0},
		{0, goldenHigh - 1}, {goldenWide - 1, goldenHigh - 1}} {
		x, y := at[0], at[1]
		if with.At(x, y).Depth != without.At(x, y).Depth {
			t.Errorf("corner %v sees a letter", at)
		}
	}
}

// channels is a pixel as a golden file keeps it: red, green, blue, how
// much of it is covered, and how far its ray went.
func channels(pixel frame.Pixel) [5]float64 {
	return [5]float64{float64(pixel.R), float64(pixel.G), float64(pixel.B),
		float64(pixel.Cover), float64(pixel.Depth)}
}

// written is a frame as a golden file's text, as trace writes its own:
// a line a pixel, row by row from the top left, to seven figures.
func written(picture *frame.Frame) string {
	var text strings.Builder
	for _, pixel := range picture.Pix {
		n := channels(pixel)
		fmt.Fprintf(&text, "%.7g %.7g %.7g %.7g %.7g\n", n[0], n[1],
			n[2], n[3], n[4])
	}
	return text.String()
}

// read is a golden file's pixels, five numbers each.
func read(path string) ([][5]float64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pixels [][5]float64
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	for row, line := range lines {
		numbers, err := parsed(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, row+1, err)
		}
		pixels = append(pixels, numbers)
	}
	return pixels, nil
}

// parsed is one line of a golden file as its five numbers.
func parsed(line string) ([5]float64, error) {
	var numbers [5]float64
	fields := strings.Fields(line)
	if len(fields) != len(numbers) {
		return numbers, fmt.Errorf("%d numbers, not 5", len(fields))
	}
	for i, field := range fields {
		number, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return numbers, err
		}
		numbers[i] = number
	}
	return numbers, nil
}

// apart is how far a channel is from its golden value: zero when it is the
// same, equal as two endless depths are or within the tolerances, and endlessly
// far when it is a NaN.
func apart(got, want float64) float64 {
	gap := math.Abs(got - want)
	switch {
	case got == want || gap <= goldenAbsolute ||
		gap <= goldenRelative*math.Abs(want):
		return 0
	case math.IsNaN(gap):
		return math.Inf(1)
	}
	return gap
}

// worst is the channel furthest from its golden value as a sentence,
// and whether any is off.
func worst(picture *frame.Frame, want [][5]float64) (string, bool) {
	if len(want) != len(picture.Pix) {
		return fmt.Sprintf("%d pixels, the golden %d", len(picture.Pix),
			len(want)), true
	}
	said, furthest := "", 0.0
	for i, pixel := range picture.Pix {
		for k, got := range channels(pixel) {
			gap := apart(got, want[i][k])
			if gap <= furthest {
				continue
			}
			furthest = gap
			x, y := i%picture.W, i/picture.W
			said = fmt.Sprintf("pixel %d,%d channel %d is %.7g, "+
				"not %.7g", x, y, k, got, want[i][k])
		}
	}
	return said, furthest > 0
}
