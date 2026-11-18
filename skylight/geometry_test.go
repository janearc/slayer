package skylight

import (
	"fmt"
	"math"
	"testing"

	"pgregory.net/rapid"
)

// The rays and shells the geometry is checked on.
const (
	exitShare = 1e-7 // picked: the squares near six thousand kilometres
	// carry a few thousandths of a square metre; a millimetre's error in
	// the shell moves the grazing chord by five parts in ten thousand
	quarter    = 0.25 // picked: a quarter metre off the tangent
	millimetre = 1e-3 // defined: a thousandth of a metre
	onSphere   = 1e-6 // picked: a micrometre; the radii's own rounding
	// near six thousand kilometres is under a nanometre
	thickShell = 1e3   // picked: a kilometre, for random rays
	basisShare = 1e-12 // picked: a few roundings of numbers near one
)

// a ray is a start and a way, and the distance it should go in the
// shell, and whether it should leave by the outer sphere.
type ray struct {
	name              string
	position, heading vector
	distance          float64
	outward           bool
}

// metreRays are rays through a shell one metre thick at Earth's radius:
// along the ground's tangent, from the top a quarter metre inside the
// tangent and a quarter metre outside it, and straight down.
//
// From the top, radius s, a line whose nearest approach to the centre
// is b goes the root of s squared less b squared to that nearest point,
// and meets the ground, radius R, the root of R squared less b squared
// before it.
func metreRays(thickness float64) []ray {
	ground, top := earthRadius, earthRadius+thickness
	inside, outside := ground-quarter, ground+quarter
	return []ray{
		{"along the tangent", vector{ground, none, none},
			vector{none, none, one},
			math.Sqrt((top - ground) * (top + ground)), true},
		{"inside the tangent", vector{top, none, none},
			aimed(top, inside),
			math.Sqrt((top-inside)*(top+inside)) -
				math.Sqrt((ground-inside)*(ground+inside)), false},
		{"outside the tangent", vector{top, none, none},
			aimed(top, outside),
			two * math.Sqrt((top-outside)*(top+outside)), true},
		{"straight down", vector{top, none, none},
			vector{-one, none, none}, thickness, false},
	}
}

// aimed is the way from a point at radius on the x axis whose line
// passes nearest the centre at nearest, going inward and up.
func aimed(radius, nearest float64) vector {
	across := math.Sqrt((radius-nearest)*(radius+nearest)) / radius
	return vector{-across, none, nearest / radius}
}

// leaves is nil when every ray goes its distance in the layer and
// leaves by the sphere it should.
func leaves(through layer, rays []ray) error {
	for _, each := range rays {
		got, outward := exit(each.position, each.heading, &through)
		if outward != each.outward || !(math.Abs(got-each.distance) <=
			exitShare*each.distance) {
			return fmt.Errorf("%s: %.12g outward %v, want %.12g %v",
				each.name, got, outward, each.distance, each.outward)
		}
	}
	return nil
}

// a ray in a shell one metre thick at Earth's radius goes its exact
// distance: along the tangent, grazing just inside and just outside
// it, and straight down.
func TestGrazingRaysThroughAMetre(t *testing.T) {
	metre := layerOf(earthRadius, Shell{Top: one}, none, one)
	if err := leaves(metre, metreRays(one)); err != nil {
		t.Error(err)
	}
}

// the anti-test: the tangent ray worked out for a shell a millimetre
// thicker does not fit the metre's, so the check sees a millimetre at
// grazing. The other rays start at the thicker top, outside the metre.
func TestAMillimetreIsSeen(t *testing.T) {
	metre := layerOf(earthRadius, Shell{Top: one}, none, one)
	tangent := metreRays(one + millimetre)[none]
	if leaves(metre, []ray{tangent}) == nil {
		t.Errorf("%s: a millimetre is not seen", tangent.name)
	}
}

// landing is how far from the sphere it names a ray in the layer ends,
// and how far outside the layer the middle of its path is.
func landing(through layer, position, heading vector) (float64, float64) {
	distance, outward := exit(position, heading, &through)
	target := math.Sqrt(through.innerSquared)
	if outward {
		target = math.Sqrt(through.outerSquared)
	}
	end := position.plus(heading.scaled(distance)).length()
	middle := position.plus(heading.scaled(distance / two)).length()
	outside := max(middle-math.Sqrt(through.outerSquared),
		math.Sqrt(through.innerSquared)-middle, none)
	return math.Abs(end - target), outside
}

