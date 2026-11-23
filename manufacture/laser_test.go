package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// testLaser is a beam from the origin straight up, a metre across at its
// waist, a kilometre long and spreading a degree.
func testLaser() Laser {
	return Laser{At: maths.Vec{}, Toward: maths.YAxis,
		Radius: measure.Metres(1), Reach: measure.Metres(1000),
		Divergence: measure.Degrees(1)}
}

func TestALasersBoxReachesTheFarEndAndAThousandthOfItsProfileOut(t *testing.T) {
	box, err := testLaser().Extent()
	if err != nil {
		t.Fatal(err)
	}
	width := 1 + 1000*math.Tan(measure.Degrees(1).InRadians())
	edge := math.Sqrt(math.Log(1000)/2) * width
	if math.Abs(box.Max.Y-(1000+edge)) > 1e-6 ||
		math.Abs(box.Max.X-edge) > 1e-6 || math.Abs(box.Min.Y+edge) > 1e-6 {
		t.Errorf("a laser's box %+v, not %v wide of a kilometre", box, edge)
	}
}

func TestALaserWithNoReachOrNoWaistHasNoBox(t *testing.T) {
	none := testLaser()
	none.Reach = measure.Metres(0)
	thin := testLaser()
	thin.Radius = measure.Metres(0)
	for name, laser := range map[string]Laser{"no reach": none,
		"no waist": thin} {
		if _, err := laser.Extent(); err == nil {
			t.Errorf("a laser with %s was given a box", name)
		}
	}
}

func TestALaserAheadOfAnEyeIsSeenAndOneBehindIsNot(t *testing.T) {
	ahead := maths.Subspace{Origin: maths.V(0, 0, -50),
		Orientation: maths.Identity, Scale: 1}
	got, err := testLaser().seen(sow(0), ahead, lookingDown())
	if err != nil || !got.Visible {
		t.Errorf("a laser ahead is %+v (%v)", got, err)
	}
	behind := maths.Subspace{Origin: maths.V(0, 0, 5000),
		Orientation: maths.Identity, Scale: 1}
	got, err = testLaser().seen(sow(0), behind, lookingDown())
	if err != nil || got.Visible {
		t.Errorf("a laser behind is %+v (%v)", got, err)
	}
}
