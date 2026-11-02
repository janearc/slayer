package shortcut

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// ramp is a medium of known numbers: x squared, y squared, four z
// squared and x y z, over the box from (-1, 0, 2) to (2, 6, 3.5).
//
// at a bake of 4 its cells are 1 apart in x, 2 in y and 0.5 in z, so
// every lattice point has a whole or a half number for a coordinate and
// the density there is worked by hand. every lattice value is positive.
type ramp struct{}

func (ramp) Density(p maths.Vec) float64 {
	return p.X*p.X + p.Y*p.Y + 4*p.Z*p.Z + p.X*p.Y*p.Z
}

func (ramp) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-1, 0, 2), Max: maths.V(2, 6, 3.5)}
}

// slope is a medium thick in x alone, over (-1, 0, 0) to (1, 1, 1), and
// negative on the left of the middle.
type slope struct{}

func (slope) Density(p maths.Vec) float64 { return p.X }
func (slope) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-1, 0, 0), Max: maths.V(1, 1, 1)}
}

// spanned is a ramp that says its own span, from its ray and its range
// in a way a box could not: it enters at the ray's x, leaves at far
// less near, and is hit when the ray points up z.
type spanned struct{ ramp }

func (spanned) Span(r maths.Ray, near, far float64) (enter, leave float64, ok bool) {
	return r.At.X, far - near, r.Dir.Z > 0
}

// clock is a ramp that gets 1000 thicker each second, and ticks into
// the same ramp at the time it is told.
type clock struct {
	ramp
	at time.Duration
}

func (c clock) Density(p maths.Vec) float64 {
	return c.ramp.Density(p) + 1000*c.at.Seconds()
}

func (clock) Tick(t time.Duration) medium.Medium { return clock{at: t} }

// tripwire is a medium whose box cannot be asked for. Bake asks for it
// right after its guard and before it asks for any memory, so a Bake
// that gets past the guard stops here, for nothing.
type tripwire struct{ ramp }

func (tripwire) Bounds() maths.Box {
	panic("tripwire: the guard let the bake through")
}

// closeTo is whether two numbers are within e of each other. a NaN is
// close to nothing.
func closeTo(a, b, e float64) bool { return math.Abs(a-b) <= e }

// panicOf is what call panics with, as text, or "" when it does not.
func panicOf(call func()) (got string) {
	defer func() {
		if r := recover(); r != nil {
			got, _ = r.(string)
			if got == "" {
				got = "a panic that is not text"
			}
		}
	}()
	call()
	return ""
}

// at the lattice points the bake is the medium itself. the eight
// corners of ramp's box, worked from x squared plus y squared plus four
// z squared plus x y z, with x from -1 or 2, y from 0 or 6, z from 2 or
// 3.5:
//
//	(-1,0,2)    1 +  0 + 16 +   0 =  17
//	(2,0,2)     4 +  0 + 16 +   0 =  20
//	(-1,6,2)    1 + 36 + 16 -  12 =  41
//	(2,6,2)     4 + 36 + 16 +  24 =  80
//
//	(-1,0,3.5)  1 +  0 + 49 +   0 =  50
//	(2,0,3.5)   4 +  0 + 49 +   0 =  53
//	(-1,6,3.5)  1 + 36 + 49 -  21 =  65
//	(2,6,3.5)   4 + 36 + 49 +  42 = 131
//
// the far corner is the largest index on every axis; the corners on
// each face, one axis at a time, are the largest index on that axis
// alone.
func TestABakeIsTheMediumAtItsCorners(t *testing.T) {
	b := bakeIn(t, ramp{}, 4)
	corners := []struct {
		at   maths.Vec
		want float64
	}{
		{maths.V(-1, 0, 2), 17},
		{maths.V(2, 0, 2), 20},
		{maths.V(-1, 6, 2), 41},
		{maths.V(2, 6, 2), 80},
		{maths.V(-1, 0, 3.5), 50},
		{maths.V(2, 0, 3.5), 53},
		{maths.V(-1, 6, 3.5), 65},
		{maths.V(2, 6, 3.5), 131},
	}
	for _, c := range corners {
		if got := b.Density(c.at); !closeTo(got, c.want, 1e-9) {
			t.Errorf("at the corner %v the bake is %v, want %v",
				c.at, got, c.want)
		}
	}
}

