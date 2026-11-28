package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// rampLook is a look whose every class is its own grey, so a change of
// class is a change of paint.
func rampLook() Look {
	var look Look
	for biome := range Biomes {
		look.Paints[biome] = maths.Grey(float64(biome+1) / float64(Biomes+1))
	}
	return look
}

// The test's numbers: the footprints, in radians, of a distant eye, a
// near one and one between; how many points walk a path; and how much a
// patch's mean paint may change as the eye comes down.
const (
	distantFootprint = 0.05
	nearFootprint    = 1e-6
	middleFootprint  = 1e-3
	pathPoints       = 40000
	ringPoints       = 2000
	arcHalf          = 0.1 // radians either side of a coast a walk covers
	meanAllowed      = 0.06
	patchSide        = 40
	landOver         = 0.005 // how far over the sea's grey a paint is land
	edgeTiles        = 400   // how many tiles of a level an edge test asks
)

// grounded is the test world's ground in the ramp look, mixed as a
// generated planet's is.
func grounded(world *World) *scrooged {
	return world.Scrooged(rampLook(), 0, 0.35).(*scrooged)
}

// A footprint in the length a point is in is an angle: over the point's
// distance from the middle; none for no footprint or no distance.
func TestAFootprintIsAnAngle(t *testing.T) {
	far := maths.V(0, 0, 6.4e6)
	if got := angular(far, 6.4); math.Abs(got-1e-6) > 1e-12 {
		t.Errorf("6.4 m at 6.4e6 m is %v, not 1e-6", got)
	}
	if angular(far, 0) != 0 || angular(maths.V(0, 0, 0), 5) != 0 {
		t.Error("no footprint or no distance still has an angle")
	}
	if got := angular(maths.V(0, 1, 0), 0.25); got != 0.25 {
		t.Errorf("a footprint on the unit sphere is %v, not 0.25", got)
	}
}

// A level is seen by a pixel when its tiles are further apart than the
// pixel is wide: none of it at one pixel, all of it at two, and more of
// it the narrower the pixel, never less.
func TestALevelFadesInAsThePixelNarrows(t *testing.T) {
	if weight(1, 1) != 0 || weight(2, 1) != 1 || weight(0.5, 1) != 0 {
		t.Errorf("weights at one, two and a half: %v %v %v", weight(1, 1),
			weight(2, 1), weight(0.5, 1))
	}
	last := -1.0
	for width := 1.0; width >= 0.5; width -= 0.01 {
		got := weight(1, width)
		if got < last || got < 0 || got > 1 {
			t.Fatalf("at a width of %v the weight is %v after %v", width,
				got, last)
		}
		last = got
	}
}

// A coast breaks into finer coast as the eye comes down: walking a path
// over the world, a near footprint finds more places where land turns to
// sea than a distant one.
func TestACoastBreaksFinerAsTheEyeComesDown(t *testing.T) {
	world := New(terran)
	look := rampLook()
	fine := grounded(world)
	around := coastOn(fine, look)
	far := coasts(fine, look, distantFootprint, around)
	near := coasts(fine, look, nearFootprint, around)
	if near <= far || near < 3 {
		t.Errorf("coast crossings: %d from far, %d from near", far, near)
	}
}

// coastOn is the turn, round a ring of the world, at which the finest
// paint first goes from sea to land.
func coastOn(fine *scrooged, look Look) float64 {
	sea := look.Paints[Sea].R + landOver
	lastLand := false
	for step := range ringPoints {
		turn := 2 * math.Pi * float64(step) / ringPoints
		now := fine.Within(onRing(turn), nearFootprint).R > sea
		if step > 0 && now && !lastLand {
			return turn
		}
		lastLand = now
	}
	return 0
}

// onRing is the point a turn of the way round a ring of the world.
func onRing(turn float64) maths.Vec {
	return maths.V(math.Cos(turn), 0.3, math.Sin(turn)).Unit()
}

// coasts is how many times a walk along a short arc of the ring, about
// a turn, turns from sea to land or back, painting at a footprint.
func coasts(fine *scrooged, look Look, footprint, about float64) int {
	sea := look.Paints[Sea].R + landOver
	turns, lastLand := 0, false
	for step := range pathPoints {
		turn := about + arcHalf*(2*float64(step)/pathPoints-1)
		now := fine.Within(onRing(turn), footprint).R > sea
		if step > 0 && now != lastLand {
			turns++
		}
		lastLand = now
	}
	return turns
}

// Anti-test: no tile's edge shows in a class boundary, at the world's
// own level or the tree's. Across the edge between two tiles the paint
// changes no more than it does along the edge the same distance, on the
// whole, as it would if a tile decided its class.
func TestNoTileEdgeIsAClassBoundary(t *testing.T) {
	world := New(terran)
	fine := grounded(world)
	for _, level := range []int{0, 2, 4} {
		tiling := fine.tree.levels[level]
		across, along := edgeContrast(fine, tiling)
		if across > 1.5*along+1e-9 {
			t.Errorf("level %d: paint changes %v across tile edges, %v "+
				"along them", level, across, along)
		}
	}
}

