package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// The witness for the black sawtooth inside the Hubble and Spitzer tubes
// (heard v6, frames 42 and 46). The housing is polished; each of
// its 96 flat strips mirrored the world at its own angle, so a shadow's
// edge seen in the wall broke into teeth, one a strip. Its normals now
// stand for the round tube, and turn smoothly from strip to strip.

// housingAperture is the aperture the tube is made for, in metres:
// the heard shoot's optical-aperture-metres.
const housingAperture = 8 // picked: as the frames that showed it

// housingSeed is the seed the spacecraft is sown from.
const housingSeed = 5 // picked: as the shoot's factory sows it

// housingDetail is the detail the frames were drawn at, 96 sides a tube.
const housingDetail = "demo" // picked: as the shoot's factory makes it

// stripNudge is how far either side of the strip edge at angle nought
// the two rays look, in radians round the tube's axis.
const stripNudge = 1e-5 // picked: far inside a strip, 0.065 radians

// smoothTurnLimit is the most the round wall's normal may turn between
// the two rays, in degrees: they are 2e-5 radians, 0.0011 degrees, apart.
const smoothTurnLimit = 0.01 // picked: ten times that, a tenth of a strip's

// stripTurnLeast is the least the faces' own normals must turn between
// the two rays, in degrees, for the rays to be either side of an edge:
// a 96-sided tube's strips meet at 3.75 degrees.
const stripTurnLeast = 1 // picked: well under 3.75 and well over nought

// housingLevel is defined as the y component of a ray kept level.
const housingLevel = 0

// housingHalf is defined as half, for the middle of a box.
const housingHalf = 0.5

// housingNear and housingFar are defined as the whole of a ray.
const (
	housingNear = 0
	housingFar  = 1
)

// rimStep is how far out from the axis, over the inner wall's distance,
// a ray down onto the rim falls: into the rim's top, which is the wall's
// thickness wide, 0.28 m of a 5.5 to 7.7 m radius at this aperture.
const rimStep = 1.02 // picked: 0.11 to 0.15 m out, inside the rim

// rimNormalTolerance is how far from straight up the rim's normal may be.
const rimNormalTolerance = 1e-9 // picked: rounding, far under a strip

// TestRoundHousingTurnsSmoothlyAcrossItsStrips is the witness: either
// side of a strip edge the housing's shading normal is the round tube's,
// nearly the same. On 733bdc72 it turned by the whole 3.75 degrees, and
// a mirrored ray by twice that.
func TestRoundHousingTurnsSmoothlyAcrossItsStrips(test *testing.T) {
	for _, design := range []string{"hubble", "spitzer"} {
		shading, _ := housingTurns(test, design)
		test.Logf("%s: the shading normal turns %.5f degrees", design,
			shading)
		if !(shading < smoothTurnLimit) {
			test.Fatalf("%s: the housing's normal turns %.4f degrees "+
				"across a strip edge; a round wall turns under %v",
				design, shading, smoothTurnLimit)
		}
	}
}

// TestRoundHousingKeepsItsFlatStrips is the anti-test: the two rays do
// fall either side of an edge, where the faces themselves turn, so the
// witness is not passing on one face; and the rim stays a sharp corner,
// its top facing straight up, not blended with the wall below it.
func TestRoundHousingKeepsItsFlatStrips(test *testing.T) {
	for _, design := range []string{"hubble", "spitzer"} {
		_, faces := housingTurns(test, design)
		if !(faces > stripTurnLeast) {
			test.Fatalf("%s: the faces turn %.4f degrees; the rays are "+
				"not either side of a strip edge", design, faces)
		}
		housing, from := housingAt(test, design)
		top := housing.Bounds().Max.Y
		wall := housingWall(test, housing, from, housingNear)
		rim := from.Add(wall.Sub(from).Scale(rimStep))
		down := maths.Ray{At: maths.V(rim.X, top+housingAperture, rim.Z),
			Dir: maths.V(housingLevel, -housingFar, housingLevel)}
		met, hit := housing.Meet(down, housingNear, math.Inf(housingFar))
		if !hit {
			test.Fatalf("%s: a ray down onto the rim missed it", design)
		}
		up := housing.NormalAt(met)
		if math.Abs(up.Y-housingFar) > rimNormalTolerance {
			test.Fatalf("%s: the rim's normal is %v, not straight up",
				design, up)
		}
	}
}

// housingTurns is how far the housing's shading normal and its faces'
// own normals turn, in degrees, between two level rays from the tube's
// axis to its inner wall, either side of the strip edge at angle nought.
func housingTurns(test *testing.T, design string) (float64, float64) {
	test.Helper()
	housing, from := housingAt(test, design)
	var shading, faces [2]maths.Vec
	for index, angle := range []float64{-stripNudge, stripNudge} {
		ray := maths.Ray{At: from, Dir: maths.V(math.Cos(angle),
			housingLevel, -math.Sin(angle))}
		met, hit := housing.Meet(ray, housingNear, math.Inf(housingFar))
		if !hit {
			test.Fatalf("%s: a ray from the axis missed the wall", design)
		}
		shading[index] = housing.NormalAt(met)
		one, two, three := housing.Corners(met.Face())
		faces[index] = two.Sub(one).Cross(three.Sub(one)).Unit()
	}
	return between(shading), between(faces)
}

// housingAt is the made spacecraft's housing mesh and a point on the
// tube's axis an aperture above the primary mirror, inside the tube.
func housingAt(test *testing.T, design string) (*mesh.Mesh, maths.Vec) {
	test.Helper()
	housing, mirror := spacecraftMeshes(test, design)
	box := mirror.Bounds()
	from := maths.V((box.Min.X+box.Max.X)*housingHalf,
		box.Max.Y+housingAperture, (box.Min.Z+box.Max.Z)*housingHalf)
	return housing, from
}

// spacecraftMeshes is the housing and primary mirror meshes of the
// design made as the shoot makes it.
func spacecraftMeshes(test *testing.T, design string) (*mesh.Mesh,
	*mesh.Mesh,
) {
	test.Helper()
	row, err := manufacture.Spacecraft(design,
		measure.Metres(housingAperture))
	if err != nil {
		test.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(*row,
		manufacture.Sow(housingSeed), housingDetail)
	if err != nil {
		test.Fatal(err)
	}
	things, err := made.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	meshes := map[string]*mesh.Mesh{}
	for _, object := range things {
		if polygon, isMesh := object.Shape.(*mesh.Mesh); isMesh {
			meshes[object.Material.Name] = polygon
		}
	}
	housing, mirror := meshes["telescope housing"], meshes["primary mirror"]
	if housing == nil || mirror == nil {
		test.Fatalf("%s: no housing or primary mirror mesh", design)
	}
	return housing, mirror
}

// housingWall is where a level ray from the axis at an angle meets the
// housing's inner wall.
func housingWall(test *testing.T, housing *mesh.Mesh, from maths.Vec,
	angle float64,
) maths.Vec {
	test.Helper()
	ray := maths.Ray{At: from, Dir: maths.V(math.Cos(angle),
		housingLevel, -math.Sin(angle))}
	met, hit := housing.Meet(ray, housingNear, math.Inf(housingFar))
	if !hit {
		test.Fatal("a ray from the axis missed the wall")
	}
	return ray.Point(met.Along())
}

// between is the angle between two ways one long, in degrees.
func between(ways [2]maths.Vec) float64 {
	cosine := math.Max(-housingFar, math.Min(housingFar,
		ways[housingNear].Dot(ways[housingFar])))
	return math.Acos(cosine) / maths.Degree
}
