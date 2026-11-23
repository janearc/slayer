package session

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	scenes "github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/titan/over"
)

// smallWindow is a window of five samples by four, 4 m apart, its heights
// a rise eastward with a ripple, written as the bytes a window keeps: the
// west, north and step as numbers, the samples across and down as counts,
// and the heights as numbers.
func smallWindow(t *testing.T) terrain.Window {
	t.Helper()
	var raw []byte
	for _, number := range []float64{0, 0, 4} {
		raw = binary.LittleEndian.AppendUint64(raw, math.Float64bits(number))
	}
	for _, count := range []uint64{5, 4} {
		raw = binary.LittleEndian.AppendUint64(raw, count)
	}
	for row := range 4 {
		for column := range 5 {
			height := 3*float64(column) + math.Sin(float64(column*row))
			raw = binary.LittleEndian.AppendUint64(raw,
				math.Float64bits(height))
		}
	}
	window, err := terrain.WindowFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	return window
}

// A set that asks no made ground stands on the survey's own triangles:
// the same points and the same faces as the window gives, and no others.
func TestNoMadeGroundIsTheSurvey(t *testing.T) {
	window := smallWindow(t)
	stage := &staging{ground: window, lowest: window.Lowest()}
	got, err := stage.groundSurface(nil)
	if err != nil {
		t.Fatal(err)
	}
	want, err := window.GroundOn(0, 0, window.Lowest(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Points(), want.Points()) ||
		!slices.Equal(got.Faces(), want.Faces()) {
		gotPoints := len(got.Points())
		gotFaces := len(got.Faces())
		wantPoints := len(want.Points())
		wantFaces := len(want.Faces())
		t.Fatalf("%d points and %d faces, the survey has %d and %d",
			gotPoints, gotFaces, wantPoints, wantFaces)
	}
}

// The made ground's receipt follows a placed window's own sentence and
// does not take its place; with no sentence it stands alone.
func TestReceiptKeepsTheWindowsSentence(t *testing.T) {
	receipt := over.Receipt{Surveyed: 4}
	kept := withReceipt("ground on a sphere", receipt)
	if !strings.HasPrefix(kept, "ground on a sphere; made ground: 4 ") {
		t.Fatalf("the tally says %q", kept)
	}
	if alone := withReceipt("", receipt); !strings.HasPrefix(alone,
		"made ground: 4 ") {
		t.Fatalf("alone the tally says %q", alone)
	}
}

// A named scene that says its notes for the reader writes them to
// standard error, and every other writes them nowhere, as before.
func TestNamedSceneNotesGoWhereTheSceneSays(t *testing.T) {
	for _, listed := range scenes.All() {
		scene, err := scenes.Get(listed.Name())
		if err != nil {
			continue // a scene drawn as maps is built by no job
		}
		want := io.Writer(io.Discard)
		if scene.Noted() {
			want = os.Stderr
		}
		if got := notesOf(scene); got != want {
			t.Errorf("%s writes its notes to %v", listed.Name(), got)
		}
	}
}

// Anti: of every named scene, none says its notes to the reader. The one
// scene that did, built over a survey with made ground, is no longer in the
// registry, which has six scenes and none of them says its notes.
func TestNoNamedSceneIsNoted(t *testing.T) {
	var noted []string
	for _, listed := range scenes.All() {
		if scene, err := scenes.Get(listed.Name()); err == nil &&
			scene.Noted() {
			noted = append(noted, listed.Name())
		}
	}
	if len(noted) != 0 {
		t.Fatalf("named scenes noted: %v", noted)
	}
}

// lookingNorth is an eye at the origin looking north, which is the
// world's minus z, through a lens a right angle wide on a film 1000
// pixels across and 500 down.
func lookingNorth() strategies.Eye {
	place := maths.Subspace{Origin: maths.V(0, 0, 0),
		Orientation: maths.LookAtUp(maths.V(0, 0, 0), maths.V(0, 0, -1),
			maths.V(0, 1, 0))}
	return strategies.Eye{Place: place, Wide: math.Pi / 2, Across: 1000,
		Down: 500}
}

// A place straight ahead is wanted as fine as a pixel spans there: a
// right angle over 1000 pixels is 500 pixels a metre at a metre.
func TestSightWantsAPixel(t *testing.T) {
	eye := sightOf(lookingNorth(), 4)
	got := eye.Wanted(maths.V(0, 0, -100))
	if want := 100.0 / 500; math.Abs(got-want) > 1e-12 {
		t.Fatalf("a place 100 m ahead is wanted at %g m, not %g", got, want)
	}
}

// Anti: a place behind the eye, or well outside its view, is not wanted
// at all.
func TestSightWantsNothingUnseen(t *testing.T) {
	eye := sightOf(lookingNorth(), 4)
	for name, point := range map[string]maths.Vec{
		"behind":   maths.V(0, 0, 100),
		"aside":    maths.V(300, 0, -100),
		"overhead": maths.V(0, 200, -100),
	} {
		if wanted := eye.Wanted(point); !math.IsInf(wanted, 1) {
			t.Errorf("%s: wanted at %g m", name, wanted)
		}
	}
}

// Anti: a place just outside the view, within a cell's diagonal of it,
// is still seen, so a cell half in the view is laid.
func TestSightReachesACellOut(t *testing.T) {
	eye := sightOf(lookingNorth(), 4)
	if wanted := eye.Wanted(maths.V(103, 0, -100)); math.IsInf(wanted, 1) {
		t.Fatal("a place 3 m outside the view was not seen")
	}
}

// Made ground the set says only the seed of makes roughness and no
// flutes or beds, with two faces a pixel.
func TestMadeRulesFromASeed(t *testing.T) {
	rules := madeRules(set.Made{}, 4, lookingNorth())
	if rules.Faces != 2*1000*500 || rules.Flutes.Depth != 0 ||
		rules.Beds.Standout != 0 {
		t.Fatalf("rules %+v", rules)
	}
	if rules.Roughness.Gain != 1 || !rules.Roughness.Continued {
		t.Fatalf("roughness %+v is not the survey's own", rules.Roughness)
	}
}
