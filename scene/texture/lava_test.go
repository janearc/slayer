package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// magma is a 3000 K blackbody's colour, red-orange, in no unit: the lava's
// light is read against it.
var magma = maths.RGB(1, 0.45, 0.12)

// sphereOf is a spiral of unit directions over the sphere, a count of them.
func sphereOf(count int) []maths.Vec {
	golden := math.Pi * (math.Sqrt(5) - 1)
	points := make([]maths.Vec, count)
	for index := range points {
		height := 1 - 2*(float64(index)+0.5)/float64(count)
		flat := math.Sqrt(1 - height*height)
		angle := golden * float64(index)
		points[index] = maths.V(flat*math.Cos(angle), height,
			flat*math.Sin(angle))
	}
	return points
}

// The lava's mean light over the ball, taken over points other than the
// ones its own mean was taken at, is the glow it was given in each colour,
// to a few per cent: a crust changes where a ball's light is, not how
// much of it there is.
func TestTheLavasMeanLightIsItsGlow(t *testing.T) {
	lava := NewLava(7, 100, magma)
	var sum maths.Light
	points := sphereOf(9001)
	for _, point := range points {
		sum = sum.Add(lava.At(point.Scale(100)))
	}
	mean := sum.Scale(1 / float64(len(points)))
	for name, pair := range map[string][2]float64{"red": {mean.R, magma.R},
		"green": {mean.G, magma.G}, "blue": {mean.B, magma.B}} {
		if math.Abs(pair[0]-pair[1]) > 0.04*pair[1] {
			t.Errorf("the mean %s is %v, want %v", name, pair[0], pair[1])
		}
	}
}

// The ball is dark plates parted by thin bright seams, as a lava lake's
// crust is, the light read against the brightest of the ball: a plate's
// middle, far from any seam and its halo, is under a tenth of it and
// redder than the magma; a seam is over half of it. The seams are a small
// part of the ball. The anti-test, which is the leopard's print: no point
// far from a seam is bright, so there is no soft dark blob with a glowing
// rim, and no point is without light.
func TestTheLavaIsDarkPlatesAndThinBrightSeams(t *testing.T) {
	lava := NewLava(7, 100, magma)
	points := sphereOf(30000)
	peak := 0.0
	for _, point := range points {
		peak = math.Max(peak, lava.At(point.Scale(100)).R)
	}
	var plate, seam, bright int
	for _, point := range points {
		found := lava.seam(point)
		light := lava.At(point.Scale(100))
		if light.R < 0 || light.G < 0 || light.B < 0 || !(light.R > 0) {
			t.Fatalf("the light at %v is %v", point, light)
		}
		switch {
		case found.Core < 0.001 && found.Halo < 0.05:
			plate++
			if light.R > 0.1*peak || light.G/light.R > magma.G/magma.R {
				t.Fatalf("the plate at %v, %+v, is %v of a peak of %v", point,
					found, light, peak)
			}
		case found.Core > 0.7:
			seam++
			if light.R < 0.5*peak {
				t.Fatalf("the seam at %v is only %v of a peak of %v", point,
					light, peak)
			}
		}
		if light.R > 0.3*peak {
			bright++
		}
	}
	t.Logf("of %d points %d are deep in a plate, %d on a seam and %d bright",
		len(points), plate, seam, bright)
	if plate < len(points)/10 || seam < 50 || bright < len(points)/100 ||
		bright > len(points)/4 {
		t.Errorf("%d plate, %d seam and %d bright points of %d", plate, seam,
			bright, len(points))
	}
}

// A plate's crust glows a dull red, not black, and its glow varies with its
// thickness: among the points deep in plates the red is over half a
// thousandth of the brightest of the ball, the brightest is over twice the
// dimmest, and the crust is redder than the magma. The anti-test: the
// crust never reaches the seams' light, so it is not drawn as a second
// kind of magma.
func TestACrustGlowsDullRedAndVariesWithItsThickness(t *testing.T) {
	lava := NewLava(7, 100, magma)
	points := sphereOf(30000)
	peak, least, most := 0.0, math.Inf(1), 0.0
	for _, point := range points {
		peak = math.Max(peak, lava.At(point.Scale(100)).R)
	}
	for _, point := range points {
		if found := lava.seam(point); found.Core < 0.001 &&
			found.Halo < 0.05 {
			light := lava.At(point.Scale(100))
			least, most = math.Min(least, light.R), math.Max(most, light.R)
		}
	}
	t.Logf("a crust of %.4f to %.4f of the peak", least/peak, most/peak)
	if least < 0.0005*peak || most < 2*least || most > 0.1*peak {
		t.Errorf("crust is %v to %v of a peak of %v", least, most, peak)
	}
}

// Seams are of different widths, by the plate they border, between the
// spread's share either side of the width: the widest is more than twice
// the narrowest. The anti-test: none is wider or narrower than that.
func TestSeamsAreOfDifferentWidths(t *testing.T) {
	lava := NewLava(7, 100, magma)
	narrow, wide := math.Inf(1), 0.0
	for _, point := range sphereOf(30000) {
		if found := lava.seam(point); found.Core > 0.5 {
			narrow, wide = math.Min(narrow, found.Width),
				math.Max(wide, found.Width)
		}
	}
	t.Logf("seams of %.4f to %.4f", narrow, wide)
	if wide < 2*narrow ||
		narrow < lavaSeamWidth*(1-lavaSeamSpread)*(1-1e-9) ||
		wide > lavaSeamWidth*(1+lavaSeamSpread)*(1+1e-9) {
		t.Errorf("seams of %v to %v wide of %v", narrow, wide, lavaSeamWidth)
	}
}

