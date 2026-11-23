package shape

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// The numbers the brute-force reference and its properties are held to, each
// a choice of the test.
const (
	// referenceSteps is how many samples the reference takes along the part
	// of a ray that is over the field's square. Two roots nearer than a
	// step are not seen by it, and the property allows for that.
	referenceSteps = 6000
	// referenceGap is how far above or below the surface a reported hit may
	// be, as a share of the field's reach, for rounding.
	referenceGap = 1e-9
	// referenceAlong is how far past the end of the reference's bracket a
	// reported hit may be, as a share of that distance, for rounding.
	referenceAlong = 1e-9
	// leastElevation and greatestElevation bound the angle in degrees a
	// drawn ray makes with the level, either side of level.
	leastElevation, greatestElevation = 0.05, 89.9
	// lowestScale and highestScale bound how long a drawn ray's direction is,
	// so the distance is counted in lengths of it, not in metres.
	lowestScale, highestScale = 0.5, 3.0
	// testLift is a share of the field's reach that a drawn ray starts off
	// the surface it leaves, as the tracer lifts a ray off the surface.
	testLift = 1e-6
	// leastLeavingDegrees is how steeply to the surface a ray drawn from a
	// point exactly on it must leave: a ray along the surface has no first
	// hit, and one a hair off it cannot be told from the surface.
	leastLeavingDegrees = 1.0
	// planeSamples and planeExtent are the planar field's grid and width.
	planeSamples, planeExtent = 5, 8.0
	// planeSlope is the largest slope a drawn plane has along either axis.
	planeSlope = 2.0
	// planeInset is how much of the plane's half width a drawn start keeps
	// within, so a short ray from it is over the plane's square.
	planeInset = 0.9
	// antiTestFar is how far the wrong-reference test follows its ray, in
	// metres: past the ray's height of 3 over the field, and not a length
	// at which a crossing, at 2 or 3, falls on one of the reference's
	// samples, where the gap is exactly nought and changes no sign.
	antiTestFar = 7.0
)

// gapAt is how far above the surface the ray is at along: the ray's height
// less the field's, over the square and held to it.
func gapAt(field *HeightField, r maths.Ray, along float64) float64 {
	at := r.Point(along)
	return at.Y - field.height(at.X, at.Z)
}

// spanOver is the part of the ray, from nought to far, that lies over the
// field's square, by the slab test of each axis.
func spanOver(field *HeightField, r maths.Ray, far float64) (float64,
	float64, bool) {
	enter, leave := 0.0, far
	reach := field.extent * half
	for _, axis := range [2][2]float64{{r.At.X, r.Dir.X}, {r.At.Z, r.Dir.Z}} {
		origin, direction := axis[0], axis[1]
		if direction == 0 {
			if math.Abs(origin) > reach {
				return 0, 0, false
			}
			continue
		}
		one, two := (-reach-origin)/direction, (reach-origin)/direction
		enter = math.Max(enter, math.Min(one, two))
		leave = math.Min(leave, math.Max(one, two))
	}
	return enter, leave, enter <= leave
}

// referenceHit samples the ray across the square and returns the first
// stretch between two samples over which it passes from one side of the
// surface to the other, and whether there is one. It is slow and it is
// plain, and shares nothing with the cell walk but the height function.
func referenceHit(field *HeightField, r maths.Ray, far float64) (low,
	high float64, found bool) {
	enter, leave, ok := spanOver(field, r, far)
	if !ok {
		return 0, 0, false
	}
	before := gapAt(field, r, enter)
	for step := 1; step <= referenceSteps; step++ {
		along := enter + (leave-enter)*float64(step)/referenceSteps
		gap := gapAt(field, r, along)
		if before*gap < 0 {
			return enter + (leave-enter)*float64(step-1)/referenceSteps,
				along, true
		}
		before = gap
	}
	return 0, 0, false
}