// inside a cell the bake blends the eight cells round the point, each
// axis by how far along its cell the point is. the density is a sum of
// a function of x, one of y, one of z and the product x y z, so the
// blend is the three blended parts and the product, which the blend
// keeps whole.
//
// at (0.25, 3.5, 2.7): x is a quarter along the cell from 0 to 1, y
// three quarters along 2 to 4, z four tenths along 2.5 to 3.
//
//	x squared  0 + (1 - 0) * 0.25            =  0.25
//	y squared  4 + (16 - 4) * 0.75           = 13
//	4z squared 25 + (36 - 25) * 0.4          = 29.4
//	x y z      0.25 * 3.5 * 2.7              =  2.3625
//
//	sum                                      = 45.0125
//
// at (1.75, 0.5, 3.4): x three quarters along 1 to 2, y a quarter
// along 0 to 2, z four fifths along 3 to 3.5.
//
//	x squared  1 + 3 * 0.75                  =  3.25
//	y squared  0 + 4 * 0.25                  =  1
//	4z squared 36 + 13 * 0.8                 = 46.4
//	x y z      1.75 * 0.5 * 3.4              =  2.975
//
//	sum                                      = 53.625
//
// the exact density at the first is 0.0625 + 12.25 + 29.16 + 2.3625,
// not the blend, so the blend is not the medium.
func TestABakeBlendsTheCellsRoundAPoint(t *testing.T) {
	b := bakeIn(t, ramp{}, 4)
	for _, c := range []struct {
		at   maths.Vec
		want float64
	}{
		{maths.V(0.25, 3.5, 2.7), 45.0125},
		{maths.V(1.75, 0.5, 3.4), 53.625},
	} {
		if got := b.Density(c.at); !closeTo(got, c.want, 1e-9) {
			t.Errorf("at %v the bake is %v, want %v", c.at, got, c.want)
		}
	}
	exact := (ramp{}).Density(maths.V(0.25, 3.5, 2.7))
	if closeTo(exact, 45.0125, 1e-3) {
		t.Errorf("the exact density %v is the blend; the test is blind", exact)
	}
}

// a bake of 2 is the eight corners and nothing between, so the middle
// of the box is the mean of the eight corner values above:
// (17 + 20 + 41 + 80 + 50 + 53 + 65 + 131) / 8 = 457 / 8 = 57.125.
//
// the exact density at the middle (0.5, 3, 2.75) is 0.25 + 9 + 30.25 +
// 4.125 = 43.625.
func TestABakeOfTwoIsTheMeanOfItsCorners(t *testing.T) {
	b := bakeIn(t, ramp{}, 2)
	if got := b.Density(maths.V(0.5, 3, 2.75)); !closeTo(got, 57.125, 1e-9) {
		t.Errorf("the middle of a bake of 2 is %v, want 57.125", got)
	}
	if got := b.Density(maths.V(2, 6, 3.5)); !closeTo(got, 131, 1e-9) {
		t.Errorf("the far corner of a bake of 2 is %v, want 131", got)
	}
	if got := b.Density(maths.V(-1, 0, 2)); !closeTo(got, 17, 1e-9) {
		t.Errorf("the near corner of a bake of 2 is %v, want 17", got)
	}
}

// the grid is read where Bake wrote it: three cells a side, built by
// hand so the cells do not come from Bake, each the product of a
// number for its x, one for its y and one for its z, with
//
//	x at 1, 3, 5      a = 1, 2, 5
//	y at -2, 0, 2     b = 1, 3, 4
//	z at 0, 3, 6      c = 2, 3, 7
//
// a blend of products is the product of the blends, so the density
// inside a cell is the blended a times the blended b times the blended
// c.
//
// (4, -1, 4.5) is half way along 3 to 5, half way along -2 to 0 and
// half way along 3 to 6:
// (2 + 3 * 0.5) * (1 + 2 * 0.5) * (3 + 4 * 0.5) = 3.5 * 2 * 5 = 35.
//
// (1.5, 1.5, 1.5) is a quarter along 1 to 3, three quarters along 0 to
// 2 and half way along 0 to 3:
// (1 + 0.25) * (3 + 0.75) * (2 + 0.5) = 1.25 * 3.75 * 2.5 = 11.71875.
//
// (4.5, -1.5, 5.25) is three quarters along 3 to 5, a quarter along -2
// to 0 and three quarters along 3 to 6:
// (2 + 3 * 0.75) * (1 + 2 * 0.25) * (3 + 4 * 0.75) = 4.25 * 1.5 * 6 =
// 38.25.
//
// on the lattice the blend is the cell. (3, 0, 3) is the middle cell,
// 2 * 3 * 3 = 18, and the far corner (5, 2, 6) is 5 * 4 * 7 = 140. the
// far edge on one axis alone is 5 * 3 * 3 = 45 at x = 5, 2 * 4 * 3 = 24
// at y = 2, and 2 * 3 * 7 = 42 at z = 6.
func TestABakeIsReadWhereItIsWritten(t *testing.T) {
	a, b, c := []float64{1, 2, 5}, []float64{1, 3, 4}, []float64{2, 3, 7}
	cells := make([]float64, 27)
	for k := 0; k < 3; k++ {
		for j := 0; j < 3; j++ {
			for i := 0; i < 3; i++ {
				cells[(k*3+j)*3+i] = a[i] * b[j] * c[k]
			}
		}
	}
	bk := &Baked{Of: slope{}, Resolution: 3,
		box:   maths.Box{Min: maths.V(1, -2, 0), Max: maths.V(5, 2, 6)},
		cells: cells}
	for _, w := range []struct {
		at   maths.Vec
		want float64
	}{
		{maths.V(4, -1, 4.5), 35},
		{maths.V(1.5, 1.5, 1.5), 11.71875},
		{maths.V(4.5, -1.5, 5.25), 38.25},
		{maths.V(3, 0, 3), 18},
		{maths.V(5, 2, 6), 140},
		{maths.V(5, 0, 3), 45},
		{maths.V(3, 2, 3), 24},
		{maths.V(3, 0, 6), 42},
		{maths.V(1, -2, 0), 1 * 1 * 2},
	} {
		if got := bk.Density(w.at); !closeTo(got, w.want, 1e-12) {
			t.Errorf("at %v the bake is %v, want %v", w.at, got, w.want)
		}
	}
}

