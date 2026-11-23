package mesh

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestAMeshWithRestPointsIsPosed(t *testing.T) {
	corners := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)}
	made, err := NewWithRestPoints(corners, corners, [][3]int{{0, 1, 2}},
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !made.Posed() {
		t.Fatal("a mesh with rest points is not posed")
	}
}

// Anti-test: a mesh made without rest points is not posed.

func TestAMeshWithoutRestPointsIsNotPosed(t *testing.T) {
	made, err := New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if made.Posed() {
		t.Fatal("a mesh without rest points is posed")
	}
}
