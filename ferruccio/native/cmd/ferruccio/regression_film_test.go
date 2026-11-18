package main

import (
	"bytes"
	"errors"
	"github.com/janearc/puzzlemouth/yamlfile"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestBoundedRegressionRefusesPhysicalMoonrise preserves the actual optical contract.
func TestBoundedRegressionRefusesPhysicalMoonrise(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "sets", "kilauea-iki-moonrise.yaml")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), original...)
	got, err := boundedRegressionFilm(path, original)
	if !errors.Is(err, errBoundedPhysicalFilm) || !strings.Contains(err.Error(), "physical film") {
		t.Fatalf("physical rig accepted: output bytes=%d error=%v", len(got), err)
	}
	if got != nil {
		t.Fatal("refused rig returned rewritten source")
	}
	if !bytes.Equal(original, before) {
		t.Fatal("source bytes changed")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("source file changed")
	}
}

// TestBoundedRegressionChangesOnlyRaster leaves ordinary scene semantics intact.
func TestBoundedRegressionChangesOnlyRaster(t *testing.T) {
	path := filepath.Join("..", "..", "..", "examples", "full-render-check.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := boundedRegressionFilm(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]any
	if err := yamlfile.Read(path, raw, &before); err != nil {
		t.Fatal(err)
	}
	if err := yamlfile.Read(path, got, &after); err != nil {
		t.Fatal(err)
	}
	film := after["scene"].(map[string]any)["film"].(map[string]any)
	if film["width"] != 160 || film["height"] != 100 || film["rays"] != 1 {
		t.Fatalf("wrong bounded film: %v", film)
	}
	originalFilm := before["scene"].(map[string]any)["film"].(map[string]any)
	for _, key := range []string{"width", "height", "rays"} {
		delete(film, key)
		delete(originalFilm, key)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("changed scene beyond raster fields")
	}
}