// outside the box the bake is zero, one face at a time: a point just past a
// face on one axis alone, the others well inside, is outside. ramp's box is x
// from -1 to 2, y from 0 to 6, z from 2 to 3.5; a point inside by the same
// small distance has a number.
func TestABakeIsNoughtPastEveryFace(t *testing.T) {
	b := bakeIn(t, ramp{}, 4)
	out := []maths.Vec{
		maths.V(-1.001, 3, 2.75), maths.V(2.001, 3, 2.75),
		maths.V(0.5, -0.001, 2.75), maths.V(0.5, 6.001, 2.75),
		maths.V(0.5, 3, 1.999), maths.V(0.5, 3, 3.501),
		maths.V(math.Inf(1), 3, 2.75), maths.V(0.5, math.Inf(-1), 2.75),
		maths.V(0.5, 3, math.NaN()),
		maths.V(-1.001, -0.001, 1.999), maths.V(2.001, 6.001, 3.501),
	}
	for _, p := range out {
		if got := b.Density(p); got != 0 {
			t.Errorf("at %v, outside, the bake is %v", p, got)
		}
	}
	in := []maths.Vec{
		maths.V(-0.999, 3, 2.75), maths.V(1.999, 3, 2.75),
		maths.V(0.5, 0.001, 2.75), maths.V(0.5, 5.999, 2.75),
		maths.V(0.5, 3, 2.001), maths.V(0.5, 3, 3.499),
	}
	for _, p := range in {
		if got := b.Density(p); !(got > 10) {
			t.Errorf("at %v, inside, the bake is %v", p, got)
		}
	}
}

// the bake is never less than zero, though a medium's numbers may be: a slope
// of x over -1 to 1 at a bake of 3 has the cells -1, 0 and 1 in x, so half way
// between the first two is -0.5 blended, which is read as zero; half way
// between the last two is 0.5.
func TestABakeIsNeverNegative(t *testing.T) {
	b := bakeIn(t, slope{}, 3)
	if got := b.Density(maths.V(-0.5, 0.5, 0.5)); got != 0 {
		t.Errorf("a blend of -0.5 is read as %v, want 0", got)
	}
	if got := b.Density(maths.V(0.5, 0.5, 0.5)); !closeTo(got, 0.5, 1e-12) {
		t.Errorf("a blend of 0.5 is read as %v, want 0.5", got)
	}
}

// a bake keeps the medium it was made from and the resolution it was
// asked for, from 2 up; under 2 is the default, 96 cells a side.
func TestABakeKeepsItsResolution(t *testing.T) {
	for _, n := range []int{2, 3, 5} {
		b := bakeIn(t, ramp{}, n)
		if b.Resolution != n || b.Of != (medium.Medium)(ramp{}) ||
			len(b.cells) != n*n*n {
			t.Errorf("a bake of %d has resolution %d, %d cells",
				n, b.Resolution, len(b.cells))
		}
	}
	for _, n := range []int{1, 0} {
		b := bakeIn(t, ramp{}, n)
		if b.Resolution != 96 || len(b.cells) != 96*96*96 {
			t.Errorf("a bake of %d has resolution %d, %d cells; want 96 and 884736",
				n, b.Resolution, len(b.cells))
		}
	}
}

