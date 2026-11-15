//go:build draw

package main_test

import (
	"bytes"
	_ "embed"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// channels is defined by image.Color.RGBA as four returned components.
const channels = 4

// distinct is defined as the smallest count proving colour variation.
const distinct = 2

func TestSmokeAndNamedSceneProduceActualPNGs(test *testing.T) {
	for _, selector := range []string{"1", "swarm"} {
		directory := test.TempDir()
		output, code := call("show", selector, "--width", "16", "--height", "12",
			"--rays", "1", "--out", directory)
		if code != success {
			test.Fatal(output)
		}
		name := selector
		if selector == "1" {
			name = "smoke-1"
		}
		readPNG(test, filepath.Join(directory, name+".png"), picked(test, "width"), picked(test, "height"))
	}
}

func readPNG(test *testing.T, path string, width, height int) {
	test.Helper()
	file, err := os.Open(path)
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	picture, err := png.Decode(file)
	if err != nil || picture.Bounds().Dx() != width || picture.Bounds().Dy() != height {
		test.Fatalf("bad PNG %s: %v", path, err)
	}
	colours := map[[channels]uint32]bool{}
	for y := range height {
		for x := range width {
			red, green, blue, alpha := picture.At(x, y).RGBA()
			colours[[channels]uint32{red, green, blue, alpha}] = true
		}
	}
	if len(colours) < distinct {
		test.Fatal("render contains no visible scene variation")
	}
}

func TestTerminalPreviewAlsoKeepsThePNG(test *testing.T) {
	directory := test.TempDir()
	output, code := call("show", "1", "--width", "16", "--height", "12",
		"--terminal", "--cells", "8x4", "--out", directory)
	if code != success || !bytes.Contains([]byte(output), []byte("\x1b[38;2;")) {
		test.Fatal(code, output)
	}
	readPNG(test, filepath.Join(directory, "smoke-1.png"), picked(test, "width"), picked(test, "height"))
}

// picked reads one dimension of the deliberately tiny command fixture.
func picked(test *testing.T, name string) int {
	test.Helper()
	value, err := choices.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return int(value.Value)
}
