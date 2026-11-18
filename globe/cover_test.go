package globe

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestGlobeSaysItsRadiusAndBounds checks a globe of radius 2 reports 2 and the cube from -2 to 2.
func TestGlobeSaysItsRadiusAndBounds(t *testing.T) {
	ball, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	if ball.Radius() != 2 {
		t.Errorf("radius %v, want 2", ball.Radius())
	}
	want := maths.Box{Min: maths.V(-2, -2, -2), Max: maths.V(2, 2, 2)}
	if got := ball.Bounds(); got != want {
		t.Errorf("bounds %v, want %v", got, want)
	}
}

// TestARayWhoseRootsLieOutsideTheRangeMisses checks a ray that crosses the globe at distances 8 and 12 misses when asked only about 0 to 5, and hits at 8 when the range is widened.
func TestARayWhoseRootsLieOutsideTheRangeMisses(t *testing.T) {
	ball, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{At: maths.V(-10, 0, 0), Dir: maths.V(1, 0, 0)}
	if root, hit := ball.Hit(ray, 0, 5); hit || root != 0 {
		t.Errorf("hit=%v root=%v inside a range that holds neither root", hit, root)
	}
	if root, hit := ball.Hit(ray, 0, 20); !hit || root != 8 {
		t.Errorf("hit=%v root=%v, want the entry at 8", hit, root)
	}
	if root, hit := ball.Hit(ray, 9, 20); !hit || root != 12 {
		t.Errorf("hit=%v root=%v, want the exit at 12 when the entry is nearer than the range", hit, root)
	}
}