// any ray from inside a shell a kilometre thick ends on the sphere it
// says it leaves by, and goes there without leaving the shell.
func TestExitLandsOnItsSphere(t *testing.T) {
	shell := layerOf(earthRadius, Shell{Top: thickShell}, none,
		thickShell)
	rapid.Check(t, func(draw *rapid.T) {
		height := rapid.Float64Range(none, thickShell).Draw(draw, "height")
		rise := rapid.Float64Range(-one, one).Draw(draw, "rise")
		spin := rapid.Float64Range(none, one).Draw(draw, "spin")
		position := vector{earthRadius + height, none, none}
		heading := turned(vector{one, none, none}, rise, spin)
		off, outside := landing(shell, position, heading)
		if !(off <= onSphere) || !(outside <= onSphere) {
			draw.Fatalf("ends %g m off its sphere, %g m outside", off,
				outside)
		}
	})
}

// the anti-test: a ray that leaves by the top is a metre from the
// ground's sphere, so the check would see the wrong sphere named.
func TestTheOtherSphereIsSeen(t *testing.T) {
	metre := layerOf(earthRadius, Shell{Top: one}, none, one)
	outside := metreRays(one)[two]
	distance, _ := exit(outside.position, outside.heading, &metre)
	end := outside.position.plus(outside.heading.scaled(distance))
	if off := math.Abs(end.length() - earthRadius); off <= onSphere {
		t.Errorf("a ray out of the top ends %g m from the ground", off)
	}
}

// orthonormal is nil when the three are each of length one and square
// to each other.
func orthonormal(axis, first, second vector) error {
	for _, pair := range [][two]vector{{axis, first}, {axis, second},
		{first, second}} {
		if !(math.Abs(pair[none].dot(pair[one])) <= basisShare) {
			return fmt.Errorf("%v and %v are not square", pair[none],
				pair[one])
		}
	}
	for _, each := range []vector{axis, first, second} {
		if !(math.Abs(each.length()-one) <= basisShare) {
			return fmt.Errorf("%v is not of length one", each)
		}
	}
	return nil
}

// axes are ways to build a basis about, the two poles among them.
var axes = []vector{{none, none, one}, {none, none, -one},
	{one, none, none}, vector{0.3, -0.5, 0.6}.unit(), // picked: oblique
	vector{1e-9, 2e-9, -1}.unit()} // picked: a hair from the south pole

// every basis built about an axis is orthonormal with it, at the poles
// and away from them.
func TestTheBasisIsOrthonormal(t *testing.T) {
	for _, axis := range axes {
		first, second := basis(axis)
		if err := orthonormal(axis, first, second); err != nil {
			t.Errorf("about %v: %v", axis, err)
		}
	}
}

// the anti-test: a basis with one vector a thousandth long is refused,
// so the check would see a basis that stretched the turns.
func TestAStretchedBasisIsCaught(t *testing.T) {
	stretch := 1.001 // picked: a thousandth too long
	first, second := basis(axes[two])
	if orthonormal(axes[two], first.scaled(stretch), second) == nil {
		t.Error("a stretched basis is orthonormal")
	}
}

// a crossing nearer the horizon than the grazing cosine is weighed by
// the mean of one over the cosine over the crossings in that band, two
// over the grazing cosine, whichever way it goes.
func TestGrazingCrossingsAreCapped(t *testing.T) {
	near := grazing / two / two // picked: well inside the grazing band
	for _, cosine := range []float64{near, -near, none} {
		if got := slant(cosine); got != two/grazing {
			t.Errorf("cosine %g weighs %g, want %g", cosine, got,
				two/grazing)
		}
	}
}

// the anti-test: a crossing above the grazing band is weighed by its own
// cosine, not the band's, so the cap reaches no further than it says.
func TestASteepCrossingIsNotCapped(t *testing.T) {
	steep := half // picked: sixty degrees from the vertical
	if got := slant(-steep); got != one/steep {
		t.Errorf("cosine %g weighs %g, want %g", steep, got, one/steep)
	}
}
