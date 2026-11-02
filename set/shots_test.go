package set

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// at puts every asset at the origin, the way the shoebox stands.
func at(asset string) (maths.Vec, bool) {
	return maths.Vec{}, asset == "the box"
}

// The shoebox camera stands 10 m east and 10 m north of the box, at its
// foot, faces it across the ground and tips up 25 degrees.
func TestShoeboxCameraFacesTheBox(t *testing.T) {
	source, err := os.ReadFile("testdata/shoebox.yaml")
	if err != nil {
		t.Fatal(err)
	}
	read, err := Read("shoebox.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	shots, err := read.Shots(at)
	if err != nil {
		t.Fatal(err)
	}
	if len(shots) != 1 || shots[0].Name() != "shoot_1" {
		t.Fatalf("shots = %v, want the one camera, shoot_1", shots)
	}
	eye := shots[0].Eye()
	if want := maths.V(10, 0, -10); !eye.Origin.Near(want, 1e-9) {
		t.Errorf("eye stands at %v, want %v", eye.Origin, want)
	}
	tip := 25 * math.Pi / 180
	toward := maths.V(-1, 0, 1).Unit().Scale(math.Cos(tip))
	want := toward.Add(maths.V(0, math.Sin(tip), 0))
	if got := eye.Forward(); !got.Near(want, 1e-9) {
		t.Errorf("eye faces %v, want %v", got, want)
	}
}

// A camera says which asset it cannot find, and a camera straight above
// what it points to says it has no heading.
func TestShotsSayWhatIsWrong(t *testing.T) {
	missing := camera{name: "far", from: "the shed", pointTo: "the box"}
	if _, err := (Set{cameras: []camera{missing}}).Shots(at); err == nil ||
		!strings.Contains(err.Error(), "the shed") {
		t.Errorf("a camera off a missing asset: %v", err)
	}
	above := camera{name: "drone", from: "the box", pointTo: "the box",
		standoff: [3]float64{0, 0, 50}, alpha: -90}
	if _, err := (Set{cameras: []camera{above}}).Shots(at); err == nil ||
		!strings.Contains(err.Error(), "no heading") {
		t.Errorf("a camera straight above: %v", err)
	}
}
