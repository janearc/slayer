package session

import (
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/trace"
)

// dropOfTheSet is the source of a pair of stars with a lava drop of a
// given age, which may make its temperature made up.
func dropOfTheSet(age string) string {
	return `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: a drop
  film: {width: 64, height: 36, rays: 1, exposure: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -5, 2]}
  world:
    unit: 1
    binaries:
      pair:
        generate:
          primary: {mass: 2.45, age: 1.2e7}
          companion: {mass: 0.62, temperature: 25000, radius: 0.0122}
          separation: 9.2
          eccentricity: 0.16
          drop: {mass: 0.1, temperature: 3000, age: ` + age +
		`, distance: 48, made_up: true}
        drop_at: [0, 0, 0]
        time: 4
`
}

// starsOf is what the stars of a set come to: the lines they say, and the
// things and lamps they put in the world.
func starsOf(t *testing.T, source string) ([]string, *thing.World) {
	t.Helper()
	read, err := set.Read("drop.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	making := &staging{given: asked{set: "drop.yaml", width: 64,
		samples: 1, wide: 20}, world: thing.NewWorld(),
		places: map[string]maths.Vec{}}
	lines, err := making.starred(read, making.given, maths.V(0, 0, -1e12),
		1)
	if err != nil {
		t.Fatal(err)
	}
	return lines, making.world
}

// A lava drop is drawn, a third ball and a third lamp, and where its
// temperature is made up the draw says so in a line that begins "made
// up:" and names the drop. The anti-test: a drop that is not made up says
// no such line, so the line is not said of everything.
func TestAMadeUpDropIsDrawnAndSaidSo(t *testing.T) {
	lines, world := starsOf(t, dropOfTheSet("1.0e9"))
	if len(world.Things) != 3 || len(world.Lamps) != 3 {
		t.Errorf("the pair and its drop are %d things and %d lamps, want 3 "+
			"and 3", len(world.Things), len(world.Lamps))
	}
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "made up:") ||
		!strings.Contains(lines[0], "the lava drop") {
		t.Errorf("the draw says %q", lines)
	}
	if lines, _ := starsOf(t, dropOfTheSet("1.0e6")); len(lines) != 0 {
		t.Errorf("a young drop says %q", lines)
	}
}

// A set staged for a job's draw carries its "made up:" lines in what was
// asked, and the draw says them before it draws: toldOf writes each to the
// stream. The anti-test: a set with nothing made up, and one asked nothing,
// say nothing, so the line is never said of what is not made up.
func TestAJobsDrawSaysWhatIsMadeUpBeforeItDraws(t *testing.T) {
	ofAge := func(age string) drawing {
		read, err := set.Read("drop.yaml", []byte(dropOfTheSet(age)))
		if err != nil {
			t.Fatal(err)
		}
		return stagedWorld(read, asked{set: "drop.yaml", width: 64,
			samples: 1, wide: 20})
	}
	old := ofAge("1.0e9")
	if old.err != nil {
		t.Fatal(old.err)
	}
	var said strings.Builder
	toldOf(&said, old.given)
	if !strings.HasPrefix(said.String(), "made up:") ||
		!strings.Contains(said.String(), "3000 K") {
		t.Errorf("the draw says %q", said.String())
	}
	young := ofAge("1.0e6")
	var silent strings.Builder
	toldOf(&silent, young.given)
	toldOf(&silent, asked{})
	if silent.Len() != 0 {
		t.Errorf("a set with nothing made up says %q", silent.String())
	}
}

// A molten star is drawn with its crust and not a flat glow: its material
// shines by the lava texture, glows by nothing else, and is darkened at its
// limb as a star is. The anti-test: a star that is not molten is a flat
// glow with no texture.
func TestAMoltenStarIsDrawnWithACrust(t *testing.T) {
	molten := shining(5, 3000, 1e3)
	molten.Molten = true
	ball, _, err := starOf(molten, maths.Vec{}, 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, crusted := ball.Material.Shine.(texture.Lava); !crusted ||
		ball.Material.Glow != (maths.Light{}) || !(ball.Material.Limb > 0) {
		t.Errorf("a molten ball's material is %+v", ball.Material)
	}
	flat, _, _ := starOf(shining(5, 3000, 1e3), maths.Vec{}, 1, 1, 0)
	if flat.Material.Shine != nil || flat.Material.Glow == (maths.Light{}) {
		t.Errorf("a star that is not molten is %+v", flat.Material)
	}
}

// The anti-test: a drop lights what faces it and not what is behind a
// planet from it. A white ball is lit by a drop on the side toward it,
// and not on its far side, which the ball itself shades.
func TestTheDropDoesNotLightWhatIsBehindThePlanet(t *testing.T) {
	lit := func(from float64) float64 {
		world := thing.NewWorld()
		drop := generate.Shine{Name: "drop", Mass: measure.SolarMasses(0.1),
			Temperature: measure.Kelvin(3000),
			Luminosity:  measure.Watts(1e6), Radius: measure.Metres(1),
			Molten: true}
		ball, lamp, err := starOf(drop, maths.V(0, 0, 50), 1, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		world.Add(ball)
		world.Lamp(lamp)
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 3},
			Material: &thing.Material{Name: "planet", Colour: maths.Grey(1)},
			Motion:   thing.Still(maths.Subspace{})})
		at := maths.V(0, 0, from)
		eye := maths.Subspace{Origin: at,
			Orientation: maths.LookAtUp(at, maths.Vec{}, maths.Up)}
		const side = 5
		picture, err := trace.Render(world, eye,
			film.Pinhole{Wide: 10 * maths.Degree}, 0, side, side)
		if err != nil {
			t.Fatal(err)
		}
		return picture.At(side/2, side/2).Light().R
	}
	if near := lit(10); !(near > 0) {
		t.Fatalf("the planet's near side is lit by %v", near)
	}
	if far := lit(-10); far != 0 {
		t.Errorf("the planet's far side is lit by %v", far)
	}
}