// the ceiling is 256 cells a side: 257 and past are refused by name,
// 256 and under are not.
//
// tripwire stops a bake that gets past the guard before it asks for any
// memory, so no bake of any size is made here.
func TestTheCeilingIsTwoHundredAndFiftySix(t *testing.T) {
	for _, n := range []int{257, 1000, 3000000} {
		got := panicOf(func() { Bake(tripwire{}, n) })
		if !strings.Contains(got, "past 256") {
			t.Errorf("a bake of %d panics with %q, want the guard", n, got)
		}
	}
	for _, n := range []int{256, 255, 2} {
		got := panicOf(func() { Bake(tripwire{}, n) })
		if !strings.HasPrefix(got, "tripwire") {
			t.Errorf("a bake of %d panics with %q, want the guard passed", n, got)
		}
	}
}

// a bake is over its medium's bounds, and says so.
func TestABakeHasItsMediumsBounds(t *testing.T) {
	want := maths.Box{Min: maths.V(-1, 0, 2), Max: maths.V(2, 6, 3.5)}
	if got := bakeIn(t, ramp{}, 3).Bounds(); got != want {
		t.Errorf("the bounds are %v, want %v", got, want)
	}
	flat := maths.Box{Min: maths.V(-1, 0, 0), Max: maths.V(1, 1, 1)}
	if got := bakeIn(t, slope{}, 3).Bounds(); got != flat {
		t.Errorf("the bounds of the slope are %v", got)
	}
}

// a bake spans as its medium does when the medium says. spanned enters
// at the ray's x, leaves at far less near and is hit when the ray
// points up z, and a bake of it says exactly that, a miss included.
//
// a medium that does not say is spanned by the box. a ray along x from
// (-5, 3, 2.75) meets ramp's box where x is -1 and 2, which is 4 and 7
// along it; held to a near of 5 and a far of 6 it is 5 and 6; and a ray
// at y of 9 misses.
func TestABakeSpansAsItsMediumDoes(t *testing.T) {
	sp := bakeIn(t, spanned{}, 2)
	up := maths.Ray{At: maths.V(7, 0, 0), Dir: maths.V(0, 0, 1)}
	if e, l, ok := sp.Span(up, 10, 20); e != 7 || l != 10 || !ok {
		t.Errorf("the medium's span is %v, %v, %v; want 7, 10, true", e, l, ok)
	}
	down := maths.Ray{At: maths.V(7, 0, 0), Dir: maths.V(0, 0, -1)}
	if e, l, ok := sp.Span(down, 10, 20); e != 7 || l != 10 || ok {
		t.Errorf("the medium's miss is %v, %v, %v; want 7, 10, false", e, l, ok)
	}
	b := bakeIn(t, ramp{}, 2)
	along := maths.Ray{At: maths.V(-5, 3, 2.75), Dir: maths.V(1, 0, 0)}
	if e, l, ok := b.Span(along, 0, 100); e != 4 || l != 7 || !ok {
		t.Errorf("the box's span is %v, %v, %v; want 4, 7, true", e, l, ok)
	}
	if e, l, ok := b.Span(along, 5, 6); e != 5 || l != 6 || !ok {
		t.Errorf("the box's span from 5 to 6 is %v, %v, %v; want 5, 6, true", e, l, ok)
	}
	over := maths.Ray{At: maths.V(-5, 9, 2.75), Dir: maths.V(1, 0, 0)}
	if _, _, ok := b.Span(over, 0, 100); ok {
		t.Error("a ray over the box met it")
	}
}

// a bake of a medium that moves ticks into a bake of the medium at that
// time, at the same resolution, and leaves itself as it was. clock is
// ramp plus 1000 a second, so at 3 seconds the near corner is 17 +
// 3000 = 3017, where the first bake still has 17.
//
// a medium that does not move ticks into the same bake.
func TestABakeTicksAsItsMediumDoes(t *testing.T) {
	b := bakeIn(t, clock{}, 4)
	next, ok := b.Tick(3 * time.Second).(*Baked)
	if !ok || next == b {
		t.Fatalf("a moving medium ticked into %T, the same bake: %v", next, next == b)
	}
	if next.Of != (medium.Medium)(clock{at: 3 * time.Second}) {
		t.Errorf("the new bake is of %v, want the clock at 3 seconds", next.Of)
	}
	if next.Resolution != 4 {
		t.Errorf("the new bake has resolution %d, want 4", next.Resolution)
	}
	near := maths.V(-1, 0, 2)
	if got := next.Density(near); !closeTo(got, 3017, 1e-9) {
		t.Errorf("at 3 seconds the near corner is %v, want 3017", got)
	}
	if got := b.Density(near); !closeTo(got, 17, 1e-9) {
		t.Errorf("the first bake moved to %v, want 17", got)
	}
	still := bakeIn(t, ramp{}, 4)
	if got := still.Tick(time.Hour); got != medium.Medium(still) {
		t.Errorf("a medium that does not move ticked into %v", got)
	}
}
