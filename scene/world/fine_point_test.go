package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The test's step and its allowance: how far apart, in coarse spacings,
// two near points are, how much their paints may differ, and how many
// places are asked.
const (
	nearStep    = 0.00001
	nearAllowed = 0.01
	pointsAsked = 300
)

// flatLook is a look whose every class is the same grey, so only the
// light's grain can change a paint, never a class.
func flatLook() Look {
	var look Look
	for biome := range Biomes {
		look.Paints[biome] = maths.Grey(0.5)
	}
	return look
}

// With no lines, a fine paint is worked at the point: two points a hair
// apart have paints a hair apart, no tile steps between them, and over
// the world the paint does vary.
func TestAFinePointHasNoTileSteps(t *testing.T) {
	world := New(terran)
	fine := world.Scrooged(flatLook(), 0.3, 0.3)
	step := nearStep * world.Tiling.Spacing()
	least, most := math.Inf(1), math.Inf(-1)
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		near := point.Add(maths.V(step, 0, 0))
		a, b := fine.At(point), fine.At(near)
		if math.Abs(a.R-b.R) > nearAllowed {
			t.Fatalf("a step of %v changes the paint from %v to %v", step, a, b)
		}
		least, most = math.Min(least, a.R), math.Max(most, a.R)
	}
	if most-least < nearAllowed {
		t.Errorf("the paint never varies: %v to %v", least, most)
	}
}

// Anti-test: with lines, the paint is the tile's, flat within it, so a
// point and one a hair off are the same paint.
func TestLinedFinePaintStaysTiled(t *testing.T) {
	world := New(terran)
	look := flatLook()
	look.Lines = 0.5
	fine := world.Fine(look, 3, 0.3, 0)
	point := maths.V(0.3, 0.5, 0.8).Unit()
	near := point.Add(maths.V(nearStep*world.Tiling.Spacing()*0.01, 0, 0))
	if fine.At(point) != fine.At(near) {
		t.Errorf("a lined tile's paint changes inside it: %v, %v",
			fine.At(point), fine.At(near))
	}
}

// With no mix the ground is the coarse tiles' own: every point is the
// paint of the class of the tile it is in.
func TestNoMixIsTheCoarseTiles(t *testing.T) {
	world := New(terran)
	look := rampLook()
	fine := world.Scrooged(look, 0, 0)
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		want := look.Paints[world.Biome(world.Tiling.Nearest(point))]
		if got := fine.At(point); got != want {
			t.Fatalf("at %v the paint is %v, the tile's is %v", point, got,
				want)
		}
	}
}

// scatterOnSphere is the place-th of a spread of points over the sphere,
// by the golden angle, so a test samples the world and not one patch.
func scatterOnSphere(place int) maths.Vec {
	const goldenTurn = 2.39996322972865332
	height := 1 - 2*(float64(place)+0.5)/pointsAsked
	round := math.Sqrt(1 - height*height)
	turn := goldenTurn * float64(place)
	return maths.V(round*math.Cos(turn), height, round*math.Sin(turn))
}
