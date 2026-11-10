package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// drawViewer uses the actual offscreen pipeline with the three typed cgo meshes.
func drawViewer[M any](draw func(string, [3]M), path string, sky, back, earth M) {
	draw(path, [3]M{sky, back, earth})
}

func TestPaintingKeysUpdateRealStacksAndNativeBuffers(t *testing.T) {
	viewerFixture(t)
	data, earth, sky, themes := viewerFiles(t)
	hudThemes = nil
	readThemes(themes, "twilight")
	view.sources = []source{{name: "warm", path: filepath.Join(data, "data-irac/GLM_00000+0000_mosaic_I4.fits"), nm: 8000, floor: 0, ceil: 1, strength: 1}, {name: "cool", path: filepath.Join(data, "data-irac/GLM_00000+0000_mosaic_I1.fits"), nm: 3600, floor: 0, ceil: 1, strength: .5}}
	skyCorners := buildSky(0)
	backCorners := buildBack(sky, 0)
	buildEarth(earth, 0)
	view.ra, view.dec, view.fov = 180, 0, 60
	recolour()
	shot := filepath.Join(t.TempDir(), "controls.png")
	// The real shot allocates the native buffers before any repaint callback.
	drawViewer(writeShot, shot, meshOf(skyCorners, view.skyColours), meshOf(backCorners, view.backColours), meshOf(view.earth.corners, view.earth.colours))
	file, err := os.Open(shot)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if picture.Bounds().Dx() != 1400 || picture.Bounds().Dy() != 1000 {
		t.Fatal("native frame size")
	}
	before := append([]byte{}, view.skyColours...)
	goKey('1')
	if view.on[0] || bytes.Equal(before, view.skyColours) {
		t.Fatal("sky layer toggle did not repaint")
	}
	goKey('1')
	if !view.on[0] {
		t.Fatal("sky layer did not return")
	}
	before = append([]byte{}, view.backColours...)
	goKey('5')
	if view.wideOn[0] || bytes.Equal(before, view.backColours) {
		t.Fatal("wide layer toggle did not repaint")
	}
	goKey('5')
	goKey('e')
	if view.mode != earthView {
		t.Fatal("earth mode")
	}
	view.lon, view.lat = 30, 20
	goKey('p')
	if !view.placed || view.placeLon != 30 || view.placeLat != 20 || view.earth.marked < 0 {
		t.Fatal("observer not placed")
	}
	goKey('f')
	if view.mode != placeView {
		t.Fatal("standing mode")
	}
	goKey('f')
	if view.mode != earthView {
		t.Fatal("back to earth")
	}
	goKey('e')
	if view.mode != skyView {
		t.Fatal("back to sky")
	}
	lightness := view.lightness
	goKey('+')
	if view.lightness <= lightness {
		t.Fatal("lightness increase")
	}
	goKey('-')
	closeFloat(t, view.lightness, lightness)
	goKey('=')
	if view.lightness <= lightness {
		t.Fatal("equals lightness")
	}
	goKey('m')
	if view.earth.showing != 1 {
		t.Fatal("photograph cycle")
	}
	goKey('m')
	if view.earth.showing != 2 {
		t.Fatal("measured earth cycle")
	}
	goKey('m')
	if view.earth.showing != 0 {
		t.Fatal("photograph wrap")
	}
	goKey('c')
	if view.chroma != 1 {
		t.Fatal("chroma cycle")
	}
	goKey('b')
	if view.ground != 1 {
		t.Fatal("background cycle")
	}
	goKey('h')
	if view.hudSize != 0 {
		t.Fatal("HUD enabled")
	}
	showHud(800, 600)
	first := view.hudSaid
	showHud(800, 600)
	if first == "" || first == "off" || view.hudSaid != first {
		t.Fatal("HUD cache")
	}
	for _, mode := range []int{earthView, placeView} {
		view.mode = mode
		showHud(800, 600)
	}
}
