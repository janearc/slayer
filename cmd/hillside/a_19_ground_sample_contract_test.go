package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Every sample location must retain the DEM's own height, including its
// first and last rows. Outside requests clamp to the nearest terrain edge.
func TestGroundInterpolationRetainsDEMSamplesAndEveryClampedEdge(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	pinNonplanarTerrainSamples(t, given)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	basaltSurfaces := 0
	for _, item := range stage.world.Things {
		if item.Material != nil && item.Material.Name == "basalt" {
			basaltSurfaces++
			// An authored terrain palette, not a measured rock spectrum.
			if item.Material.Colour != maths.RGB(.07, .065, .06) {
				t.Fatal("terrain changed its chosen dark basalt reflectance", item.Material.Colour)
			}
		}
	}
	if basaltSurfaces != 1 {
		t.Fatal("fixture lost its actual basalt terrain surface")
	}
	if !(stage.lowest > 0) {
		t.Fatal("owned terrain lost its nonzero absolute elevation control")
	}
	w, h, step := stage.ground.Size()
	varied := false
	base := stage.ground.At(0, 0)
	check := func(x, z float64, i, j int) {
		t.Helper()
		want := stage.ground.At(i, j) - stage.lowest
		if got := stage.groundAt(x, z); !(math.Abs(got-want) <= 1e-9) {
			t.Fatalf("ground(%g,%g)=%g, want DEM sample(%d,%d)=%g", x, z, got, i, j, want)
		}
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			check(float64(i)*step, float64(j)*step, i, j)
			varied = varied || stage.ground.At(i, j) != base
		}
		check(-step/4, float64(j)*step, 0, j)
		check((float64(w-1)+.25)*step, float64(j)*step, w-1, j)
	}
	for i := 0; i < w; i++ {
		check(float64(i)*step, -step/4, i, 0)
		check(float64(i)*step, (float64(h-1)+.25)*step, i, h-1)
	}
	if !varied {
		t.Fatal("flat fixture would hide ground interpolation mistakes")
	}
}

// Raise four finite pixels in the owned uncompressed TIFF and repin its
// actual bytes. Interior and last-row/column perturbations expose mistakes
// that an affine plane hides, including extrapolation from the wrong cell.
func pinNonplanarTerrainSamples(t *testing.T, given asked) {
	t.Helper()
	changes := []fixtureTerrainSample{{0, 0, .0625}}
	for _, sample := range [][2]int{{24, 20}, {75, 20}, {24, 50}, {75, 50}} {
		column, row := sample[0], sample[1]
		changes = append(changes, fixtureTerrainSample{column, row,
			float32(column+1000*row)/8 + 256})
	}
	pinFixtureTerrainSamples(t, given, changes)
}

type fixtureTerrainSample struct {
	column, row int
	height      float32
}

