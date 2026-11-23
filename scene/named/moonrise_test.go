package named_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/named"
)

// the moon as nasa's moon fact sheet gives it, in metres: its mean
// radius; and the width it shows at its mean distance, 384,400 km, in
// degrees, to the third place
const (
	moonRadius  = 1737.4e3
	moonAcross  = 0.5179
	acrossPlace = 1e-3
)

// the moon's and the sun's altitudes the tests ask for, in degrees,
// picked by the author of this package, 2026-11-12
const (
	moonUp = 0.3
	sunUp  = 2.0
)

// the bounds cmd/moonrise held the moon and the sun to, in degrees: the
// moon's lowest, the sun's lowest, and both's highest; and how far past
// an edge a near miss is, picked by the author of this package,
// 2026-11-12
const (
	moonLowest, sunLowest, highest = -5, 0, 90
	nearMiss                       = 0.01
)

// pair is two of a thing, defined: the earth and the moon
const pair = 2

// globes are a built scene's things that are globes, and where each is
// at the scene's zero.
func globes(built *named.Built) []maths.Vec {
	var found []maths.Vec
	for _, each := range built.World().Things {
		if _, round := each.Shape.(globe.Globe); round {
			found = append(found, each.Motion.At(none).Origin)
		}
	}
	return found
}

// moonOf is where the moon's middle is: the globe furthest from the eye.
func moonOf(t *testing.T, built *named.Built) maths.Vec {
	t.Helper()
	found := globes(built)
	if len(found) == none {
		t.Fatal("there is no globe")
	}
	eye := built.Eye().Origin
	return slices.MaxFunc(found, func(one, other maths.Vec) int {
		gap := one.Sub(eye).Len() -
			other.Sub(eye).Len()
		return int(math.Copysign(once, gap))
	})
}

// the moon stands at the altitude asked, straight ahead, and is 0.518
// degrees across: its radius and distance as nasa's fact sheet gives
// them. There are two globes, the earth and the moon.
func TestTheMoonIsWhereAndAsLargeAsAsked(t *testing.T) {
	built := mustBuild(t, "moonrise", map[string]any{"moon": moonUp})
	if count := len(globes(built)); count != pair {
		t.Errorf("there are %d globes; the earth and the moon are two",
			count)
	}
	toward := moonOf(t, built).Sub(built.Eye().Origin)
	altitude := math.Asin(toward.Unit().Y) / maths.Degree
	across := 2 * math.Atan(moonRadius/toward.Len()) / maths.Degree
	if math.Abs(altitude-moonUp) > closeEnough ||
		math.Abs(across-moonAcross) > acrossPlace {
		t.Errorf("the moon is %.6f deg up and %.4f across", altitude,
			across)
	}
}

// the sun is distant and behind the eye, at the altitude asked: the way
// to it points back past the eye, away from the moon.
func TestTheSunIsDistantAndBehind(t *testing.T) {
	built := mustBuild(t, "moonrise", map[string]any{"sun": sunUp})
	sun, distant := built.World().Lamps[none].(emit.Distant)
	if !distant {
		t.Fatalf("the sun is a %T", built.World().Lamps[none])
	}
	toward, far, _ := sun.Light(maths.Vec{}, maths.Up)
	ahead := moonOf(t, built).Sub(built.Eye().Origin)
	if !math.IsInf(far, once) || toward.Dot(ahead) >= none ||
		math.Abs(math.Asin(toward.Y)/maths.Degree-sunUp) > closeEnough {
		t.Errorf("the sun is toward %v, %g away", toward, far)
	}
}

// anti-tests: a tree of no kind there is, a near miss of one, is refused
// as not among the choices, which are listed.
func TestATreeOfNoKindIsRefused(t *testing.T) {
	err := mustGet(t, "moonrise").Check(map[string]any{"kind": "Norway"})
	if !errors.Is(err, named.ErrChoice) {
		t.Errorf("a tree of kind Norway: %v; want ErrChoice", err)
	}
}

// mustParameter is the scene's parameter of that name, or the test stops.
func mustParameter(t *testing.T, scene *named.Scene,
	name string) named.Parameter {
	t.Helper()
	for _, one := range scene.Parameters() {
		if one.Name() == name {
			return one
		}
	}
	t.Fatalf("%s has no %s", scene.Name(), name)
	return named.Parameter{}
}

// edges are values at the very bounds of the moonrise's parameters,
// which are held, and just past them, which are refused.
var edges = []struct {
	name         string
	held, beyond float64
}{
	{"moon", moonLowest, moonLowest - nearMiss},
	{"moon", highest, highest + nearMiss},
	{"sun", sunLowest, sunLowest - nearMiss},
	{"sun", highest, highest + nearMiss},
	{"away", math.SmallestNonzeroFloat64, none},
}

// A value at the edge of its bounds is held.
func TestAValueAtTheEdgeIsHeld(t *testing.T) {
	scene := mustGet(t, "moonrise")
	for _, edge := range edges {
		if err := scene.Check(map[string]any{edge.name: edge.held}); err !=
			nil {
			t.Errorf("%s at %v: %v", edge.name, edge.held, err)
		}
	}
}

// anti-tests: a value just past the edge of its bounds is refused, and
// the refusal says the bounds; a seed below nought is too.
func TestAValuePastTheEdgeIsRefused(t *testing.T) {
	scene := mustGet(t, "moonrise")
	for _, edge := range edges {
		err := scene.Check(map[string]any{edge.name: edge.beyond})
		bounds := mustParameter(t, scene, edge.name).Bounds().String()
		if !errors.Is(err, named.ErrBounds) ||
			!containsAll(err.Error(), edge.name, bounds) {
			t.Errorf("%s at %v: %v; want ErrBounds, saying %q",
				edge.name, edge.beyond, err, bounds)
		}
	}
	err := scene.Check(map[string]any{"seed": -once})
	if !errors.Is(err, named.ErrBounds) {
		t.Errorf("a seed of -1: %v; want ErrBounds", err)
	}
}

// The bounds say themselves in words, and give their edges.
func TestBoundsSayThemselves(t *testing.T) {
	scene := mustGet(t, "moonrise")
	for name, want := range map[string]string{"moon": "from -5 to 90",
		"away": "above 0", "seed": "at least 0", "kind": "any number"} {
		if got := mustParameter(t, scene, name).Bounds().String(); got !=
			want {
			t.Errorf("%s's bounds are %q, want %q", name, got, want)
		}
	}
	least, open, set := mustParameter(t, scene, "away").Bounds().Least()
	most, high := mustParameter(t, scene, "sun").Bounds().Most()
	if least != none || !open || !set || most != highest || !high {
		t.Errorf("away's least is %v open %v set %v; sun's most %v "+
			"set %v", least, open, set, most, high)
	}
}

// anti-tests: a parameter with no bounds has no edges, and holds any
// number.
func TestNoBoundsHoldEverything(t *testing.T) {
	bounds := mustParameter(t, mustGet(t, "swarm"), "wear").Bounds()
	_, _, low := bounds.Least()
	_, high := bounds.Most()
	if low || high || !bounds.Holds(-math.MaxFloat64) ||
		!bounds.Holds(math.MaxFloat64) {
		t.Errorf("an unbounded parameter has edges: %+v", bounds)
	}
}
