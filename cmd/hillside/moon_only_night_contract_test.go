package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/set"
)

// A Sun below the horizon still adds solar skylight. These night recipes
// deliberately use only the full-Moon source, so an extra Sun is a regression.
func TestMoonOnlyNightInputsExcludeSolarSky(test *testing.T) {
	for _, name := range []string{"quality-summit-moon.yaml",
		"quality-summit-moon-preview.yaml", "quality-summit-moon-4k.yaml"} {
		test.Run(name, func(test *testing.T) {
			read := readLightingInput(test, name)
			if read.Light() != "moon" {
				test.Fatal("night scene lost its Moon lighting")
			}
			moon, present := read.Moonlit()
			if !present || !moon.Disc() {
				test.Fatal("night scene lost its visible Moon")
			}
			if _, present := read.Sunlit(); present {
				test.Fatal("Moon-only night scene also adds solar skylight")
			}
		})
	}
}

// The guard is specific to night intent; ordinary solar scenes keep their Sun.
func TestMoonOnlyGuardDoesNotRemoveDaylightInputs(test *testing.T) {
	read := readLightingInput(test, "quality-summit-golden-preview.yaml")
	if _, present := read.Sunlit(); !present || read.Light() != "sun" {
		test.Fatal("golden-hour scene lost its explicit Sun")
	}
}

// readLightingInput checks the shipped recipe through the ordinary set reader.
func readLightingInput(test *testing.T, name string) set.Set {
	test.Helper()
	path := filepath.Join("..", "..", "doc", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	read, err := set.Read(path, raw)
	if err != nil {
		test.Fatal(err)
	}
	return read
}
