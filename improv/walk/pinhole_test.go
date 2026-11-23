package walk

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// pinholeDir is the rigless fixture: the marker set with its film's rig
// taken off for a pinhole 45 degrees across, and a mask of each
// camera's still traced from it by the session, white where a marker
// glows. The places are the marker set's own.
var pinholeDir = filepath.Join("testdata", "pinhole")

// pinholeWide is the fixture's film's angle across, in degrees. Picked
// for the fixture: about what the marker set's 85 mm sees across its
// frame.
const pinholeWide = "    wide: 45\n"

// pinholeSet reads the rigless fixture's set as written, or with its
// film's lines changed first, and the marker set's places.
func pinholeSet(t *testing.T, from, to string) (set.Set, Places) {
	t.Helper()
	path := filepath.Join(pinholeDir, "set.yaml")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if from != "" {
		if !strings.Contains(string(source), from) {
			t.Fatalf("the fixture has no %q to change", from)
		}
		replaced := strings.Replace(string(source), from, to, 1)
		source = []byte(replaced)
	}
	read, err := set.Read(path, source)
	if err != nil {
		t.Fatal(err)
	}
	_, places := markerSet(t)
	return read, places
}

// TestRiglessMarkers: on a set whose film has no rig, only how wide it
// sees, every marker the tracer drew lies within markerPixels of a
// marker's projected pixel, and the camera makes the still's rows.
func TestRiglessMarkers(t *testing.T) {
	read, places := pinholeSet(t, "", "")
	for _, name := range []string{"down", "above"} {
		middles, size := blobs(t, filepath.Join(pinholeDir, name+".png"))
		view, err := CameraOf(read, name, places, size.X)
		if err != nil {
			t.Fatal(err)
		}
		if _, height := view.size(); height != size.Y {
			t.Fatalf("%s: the camera makes %d rows, the still has %d",
				name, height, size.Y)
		}
		if len(middles) < leastMarkers {
			t.Errorf("%s: %d markers drawn; the test sees too little",
				name, len(middles))
		}
		for _, middle := range middles {
			nearest, which := nearestMarker(view, places, middle)
			if nearest > markerPixels {
				t.Errorf("%s: a marker drawn at %.1f, %.1f is %.2f pixels "+
					"from the nearest projected, %s", name, middle[0],
					middle[1], nearest, which)
			}
		}
	}
}

// TestRiglessNearMiss: the same film said 50 degrees wide, and the same
// camera with its eye 5 m to the side, each miss the drawn markers, so
// the test can tell a wrong angle and a wrong eye.
func TestRiglessNearMiss(t *testing.T) {
	middles, size := blobs(t, filepath.Join(pinholeDir, "down.png"))
	wider, places := pinholeSet(t, pinholeWide, "    wide: 50\n")
	view, err := CameraOf(wider, "down", places, size.X)
	if err != nil {
		t.Fatal(err)
	}
	read, _ := pinholeSet(t, "", "")
	right, err := CameraOf(read, "down", places, size.X)
	if err != nil {
		t.Fatal(err)
	}
	moved := right
	moved.eye.Origin = moved.eye.Origin.Add(maths.V(0, 0, 5))
	for name, wrong := range map[string]Camera{"50 degrees": view,
		"5 m off": moved} {
		worst := 0.0
		for _, middle := range middles {
			nearest, _ := nearestMarker(wrong, places, middle)
			worst = math.Max(worst, nearest)
		}
		if worst <= markerPixels {
			t.Errorf("a camera %s lands every marker within %.2f pixels",
				name, worst)
		}
	}
}

// TestRiglessRefuses: a film with neither a rig nor a wide, and a
// picture under a pixel across, are refused by name; the rigged set
// still takes its rig, not a pinhole.
func TestRiglessRefuses(t *testing.T) {
	bare, places := pinholeSet(t, pinholeWide, "")
	if _, err := CameraOf(bare, "down", places, 320); err == nil ||
		!errors.Is(err, errWalk) || !strings.Contains(err.Error(), "wide") {
		t.Errorf("a film with no rig and no wide gave %v", err)
	}
	read, _ := pinholeSet(t, "", "")
	if _, err := CameraOf(read, "down", places, 0); err == nil ||
		!errors.Is(err, errWalk) {
		t.Errorf("a picture no pixels across gave %v", err)
	}
	rigged, _ := markerSet(t)
	view, err := CameraOf(rigged, "down", places, 320)
	if err != nil {
		t.Fatal(err)
	}
	if view.glass.pinhole {
		t.Error("a set with a rig was given a pinhole")
	}
}
