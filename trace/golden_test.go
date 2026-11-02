package trace

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	skies "github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// update rewrites the golden pictures from what the tracer draws now.
// No gate passes it: a picture is moved by hand, by a change that meant
// to move it.
var update = flag.Bool("update", false,
	"rewrite trace/testdata/golden from the pictures drawn now")

// goldenSide is how many pixels a golden picture is across and down.
const goldenSide = 16

// goldenRelative and goldenAbsolute are how far a channel may be from
// its golden value: a part in a hundred thousand of it, or a ten
// millionth outright.
//
// arm64 fuses a multiply and an add, so reordering an expression moves
// a number's last bit. That is not a change of picture; a part in a
// thousand is.
const (
	goldenRelative = 1e-5
	goldenAbsolute = 1e-7
)

// lookingAt is an eye at a place, looking at a target.
func lookingAt(from, target maths.Vec) maths.Subspace {
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, target)}
}

// tenBack is the eye most of these tests look from: ten units up z,
// looking at the middle of the world.
func tenBack() maths.Subspace {
	return lookingAt(maths.V(0, 0, 10), maths.Vec{})
}

// golden is one frozen picture: its file's name and how it is drawn.
type golden struct {
	name string
	draw func() *frame.Frame
}

// goldens is every frozen picture, each a scene the tests already use,
// small, drawn the way the tests draw it.
func goldens() []golden {
	return []golden{
		{name: "solid", draw: goldenSolid},
		{name: "gloss", draw: goldenGloss},
		{name: "haze", draw: goldenHaze},
		{name: "puff", draw: goldenPuff},
		{name: "gas", draw: goldenGas},
		{name: "shell", draw: goldenShell},
		{name: "hole", draw: goldenHole},
		{name: "gamma", draw: goldenGamma},
		{name: "radar", draw: goldenRadar},
		{name: "sampled", draw: goldenSampled},
	}
}

// squared is a world drawn at zero, goldenSide pixels each way.
func squared(world *thing.World, eye maths.Subspace,
	lens film.Sensor) *frame.Frame {
	return draw(world, eye, lens, 0, goldenSide, goldenSide)
}

// goldenSolid is smoking(): a solid ball on a floor, a sun, a shadow.
func goldenSolid() *frame.Frame {
	eye := lookingAt(maths.V(0, 2, 6), centre)
	return squared(smoking(), eye, film.Pinhole{})
}

// goldenGloss is a glossy floor under a sky that runs from white at
// the horizon to blue overhead, seen low, so the mirror's share runs
// across the picture.
func goldenGloss() *frame.Frame {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.1), Gloss: 1}})
	world.Sky = skies.Gradient{Horizon: maths.Grey(1),
		Zenith: maths.RGB(0.2, 0.4, 1)}
	eye := lookingAt(maths.V(0, 1, 0), maths.V(0, 0.5, 5))
	return squared(world, eye, film.Pinhole{})
}

// goldenHaze is the glowing floor fading into a sky's own colour.
func goldenHaze() *frame.Frame {
	world := glowing()
	world.Sky = skies.Gradient{Horizon: maths.RGB(0.9, 0.7, 0.5),
		Zenith: maths.RGB(0.2, 0.4, 1)}
	world.Haze = thing.Haze{Density: 0.1, FromSky: true}
	eye := lookingAt(maths.V(0, 1, 0), maths.V(0, 0.5, 5))
	return squared(world, eye, film.Pinhole{})
}

// goldenPuff is puffed(2): smoke over a glowing floor, under a sun so
// the smoke is lit and shades itself.
func goldenPuff() *frame.Frame {
	world := puffed(2)
	world.Lamp(thing.Sun(sunwards, maths.Grey(1)))
	eye := lookingAt(maths.V(0, 2, 3), centre)
	return squared(world, eye, film.Pinhole{Wide: 40 * maths.Degree})
}

// goldenGas is earthlike(): the ball, its lit limb of air and its
// night side, seen from three radii out, side on to the sun.
func goldenGas() *frame.Frame {
	eye := lookingAt(maths.V(0, 0, 3), maths.Vec{})
	return squared(earthlike(), eye, film.Pinhole{Wide: 50 * maths.Degree})
}

// goldenShell is shell() round an opaque giant, two volumes that
// overlap.
func goldenShell() *frame.Frame {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Giant{Radius: 1,
		Depth: 0.08, Thick: 500}, Material: unlit})
	world.AddVolume(shell())
	eye := lookingAt(maths.V(0, 0, 20), maths.Vec{})
	return squared(world, eye, film.Pinhole{Wide: 50 * maths.Degree})
}

// goldenHole is hole() under its compass sky from a hundred masses
// out. The picture's middle column runs from the hole itself, where
// rays are captured, to about thirty-nine masses off it, and its top
// corners pass outside the reach.
func goldenHole() *frame.Frame {
	eye := lookingAt(maths.V(0, 0, -100), maths.V(0, 19, 0))
	lens := film.Pinhole{Wide: 22.6 * maths.Degree}
	return squared(holeWorld(nil), eye, lens)
}

