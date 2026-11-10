package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"bytes"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

func TestSyntheticEarthDeduplicatesVerticesAndReflectsKnownSpectra(t *testing.T) {
	viewerFixture(t)
	_, earth, _, _ := viewerFiles(t)
	buildEarth(earth, 1)
	if len(view.earth.corners) != 80*9 || len(view.earth.hexOf) != 80*3 || len(view.earth.hexes) != 42 || len(view.earth.triangleOf) != 80 {
		t.Fatalf("earth faces/vertices %d/%d", len(view.earth.corners)/9, len(view.earth.hexes))
	}
	_, landY, _ := view.earth.land.XYZ()
	_, seaY, _ := view.earth.sea.XYZ()
	closeFloat(t, landY, .6)
	closeFloat(t, seaY, .2)
	white := swatch.Illuminant(swatch.D65())
	wx, wy, wz := white.XYZ()
	for _, sample := range []struct {
		shade swatch.Swatch
		share float64
	}{{view.earth.land, .6}, {view.earth.sea, .2}} {
		x, y, z := sample.shade.XYZ()
		closeFloat(t, x, sample.share*wx)
		closeFloat(t, y, sample.share*wy)
		closeFloat(t, z, sample.share*wz)
	}

	if len(view.earth.images) != 2 {
		t.Fatal("synthetic photographs were not read")
	}
	landCount := 0
	for _, land := range view.earth.onLand {
		if land {
			landCount++
		}
	}
	if landCount == 0 || landCount == len(view.earth.onLand) {
		t.Fatal("land polygon did not distinguish sea")
	}
	for _, at := range [][2]float64{{3.1, 4.2}, {81.3, -22.7}, {-175.2, 68.3}, {123.4, -76.2}} {
		x, y, z := unit(at[0], at[1])
		direction := [3]float64{x, y, z}
		got := nearestHex(at[0], at[1])
		best := -2.0
		for _, hex := range view.earth.hexes {
			best = math.Max(best, dot(hex, direction))
		}
		closeFloat(t, dot(view.earth.hexes[got], direction), best)
	}
	view.mode = earthView
	view.earth.marked = 0
	view.earth.showing = len(view.earth.images)
	recolourEarth()
	if len(view.earth.colours) != len(view.earth.hexOf)*4 {
		t.Fatal("earth colour count")
	}
	for i := 3; i < len(view.earth.colours); i += 4 {
		if view.earth.colours[i] != 255 {
			t.Fatal("earth alpha")
		}
	}
	measured := append([]byte{}, view.earth.colours...)
	landBrightness, seaBrightness := -1, -1
	for corner, hex := range view.earth.hexOf {
		if hex == view.earth.marked {
			continue
		}
		rgb := view.earth.colours[corner*4 : corner*4+3]
		brightness := int(rgb[0]) + int(rgb[1]) + int(rgb[2])
		if view.earth.onLand[hex] {
			landBrightness = brightness
		} else {
			seaBrightness = brightness
		}
	}
	if seaBrightness < 0 || landBrightness <= seaBrightness {
		t.Fatalf("measured land/sea brightness %d/%d", landBrightness, seaBrightness)
	}
	view.earth.showing = 0
	recolourEarth()
	if bytes.Equal(measured, view.earth.colours) {
		t.Fatal("photograph did not replace measured surface")
	}

}

