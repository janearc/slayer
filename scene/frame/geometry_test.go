package frame_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
)

type cardinal struct {
	name                string
	latitude, longitude float64
	point, nadir        maths.Vec
	east, north         maths.Vec
}

// These points are the six axes of a radius-ten sphere whose centre is
// (0,-10,0), including both signed-zero antipodal representations. Pole
// longitude follows the documented convention: the anchor's meridian, zero.
// They are geometric fixtures, not positions captured from code.
var cardinals = []cardinal{
	{"anchor", 0, 0, maths.V(0, 0, 0), maths.V(0, -1, 0),
		maths.V(1, 0, 0), maths.V(0, 0, -1)},
	{"east", 0, math.Pi / 2, maths.V(10, -10, 0), maths.V(-1, 0, 0),
		maths.V(0, -1, 0), maths.V(0, 0, -1)},
	{"west", 0, -math.Pi / 2, maths.V(-10, -10, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0), maths.V(0, 0, -1)},
	{"north", math.Pi / 2, 0, maths.V(0, -10, -10), maths.V(0, 0, 1),
		maths.V(1, 0, 0), maths.V(0, -1, 0)},
	{"south", -math.Pi / 2, 0, maths.V(0, -10, 10), maths.V(0, 0, -1),
		maths.V(1, 0, 0), maths.V(0, 1, 0)},
	{"opposite", 0, math.Pi, maths.V(0, -20, 0), maths.V(0, 1, 0),
		maths.V(-1, 0, 0), maths.V(0, 0, -1)},
	{"opposite negative zero", 0, math.Pi,
		maths.V(math.Copysign(0, -1), -20, 0), maths.V(0, 1, 0),
		maths.V(-1, 0, 0), maths.V(0, 0, -1)},
}

func location(latitude, longitude, height float64) frame.Location {
	return frame.Location{Latitude: measure.Radians(latitude),
		Longitude: measure.Radians(longitude), Height: measure.Metres(height)}
}

func sphereAt(t *testing.T, radius float64,
	anchor frame.Location,
) frame.Sphere {
	t.Helper()
	sphere, err := frame.NewSphere(measure.Metres(radius), anchor)
	if err != nil {
		t.Fatal(err)
	}
	return sphere
}

func sameVector(t *testing.T, got, want maths.Vec) {
	t.Helper()
	if !(got.Sub(want).Len() <= 1e-12) {
		t.Fatalf("vector %v, want independent fixture %v", got, want)
	}
}

func TestSphericalCardinalsHaveTheAuthoredPositions(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	for _, row := range cardinals {
		t.Run(row.name, func(t *testing.T) {
			point, err := sphere.Position(location(row.latitude, row.longitude, 0))
			if err != nil {
				t.Fatal(err)
			}
			sameVector(t, point, row.point)
		})
	}
}

func TestIndependentCardinalsGiveExpectedGeography(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	for _, row := range cardinals {
		t.Run(row.name, func(t *testing.T) {
			got, err := sphere.Location(row.point)
			if err != nil {
				t.Fatal(err)
			}
			longitude := got.Longitude.InRadians()
			delta := math.Remainder(longitude-row.longitude, 2*math.Pi)
			if !(math.Abs(got.Latitude.InRadians()-row.latitude) <= 1e-13 &&
				math.Abs(longitude) <= math.Pi && math.Abs(delta) <= 1e-13 &&
				math.Abs(got.Height.InMetres()) <= 1e-12) {
				t.Fatalf("wrong geography for %v: %v", row.point, got)
			}
		})
	}
}

func TestNadirAndTangentFollowTheRadialAxes(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	for _, row := range cardinals {
		t.Run(row.name, func(t *testing.T) {
			nadir, err := sphere.Nadir(row.point)
			if err != nil {
				t.Fatal(err)
			}
			sameVector(t, nadir, row.nadir)
			basis, err := sphere.Tangent(row.point)
			if err != nil {
				t.Fatal(err)
			}
			sameVector(t, basis.X, row.east)
			sameVector(t, basis.Y, row.nadir.Neg())
			sameVector(t, basis.Z, row.north.Neg())
		})
	}
}

func TestCompassUsesTheLocalTangent(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	for _, row := range cardinals {
		t.Run(row.name, func(t *testing.T) {
			for _, direction := range []struct {
				azimuth, elevation float64
				want               maths.Vec
			}{{0, 0, row.north}, {math.Pi / 2, 0, row.east},
				{0, math.Pi / 2, row.nadir.Neg()}} {
				got, err := sphere.Compass(row.point,
					measure.Radians(direction.azimuth),
					measure.Radians(direction.elevation))
				if err != nil {
					t.Fatal(err)
				}
				sameVector(t, got, direction.want)
			}
		})
	}
}

func TestAQuarterTurnAcrossTheDatelineRemainsEast(t *testing.T) {
	sphere := sphereAt(t, 10, location(0, 3*math.Pi/4, 0))
	point, err := sphere.Position(location(0, -3*math.Pi/4, 0))
	if err != nil {
		t.Fatal(err)
	}
	sameVector(t, point, maths.V(10, -10, 0))
}

