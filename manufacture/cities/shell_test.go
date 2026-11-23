package cities

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestAShellThatLeansFarFromItsMiddleIsStillAClosedSolidWoundOutward(
	t *testing.T) {
	wide := []maths.Vec{maths.V(-0.2, 0, 3), maths.V(0.2, 0, 3),
		maths.V(0.2, 0, 3.02), maths.V(-0.2, 0, 3.02)}
	narrow := []maths.Vec{maths.V(-0.04, 1.2, 0.7), maths.V(0.04, 1.2, 0.7),
		maths.V(0.04, 1.2, 0.72), maths.V(-0.04, 1.2, 0.72)}
	mesh, err := shell{Bottom: wide, Top: narrow}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	if !closed(mesh) || volume(mesh) <= 0 {
		t.Errorf("a leaning shell is not closed, or is turned inside out: "+
			"volume %v", volume(mesh))
	}
}

func TestAShellOfRingsThatDoNotMatchIsRefused(t *testing.T) {
	three := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 0, 1)}
	four := append([]maths.Vec{maths.V(1, 0, 1)}, three...)
	if _, err := (shell{Bottom: three, Top: four}).Mesh("quick"); err == nil {
		t.Error("a shell of a triangle and a square was made")
	}
	if _, err := (shell{Bottom: three[:2], Top: three[:2]}).Mesh("quick"); err == nil {
		t.Error("a shell of two corners was made")
	}
}
