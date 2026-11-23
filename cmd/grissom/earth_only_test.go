package main

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestEarthOnlyNeedsNoSkyDirectoriesAndBuildsNoSkyTriangles(t *testing.T) {
	viewerFixture(t)
	_, earth, _, _ := viewerFiles(t)
	skyCorners, backCorners := start(options{earth: earth, earthDepth: 1,
		earthOnly: true, distance: defaultDistance, fov: defaultFov})
	if len(skyCorners) != 0 || len(backCorners) != 0 || len(view.cells) != 0 ||
		len(view.sources) != 0 || len(view.wideSources) != 0 {
		t.Fatalf("sky built: %d, %d corners, %d cells", len(skyCorners), len(backCorners), len(view.cells))
	}
	if len(view.earth.corners) != 80*9 || view.mode != earthView {
		t.Fatalf("earth %d corners, mode %d", len(view.earth.corners), view.mode)
	}
	for _, empty := range []struct {
		count   int
		corners bool
	}{
		{int(meshOf(skyCorners, view.skyColours).count), meshOf(skyCorners, view.skyColours).corners != nil},
		{int(meshOf(backCorners, view.backColours).count), meshOf(backCorners, view.backColours).corners != nil},
	} {
		if empty.count != 0 || empty.corners {
			t.Fatal("an empty sky was handed to Metal as a mesh")
		}
	}
}

func TestEarthOnlyShotDrawsOnlyTheEarthViewAndKeepsToIt(t *testing.T) {
	viewerFixture(t)
	_, earth, _, themes := viewerFiles(t)
	prefix := filepath.Join(t.TempDir(), "only")
	err := run(options{earth: earth, colourways: themes, colourway: "twilight",
		earthDepth: 1, earthOnly: true, noHud: true, distance: 4.2,
		fov: defaultFov, shot: prefix, view: earthView, look: "-122.4,37.7"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(prefix + "-earth.png")
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(file)
	file.Close()
	if err != nil || picture.Bounds().Dx() != 1400 || picture.Bounds().Dy() != 1000 {
		t.Fatalf("earth shot %v: %v", picture.Bounds(), err)
	}
	for _, other := range []string{"sky", "standing"} {
		if _, err := os.Stat(prefix + "-" + other + ".png"); err == nil {
			t.Fatalf("a %s shot was drawn", other)
		}
	}
	goKey('e')
	if view.mode != earthView {
		t.Fatal("e flipped to a sky that is not loaded")
	}
	goKey('c')
	if view.chroma != 1 {
		t.Fatal("a colour key did not reach the earth-only viewer")
	}
}

func TestNoHudHidesTheCrosshairAndTheText(t *testing.T) {
	viewerFixture(t)
	view.mode, view.hudSize = earthView, 0
	got, _ := uniforms(goView, 800, 400)
	if got[34] != 1 || view.hudSaid == "off" {
		t.Fatalf("the hud was not shown: crosshair %v, said %q", got[34], view.hudSaid)
	}
	view.noHud, view.hudSaid = true, ""
	got, _ = uniforms(goView, 800, 400)
	if got[34] != 0 || view.hudSaid != "off" {
		t.Fatalf("the hud was shown: crosshair %v, said %q", got[34], view.hudSaid)
	}
}

// limbHeight is where the top of the globe's outline falls in a frame, as
// a share of the frame's half height: past 1 it is cut off.
func limbHeight(t *testing.T, distance float64, width, height int) float64 {
	t.Helper()
	viewerFixture(t)
	view.mode, view.distance = earthView, distance
	got, _ := uniforms(goView, width, height)
	toward := vec(unit(view.lon, view.lat))
	right := norm(cross(scaled(toward, -1), [3]float64{0, 0, 1}))
	up := cross(right, scaled(toward, -1))
	// the point where a sight line from the camera just grazes the earth.
	limb := plus(scaled(toward, 1/distance), scaled(up, math.Sqrt(1-1/(distance*distance))))
	row := func(first int) float64 {
		return float64(got[first])*limb[0] + float64(got[first+4])*limb[1] + float64(got[first+8])*limb[2] + float64(got[first+12])
	}
	return row(1) / row(3)
}

func TestTheDistanceDecidesWhetherTheWholeGlobeIsInTheFrame(t *testing.T) {
	if cut := limbHeight(t, defaultDistance, 1400, 1000); cut <= 1 {
		t.Fatalf("at %g radii the globe fits (%.3f), but a shot cuts it off", defaultDistance, cut)
	}
	fit := limbHeight(t, 4.2, 1400, 1000)
	if fit >= 1 || fit < 0.9 {
		t.Fatalf("at 4.2 radii the globe's top is at %.3f of the half height", fit)
	}
}

func TestDistanceAndNoHudReachTheView(t *testing.T) {
	viewerFixture(t)
	configure(options{distance: 4.2, noHud: true, earthOnly: true, fov: defaultFov})
	if view.distance != 4.2 || !view.noHud || !view.earthOnly || view.mode != earthView {
		t.Fatalf("view %v %v %v %d", view.distance, view.noHud, view.earthOnly, view.mode)
	}
}
