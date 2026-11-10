package smoke_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/smoke"
)

// firstInput is defined as the index of the sole declared input.
const firstInput = 0

// soleInput is defined as the count of files required by this photo smoke.
const soleInput = 1

func TestRequestedPhotoSmokeNeedsItsRealInput(test *testing.T) {
	var scene smoke.Smoke
	exists := false
	for _, candidate := range smoke.All() {
		for _, input := range candidate.Inputs {
			if input == "$SLAYER_MARBLE/blue.jpg" {
				scene, exists = candidate, true
			}
		}
	}
	if !exists {
		test.Fatal("photo smoke disappeared")
	}
	test.Setenv("SLAYER_MARBLE", "")
	if _, err := scene.RequireInputs(); err == nil || !strings.Contains(err.Error(), "SLAYER_MARBLE") {
		test.Fatal("missing data directory was not explained", err)
	}
	directory := test.TempDir()
	test.Setenv("SLAYER_MARBLE", directory)
	if _, err := scene.RequireInputs(); !errors.Is(err, os.ErrNotExist) {
		test.Fatal("missing file was accepted", err)
	}
	path := filepath.Join(directory, "blue.jpg")
	if err := os.WriteFile(path, []byte("availability fixture, not a decoded photo"), 0600); err != nil {
		test.Fatal(err)
	}
	paths, err := scene.RequireInputs()
	if err != nil || len(paths) != soleInput || paths[firstInput] != path {
		test.Fatal("declared asset was not returned", paths, err)
	}
}
