package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// detailSkins are the skins the demo level cuts into parts, with the parts
// each draws at that level that the quick level does not.
var detailSkins = map[Skin][]string{
	skinCurtain: {"backpan", "backpan fold", "slab edge", "sill", "transom",
		"vent", "mullion cap"},
	skinMirror: {"backpan", "backpan fold", "slab edge", "sill", "transom",
		"vent", "mullion cap"},
	skinGranite: {"apron", "sill", "lintel", "frame", "spandrel"},
	skinConcrete: {"cold joint", "drip", "tie hole", "slab edge", "frame",
		"pier"},
	skinGothic: {"offset", "hood", "lancet head", "spandrel panel",
		"tracery lozenge", "pinnacle shaft", "finial"},
	skinCantilever: {"soffit", "post", "handrail", "balustrade glass",
		"anchor", "cable"},
}

// sideAllowance is how far past the quick tower's walls a demo part may
// stand, by skin, and topAllowance how far over its top: the deepest the
// skin's parts stand out, a cantilevered plate the furthest.
var sideAllowance = map[Skin]float64{skinCurtain: 6, skinMirror: 6,
	skinGranite: 6, skinConcrete: 6, skinGothic: 6, skinCantilever: 8}

// topAllowance is how far over the quick tower's top the demo level may
// reach: a gothic pinnacle stands on a shaft, and nothing else rises.
const topAllowance = 6.0

// boundsOf is the box round every part drawn.
func boundsOf(drawn []Rendered) (low, high maths.Vec) {
	inf := math.Inf(1)
	low, high = maths.V(inf, inf, inf), maths.V(-inf, -inf, -inf)
	for _, one := range drawn {
		for _, face := range one.Mesh.Triangles {
			for _, corner := range face {
				at := one.Place.Out(corner)
				low = maths.V(math.Min(low.X, at.X), math.Min(low.Y, at.Y),
					math.Min(low.Z, at.Z))
				high = maths.V(math.Max(high.X, at.X), math.Max(high.Y, at.Y),
					math.Max(high.Z, at.Z))
			}
		}
	}
	return low, high
}

func TestTheDemoLevelHasMorePartsThanTheQuickLevelAndTheNamedOnes(
	t *testing.T) {
	for skin, wanted := range detailSkins {
		pins := plainPins(skin)
		pins.Style = stylePunched
		quick := drawnTower(t, towerOf(t, 3, 0, pins), "quick", 0)
		demo := drawnTower(t, towerOf(t, 3, 0, pins), "demo", 0)
		if len(demo) <= len(quick) {
			t.Errorf("%s: demo has %d parts, quick %d", skin, len(demo),
				len(quick))
		}
		found := counts(demo)
		for _, name := range wanted {
			if found[name] == 0 {
				t.Errorf("%s: no %q at the demo level among %v", skin, name,
					found)
			}
		}
		if twice := repeats(demo, wanted); len(twice) > 0 {
			t.Errorf("%s: %d parts stand twice: %v", skin, len(twice),
				twice[:1])
		}
	}
}

func TestEveryPartOfADemoTowerIsInsideTheTowersEnvelope(t *testing.T) {
	for skin := range detailSkins {
		pins := plainPins(skin)
		pins.Style = stylePunched
		quickTower := towerOf(t, 3, 0, pins)
		quickLow, quickHigh := boundsOf(drawnTower(t,
			quickTower, "quick", 0))
		demoTower := towerOf(t, 3, 0, pins)
		demoLow, demoHigh := boundsOf(drawnTower(t,
			demoTower, "demo", 0))
		side := sideAllowance[skin]
		if demoLow.X < quickLow.X-side || demoHigh.X > quickHigh.X+side ||
			demoLow.Z < quickLow.Z-side || demoHigh.Z > quickHigh.Z+side {
			t.Errorf("%s: demo %v to %v is out of quick %v to %v by more "+
				"than %v", skin, demoLow, demoHigh, quickLow, quickHigh, side)
		}
		if demoLow.Y < quickLow.Y-epsilonHeight ||
			demoHigh.Y > quickHigh.Y+topAllowance {
			t.Errorf("%s: demo stands from %v to %v, quick from %v to %v",
				skin, demoLow.Y, demoHigh.Y, quickLow.Y, quickHigh.Y)
		}
	}
}

// epsilonHeight is how far under the quick tower's foot a part may stand,
// a lining under the lowest plate: a tenth of a metre.
const epsilonHeight = 0.1

func TestTheSameSeedMakesTheSameDemoTowerAndAnotherSeedAnother(
	t *testing.T) {
	for skin := range detailSkins {
		pins := plainPins(skin)
		pins.Style = stylePunched
		firstTower := towerOf(t, 3, 0, pins)
		first := heldKey(drawnTower(t, firstTower, "demo", 0))
		againTower := towerOf(t, 3, 0, pins)
		again := heldKey(drawnTower(t, againTower, "demo", 0))
		otherTower := towerOf(t, 4, 0, pins)
		other := heldKey(drawnTower(t, otherTower, "demo", 0))
		if first != again {
			t.Errorf("%s: one seed made %q and %q", skin, first, again)
		}
		if first == other {
			t.Errorf("%s: seeds 3 and 4 made the same tower %q", skin, first)
		}
	}
}

func TestAPointedArchIsConvexAndRisesToItsApexInTheFace(t *testing.T) {
	const width, pieces = 1.3, 5
	corners := archCorners(width, pieces)
	if len(corners) != 2*pieces+1 {
		t.Fatalf("%d corners, wanted %d", len(corners), 2*pieces+1)
	}
	apex := width * math.Sin(archAngle*radiansPerDegree)
	least := 0.0
	for _, corner := range corners {
		least = math.Min(least, corner.Z)
	}
	if math.Abs(least+apex) > 1e-9 {
		t.Errorf("the arch rises to %v, wanted %v", -least, apex)
	}
	if !convex(corners) {
		t.Errorf("the arch is not convex")
	}
	if _, err := (prism{Bottom: polygon{corners}, Top: polygon{corners},
		Height: 0.2}).Mesh("demo"); err != nil {
		t.Errorf("the arch is not a prism: %v", err)
	}
}
