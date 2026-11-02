package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the plain share is the distance over the reach, and a reach of zero or under
// is two, past the edge, whether the point is at the heart or not: 3 from the
// heart over a reach of 2 is 1.5.
func TestTheRoundShareIsDistanceOverReach(t *testing.T) {
	at := maths.V(1, 1, 1)
	for _, c := range []struct {
		p     maths.Vec
		reach float64
		want  float64
	}{
		{maths.V(4, 1, 1), 2, 1.5},
		{maths.V(1, 1, 1), 2, 0},
		{maths.V(2, 1, 1), 0, 2},
		{at, 0, 2},
		{maths.V(2, 1, 1), -1, 2},
	} {
		if got := round(c.p, at, c.reach); got != c.want {
			t.Errorf("round(%v, %v) is %v, want %v", c.p, c.reach, got, c.want)
		}
	}
}

// a volcano's height, by share, at a rise of 2 on ground of 0.25. to a
// share of 0.2, 0.65 plus the share, of the rise: 0.65 * 2 at the heart,
// 0.75 * 2 at 0.1, 0.85 * 2 at 0.2, the crater's wall.
//
// past it the cone, the rise times (1 - (share - 0.2) / 0.8) to the 1.6th.
// at a share of 0.975 that is (1/32) to the 1.6th = 2^-8, at a hair over
// 0.2 the full rise, and at the edge, 1, the ground.
func TestAVolcanoRisesAsAConeWithACrater(t *testing.T) {
	v := Volcano{Rise: 2}
	for _, c := range []struct{ share, want float64 }{
		{0, 0.25 + 1.3},
		{0.1, 0.25 + 1.5},
		{0.2, 0.25 + 1.7},
		{0.2000001, 0.25 + 2},
		{0.975, 0.25 + 2.0/256},
		{1, 0.25},
	} {
		if got := v.Shape(0.25, c.share); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("a volcano at a share of %v is %v, want %v", c.share, got, c.want)
		}
	}
}

// the crater and the slopes: lava to a share of 0.2 and ash to 0.55, the
// rules beyond. a dead one is rock instead of both, and the rules beyond
// as well.
func TestAVolcanoFixesLavaAndAsh(t *testing.T) {
	for _, c := range []struct {
		share      float64
		live, dead Biome
	}{
		{0, Lava, Rock},
		{0.2, Lava, Rock},
		{0.2000001, Ash, Rock},
		{0.55, Ash, Rock},
		{0.5500001, Unfixed, Unfixed},
		{0.6, Unfixed, Unfixed},
		{1, Unfixed, Unfixed},
	} {
		if got := (Volcano{}).Class(c.share); got != c.live {
			t.Errorf("a share of %v is %d, want %d", c.share, got, c.live)
		}
		if got := (Volcano{Dead: true}).Class(c.share); got != c.dead {
			t.Errorf("a dead one at %v is %d, want %d", c.share, got, c.dead)
		}
	}
}

// a canyon from (0,0,0) to (4,0,0), 2 wide: its middle is (2,0,0), its reach
// half of 4, 2, and the width, 4. the share of a point is its distance from the
// line over the width, 3 off the line is 1.5, and a width of zero or under is
// two.
func TestACanyonIsAPathAndAWidth(t *testing.T) {
	c := Canyon{From: maths.V(0, 0, 0), To: maths.V(4, 0, 0), Width: 2}
	if at, reach := c.Footprint(); at != maths.V(2, 0, 0) || reach != 4 {
		t.Errorf("the footprint is %v and %v, want (2,0,0) and 4", at, reach)
	}
	if got := c.Share(maths.V(1, 3, 0), maths.Vec{}, 0); got != 1.5 {
		t.Errorf("3 off the line, with a width of 2, is %v, want 1.5", got)
	}
	for _, width := range []float64{0, -1} {
		c.Width = width
		if got := c.Share(maths.V(2, 0, 0), maths.Vec{}, 0); got != 2 {
			t.Errorf("on the line with a width of %v is %v, want 2", width, got)
		}
	}
}

// a canyon's floor is Depth below the ground on the line and the ground at
// the walls' edge, with the smooth step between: at a share of 1/4 the
// step is t * t * (3 - 2 t) for t = 1/4, 1/16 * 5/2 = 5/32, so the floor is
// 1 - 5/32 = 27/32 of the depth down, 3.375 for a depth of 4.
func TestACanyonFallsAwayAlongASmoothStep(t *testing.T) {
	c := Canyon{Depth: 4}
	for _, k := range []struct{ share, want float64 }{
		{0, 10 - 4},
		{0.25, 10 - 3.375},
		{0.5, 10 - 2},
		{1, 10},
	} {
		if got := c.Shape(10, k.share); !close12(got, k.want) {
			t.Errorf("at a share of %v the ground is %v, want %v", k.share, got, k.want)
		}
	}
}