// nextTo is a point on the sphere a step of an angle in radians from a
// direction, along a tangent to it.
func nextTo(point maths.Vec, step float64) maths.Vec {
	along := point.Cross(maths.V(1, 0, 0))
	if along.Len() < 0.5 {
		along = point.Cross(maths.V(0, 1, 0))
	}
	return point.Add(along.Unit().Scale(step)).Unit()
}

// Plates are of many sizes and the sizes run into one another: the
// largest plates are more than twice the size of the smallest, and the
// size changes smoothly, by under a twentieth between points a hundredth of
// a radian apart, with no band where it jumps. The anti-test: the lattice
// is stretched and never folded, so its scale never falls to nothing or
// turns round, which would make a plate's size go negative or leave a
// patch of the ball with no plates.
func TestPlatesComeInManySizesThatRunIntoOneAnother(t *testing.T) {
	lava := NewLava(7, 100, magma)
	smallest, largest, steepest := math.Inf(1), 0.0, 0.0
	for _, point := range sphereOf(40000) {
		size := lava.seam(point).Size
		smallest, largest = math.Min(smallest, size), math.Max(largest, size)
		near := nextTo(point, 0.01)
		steepest = math.Max(steepest,
			math.Abs(lava.seam(near).Size/size-1))
		if !(size > 0) {
			t.Fatalf("a plate of size %v at %v", size, point)
		}
	}
	t.Logf("plates from %.2f to %.2f of the middle size, the steepest "+
		"change %.3f in a hundredth of a radian", smallest, largest, steepest)
	if largest < 2*smallest || steepest > 0.05 {
		t.Errorf("sizes %v to %v, steepest change %v", smallest, largest,
			steepest)
	}
}

// The plates' lattice is not folded over itself: stepping a hundredth of a
// radian over the ball moves the lattice point on, by between a quarter
// and five times as far as a step of the lattice at its middle scale: the
// scale changes the step, by the size's range, and does not stop or
// reverse it.
func TestThePlatesLatticeIsNeverFolded(t *testing.T) {
	lava := NewLava(7, 100, magma)
	step := 0.01
	for _, point := range sphereOf(20000) {
		near := nextTo(point, step)
		moved := lava.bent(near).Sub(lava.bent(point)).Len()
		cell := step * lavaFrequency
		if moved < cell/4 || moved > 5*cell {
			t.Fatalf("a step of %v moved the lattice %v at %v, a step "+
				"of the lattice at its middle being %v", step, moved, point,
				cell)
		}
	}
}

// The plates' lattice keeps its orientation: the two tangent steps of the
// ball, a thousandth of a radian each, map to two steps of the lattice
// whose cross product points out of the ball, as the steps' own do, at
// every point of it. A fold would turn the cross product
// round, or take its length to nothing, and squeeze plates into streaks;
// and the lattice never stretches a step one way more than lavaMostStretch
// times the other. The anti-test: a size spread of three stretches over
// twice as much as the picked one, so the check can tell.
func TestThePlatesLatticeKeepsItsOrientation(t *testing.T) {
	count := func(spread float64) (reversed, squeezed, points int,
		worst float64) {
		saved := lavaSizeSpread
		lavaSizeSpread = spread
		defer func() { lavaSizeSpread = saved }()
		lava := NewLava(7, 100, magma)
		const step = 1e-3
		for _, point := range sphereOf(40000) {
			first := nextTo(point, step).Sub(point).Unit()
			second := point.Cross(first)
			one := lava.bent(point.Add(first.Scale(step))).Sub(
				lava.bent(point))
			other := lava.bent(point.Add(second.Scale(step))).Sub(
				lava.bent(point))
			normal := one.Cross(other)
			// the Gram matrix's two roots are the squares of how much the
			// lattice stretches the ball's steps, most and least
			a, b, c := one.Dot(one), one.Dot(other), other.Dot(other)
			middle := (a + c) / 2
			spread := math.Sqrt(math.Max(0, middle*middle-(a*c-b*b)))
			longest, shortest := math.Sqrt(middle+spread),
				math.Sqrt(math.Max(middle-spread, 1e-30))
			if normal.Dot(point) <= 0 {
				reversed++
			} else if longest/shortest > lavaMostStretch {
				squeezed++
			}
			worst = math.Max(worst, longest/shortest)
			points++
		}
		t.Logf("spread %v: the lattice stretches a step %.2f times more one "+
			"way than the other at worst", spread, worst)
		return reversed, squeezed, points, worst
	}
	reversed, squeezed, points, worst := count(lavaSizeSpread)
	t.Logf("%d of %d points reversed and %d squeezed", reversed, points,
		squeezed)
	if reversed != 0 || squeezed != 0 {
		t.Errorf("the lattice is reversed at %d and squeezed at %d of %d",
			reversed, squeezed, points)
	}
	if _, _, _, more := count(3); !(more > 2*worst) {
		t.Errorf("a size spread of three stretches no more than the "+
			"picked one, %v against %v", more, worst)
	}
}

// A pixel that covers many plates sees their mean, the glow, and not
// speckle: the same point asked at a wide footprint is the glow, and at no
// footprint it is the plate's own.
func TestAFarLavaBallIsItsMeanLight(t *testing.T) {
	lava := NewLava(7, 100, magma)
	for _, point := range sphereOf(50) {
		at := point.Scale(100)
		if got := lava.Within(at, 100); got != magma {
			t.Fatalf("a footprint of a radius gave %v, want the glow", got)
		}
		if got := lava.Within(at, 0); got != lava.At(at) {
			t.Fatalf("no footprint gave %v, want the plate's %v", got,
				lava.At(at))
		}
	}
}