// edgeContrast is how much the paint changes across the edges of a
// tiling's tiles, a hair each way, and how much along them, summed over
// the edges of tiles spread over the sphere, painted for a pixel a
// sixteenth of the tiling's spacing wide.
func edgeContrast(fine *scrooged, tiling hexes.Tiling) (across,
	along float64) {
	spacing := tiling.Spacing()
	hair, pixel := 0.02*spacing, spacing/16
	paint := func(p maths.Vec) float64 { return fine.Within(p, pixel).R }
	for place := range edgeTiles {
		tile := tiling.Nearest(scatterOn(place, edgeTiles))
		a := tiling.Centre(tile)
		for _, other := range tiling.Neighbours(tile) {
			b := tiling.Centre(other)
			edge, way := a.Add(b).Unit(), b.Sub(a).Unit()
			side := edge.Cross(way).Unit()
			across += math.Abs(paint(edge.Add(way.Scale(hair))) -
				paint(edge.Sub(way.Scale(hair))))
			along += math.Abs(paint(edge.Add(side.Scale(hair))) -
				paint(edge.Sub(side.Scale(hair))))
		}
	}
	return across, along
}

// scatterOn is the place-th of count points spread over the sphere by
// the golden angle.
func scatterOn(place, count int) maths.Vec {
	const goldenTurn = 2.39996322972865332
	height := 1 - 2*(float64(place)+0.5)/float64(count)
	round := math.Sqrt(1 - height*height)
	turn := goldenTurn * float64(place)
	return maths.V(round*math.Cos(turn), height, round*math.Sin(turn))
}

// Children keep their parent's light: the mean paint of a patch, a few
// tiles wide, is the same however fine the eye, within a tolerance.
func TestAPatchKeepsItsLightAsTheEyeComesDown(t *testing.T) {
	world := New(terran)
	fine := grounded(world)
	mean := func(footprint float64) float64 {
		sum := 0.0
		for across := range patchSide {
			for down := range patchSide {
				turn := 0.4 * float64(across) / patchSide
				lift := 0.4 * float64(down) / patchSide
				point := maths.V(math.Cos(turn), lift, math.Sin(turn)).Unit()
				sum += fine.Within(point, footprint).R
			}
		}
		return sum / (patchSide * patchSide)
	}
	far, near := mean(distantFootprint/10), mean(middleFootprint)
	if math.Abs(far-near) > meanAllowed {
		t.Errorf("a patch's mean paint is %v from far, %v from near", far,
			near)
	}
}

// A fixed tile's class is held round its middle, not by its hexagon: the
// middle of a volcano's tile is lava, and a point most of a tile away is
// not held.
func TestAFixedTileIsHeldRound(t *testing.T) {
	plain := New(terran)
	heart, ok := plain.Find(maths.V(0.6, 0.2, 0.8), Land)
	if !ok {
		t.Fatal("no land to put a volcano on")
	}
	at := plain.Tiling.Centre(heart)
	with := terran
	with.Features = []Feature{Volcano{At: at,
		Reach: 4 * plain.Tiling.Spacing(), Rise: 0.5}}
	world := New(with)
	look := rampLook()
	fine := grounded(world)
	if got := fine.Within(at, nearFootprint); got != look.Paints[Lava] {
		t.Errorf("the heart is %v, not lava %v", got, look.Paints[Lava])
	}
	if !fine.holds(at, world.Smoothed(at)) {
		t.Error("the heart is not held")
	}
	far := at.Add(maths.V(0.6*world.Tiling.Spacing(), 0, 0)).Unit()
	if fine.holds(far, world.Smoothed(far)) &&
		world.Smoothed(far).Nearest == world.index(heart) {
		t.Error("a point most of a tile off is held")
	}
}

// heightStep is the most a 1 per cent narrower pixel may change the
// height at a point: the newest level fades in by about two per cent of
// its detail, which is a few tenths at most.
const heightStep = 0.01

// Anti-test: no step where a level opens. As the pixel narrows by one
// per cent at a time across the width at which a level opens, the height
// at a point changes a little each time, never by a jump.
func TestNoStepWhereALevelOpens(t *testing.T) {
	world := New(terran)
	tree := world.tree()
	from, to := tree.spacing(3), tree.spacing(4)/2
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		last := tree.gather(point, from).detail[heightAttribute]
		for angle := from * 0.99; angle > to; angle *= 0.99 {
			now := tree.gather(point, angle).detail[heightAttribute]
			if math.Abs(now-last) > heightStep {
				t.Fatalf("at %v the height steps from %v to %v at %v",
					point, last, now, angle)
			}
			last = now
		}
	}
}

// BenchmarkWithin is what a painted place costs at a pixel a hundred
// metres across on a world of Earth's size, the places a pixel apart
// along a line, as a picture asks them.
func BenchmarkWithin(b *testing.B) {
	world := New(terran)
	fine := world.Scrooged(rampLook(), 0.25, 0.35)
	scale := 6.4e6
	start := maths.V(0.6, 0.2, 0.8).Unit()
	step := maths.V(0, 1, 0).Scale(100 / scale)
	for index := 0; index < b.N; index++ {
		point := start.Add(step.Scale(float64(index % 1_000_000))).Unit()
		fine.(*scrooged).Within(point.Scale(scale), 100)
	}
}
