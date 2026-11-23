package manufacture

import (
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func TestHumboldtSquidThreeCorrespondence(t *testing.T) {
	a, err := HumboldtSquidThree(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := HumboldtSquidThree(math.Pi / 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) || len(a) < 14 {
		t.Fatalf("pieces: %d %d", len(a), len(b))
	}
	max := 0.
	meshes := 0
	for i := range a {
		x, ok := a[i].Shape.(*mesh.Mesh)
		if !ok {
			continue
		}
		y, ok := b[i].Shape.(*mesh.Mesh)
		if !ok {
			t.Fatal("mesh type changed")
		}
		p, q := x.Points(), y.Points()
		if len(p) != len(q) || x.FaceCount() != y.FaceCount() {
			t.Fatal("lost correspondence")
		}
		meshes++
		for j := range p {
			max = math.Max(max, p[j].Sub(q[j]).Len())
		}
	}
	if max < .025 {
		t.Fatalf("no visible movement: %.5f", max)
	}
	t.Logf("%d indexed mesh pieces, maximum movement %.5f m", meshes, max)
}