// goldenGamma is a source of gamma behind a ball that passes some of
// it, in the gamma band.
func goldenGamma() *frame.Frame {
	world := thing.NewWorld()
	world.Add(source())
	passing := map[radiation.Range]float64{radiation.Gamma(): 0.5}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5},
		Material: &thing.Material{Name: "glass",
			Radiant: thing.Spectrum{Passes: passing}}})
	eye := tenBack()
	lens := film.Pinhole{Wide: 10 * maths.Degree}
	return drawIn(world, eye, lens, 0, goldenSide, goldenSide, gammaBand)
}

// goldenRadar is a ball that returns radio under a radar at the eye,
// in magellan's band.
func goldenRadar() *frame.Frame {
	world := radarWorld(radarGround(nil), tenBack().Origin)
	lens := film.Pinhole{Wide: 30 * maths.Degree}
	return drawIn(world, tenBack(), lens, 0, goldenSide, goldenSide,
		magellan)
}

// goldenSampled is smoking() with two rays per pixel each way.
func goldenSampled() *frame.Frame {
	eye := lookingAt(maths.V(0, 2, 6), centre)
	return sampled(smoking(), eye, film.Pinhole{}, 0, goldenSide,
		goldenSide, 2)
}

// channels is a pixel as the five numbers a golden file keeps of it:
// red, green, blue, cover and depth.
func channels(pixel frame.Pixel) [5]float64 {
	return [5]float64{float64(pixel.R), float64(pixel.G),
		float64(pixel.B), float64(pixel.Cover), float64(pixel.Depth)}
}

// written is a frame as a golden file's text: a line a pixel, row by
// row from the top left, five numbers to seven figures.
func written(picture *frame.Frame) string {
	var text strings.Builder
	for _, pixel := range picture.Pix {
		numbers := channels(pixel)
		fmt.Fprintf(&text, "%.7g %.7g %.7g %.7g %.7g\n", numbers[0],
			numbers[1], numbers[2], numbers[3], numbers[4])
	}
	return text.String()
}

// read is a golden file's pixels, five numbers each, or why it could
// not be read.
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

// same is whether a channel is the value wanted of it: equal, as two
// infinite depths are, or within a share of it, or within the golden
// absolute tolerance. A NaN is never the same as anything, itself
// included.
func same(got, want, share float64) bool {
	if got == want {
		return true
	}
	gap := math.Abs(got - want)
	return gap <= goldenAbsolute || gap <= share*math.Abs(want)
}

// apart is how far a channel is from the value wanted of it: zero when it is
// the same to a share, and infinitely far when either is a NaN or only one is
// infinite.
func apart(got, want, share float64) float64 {
	if same(got, want, share) {
		return 0
	}
	gap := math.Abs(got - want)
	if math.IsNaN(gap) {
		return math.Inf(1)
	}
	return gap
}

// worst is the channel of a picture furthest from its golden value,
// as a sentence, and whether any channel is off at all.
func worst(picture *frame.Frame, pixels [][5]float64) (string, bool) {
	return worstBy(picture, pixels, goldenRelative)
}

// worstBy is worst with the share of a value a channel may be off by.
func worstBy(picture *frame.Frame, pixels [][5]float64,
	share float64) (string, bool) {
	said, furthest := "", 0.0
	for i, pixel := range picture.Pix {
		got := channels(pixel)
		for k, want := range pixels[i] {
			gap := apart(got[k], want, share)
			if gap <= furthest {
				continue
			}
			furthest = gap
			x, y := i%picture.W, i/picture.W
			said = fmt.Sprintf("pixel %d,%d %s is %.7g, not %.7g",
				x, y, channelName(k), got[k], want)
		}
	}
	return said, furthest > 0
}

// channelName is the name of a golden file's k-th number.
func channelName(k int) string {
	return [5]string{"red", "green", "blue", "cover", "depth"}[k]
}

// The tracer draws its own test scenes as it did when they were
// frozen. A refactor that moves a picture fails here, with the pixel
// that moved most.
func TestGoldenPictures(t *testing.T) {
	for _, scene := range goldens() {
		t.Run(scene.name, func(t *testing.T) { compare(t, scene) })
	}
}

// compare draws one golden scene and holds it to its file, or with
// -update writes the file.
func compare(t *testing.T, scene golden) {
	picture := scene.draw()
	path := filepath.Join("testdata", "golden", scene.name+".txt")
	if *update {
		text := []byte(written(picture))
		if err := os.WriteFile(path, text, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	pixels, err := read(path)
	if err != nil {
		t.Fatalf("%v; draw it once with -update", err)
	}
	if len(pixels) != len(picture.Pix) {
		t.Fatalf("%s holds %d pixels, the picture %d", path,
			len(pixels), len(picture.Pix))
	}
	if said, off := worst(picture, pixels); off {
		t.Errorf("the picture moved; furthest: %s", said)
	}
}