// The field and ray a property run once drew and shrank to: seven samples
// fifty metres across, every height one metre but a pit of 2^-30 m at the
// centre sample and 1/32 m in front of it, and a ray starting 2.8e-9 m over
// the pit's floor, rising a degree. The surface climbs from the pit at about
// 0.17 a metre of the ray's run and the ray at 0.017, so it is crossed at
// 2.8e-9 / 0.15, 1.9e-8 along it.
const (
	nearSamples, nearExtent = 7, 50.0
	nearFlat                = 1.0
	nearPitAt, nearPit      = 24, 9.313225746154785e-10
	nearSlopeAt, nearSlope  = 17, 0.03125
)

// nearRay is that ray, and nearField that field.
var nearRay = maths.Ray{
	At: maths.V(7.450580596923828e-09, 3.725290298461914e-09,
		-4.76837158203125e-07),
	Dir: maths.V(0.5402200152099431, 0.01745240643728351,
		0.8413428247011541)}

// nearField is the field the ray started over.
func nearField(t *testing.T) *HeightField {
	t.Helper()
	heights := make([]float64, nearSamples*nearSamples)
	for place := range heights {
		heights[place] = nearFlat
	}
	heights[nearPitAt], heights[nearSlopeAt] = nearPit, nearSlope
	field, err := NewHeightField(heights, nearSamples, nearExtent)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// A ray that starts closer to the surface than the departure is leaving it,
// and the crossing at its start is not a hit, by the rule Hit documents: the
// reference sees the crossing and Hit does not. The same ray lifted off the
// surface as a tracer lifts one, a millionth of the field's reach, is met
// where the reference puts the crossing: no hit is lost by the rule.
func TestARayStartedWithinTheDepartureLeavesTheSurface(t *testing.T) {
	field := nearField(t)
	far := shapePropertyRangesIn(t).far
	low, _, found := referenceHit(field, nearRay, far)
	_, hit := field.Hit(nearRay, 0, far)
	if !found || hit || !(low < departureEnd(field, nearRay)) {
		t.Fatalf("crossing found %t from %g, Hit %t, the departure ends at %g",
			found, low, hit, departureEnd(field, nearRay))
	}
	if why := disagreement(field, nearRay, far); why != "" {
		t.Errorf("the property held to a ray it was never to be held to: %s", why)
	}
	lifted := nearRay
	lifted.At.Y += testLift * nearExtent
	if why := disagreement(field, lifted, far); why != "" {
		t.Errorf("the ray lifted off the surface: %s", why)
	}
	if _, hit := field.Hit(lifted, 0, far); !hit {
		t.Error("the ray lifted off the surface met nothing")
	}
}

// departureEnd is how far along the ray, from its start, a crossing is the
// surface the ray is leaving and not a hit, by the rule Hit documents: the
// share departure of the larger of the field's extent, its height range and
// how far the ray starts from the field's zero, in lengths of the ray's
// direction.
func departureEnd(field *HeightField, r maths.Ray) float64 {
	reach := math.Max(math.Max(field.extent, field.top-field.bottom),
		r.At.Len())
	return departure * reach / r.Dir.Len()
}

// disagreement is what is wrong with the answer Hit gives for a ray, against
// the reference, in words, or nothing when it agrees. A reference bracket
// that begins inside the departure may hold the surface the ray is leaving,
// which Hit does not find by its own rule, so it is no complaint.
func disagreement(field *HeightField, r maths.Ray, far float64) string {
	low, high, found := referenceHit(field, r, far)
	got, hit := field.Hit(r, 0, far)
	reach := math.Max(field.extent, field.top-field.bottom)
	leaving := low < departureEnd(field, r)
	switch {
	case found && !hit && !leaving:
		return fmt.Sprintf("a crossing between %g and %g was missed", low, high)
	case hit && math.Abs(gapAt(field, r, got)) > referenceGap*reach:
		return fmt.Sprintf("the hit at %g is %g off the surface", got,
			gapAt(field, r, got))
	case found && !leaving && got > high+referenceAlong*(1+high):
		return fmt.Sprintf("the hit at %g is past the crossing at %g", got,
			high)
	}
	return ""
}

// drawnRay draws a ray of any direction but level, from over, beside or
// under the field, of any length of direction.
func drawnRay(t *rapid.T, ranges shapePropertyRanges,
	field *HeightField) maths.Ray {
	reach := field.extent * half
	elevation := rapid.Float64Range(leastElevation, greatestElevation).
		Draw(t, "elevation") * math.Pi / 180
	if rapid.Bool().Draw(t, "down") {
		elevation = -elevation
	}
	azimuth := rapid.Float64Range(0, 2*math.Pi).Draw(t, "azimuth")
	scale := rapid.Float64Range(lowestScale, highestScale).Draw(t, "scale")
	dir := maths.V(math.Cos(elevation)*math.Cos(azimuth), math.Sin(elevation),
		math.Cos(elevation)*math.Sin(azimuth)).Scale(scale)
	at := maths.V(
		rapid.Float64Range(-2*reach, 2*reach).Draw(t, "x"),
		rapid.Float64Range(field.bottom-ranges.rise, field.top+ranges.rise).
			Draw(t, "y"),
		rapid.Float64Range(-2*reach, 2*reach).Draw(t, "z"))
	return maths.Ray{At: at, Dir: dir}
}

// TestHeightFieldHitAgreesWithBruteForce says, for fields drawn at random
// and rays of any angle drawn at random, that Hit meets the surface where
// a plain sampling of the ray says it crosses, and nowhere that is not on
// the surface.
func TestHeightFieldHitAgreesWithBruteForce(t *testing.T) {
	ranges := shapePropertyRangesIn(t)
	rapid.Check(t, func(t *rapid.T) {
		field := ranges.drawnField(t)
		ray := drawnRay(t, ranges, field)
		if why := disagreement(field, ray, ranges.far); why != "" {
			t.Fatalf("ray %v over a field of %d samples %g across: %s", ray,
				field.samples, field.extent, why)
		}
	})
}

// TestHeightFieldHitDoesNotAgreeWithAWrongReference is the anti-test of the
// property: a reference that is told the surface is a metre higher than it
// is must disagree with Hit for some ray, so the property can fail.
//
// The reference's bracket is the ray's span over the field cut in
// referenceSteps, so the span is held to antiTestFar: the bracket is about a
// thousandth of a metre wide, and cannot hold both the crossing of the
// raised field, at 2 along the ray, and that of the field, at 3. With the
// property's far, a million metres, the bracket would be 166 metres wide
// and hold both.
func TestHeightFieldHitDoesNotAgreeWithAWrongReference(t *testing.T) {
	field := flat(t, 0)
	raised := flat(t, 1)
	ray := maths.Ray{At: maths.V(0, 3, 0), Dir: maths.V(0, -1, 0)}
	far := antiTestFar
	if why := disagreement(field, ray, far); why != "" {
		t.Fatalf("a flat field and a vertical ray disagree: %s", why)
	}
	low, high, found := referenceHit(raised, ray, far)
	got, hit := field.Hit(ray, 0, far)
	if !found || !hit {
		t.Fatalf("the reference found %t and Hit %t; both should meet it",
			found, hit)
	}
	if got >= low && got <= high {
		t.Errorf("a reference over a field a metre higher met the ray at %g "+
			"to %g, where the field is met at %g", low, high, got)
	}
}

// slopedField is a planar field: heights a plane of slope along x and along z
// through the origin at a height, so every cell of it is the same plane.
func slopedField(t *rapid.T, slopeX, slopeZ, level float64) *HeightField {
	step := planeExtent / float64(planeSamples-unity)
	heights := make([]float64, planeSamples*planeSamples)
	for row := range planeSamples {
		for column := range planeSamples {
			x := -planeExtent*half + float64(column)*step
			z := -planeExtent*half + float64(row)*step
			heights[row*planeSamples+column] = level + slopeX*x + slopeZ*z
		}
	}
	field, err := NewHeightField(heights, planeSamples, planeExtent)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// leavingRay draws a ray that starts on the plane the field is, lifted off
// it by lift of the field's reach to the side it leaves to, and leaves to
// that side at least degrees from the plane. The side is one for above, and
// minus one for below.
func leavingRay(t *rapid.T, field *HeightField,
	side, lift, degrees float64) maths.Ray {
	reach := field.extent * half * planeInset
	x := rapid.Float64Range(-reach, reach).Draw(t, "x")
	z := rapid.Float64Range(-reach, reach).Draw(t, "z")
	normal := field.GeometricNormal(maths.V(x, 0, z))
	rise := field.height(1, 0) - field.height(0, 0)
	along := maths.V(1, rise, 0).Unit()
	across := normal.Cross(along).Unit()
	turn := rapid.Float64Range(0, 2*math.Pi).Draw(t, "turn")
	angle := rapid.Float64Range(degrees, greatestElevation).Draw(t, "leave") *
		math.Pi / 180
	crosswise := across.Scale(math.Sin(turn))
	toward := along.Scale(math.Cos(turn)).Add(crosswise)
	lifting := normal.Scale(side * math.Sin(angle))
	dir := toward.Scale(math.Cos(angle)).Add(lifting)
	start := maths.V(x, field.height(x, z), z).
		Add(normal.Scale(side * lift * field.extent))
	return maths.Ray{At: start, Dir: dir}
}

// TestARayLeavingAPlaneNeverHitsTheCellItLeft says, for planes of any slope
// and rays that start on one, from either side and lifted off it as a
// tracer lifts them, that going away from the plane is a miss.
func TestARayLeavingAPlaneNeverHitsTheCellItLeft(t *testing.T) {
	far := shapePropertyRangesIn(t).far
	rapid.Check(t, func(t *rapid.T) {
		slopeX := rapid.Float64Range(-planeSlope, planeSlope).Draw(t, "slopeX")
		slopeZ := rapid.Float64Range(-planeSlope, planeSlope).Draw(t, "slopeZ")
		field := slopedField(t, slopeX, slopeZ, 0)
		side := 1.0
		if rapid.Bool().Draw(t, "below") {
			side = -1
		}
		ray := leavingRay(t, field, side, testLift, 0)
		if got, hit := field.Hit(ray, 0, far); hit {
			t.Fatalf("a ray lifted off a plane and leaving it hit it at %g: %v",
				got, ray)
		}
		onIt := leavingRay(t, field, side, 0, leastLeavingDegrees)
		if got, hit := field.Hit(onIt, 0, far); hit {
			t.Fatalf("a ray on a plane and leaving it hit it at %g: %v", got,
				onIt)
		}
	})
}

// TestARayAimedBackAtAPlaneHitsItAtTheLiftOverTheCosine is the anti-test of
// the one above: the same ray, lifted off the plane the same, aimed into it
// is met where geometry puts it, so the guard against the cell a ray left does
// not swallow the hit a ray makes coming back.
func TestARayAimedBackAtAPlaneHitsItAtTheLiftOverTheCosine(t *testing.T) {
	far := shapePropertyRangesIn(t).far
	rapid.Check(t, func(t *rapid.T) {
		slopeX := rapid.Float64Range(-planeSlope, planeSlope).Draw(t, "slopeX")
		slopeZ := rapid.Float64Range(-planeSlope, planeSlope).Draw(t, "slopeZ")
		field := slopedField(t, slopeX, slopeZ, 0)
		ray := leavingRay(t, field, 1, testLift, leastLeavingDegrees)
		normal := field.GeometricNormal(maths.V(ray.At.X, 0, ray.At.Z))
		bounce := 2 * ray.Dir.Dot(normal)
		ray.Dir = ray.Dir.Sub(normal.Scale(bounce))
		lift := testLift * field.extent
		want := lift / -ray.Dir.Dot(normal)
		got, hit := field.Hit(ray, 0, far)
		if !hit || math.Abs(got-want) > referenceGap*field.extent*(1+want) {
			t.Fatalf("a ray aimed back at a plane hit at %g, %t; want %g: %v",
				got, hit, want, ray)
		}
	})
}
