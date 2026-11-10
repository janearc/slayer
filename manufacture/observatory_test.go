package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func opticalSpec() Optical {
	return Optical{Diameter: measure.Metres(24), Aperture: measure.Metres(8),
		Opening: measure.Degrees(35), Azimuth: measure.Degrees(180),
		Altitude: measure.Degrees(55), Segmented: true}
}

func TestObservatoryMakesValidMeshesAtEveryComplexity(t *testing.T) {
	row, err := Observatory(opticalSpec())
	if err != nil {
		t.Fatal(err)
	}
	for _, detail := range Complexities() {
		made, err := MakeWithComplexity(row, Sow(5), detail)
		if err != nil {
			t.Fatal(err)
		}
		things, err := made.Things(maths.Subspace{})
		if err != nil {
			t.Fatalf("%s: %v", detail, err)
		}
		if len(things) != 3 {
			t.Errorf("%s has %d materials", detail, len(things))
		}
		if math.Abs(made.Bounds().Max.Y-16.56) > 1e-9 {
			t.Errorf("unexpected enclosure extent %v", made.Bounds())
		}
	}
}

func TestDomeHasAnOpenSlitAndAClosedShell(t *testing.T) {
	shape := domeShell{radius: 12, thick: 0.3, opening: 35 * maths.Degree}
	skin := shape.skin(cut{around: 40})
	edges := map[[2]maths.Vec]int{}
	for _, face := range skin {
		for i := range 3 {
			edges[[2]maths.Vec{face[i], face[(i+1)%3]}]++
		}
	}
	for edge, count := range edges {
		if count != 1 || edges[[2]maths.Vec{edge[1], edge[0]}] != 1 {
			t.Fatalf("open or multiply joined shell edge: %v", edge)
		}
	}
	mesh, err := skin.meshed()
	if err != nil {
		t.Fatal(err)
	}
	// Along the middle of the slit no dome is hit; beside it the shell is.
	for _, test := range []struct {
		x   float64
		hit bool
	}{{0, false}, {6, true}} {
		ray := maths.Ray{At: maths.V(test.x, 4, -20), Dir: maths.V(0, 0, 1)}
		_, hit := mesh.Hit(ray, 0, 20)
		if hit != test.hit {
			t.Errorf("x=%v hit=%v", test.x, hit)
		}
	}
}

func TestObservatoryRejectsAnInstrumentLargerThanItsEnclosure(t *testing.T) {
	spec := opticalSpec()
	spec.Aperture = measure.Metres(24)
	if _, err := Observatory(spec); err == nil {
		t.Error("accepted an oversized telescope")
	}
}
