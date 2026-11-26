package geometry

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// Fish, birds and orcas normalize the area-weighted sum and retain unused
// zero vertices. Seals retain the sum and give unused vertices an up normal.
func TestAnimalSurfaceNormalsKeepAreaAndUnusedVertexContracts(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(2, 0, 0), maths.V(0, 2, 0), maths.V(0, 0, 1), maths.V(4, 4, 4)}
	faces := [][3]int{{0, 1, 2}, {0, 3, 1}}
	seal := SmoothNormals(points, faces)
	unit := UnitNormals(points, faces)
	want := maths.V(0, 2, 4)
	if seal[0] != want {
		t.Fatalf("seal weighted sum = %v, want %v", seal[0], want)
	}
	if unit[0] != want.Unit() {
		t.Fatalf("unit normal = %v, want %v", unit[0], want.Unit())
	}
	if seal[4] != maths.Up || unit[4] != (maths.Vec{}) {
		t.Fatalf("unused vertex: seal %v unit %v", seal[4], unit[4])
	}
}
