package frame_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
)

// TestSphereReportsItsReferenceAndAnchor checks that the radius and anchor
// given to NewSphere come back as given.
func TestSphereReportsItsReferenceAndAnchor(t *testing.T) {
	anchor := location(0.5, -1.25, 30)
	sphere := sphereAt(t, 6.4e6, anchor)
	if got := sphere.Radius().InMetres(); got != 6.4e6 {
		t.Errorf("radius %v, want 6.4e6", got)
	}
	got := sphere.Anchor()
	if got.Latitude.InRadians() != 0.5 || got.Longitude.InRadians() != -1.25 ||
		math.Abs(got.Height.InMetres()-30) > 30*1e-12 {
		t.Errorf("anchor %+v, want latitude 0.5, longitude -1.25, height 30", got)
	}
}

// TestCompassPointsAlongTheLocalHorizon checks directions at the anchor of a
// sphere on the equator: north is minus z, east is plus x and straight up is
// plus y.
func TestCompassPointsAlongTheLocalHorizon(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	here := maths.V(0, 0, 0)
	cases := []struct {
		name               string
		azimuth, elevation float64
		want               maths.Vec
	}{
		{"north", 0, 0, maths.V(0, 0, -1)},
		{"east", math.Pi / 2, 0, maths.V(1, 0, 0)},
		{"south", math.Pi, 0, maths.V(0, 0, 1)},
		{"up", 0, math.Pi / 2, maths.V(0, 1, 0)},
		{"down", 0, -math.Pi / 2, maths.V(0, -1, 0)},
	}
	for _, c := range cases {
		got, err := sphere.Compass(here, measure.Radians(c.azimuth), measure.Radians(c.elevation))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !(got.Sub(c.want).Len() <= 1e-12) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestCompassRefusesBadDirectionsAndPoints checks that an azimuth or
// elevation that is not finite, an elevation past the zenith or nadir, and a
// point with no radial direction are each refused.
func TestCompassRefusesBadDirectionsAndPoints(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	here := maths.V(0, 0, 0)
	cases := map[string]func() error{
		"azimuth not a number": func() error {
			azimuth := measure.Radians(math.NaN())
			_, err := sphere.Compass(here, azimuth, measure.Radians(0))
			return err
		},
		"elevation infinite": func() error {
			elevation := measure.Radians(math.Inf(1))
			_, err := sphere.Compass(here, measure.Radians(0), elevation)
			return err
		},
		"past the zenith": func() error {
			_, err := sphere.Compass(here, measure.Radians(0), measure.Radians(math.Pi/2+0.01))
			return err
		},
		"past the nadir": func() error {
			_, err := sphere.Compass(here, measure.Radians(0), measure.Radians(-math.Pi/2-0.01))
			return err
		},
		"at the centre": func() error {
			_, err := sphere.Compass(maths.V(0, -10, 0), measure.Radians(0), measure.Radians(0))
			return err
		},
	}
	for name, call := range cases {
		if err := call(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// TestRadialOperationsRefuseBadSpheresAndPoints checks that the zero sphere
// and a point that is not finite are refused by Nadir, Tangent and Location.
func TestRadialOperationsRefuseBadSpheresAndPoints(t *testing.T) {
	good := sphereAt(t, 10, frame.Location{})
	bad := maths.V(math.NaN(), 0, 0)
	for name, call := range map[string]func() error{
		"zero sphere nadir":    func() error { _, err := (frame.Sphere{}).Nadir(maths.V(1, 1, 1)); return err },
		"zero sphere tangent":  func() error { _, err := (frame.Sphere{}).Tangent(maths.V(1, 1, 1)); return err },
		"zero sphere location": func() error { _, err := (frame.Sphere{}).Location(maths.V(1, 1, 1)); return err },
		"nadir of NaN":         func() error { _, err := good.Nadir(bad); return err },
		"tangent of NaN":       func() error { _, err := good.Tangent(bad); return err },
		"location of NaN":      func() error { _, err := good.Location(bad); return err },
	} {
		if err := call(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// TestLocationFarFromTheSphereStillGivesAnAngle checks that a point so far out
// that its radius and vertical offset cannot be added is still placed on the
// anchor's meridian and equator, straight above the anchor, and is not
// refused. The height is too large for a length and saturates.
func TestLocationFarFromTheSphereStillGivesAnAngle(t *testing.T) {
	sphere := sphereAt(t, 10, frame.Location{})
	got, err := sphere.Location(maths.V(0, 0.9e308, 0))
	if err != nil {
		t.Fatal(err)
	}
	if got.Latitude.InRadians() != 0 || got.Longitude.InRadians() != 0 {
		t.Errorf("angles %v, %v; want 0, 0", got.Latitude.InRadians(), got.Longitude.InRadians())
	}
	if !(got.Height.InMetres() > 1e300) {
		t.Errorf("height %v, want over 1e300", got.Height.InMetres())
	}
}

// TestPlacementRefusesABadOrigin checks that a pose whose origin is not
// finite is refused.
func TestPlacementRefusesABadOrigin(t *testing.T) {
	relative, err := frame.NewRelative(maths.Vec{})
	if err != nil {
		t.Fatal(err)
	}
	infinite := maths.V(0, math.Inf(1), 0)
	pose, err := relative.Placement(maths.Subspace{Origin: infinite, Scale: 1})
	if err == nil || pose != (maths.Subspace{}) {
		t.Errorf("got %v, %v", pose, err)
	}
}
