package stage_test

import (
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// yard is the yard's set file as written, changed by each pair of was
// and now.
func yard(t *testing.T, changes ...string) string {
	t.Helper()
	source, err := os.ReadFile("testdata/yard.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := string(source)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(written, changes[i]) {
			t.Fatalf("%q is not in the yard", changes[i])
		}
		written = strings.Replace(written, changes[i], changes[i+1], 1)
	}
	return written
}

// read is the yard's set, as written or changed.
func read(t *testing.T, changes ...string) set.Set {
	t.Helper()
	made, err := set.Read("yard.yaml", []byte(yard(t, changes...)))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// built is the yard's world, built, as written or changed.
func built(t *testing.T, changes ...string) *thing.World {
	t.Helper()
	said, _ := read(t, changes...).World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	return world
}

// near is two numbers alike to a part in a trillion.
func near(a, b float64) bool { return math.Abs(a-b) <= 1e-12*math.Abs(b) }

// saturn is the yard's saturn placed alone: its template, tilted 12
// degrees, its aurora as said, at the yard's place for it.
func saturn(t *testing.T, aurora bool) *thing.World {
	t.Helper()
	body, _ := template.Named("saturn")
	body.Tilt = 12 * maths.Degree
	if !aurora {
		body.Aurora = nil
	}
	alone := thing.NewWorld()
	if err := body.Place(alone, maths.V(0, 0, -600000000), 1e3,
		1); err != nil {
		t.Fatal(err)
	}
	return alone
}

// a box stands where its set says, as big as it says, in the material
// it wears.
func TestABoxIsWhereItSays(t *testing.T) {
	lawn := built(t).Things[0]
	bounds := lawn.Shape.(*mesh.Mesh).Bounds()
	if lawn.Material.Name != "grass" ||
		lawn.Material.Colour != maths.RGB(0.05, 0.06, 0.04) ||
		lawn.Material.Glow != (maths.Light{}) ||
		lawn.At(0).Origin != maths.V(1, 3, -2) ||
		bounds.Max.Sub(bounds.Min) != maths.V(4, 6, 5) ||
		!lawn.Shape.(*mesh.Mesh).Closed() {
		t.Errorf("the lawn is %+v in %+v, %v", lawn, lawn.Material, bounds)
	}
}

// a glow is a black body's light at the luminance asked, warm for a
// bulb's heat; a material may glow and have a colour both.
func TestAGlowIsABlackBodysLight(t *testing.T) {
	world := built(t)
	pane, coal := world.Things[1].Material, world.Things[2].Material
	if pane.Name != "lit-window" || !near(pane.Glow.Luminance(), 0.45) ||
		!(pane.Glow.R > pane.Glow.G && pane.Glow.G > pane.Glow.B) ||
		pane.Colour != (maths.Light{}) {
		t.Errorf("the pane is %+v", pane)
	}
	if coal.Name != "ember" || !near(coal.Glow.Luminance(), 0.02) ||
		coal.Colour != maths.RGB(0.3, 0.1, 0) ||
		!(coal.Glow.R/coal.Glow.G > pane.Glow.R/pane.Glow.G) {
		t.Errorf("the coal is %+v, the pane %+v", coal, pane)
	}
}

// a lamp is a bulb at its place, as bright as asked at a metre, lighting
// from beyond its radius.
func TestALampIsABulb(t *testing.T) {
	porch, ok := built(t).Lamps[0].(emit.Lit)
	if !ok || porch.At != maths.V(-2, 7.5, -3) || porch.Beyond != 0.05 ||
		!near(porch.Strength.Luminance(), 0.35) ||
		!(porch.Strength.R > porch.Strength.B) {
		t.Errorf("the porch lamp is %+v", built(t).Lamps[0])
	}
}

// a flood, a sky and a sun are each the light their set says, the
// flood's colour as strong as it says, the sun's way made one long.
func TestTheLightFromAfarIsWhatItSays(t *testing.T) {
	world := built(t)
	if flood := world.Lamps[1]; flood != (film.Flood{
		Strength: maths.RGB(0.02, 0.026, 0.05)}) {
		t.Errorf("the flood is %+v", flood)
	}
	if world.Sky != (sky.Gradient{Horizon: maths.RGB(0.05, 0.07, 0.12),
		Zenith: maths.RGB(0.006, 0.01, 0.03),
		Ground: maths.RGB(0.01, 0.01, 0.012)}) {
		t.Errorf("the sky is %+v", world.Sky)
	}
	sun := world.Lamps[len(world.Lamps)-1]
	want := thing.Sun(maths.V(0, 0.8, -0.6), maths.RGB(0.35, 0.34, 0.32))
	if !reflect.DeepEqual(sun, want) {
		t.Errorf("the sun is %+v, not %+v", sun, want)
	}
}

// a body is its template placed where its set says, tilted as it says,
// its aurora shown or not; and the yard's other pieces come before and
// after it as the set gives them.
func TestABodyIsItsTemplatePlaced(t *testing.T) {
	world, alone := built(t), saturn(t, false)
	if len(world.Things) != 3+len(alone.Things) ||
		len(world.Volumes) != len(alone.Volumes) ||
		len(world.Lamps) != 3+len(alone.Lamps) {
		t.Fatalf("%d things, %d volumes, %d lamps; saturn alone has %d, "+
			"%d, %d", len(world.Things), len(world.Volumes),
			len(world.Lamps), len(alone.Things), len(alone.Volumes),
			len(alone.Lamps))
	}
	if world.Things[3].At(0) != alone.Things[0].At(0) {
		t.Errorf("saturn stands at %+v, not %+v", world.Things[3].At(0),
			alone.Things[0].At(0))
	}
	lit := built(t, "aurora: false", "aurora: true")
	glowing := saturn(t, true)
	parts := func(w *thing.World) int {
		return len(w.Things) + len(w.Volumes) + len(w.Fields) +
			len(w.Lamps)
	}
	if parts(glowing) == parts(alone) || parts(lit)-parts(world) !=
		parts(glowing)-parts(alone) {
		t.Errorf("an aurora adds %d parts here and %d alone",
			parts(lit)-parts(world), parts(glowing)-parts(alone))
	}
}

// a set with no sky leaves the world's as none.
func TestNoSkyIsNone(t *testing.T) {
	world := built(t, "    sky:\n"+
		"      horizon: {srgb_linear: [0.05, 0.07, 0.12]}\n"+
		"      zenith: {srgb_linear: [0.006, 0.01, 0.03]}\n"+
		"      ground: {srgb_linear: [0.01, 0.01, 0.012]}\n"+
		"      source: eye\n", "")
	if world.Sky != nil {
		t.Errorf("the sky is %+v", world.Sky)
	}
}

// the lens is a pinhole as wide as the film says, or the pinhole's own
// where the film does not say; the print is the film's exposure, knee
// and top, or the exposure alone.
func TestTheLensAndThePrintAreTheFilms(t *testing.T) {
	film, _ := read(t).Film()
	if lens := stage.Lens(film); !near(lens.Wide, math.Pi/2) ||
		lens.Pixel != 0 {
		t.Errorf("the lens is %+v", lens)
	}
	if print := stage.Print(film); print != (colour.Tone{Exposure: 2,
		Knee: 0.7, Top: 8}) {
		t.Errorf("the print is %+v", print)
	}
	plain, _ := read(t, "    wide: 90\n", "", "    print:\n      knee: 0.7\n"+
		"      top: 8\n      source: eye\n", "").Film()
	if lens := stage.Lens(plain); lens.Wide != 0 {
		t.Errorf("a film that says no width has a lens %+v", lens)
	}
	if print := stage.Print(plain); print != (colour.Tone{Exposure: 2}) {
		t.Errorf("a film with no print prints %+v", print)
	}
}

// what the set says and the tracer cannot hold is refused, saying which
// piece: a black body too cold to see, a box too thin to have faces, and
// a comet at its own sun, whose tail has no way to stream.
//
// a sun's way that is too short is not among them: celestial takes a way
// of any length but nought, and the set refuses nought.
func TestWhatCannotBeBuiltIsRefused(t *testing.T) {
	for _, mistake := range []struct {
		changes []string
		said    string
	}{
		{[]string{"temperature: 2700, luminance",
			"temperature: 10, luminance"},
			"material lit-window: a black body at 10 K, whose light the " +
				"kit can't carry"},
		{[]string{"temperature: 2700\n", "temperature: 10\n"},
			"lamp porch: a black body at 10 K"},
		{[]string{"size: [4, 5, 6]", "size: [5e-324, 5e-324, 6]"},
			"box lawn: "},
		{[]string{"template: saturn", "template: hale-bopp",
			"middle: [0, 600000000, 0]", "middle: [0, 0, 0]"},
			"body saturn: bodies: hale-bopp is at its sun"},
	} {
		said, _ := read(t, mistake.changes...).World()
		_, err := stage.World(said)
		if err == nil || !strings.HasPrefix(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.changes, err,
				mistake.said)
		}
	}
}