func TestAPolarAnchorRetainsItsHeightAndRotatesTheFrame(t *testing.T) {
	sphere := sphereAt(t, 10, location(math.Pi/2, 0, 7))
	sameVector(t, sphere.Centre(), maths.V(0, -17, 0))
	point, err := sphere.Position(location(0, math.Pi/2, 0))
	if err != nil {
		t.Fatal(err)
	}
	sameVector(t, point, maths.V(10, -17, 0))
	nadir, err := sphere.Nadir(maths.V(10, -17, 0))
	if err != nil {
		t.Fatal(err)
	}
	sameVector(t, nadir, maths.V(-1, 0, 0))
}

func TestSmallHeightSurvivesALargeRadius(t *testing.T) {
	// This arbitrary large sphere is a numerical fixture, not a planet.
	// Forming R+h first loses these heights; local differences must survive.
	sphere := sphereAt(t, 1e20, location(0, 0, 1.5))
	point, err := sphere.Position(location(0, 0, 1.625))
	if err != nil {
		t.Fatal(err)
	}
	sameVector(t, point, maths.V(0, 0.125, 0))
	got, err := sphere.Location(maths.V(0, 0.125, 0))
	if err != nil || got.Height.InMetres() != 1.625 {
		t.Fatalf("lost the independently given height: %v, %v", got, err)
	}
}

func TestRenderTranslationKeepsSmallOffsetsInEveryAxis(t *testing.T) {
	origin := maths.V(1<<40, -(1 << 39), 1<<38)
	relative, err := frame.NewRelative(origin)
	if err != nil {
		t.Fatal(err)
	}
	world := maths.V((1<<40)+0.125, -(1<<39)-0.5, (1<<38)+2.75)
	local, err := relative.Point(world)
	if err != nil {
		t.Fatal(err)
	}
	sameVector(t, local, maths.V(0.125, -0.5, 2.75))
	back, err := relative.Restore(maths.V(0.125, -0.5, 2.75))
	if err != nil || back != world || relative.Origin() != origin {
		t.Fatalf("wrong restored position or origin: %v, %v", back, err)
	}
}

func TestRelativePlacementPreservesAllPoseFields(t *testing.T) {
	relative, err := frame.NewRelative(maths.V(100, 200, 300))
	if err != nil {
		t.Fatal(err)
	}
	basis := maths.Orientation{X: maths.V(0, 1, 0),
		Y: maths.V(-1, 0, 0), Z: maths.V(0, 0, 1)}
	pose, err := relative.Placement(maths.Subspace{
		Origin: maths.V(101, 198, 303), Orientation: basis, Scale: 3})
	if err != nil || pose.Orientation != basis || pose.Scale != 3 {
		t.Fatalf("placement changed orientation or scale: %v, %v", pose, err)
	}
	sameVector(t, pose.Origin, maths.V(1, -2, 3))
}

func TestNonfinitePointsCannotEnterOrLeaveARelativeFrame(t *testing.T) {
	relative, err := frame.NewRelative(maths.Vec{})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []maths.Vec{maths.V(math.NaN(), 0, 0),
		maths.V(0, math.Inf(1), 0), maths.V(0, 0, math.Inf(-1))} {
		if _, err := frame.NewRelative(bad); err == nil {
			t.Fatal("accepted a non-finite origin", bad)
		}
		if _, err := relative.Point(bad); err == nil {
			t.Fatal("accepted a non-finite world point", bad)
		}
		if _, err := relative.Restore(bad); err == nil {
			t.Fatal("accepted a non-finite local point", bad)
		}
	}
}

func TestRelativeArithmeticRejectsFiniteOverflow(t *testing.T) {
	relative, err := frame.NewRelative(maths.V(math.MaxFloat64, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := relative.Point(maths.V(-math.MaxFloat64, 0, 0)); err == nil {
		t.Fatal("accepted overflowing subtraction")
	}
	if _, err := relative.Restore(maths.V(math.MaxFloat64, 0, 0)); err == nil {
		t.Fatal("accepted overflowing restoration")
	}
}

func TestTheCentreHasNoRadialDirection(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	if _, err := sphere.Nadir(maths.V(0, -10, 0)); err == nil {
		t.Fatal("the centre has a nadir")
	}
	if _, err := sphere.Tangent(maths.V(0, -10, 0)); err == nil {
		t.Fatal("the centre has a tangent")
	}
	if _, err := sphere.Location(maths.V(0, -10, 0)); err == nil {
		t.Fatal("the centre has a surface location")
	}
}

func TestInvalidGeographyCannotBecomeAReferenceOrPoint(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	for _, bad := range []frame.Location{location(math.Pi, 0, 0),
		location(0, 2*math.Pi, 0), location(0, 0, -10),
		location(0, 0, math.NaN())} {
		if _, err := sphere.Position(bad); err == nil {
			t.Fatal("accepted invalid point geography", bad)
		}
		if _, err := frame.NewSphere(measure.Metres(10), bad); err == nil {
			t.Fatal("accepted invalid anchor geography", bad)
		}
	}
}

func TestAnInvalidReferenceCannotMapPositions(t *testing.T) {
	for _, radius := range []float64{0, -1, math.Inf(1), math.NaN()} {
		if _, err := frame.NewSphere(measure.Metres(radius),
			frame.Location{}); err == nil {
			t.Fatal("accepted invalid radius", radius)
		}
	}
	if _, err := (frame.Sphere{}).Position(frame.Location{}); err == nil {
		t.Fatal("the unchecked zero sphere mapped a position")
	}
}