func TestLayerReadersStretchAndCompositeSyntheticSky(t *testing.T) {
	viewerFixture(t)
	data, _, sky, themes := viewerFiles(t)
	hudThemes = nil
	readThemes(themes, "twilight")
	if len(hudThemes) != 2 || hudThemes[view.theme].name != "twilight" {
		t.Fatalf("themes %+v selected%d", hudThemes, view.theme)
	}
	view.sources = []source{{name: "warm", path: filepath.Join(data, "data-irac/GLM_00000+0000_mosaic_I4.fits"), nm: 8000, floor: 0, ceil: 1, strength: 1}, {name: "cool", path: filepath.Join(data, "data-irac/GLM_00000+0000_mosaic_I1.fits"), nm: 3600, floor: 0, ceil: 1, strength: .5, occludes: []string{"warm"}}}
	corners := buildSky(2)
	if len(corners) != len(view.cells)*9 || len(view.cells) == 0 || len(view.skyColours) != len(view.cells)*12 {
		t.Fatal("sky storage")
	}
	previous := uint64(0)
	for i, cell := range view.cells {
		if i > 0 && cell.Key() <= previous {
			t.Fatal("sky keys not sorted and unique")
		}
		previous = cell.Key()
	}
	laid := lay(view.sources[0], 2)
	low, high := stretch(laid, 0, 1)
	if low < 1 || high > 16 || high <= low {
		t.Fatalf("stretch %g..%g", low, high)
	}
	back := buildBack(sky, 1)
	if len(view.wideSources) != 4 || len(back) != 80*9 || view.backCount != 80 {
		t.Fatal("whole sky reader")
	}
	view.ground = 1 // black: the background cannot satisfy the light assertion
	recolour()
	if len(view.backColours) != 80*12 {
		t.Fatal("back colours")
	}
	nonzero := false
	for i := 0; i < len(view.skyColours); i += 4 {
		if view.skyColours[i+3] != 255 {
			t.Fatal("sky alpha")
		}
		nonzero = nonzero || view.skyColours[i] > 0 || view.skyColours[i+1] > 0 || view.skyColours[i+2] > 0
	}
	if !nonzero {
		t.Fatal("lit sources made a black sky")
	}
	x, y, z := wideLight(nil, view.cells[0], .1, .2, .3)
	closeFloat(t, x, .1)
	closeFloat(t, y, .2)
	closeFloat(t, z, .3)
}

func TestViewerColourAndLabelControlsHaveFiniteEndpoints(t *testing.T) {
	viewerFixture(t)
	for _, sample := range []struct {
		in  float64
		out byte
	}{{-1, 0}, {0, 0}, {.5, 128}, {1, 255}, {2, 255}} {
		if got := byteOf(sample.in); got != sample.out {
			t.Fatalf("channel %g became %d", sample.in, got)
		}
	}
	for i := range grounds {
		view.ground = i
		r, g, b := lineColour()
		want := 1.0
		if grounds[i] > .5 {
			want = 0
		}
		// The endpoint is an eight-bit display colour. The colour transform's
		// floating white need not equal one before quantization.
		for _, v := range []float64{r, g, b} {
			expected := byte(0)
			if want == 1 {
				expected = 255
			}
			if byteOf(v) != expected {
				t.Fatalf("outline channel %g became %d, want %d", v, byteOf(v), expected)
			}
		}
	}
	sample := srgb.RGB8(30, 80, 150).Swatch()
	view.chroma = 0
	coloured := seen(sample)
	view.chroma = len(chromas) - 1
	grey := seen(sample)
	if coloured == grey || math.Abs(float64(grey[0])-float64(grey[1])) > 1 || math.Abs(float64(grey[1])-float64(grey[2])) > 1 {
		t.Fatalf("chroma control: %v vs %v", coloured, grey)
	}
	view.lightness = .1
	dark := seen(sample)
	view.lightness = 1
	light := seen(sample)
	sum := func(c [3]byte) int { return int(c[0]) + int(c[1]) + int(c[2]) }
	if sum(dark) >= sum(light) {
		t.Fatalf("lightness control %v vs %v", dark, light)
	}

}

func TestThemeReaderSkipsUnreadableFilesAndSelectsSortedName(t *testing.T) {
	viewerFixture(t)
	_, _, _, themes := viewerFiles(t)
	hudThemes = nil
	if err := os.Symlink(filepath.Join(themes, "not-there"), filepath.Join(themes, "broken.css")); err != nil {
		t.Fatal(err)
	}
	readThemes(themes, "twilight")
	if len(hudThemes) != 2 || hudThemes[0].name != "a" || hudThemes[1].name != "twilight" || view.theme != 1 {
		t.Fatalf("themes %+v index%d", hudThemes, view.theme)
	}
}

func TestSparseMarbleKeepsUnobservedFineCellsBlack(t *testing.T) {
	viewerFixture(t)
	_, earth, _, _ := viewerFiles(t)
	buildEarth(earth, 2)
	empty, lit := 0, 0
	for _, shade := range view.earth.images[0].light {
		x, y, z := shade.XYZ()
		if x == 0 && y == 0 && z == 0 {
			empty++
		} else {
			lit++
		}
	}
	if empty == 0 || lit == 0 {
		t.Fatalf("sparse image filled%d empty%d", lit, empty)
	}
}