// rock down the walls and the floor, to a share of 0.85 and not at it.
func TestACanyonIsRockToItsWalls(t *testing.T) {
	for _, c := range []struct {
		share float64
		want  Biome
	}{{0, Rock}, {0.5, Rock}, {0.8499, Rock}, {0.85, Unfixed}, {0.9, Unfixed}} {
		if got := (Canyon{}).Class(c.share); got != c.want {
			t.Errorf("a canyon at a share of %v is %d, want %d", c.share, got, c.want)
		}
	}
}

// the segment from (0,0,0) to (2,0,0): a point beside its middle is 1
// off, past either end it is the distance to that end, sqrt 2 from
// (-1,1,0) and (3,1,0). a segment of length 1 is not mistaken for a
// point, and a segment of no length is the point: 5 from a (3,4,0).
func TestADistanceToASegmentStopsAtItsEnds(t *testing.T) {
	a, b := maths.V(0, 0, 0), maths.V(2, 0, 0)
	for _, c := range []struct {
		p, a, b maths.Vec
		want    float64
	}{
		{maths.V(1, 1, 0), a, b, 1},
		{maths.V(-1, 1, 0), a, b, math.Sqrt2},
		{maths.V(3, 1, 0), a, b, math.Sqrt2},
		{maths.V(1, 2, 0), a, maths.V(1, 0, 0), 2},
		{maths.V(3, 4, 0), a, a, 5},
	} {
		if got := toSegment(c.p, c.a, c.b); !close12(got, c.want) {
			t.Errorf("%v to the segment %v %v is %v, want %v", c.p, c.a, c.b, got, c.want)
		}
	}
}

// a shelf lifts the sea to 0.01 and leaves land as it is, at the water
// line too, and is ice everywhere.
func TestAShelfLiftsOnlyTheSea(t *testing.T) {
	s := Shelf{}
	for _, c := range []struct{ height, want float64 }{
		{-0.3, 0.01}, {-0.001, 0.01}, {0, 0}, {0.4, 0.4},
	} {
		if got := s.Shape(c.height, 0.5); got != c.want {
			t.Errorf("a shelf on %v is %v, want %v", c.height, got, c.want)
		}
	}
	if s.Class(0) != Ice || s.Class(1) != Ice {
		t.Error("a shelf is not ice")
	}
}

// a plateau is the top where the ground was lower and the ground where
// it was higher, up to the edge, share 1 in, and the ground past it, and
// is rock past 0.8 of the reach, 0.8 itself not.
func TestAPlateauIsFlatInsideAndRockAtItsEdge(t *testing.T) {
	p := Plateau{Top: 0.7}
	for _, c := range []struct{ height, share, want float64 }{
		{0.1, 0.5, 0.7},
		{0.9, 0.5, 0.9},
		{0.1, 1, 0.7},
		{0.1, 1.5, 0.1},
		{0.1, 3, 0.1},
	} {
		if got := p.Shape(c.height, c.share); got != c.want {
			t.Errorf("a plateau at a share of %v on %v is %v, want %v", c.share,
				c.height, got, c.want)
		}
	}
	for _, c := range []struct {
		share float64
		want  Biome
	}{{0, Unfixed}, {0.5, Unfixed}, {0.8, Unfixed}, {0.81, Rock}, {1, Rock}} {
		if got := p.Class(c.share); got != c.want {
			t.Errorf("a plateau at a share of %v is %d, want %d", c.share, got, c.want)
		}
	}
}

// a patch is a class and nothing else: the height is as it was, the
// class its own everywhere, the share the plain one, the footprint what
// it was given.
func TestAPatchIsAClassAndNothingElse(t *testing.T) {
	p := Patch{At: maths.V(1, 2, 3), Reach: 4, Of: Snow}
	if at, reach := p.Footprint(); at != p.At || reach != 4 {
		t.Errorf("the footprint is %v and %v", at, reach)
	}
	if got := p.Share(maths.V(1, 2, 5), p.At, 4); got != 0.5 {
		t.Errorf("2 from the heart of a reach of 4 is %v, want 0.5", got)
	}
	for _, share := range []float64{0, 0.5, 1} {
		if got := p.Shape(0.3, share); got != 0.3 {
			t.Errorf("a patch moved 0.3 to %v at %v", got, share)
		}
		if got := p.Class(share); got != Snow {
			t.Errorf("a patch at %v is %d, not snow", share, got)
		}
	}
}

// every biome, as dry ground and as sea. dry ground is beach and the
// ones above it in the list; ice is neither, and the sea is the deep and
// the shallows.
func TestWhatIsLandAndWhatIsWater(t *testing.T) {
	land := map[Biome]bool{Beach: true, Desert: true, Grass: true, Forest: true,
		Jungle: true, Tundra: true, Rock: true, Snow: true, Lava: true, Ash: true}
	water := map[Biome]bool{Deep: true, Sea: true}
	for b := Deep; b < Biomes; b++ {
		if Land(b) != land[b] {
			t.Errorf("Land(%d) is %v", b, Land(b))
		}
		if Water(b) != water[b] {
			t.Errorf("Water(%d) is %v", b, Water(b))
		}
	}
}
