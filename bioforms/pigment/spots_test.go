package pigment

import (
	"math"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestSpotsFollowRestCoordinatesAndSeed(t *testing.T) {
	f := Spots{Seed: 71, Spacing: .004, Radius: .00085}
	if (Spots{}).At(maths.Vec{}) != 0 {
		t.Fatal("empty field has marks")
	}
	marked, blank, different := 0, 0, 0
	for i := 0; i < 4096; i++ {
		p := maths.V(float64(i%64)*.00083, float64(i/64)*.00083, 0)
		a := f.At(p)
		if a != f.At(p) || a < 0 || a > 1 {
			t.Fatal(a)
		}
		if a > .15 {
			marked++
		}
		if a == 0 {
			blank++
		}
		other := f
		other.Seed++
		if a != other.At(p) {
			different++
		}
	}
	if marked < 5 || blank < 100 || different < 5 {
		t.Fatal(marked, blank, different)
	}
}

// A posed fin web and its rest web must carry the very same spots.
func TestSpottedFinFollowsBentRay(t *testing.T) {
	rest := []maths.Vec{maths.V(0, 0, .0023), maths.V(.04, 0, .0023), maths.V(0, .03, .0023)}
	rig := skeleton.New()
	if err := rig.Add("fin ray", "", rest[0], rest[1], .001, .0002); err != nil {
		t.Fatal(err)
	}
	frames := rig.Frames(skeleton.Pose{}.With("fin ray", skeleton.NewTurn(maths.ZAxis, .72)))
	posed := make([]maths.Vec, len(rest))
	for i, p := range rest {
		posed[i] = skeleton.Carry(p, frames, []skeleton.Influence{skeleton.NewInfluence(0, 1)})
	}
	web, err := mesh.NewWithRestPoints(posed, rest, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	spots := Spots{Seed: 918, Spacing: .004, Radius: .0016}
	witnessed := false
	for i := 1; i < 32; i++ {
		for j := 1; j < 32-i; j++ {
			a, b := float64(i)/32, float64(j)/32
			point := func(c []maths.Vec) maths.Vec { return c[0].Scale(1 - a - b).Add(c[1].Scale(a)).Add(c[2].Scale(b)) }
			q := point(posed)
			hit, ok := web.Meet(maths.Ray{At: q.Add(maths.ZAxis), Dir: maths.ZAxis.Neg()}, 0, 2)
			if !ok {
				t.Fatal("missed fin")
			}
			source, ok := web.RestPointAt(hit)
			if !ok || math.Abs(spots.At(source)-spots.At(point(rest))) > 1e-11 {
				t.Fatal("paint moved")
			}
			if math.Abs(spots.At(source)-spots.At(q)) > .5 {
				witnessed = true
			}
		}
	}
	if !witnessed {
		t.Fatal("fixture did not discriminate rest from posed paint")
	}
}
