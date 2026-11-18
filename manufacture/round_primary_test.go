package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// The witness for the rings and teeth in the Hubble and Spitzer primary
// mirrors: the dish was rings of flat strips, each a flat mirror. Its
// top now carries the paraboloid's normals, which turn smoothly from
// ring to ring and from strip to strip.

// ringShare is where across the dish, as a share of its radius, the
// rays fall either side of a ring: half way, a ring for any even count
// of rings, twelve at 96 sides.
const ringShare = 0.5 // defined: ring six of twelve, three of six

// stripShare is where across the dish the two rays either side of the
// strip edge at angle nought fall, clear of every ring.
const stripShare = 0.3 // picked: 3.6 rings of 12 out

// ringTurnLeast is the least the faces' own normals turn across a ring,
// in degrees: rings of the spacecraft's dish meet at 0.48 degrees.
const ringTurnLeast = 0.2 // picked: well under 0.48, well over nought

// dishStripTurnLeast is the least the faces' own normals turn across a
// strip edge three tenths out, in degrees: they meet at 0.11.
const dishStripTurnLeast = 0.05 // picked: well under 0.11, over nought

// TestRoundPrimaryTurnsSmoothlyAcrossRingsAndStrips is the witness:
// either side of a ring and of a strip edge, the primary's shading
// normal is the paraboloid's, nearly the same. On 733bdc72 it turned by
// 0.48 degrees across the ring and 0.11 across the strip.
func TestRoundPrimaryTurnsSmoothlyAcrossRingsAndStrips(test *testing.T) {
	for _, design := range []string{"hubble", "spitzer"} {
		ring, strip, _, _ := primaryTurns(test, design)
		test.Logf("%s: the shading normal turns %.5f degrees across a "+
			"ring and %.5f across a strip", design, ring, strip)
		if !(ring < smoothTurnLimit && strip < smoothTurnLimit) {
			test.Fatalf("%s: the primary's normal turns %.4f and %.4f "+
				"degrees; a paraboloid turns under %v", design, ring,
				strip, smoothTurnLimit)
		}
	}
}

// TestRoundPrimaryKeepsItsFlatFaces is the anti-test: the faces
// themselves still turn across the ring and the strip edge, so the rays
// straddle both and the witness is not passing on one face.
func TestRoundPrimaryKeepsItsFlatFaces(test *testing.T) {
	for _, design := range []string{"hubble", "spitzer"} {
		_, _, ring, strip := primaryTurns(test, design)
		if !(ring > ringTurnLeast && strip > dishStripTurnLeast) {
			test.Fatalf("%s: the faces turn %.4f across the ring and "+
				"%.4f across the strip; the rays straddle neither",
				design, ring, strip)
		}
	}
}

// primaryTurns is how far the primary's shading normal turns across a
// ring and across a strip edge, then how far its faces' own normals do,
// in degrees, each between two rays straight down onto the dish.
func primaryTurns(test *testing.T, design string) (float64, float64,
	float64, float64,
) {
	test.Helper()
	_, mirror := spacecraftMeshes(test, design)
	box := mirror.Bounds()
	radius := (box.Max.X - box.Min.X) * housingHalf
	// a ring's edge is a chord between corners on the ring, which sags
	// inside it, a millimetre at the middle of a strip; beside a corner,
	// at the angle the strip rays take, it sags a millionth of that
	ringShading, ringFaces := dishTurn(test, mirror, [2]dishPlace{
		{out: radius * ringShare * (housingFar - stripNudge),
			angle: stripNudge},
		{out: radius * ringShare * (housingFar + stripNudge),
			angle: stripNudge}})
	stripShading, stripFaces := dishTurn(test, mirror, [2]dishPlace{
		{out: radius * stripShare, angle: -stripNudge},
		{out: radius * stripShare, angle: stripNudge}})
	return ringShading, stripShading, ringFaces, stripFaces
}

// dishPlace is where a ray falls on the dish: how far out from its axis,
// and at what angle round it, in radians.
type dishPlace struct{ out, angle float64 }

// dishTurn is how far the shading normal and the faces' own normals
// turn between two rays straight down onto the dish at two places.
func dishTurn(test *testing.T, mirror *mesh.Mesh,
	places [2]dishPlace,
) (float64, float64) {
	test.Helper()
	box := mirror.Bounds()
	middle := box.Min.Add(box.Max).Scale(housingHalf)
	var shading, faces [2]maths.Vec
	for index, place := range places {
		at := maths.V(middle.X+place.out*math.Cos(place.angle),
			box.Max.Y+housingAperture,
			middle.Z-place.out*math.Sin(place.angle))
		ray := maths.Ray{At: at,
			Dir: maths.V(housingLevel, -housingFar, housingLevel)}
		met, hit := mirror.Meet(ray, housingNear, math.Inf(housingFar))
		if !hit {
			test.Fatalf("a ray down missed the dish at %v", place)
		}
		shading[index] = mirror.NormalAt(met)
		one, two, three := mirror.Corners(met.Face())
		faces[index] = two.Sub(one).Cross(three.Sub(one)).Unit()
	}
	return between(shading), between(faces)
}