// Locate every original sample before changing bytes, so adjacent requested
// samples cannot hide each other's original three-value locator.
func pinFixtureTerrainSamples(t *testing.T, given asked, changes []fixtureTerrainSample) {
	t.Helper()
	shelf := os.Getenv("SLAYER_SHELF")
	raw, err := os.ReadFile(filepath.Join(shelf, "fixture-plain.tif"))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 2 || string(raw[:2]) != "II" {
		t.Fatal("owned terrain fixture changed byte order")
	}
	oldSum := sha256.Sum256(raw)
	offsets := make([]int, len(changes))
	for i, sample := range changes {
		first := max(0, sample.column-1)
		pattern := make([]byte, 12)
		for lane := 0; lane < 3; lane++ {
			value := float32(first+lane+1000*sample.row) / 8
			binary.LittleEndian.PutUint32(pattern[lane*4:], math.Float32bits(value))
		}
		if bytes.Count(raw, pattern) != 1 {
			t.Fatal("owned sample triple is not unique and contiguous", sample.column, sample.row)
		}
		offsets[i] = bytes.Index(raw, pattern) + 4*(sample.column-first)
	}
	for i, sample := range changes {
		binary.LittleEndian.PutUint32(raw[offsets[i]:], math.Float32bits(sample.height))
	}
	for _, name := range []string{"fixture-plain.tif", "fixture.tif"} {
		if err := os.WriteFile(filepath.Join(shelf, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	catalogue := filepath.Join(filepath.Dir(filepath.Dir(given.set)), "sources.yaml")
	text, err := os.ReadFile(catalogue)
	if err != nil {
		t.Fatal(err)
	}
	old := hex.EncodeToString(oldSum[:])
	if !strings.Contains(string(text), old) {
		t.Fatal("owned catalogue did not pin its original terrain")
	}
	sum := sha256.Sum256(raw)
	text = []byte(strings.ReplaceAll(string(text), old, hex.EncodeToString(sum[:])))
	if err := os.WriteFile(catalogue, text, 0600); err != nil {
		t.Fatal(err)
	}
}

// Rays intersect the actual triangulated ground independently of groundAt.
// Probe both face interiors of a real non-coplanar cell, not a restatement
// of the interpolation formula on a plane where either diagonal works.
func TestGroundInterpolationAgreesWithActualNonplanarTerrainHits(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	pinNonplanarTerrainSamples(t, given)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	if !(stage.lowest > 0) {
		t.Fatal("owned terrain lost its nonzero absolute elevation control")
	}
	if len(stage.world.Things) != 1 {
		t.Fatal("fixture did not build one actual ground mesh")
	}
	surface := stage.world.Things[0].Shape
	box := surface.Bounds()
	hitter, ok := surface.(shape.Hitter)
	if !ok {
		t.Fatal("actual triangulated ground lost its exact ray intersector")
	}
	w, h, step := stage.ground.Size()
	checked := 0
	for j := 0; j < h-1; j++ {
		for i := 0; i < w-1; i++ {
			a, b := stage.ground.At(i, j), stage.ground.At(i+1, j)
			c, d := stage.ground.At(i, j+1), stage.ground.At(i+1, j+1)
			if math.Abs(a+d-b-c) < 1e-6 {
				continue
			}
			for _, uv := range [][2]float64{{.75, .25}, {.25, .75}} {
				x, z := (float64(i)+uv[0])*step, (float64(j)+uv[1])*step
				ray := maths.Ray{At: maths.V(x, box.Max.Y+1, z), Dir: maths.V(0, -1, 0)}
				distance, hit := hitter.Hit(ray, 0, box.Max.Y-box.Min.Y+2)
				if !hit {
					t.Fatalf("vertical ray missed actual ground cell(%d,%d)/%v", i, j, uv)
				}
				want := ray.At.Y - distance
				if got := stage.groundAt(x, z); !(math.Abs(got-want) <= 1e-8) {
					t.Fatalf("cell(%d,%d)/%v: ground height %g differs from actual mesh hit %g", i, j, uv, got, want)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatal("owned terrain did not contain a real non-coplanar cell")
	}
}

// Both triangle formulas coincide over real numbers on their shared edge.
// A checked float32 DEM can still distinguish their evaluation order. These
// extreme finite samples are a numerical stress input, not physical terrain.
func TestGroundDiagonalKeepsTheActualFarCornerUnderCancellation(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	const high = float32(0x1p100)
	pinFixtureTerrainSamples(t, given, []fixtureTerrainSample{
		{74, 49, high}, {75, 49, 0}, {74, 50, high}, {75, 50, 1},
	})
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal("finite numeric stress terrain should construct", err)
	}
	w, h, step := stage.ground.Size()
	if w != 76 || h != 51 || stage.lowest != 0 ||
		stage.ground.At(w-2, h-2) != float64(high) ||
		stage.ground.At(w-1, h-2) != 0 ||
		stage.ground.At(w-2, h-1) != float64(high) ||
		stage.ground.At(w-1, h-1) != 1 {
		t.Fatal("actual DEM lost the intended finite cancellation input")
	}
	if got := stage.groundAt(float64(w-1)*step, float64(h-1)*step); got != 1 {
		t.Fatalf("last actual terrain corner lost its one-metre sample: %g", got)
	}
}
