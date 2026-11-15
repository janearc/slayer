//go:build draw

package hillside

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
)

// The numbers the rig film's tests pick, each by the session slice's
// author, 2026-11-12.
const (
	// rigShutter is the fixture rig's shutter, half a second, as the
	// tests write it.
	rigShutter = "0.5"
	// rigFilm is two frames half a second apart.
	rigFilm = "fps: 2\nduration: {value: 1, unit: s}\n"
	// rigStep is how far the moving camera goes in its second, in metres.
	rigStep = 20.0
	// rigOpen is the fixture rig's shutter in seconds, and rigGlimpses
	// its glimpses, as onFixtureRigged writes them: defined there.
	rigOpen     = 0.5
	rigGlimpses = 2
	// rigWidth is a width for the fixture's frame, and rigRows its rows:
	// 96 across a frame 70.41 by 51.61 mm is 70.4, so 70.
	rigWidth = 96
	rigRows  = 70
)

// a rig film's first frame, developed by the session's tone, is the
// rigged still's picture at the same time, byte for byte; and a still
// scene seen from a still camera gives every frame the same.
func TestRigFilmFrameIsTheRiggedStill(t *testing.T) {
	rigged := onFixtureRigged(t, rigShutter, false)
	still := sessionPicture(t, jobFrom(t, "scene: {set: "+rigged+"}\n"),
		Ask{})
	film := opening(t, jobFrom(t, "scene: {set: "+rigged+"}\n"+rigFilm))
	if len(film.offsets) != rigGlimpses {
		t.Fatalf("the rig film draws %d glimpses, not the rig's %d",
			len(film.offsets), rigGlimpses)
	}
	first := filmPicture(t, film, nothing)
	if !bytes.Equal(first, still) {
		t.Error("the rig film's first frame is not the rigged still")
	}
	if !bytes.Equal(filmPicture(t, film, singleRay), first) {
		t.Error("a still scene's rig film changed between frames")
	}
}

// filmPicture is a session's frame developed by its tone, its hud drawn,
// and written as save writes a still.
func filmPicture(t *testing.T, session *Session, index int) []byte {
	t.Helper()
	drawn, err := session.Frame(context.Background(), index, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	picture := session.Tone().Develop(drawn.Light())
	if err := drawn.Overlay(picture); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "frame.png")
	if err := file.PNG(out, picture, sessionSquare); err != nil {
		t.Fatal(err)
	}
	return readBytes(t, out)
}

// a rig film's frame is the lens's exposure of its glimpses, not their
// mean, and its glimpses are the rig's, at take's times.
func TestRigFilmExposesTheRigsGlimpses(t *testing.T) {
	rigged := onFixtureRigged(t, rigShutter, false)
	film := opening(t, jobFrom(t, "scene: {set: "+rigged+"}\n"+rigFilm))
	want := takes(t, 1/rigOpen, shutterWhole, rigGlimpses)
	if !slices.Equal(film.offsets, want) {
		t.Errorf("the rig's glimpses are at %v, take's at %v",
			film.offsets, want)
	}
	drawn, err := film.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	glimpses := drawn.Glimpses()
	if len(glimpses) != rigGlimpses {
		t.Fatalf("%d glimpses, not the rig's %d", len(glimpses),
			rigGlimpses)
	}
	exposed, err := film.source.(rigFramer).exposed(drawn.At(),
		film.given, glimpses)
	if err != nil || !sameLight(drawn.Light(), exposed) {
		t.Errorf("the frame's light is not the lens's exposure: %v", err)
	}
	if sameLight(drawn.Light(), glimpses[nothing]) {
		t.Error("the frame's light is a glimpse, not the exposure")
	}
}

// a moving camera's lens is focused from each glimpse's eye: the lens
// built for a time is the rig's lens focused on its point from the eye
// at that time, and two times far apart give two focus distances.
func TestRigFilmRefocusesFromEachEye(t *testing.T) {
	rigged := onFixtureRigged(t, rigShutter, false)
	still := opening(t, jobFrom(t, "scene: {set: "+rigged+"}\n"))
	from, on := still.given.eye.Origin, still.given.lensed.on
	to := from.Add(lensStep())
	path := fmt.Sprintf("camera:\n  unit: m\n  look-at: {x: %g, y: %g, "+
		"z: %g}\n  moves:\n    - from: {x: %g, y: %g, z: %g}\n"+
		"      to: {x: %g, y: %g, z: %g}\n      duration: {value: 1, "+
		"unit: s}\n", on.X, on.Y, on.Z, from.X, from.Y, from.Z, to.X, to.Y,
		to.Z)
	film := opening(t, jobFrom(t, "scene: {set: "+rigged+"}\n"+rigFilm+
		path))
	rig := film.source.(rigFramer)
	var distances []float64
	for _, at := range film.Times() {
		made, err := rig.light(at, film.given)
		if err != nil {
			t.Fatal(err)
		}
		distance := on.Sub(made.given.eye.Origin).Len()
		want, err := lens.NewThin(*film.given.frame,
			film.given.lensed.focal, film.given.lensed.stop, distance)
		if err != nil || *made.given.glass != want {
			t.Errorf("the lens at %v is not focused from its eye: %v",
				at, err)
		}
		distances = append(distances, distance)
	}
	if distances[nothing] == distances[singleRay] {
		t.Errorf("a moving camera kept one focus distance, %v", distances)
	}
}

// a size of the rig frame's shape is taken, its height the frame's; a
// size of another shape is refused, naming both shapes.
func TestRigTakesASizeOfItsFramesShape(t *testing.T) {
	rigged := onFixtureRigged(t, rigShutter, false)
	for _, height := range []int{rigRows, rigRows + singleRay} {
		session := opening(t, jobFrom(t, fmt.Sprintf("scene: {set: %s}\n"+
			"film: {width: %d, height: %d}\n", rigged, rigWidth, height)))
		if session.given.height != rigRows {
			t.Errorf("%d by %d through the rig is %d rows, not %d",
				rigWidth, height, session.given.height, rigRows)
		}
	}
	for _, each := range []struct {
		height int
		says   string
	}{
		{rigRows + singleRay + singleRay, "is wide"},
		{rigWidth * rigWidth / rigRows, "is tall"},
	} {
		_, err := Open(context.Background(), jobFrom(t, fmt.Sprintf(
			"scene: {set: %s}\nfilm: {width: %d, height: %d}\n", rigged,
			rigWidth, each.height)), Stamp{})
		if err == nil || !strings.Contains(err.Error(), each.says) ||
			!strings.Contains(err.Error(), "which is wide") {
			t.Errorf("%d by %d through a wide rig gave %v", rigWidth,
				each.height, err)
		}
	}
}

// lensStep is how the moving camera moves in its second: back along z.
func lensStep() maths.Vec { return maths.V(nothing, nothing, rigStep) }

// readBytes is a file's bytes.
func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// a rig film of a scene whose lighting is not retained from frame to
// frame is refused, as any film of one is.
func TestRigFilmRefusesLightingItCannotRetain(t *testing.T) {
	alight := onFixtureRigged(t, rigShutter, true)
	_, err := Open(context.Background(), jobFrom(t, "scene: {set: "+
		alight+"}\n"+rigFilm), Stamp{})
	if err == nil || !strings.Contains(err.Error(), "retain") {
		t.Errorf("a rig film of lighting it cannot retain gave %v", err)
	}
}
