package placement

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// ringTube is a tube a metre across and two long, its wall a tenth thick.
var ringTube = shape.Tube{Radius: 0.5, Thick: 0.1, Height: 2}

func TestTubeHitterMeetsTheWall(t *testing.T) {
	ray := maths.Ray{At: maths.V(-3, 0, 0), Dir: maths.V(1, 0, 0)}
	distance, hit := tubeHitter{tube: ringTube}.Hit(ray, 0, 100)
	if !hit {
		t.Fatal("a ray at a tube's side missed it")
	}
	if want := 3 - 0.55; distance < want-0.01 || distance > want+0.01 {
		t.Fatalf("hit at %v, want near %v", distance, want)
	}
}

func TestTubeHitterMissesAboveAndBeside(t *testing.T) {
	over := maths.Ray{At: maths.V(-3, 5, 0), Dir: maths.V(1, 0, 0)}
	if _, hit := (tubeHitter{tube: ringTube}).Hit(over, 0, 100); hit {
		t.Error("a ray over the tube hit it")
	}
	beside := maths.Ray{At: maths.V(-3, 0, 4), Dir: maths.V(1, 0, 0)}
	if _, hit := (tubeHitter{tube: ringTube}).Hit(beside, 0, 100); hit {
		t.Error("a ray beside the tube hit it")
	}
}

func TestTubeHitterStopsAtFar(t *testing.T) {
	ray := maths.Ray{At: maths.V(-3, 0, 0), Dir: maths.V(1, 0, 0)}
	if _, hit := (tubeHitter{tube: ringTube}).Hit(ray, 0, 1); hit {
		t.Fatal("a hit was found beyond far")
	}
}

func TestTubeHitterRefusesNoDirection(t *testing.T) {
	ray := maths.Ray{At: maths.V(-3, 0, 0)}
	if _, hit := (tubeHitter{tube: ringTube}).Hit(ray, 0, 100); hit {
		t.Fatal("a ray with no direction hit")
	}
}

func TestDisplayHitterGivesATubeAHitterAndKeepsASphereOwn(t *testing.T) {
	if _, ok := displayHitter(ringTube); !ok {
		t.Error("a tube has no display hitter")
	}
	if _, ok := displayHitter(shape.Sphere{Radius: 1}); !ok {
		t.Error("a sphere lost its own hitter")
	}
}
