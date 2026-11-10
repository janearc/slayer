package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
)

// This changes a real sample in the uncompressed placed TIFF and pins the
// resulting bytes. The tag structure stays valid, so failure must come from
// the geometry consumer rather than decoding or a mismatched shelf hash.
func pinInfiniteTerrainSample(t *testing.T, given asked, column, row int) {
	t.Helper()
	shelf := os.Getenv("SLAYER_SHELF")
	path := filepath.Join(shelf, "fixture-plain.tif")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw[:2]) != "II" {
		t.Fatal("fixture changed byte order")
	}
	oldSum := sha256.Sum256(raw)
	pattern := make([]byte, 12)
	for i := range 3 {
		value := float32(column-1+i+1000*row) / 8
		binary.LittleEndian.PutUint32(pattern[i*4:], math.Float32bits(value))
	}
	if bytes.Count(raw, pattern) != 1 {
		t.Fatal("fixture sample triple is not unique and contiguous")
	}
	position := bytes.Index(raw, pattern) + 4
	binary.LittleEndian.PutUint32(raw[position:], math.Float32bits(float32(math.Inf(1))))
	for _, name := range []string{"fixture-plain.tif", "fixture.tif"} {
		if err := os.WriteFile(filepath.Join(shelf, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(raw)
	catalogue := filepath.Join(filepath.Dir(filepath.Dir(given.set)), "sources.yaml")
	text, err := os.ReadFile(catalogue)
	if err != nil {
		t.Fatal(err)
	}
	previous := hex.EncodeToString(oldSum[:])
	if !strings.Contains(string(text), previous) {
		t.Fatal("fixture catalogue did not pin original terrain")
	}
	text = []byte(strings.ReplaceAll(string(text), previous, hex.EncodeToString(sum[:])))
	if err := os.WriteFile(catalogue, text, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPinnedInfiniteHeightFailsThroughTerrainMeshConstruction(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	pinInfiniteTerrainSample(t, given, 24, 20)
	stage := staging{given: given, world: thing.NewWorld()}
	if err := stage.terrain(readBoundarySet(t, given.set)); !errors.Is(err, mesh.ErrMesh) {
		t.Fatalf("terrain lost mesh refusal: %v", err)
	}
}

func TestFuelTilesPropagateARealNonfiniteWindowMeshRefusal(t *testing.T) {
	given := fuelOnFixture(t)
	pinInfiniteTerrainSample(t, given, 24, 20)
	read := readBoundarySet(t, given.set)
	ground, has := read.Ground()
	if !has {
		t.Fatal("no terrain in fixture")
	}
	raster, err := terrain.Open(filepath.Join(os.Getenv("SLAYER_SHELF"), ground.File()))
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	west, north, across, down, step := ground.Window()
	window, err := raster.Window(west, north, int(across/step)+1, int(down/step)+1, step)
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{given: given, world: thing.NewWorld(), recipes: read, ground: window, west: west, north: north}
	if err := stage.prepareBeds(read); !errors.Is(err, mesh.ErrMesh) {
		t.Fatalf("fuel tile lost mesh refusal: %v", err)
	}
}
