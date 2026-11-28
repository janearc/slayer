package mesh

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestPickerMatchesProducerHitAcrossFaceBlocks(t *testing.T) {
	var points []maths.Vec
	var faces [][triangleCorners]int
	for i := range 24 {
		first := len(points)
		shift := maths.V(float64(i%4)*2, 0, -float64(i/4)*2)
		for _, corner := range cubeCorners() {
			points = append(points, corner.Add(shift))
		}
		for _, face := range cubeFaces() {
			faces = append(faces, [triangleCorners]int{first + face[0], first + face[1], first + face[2]})
		}
	}
	made := mustMesh(t, points, faces, nil, false)
	picker := made.NewPicker()
	if len(picker.blocks) < 2 {
		t.Fatal("fixture did not span multiple picker bounds")
	}
	var picks []float64
	var found []bool
	for _, x := range []float64{0, .5, 1, 2, 6.5, 7} {
		for _, clip := range [][2]float64{{0, math.Inf(1)}, {1.5, 5}, {2.5, 5}, {0, 1.5}, {11, 20}} {
			ray := maths.NewRay(maths.V(x, 0, 2), maths.ZAxis.Neg())
			distance, hit := picker.Hit(ray, clip[0], clip[1])
			picks, found = append(picks, distance), append(found, hit)
		}
	}
	if len(made.tree.nodes) != 0 {
		t.Fatal("working picks built the producer's render tree")
	}
	i := 0
	for _, x := range []float64{0, .5, 1, 2, 6.5, 7} {
		for _, clip := range [][2]float64{{0, math.Inf(1)}, {1.5, 5}, {2.5, 5}, {0, 1.5}, {11, 20}} {
			ray := maths.NewRay(maths.V(x, 0, 2), maths.ZAxis.Neg())
			distance, hit := made.Hit(ray, clip[0], clip[1])
			if picks[i] != distance || found[i] != hit {
				t.Fatalf("x %g, clip %v: cached (%g, %t), producer (%g, %t)", x, clip, picks[i], found[i], distance, hit)
			}
			i++
		}
	}
}
