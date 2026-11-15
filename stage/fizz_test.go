package stage_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// fizzSaid is the world the fizz's set file says, as it ships or
// changed by each pair of was and now.
func fizzSaid(t *testing.T, changes ...string) set.World {
	t.Helper()
	source, err := os.ReadFile("../sets/fizz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := string(source)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(written, changes[i]) {
			t.Fatalf("%q is not in the fizz", changes[i])
		}
		written = strings.Replace(written, changes[i], changes[i+1], 1)
	}
	read, err := set.Read("fizz.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	return said
}

// fizzOf is the one volume of the fizz's world, built.
func fizzOf(t *testing.T, changes ...string) *thing.Volume {
	t.Helper()
	world, err := stage.World(fizzSaid(t, changes...))
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != 1 {
		t.Fatalf("the fizz's world holds %d volumes", len(world.Volumes))
	}
	return world.Volumes[0]
}

// inTheColumn are places inside the fizz's column, about its foot: on
// its middle line and off it, low and high.
var inTheColumn = []maths.Vec{
	maths.V(0, 0.3, 0), maths.V(0.1, 0.8, -0.1), maths.V(-0.2, 1.2, 0.15),
	maths.V(0.05, 1.6, 0.3),
}

// densities is a medium's density at each place in the column, at a
// time.
func densities(t *testing.T, of medium.Medium, at time.Duration) []float64 {
	t.Helper()
	moving, timed := of.(medium.Timed)
	if !timed {
		t.Fatalf("%T does not move with time", of)
	}
	then := moving.Tick(at)
	found := make([]float64, len(inTheColumn))
	for i, place := range inTheColumn {
		found[i] = then.Density(place)
	}
	return found
}

// a fizz is the column its set says, in its colour, standing on its
// foot.
func TestAFizzIsTheColumnItSays(t *testing.T) {
	fizz := fizzOf(t, "foot: [0, 0, 0]", "foot: [1, 2, 3]")
	if fizz.Medium != (medium.Fizz{Radius: 0.4, Height: 2, Thick: 4,
		Rise: 0.3, Churn: 0.75, Seed: 43}) ||
		fizz.Material.Name != "column" ||
		fizz.Material.Colour != swatches.Light(swatches.Paint) ||
		fizz.Motion.At(0).Origin != maths.V(1, 3, -2) {
		t.Errorf("the fizz is %+v", fizz)
	}
}

// a fizz does not walk: its foot is where it stands at every moment,
// whatever its rise. Only its churn climbs.
func TestAFizzsFootStaysWhereItStands(t *testing.T) {
	fizz := fizzOf(t, "foot: [0, 0, 0]", "foot: [1, 2, 3]")
	for _, at := range []time.Duration{0, time.Second, time.Hour} {
		if fizz.Motion.At(at).Origin != maths.V(1, 3, -2) {
			t.Errorf("at %v the foot is at %v", at, fizz.Motion.At(at).Origin)
		}
	}
}

// a fizz climbs: two seconds on, the column has another density at
// every place tried in it.
func TestAFizzClimbs(t *testing.T) {
	fizz := fizzOf(t)
	first := densities(t, fizz.Medium, 0)
	later := densities(t, fizz.Medium, 2*time.Second)
	for i, place := range inTheColumn {
		if first[i] <= 0 {
			t.Errorf("at %v there is no fizz to climb", place)
		}
		if later[i] == first[i] {
			t.Errorf("at %v the fizz is %v at both moments", place, first[i])
		}
	}
}

// a fizz with no churn, or that does not rise, stands still: two
// seconds on it is the column it was, to the bit.
func TestAFizzWithNoChurnOrNoRiseStandsStill(t *testing.T) {
	for _, still := range [][2]string{
		{"churn: 0.75", "churn: 0"}, {"rise: 0.3", "rise: 0"},
	} {
		fizz := fizzOf(t, still[0], still[1])
		first := densities(t, fizz.Medium, 0)
		later := densities(t, fizz.Medium, 2*time.Second)
		for i, place := range inTheColumn {
			if first[i] <= 0 || later[i] != first[i] {
				t.Errorf("%s: at %v the fizz is %v, then %v", still[1],
					place, first[i], later[i])
			}
		}
	}
}

// a shift moves a fizz's foot by exactly the shift.
func TestAShiftMovesAFizzsFoot(t *testing.T) {
	shift := maths.V(10, -4, 2.5)
	world := thing.NewWorld()
	if err := stage.Add(world, fizzSaid(t), shift); err != nil {
		t.Fatal(err)
	}
	if at := world.Volumes[0].Motion.At(0).Origin; at != shift {
		t.Errorf("the shifted foot is at %v, not %v", at, shift)
	}
}

// a shift moves nothing else of a fizz: its column, its rise, its churn
// and its colour are the ones it has unshifted.
func TestAShiftLeavesAFizzsColumnAlone(t *testing.T) {
	world := thing.NewWorld()
	if err := stage.Add(world, fizzSaid(t), maths.V(10, -4, 2.5)); err != nil {
		t.Fatal(err)
	}
	moved, stood := world.Volumes[0], fizzOf(t)
	if moved.Medium != stood.Medium ||
		moved.Material.Colour != stood.Material.Colour ||
		moved.Material.Name != stood.Material.Name {
		t.Errorf("shifted, the fizz is %+v, not %+v", moved, stood)
	}
}
