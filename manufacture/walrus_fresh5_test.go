package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// testWalrusMesh is a part's points and faces and whether it is closed,
// failing the test if the part is not a mesh.
func testWalrusMesh(t *testing.T, part thing.Thing) ([]maths.Vec, [][3]int, bool) {
	t.Helper()
	switch made := part.Shape.(type) {
	case *mesh.Mesh:
		return made.Points(), made.Faces(), made.Closed()
	case *shape.Mesh:
		return made.Vertices, made.Faces, made.Closed()
	}
	t.Fatalf("part %q is not a mesh", part.Material.Name)
	return nil, nil, false
}

// testWalrusFresh5 is the walrus at yaw and phase, failing the test on an
// error.
func testWalrusFresh5(t *testing.T, yaw, phase float64) []thing.Thing {
	t.Helper()
	parts, err := WalrusFresh5(yaw, phase)
	if err != nil {
		t.Fatal(err)
	}
	return parts
}

// testWalrusConnected is how many of the vertices faces name can be
// reached from the first along edges, and how many faces name.
func testWalrusConnected(points []maths.Vec, faces [][3]int) (int, int) {
	neighbours := make([][]int, len(points))
	for _, face := range faces {
		for i := 0; i < 3; i++ {
			u, v := face[i], face[(i+1)%3]
			neighbours[u] = append(neighbours[u], v)
			neighbours[v] = append(neighbours[v], u)
		}
	}
	seen := make([]bool, len(points))
	todo := []int{faces[0][0]}
	seen[faces[0][0]] = true
	reached := 0
	for len(todo) > 0 {
		u := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		reached++
		for _, v := range neighbours[u] {
			if !seen[v] {
				seen[v] = true
				todo = append(todo, v)
			}
		}
	}
	named := 0
	for _, each := range neighbours {
		if len(each) > 0 {
			named++
		}
	}
	return reached, named
}

func TestWalrusFresh5Construction(t *testing.T) {
	for _, p := range []struct{ yaw, phase float64 }{{0, 0}, {.8, 0}, {.28, math.Pi / 2}} {
		parts := testWalrusFresh5(t, p.yaw, p.phase)
		if len(parts) != walrusPartIndex(1, walrusHeadPartCount) {
			t.Fatalf("%d parts", len(parts))
		}
	}
}

func TestWalrusFresh5ConnectedAndMotion(t *testing.T) {
	rest := testWalrusFresh5(t, 0, 0)
	swim := testWalrusFresh5(t, .28, math.Pi/2)
	loop := testWalrusFresh5(t, 0, 2*math.Pi)
	a, faces, closed := testWalrusMesh(t, rest[walrusHidePart])
	c, _, swimClosed := testWalrusMesh(t, swim[walrusHidePart])
	d, _, _ := testWalrusMesh(t, loop[walrusHidePart])
	if !closed || !swimClosed {
		t.Fatal("open hide")
	}
	if reached, named := testWalrusConnected(a, faces); reached != named {
		t.Fatalf("connected %d of %d face-named vertices", reached, named)
	}
	moved, back := 0., 0.
	for i := range a {
		moved = math.Max(moved, c[i].Sub(a[i]).Len())
		back = math.Max(back, d[i].Sub(a[i]).Len())
	}
	if !(moved >= .05) || !(back <= 1e-8) {
		t.Fatalf("moved %f, a whole stroke comes back %g away", moved, back)
	}
}

func TestWalrusFresh5EyesAndNostrilsAreOnTheHide(t *testing.T) {
	for _, yaw := range []float64{0, .8} {
		parts := testWalrusFresh5(t, yaw, 0)
		hide, _, _ := testWalrusMesh(t, parts[walrusHidePart])
		for side := 0; side < 2; side++ {
			for _, part := range []int{walrusEyePart, walrusNostrilPart} {
				at := parts[walrusPartIndex(side, part)].Motion.At(0).Origin
				nearest := math.Inf(1)
				for _, v := range hide {
					nearest = math.Min(nearest, v.Sub(at).Len())
				}
				if !(nearest <= .05) {
					t.Fatalf("yaw %g side %d part %d is %.3f m off the hide",
						yaw, side, part, nearest)
				}
			}
		}
	}
}

func TestWalrusLandmarkOffTheHideIsRefused(t *testing.T) {
	outside := func(maths.Vec) float64 { return 1 }
	if _, err := walrusEyeAt(outside, 1); !errors.Is(err, errWalrusLandmark) {
		t.Fatalf("an eye with no hide gave %v", err)
	}
	if _, err := walrusNostrilAt(outside, -1); !errors.Is(err, errWalrusLandmark) {
		t.Fatalf("a nostril with no hide gave %v", err)
	}
	if _, err := walrusHeadParts(outside, func(p maths.Vec) maths.Vec { return p },
		1); !errors.Is(err, errWalrusLandmark) {
		t.Fatalf("head parts with no hide gave %v", err)
	}
}

func TestWalrusFresh5TuskIsItsLength(t *testing.T) {
	parts := testWalrusFresh5(t, .8, 0)
	points, _, _ := testWalrusMesh(t, parts[walrusPartIndex(0, walrusTuskPart)])
	length := 0.
	var previous maths.Vec
	for ring := 0; ring <= walrusTuskSections; ring++ {
		centre := maths.Vec{}
		for j := 0; j < walrusTuskRadial; j++ {
			centre = centre.Add(points[ring*walrusTuskRadial+j])
		}
		centre = centre.Scale(1. / walrusTuskRadial)
		if ring > 0 {
			length += centre.Sub(previous).Len()
		}
		previous = centre
	}
	if !(math.Abs(length-walrusTuskLength) <= .02*walrusTuskLength) {
		t.Fatalf("posed tusk %.3f m along its rings, want %.3f",
			length, walrusTuskLength)
	}
}
