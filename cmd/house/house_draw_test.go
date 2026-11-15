//go:build draw

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
)

// porch writes the porch's set file, changed by each pair of was and now,
// where the test keeps its files, and says where.
func porch(t *testing.T, changes ...string) string {
	t.Helper()
	source, err := os.ReadFile("testdata/porch.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := string(source)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(written, changes[i]) {
			t.Fatalf("%q is not in the porch", changes[i])
		}
		written = strings.Replace(written, changes[i], changes[i+1], 1)
	}
	return kept(t, "porch.yaml", written)
}

// kept writes a file where the test keeps its files, and says where.
func kept(t *testing.T, name, written string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// picture is the png at a path, decoded.
func picture(t *testing.T, path string) image.Image {
	t.Helper()
	opened, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	decoded, err := png.Decode(opened)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

// the box stands one metre north of the world's middle, for the camera
// that stands off it.
var box = places{"the box": maths.V(0, 1, 0)}

// one camera is drawn whole, at the film's frame, into a folder made for
// it that anyone may read, with its raw beside the picture, and house
// says so, with the rays a pixel the rays along each side make.
func TestOneCameraIsDrawnWhole(t *testing.T) {
	out := filepath.Join(t.TempDir(), "made", "front.png")
	said, err := shoot(porch(t, "rays: 1", "rays: 2"), []string{"front"},
		places{}, out)
	if err != nil {
		t.Fatal(err)
	}
	if said != out+": 16x9, 4 rays a pixel\n" {
		t.Errorf("house said %q", said)
	}
	made, err := os.Stat(filepath.Dir(out))
	if err != nil || made.Mode().Perm() != 0o755 {
		t.Errorf("the folder is %v, %v", made.Mode(), err)
	}
	if size := picture(t, out).Bounds().Size(); size != image.Pt(16, 9) {
		t.Errorf("the picture is %v", size)
	}
	light, err := raw.Read(strings.TrimSuffix(out, ".png") + ".pfm")
	if err != nil || light.W != 16 || light.H != 9 {
		t.Errorf("the raw is %v, %v", light, err)
	}
}

// several cameras are drawn side by side, each at half the frame, a
// black gap of a thirty-second of one between them, and house says each
// by its name and title. no raw is written for them.
func TestSeveralCamerasAreDrawnSideBySide(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pair.png")
	said, err := shoot(porch(t, "width: 16", "width: 64"),
		[]string{"front", "side"}, box, out)
	if err != nil {
		t.Fatal(err)
	}
	if said != out+": front, \"the front\"\n"+out+": side, \"the side\"\n" {
		t.Errorf("house said %q", said)
	}
	drawn := picture(t, out)
	if size := drawn.Bounds().Size(); size != image.Pt(65, 4) {
		t.Errorf("the picture is %v", size)
	}
	black := color.RGBAModel.Convert(color.Black)
	for y := range 4 {
		if got := color.RGBAModel.Convert(drawn.At(32, y)); got != black {
			t.Errorf("the gap at row %d is %v", y, got)
		}
	}
	for _, x := range []int{0, 31, 33, 64} {
		if got := color.RGBAModel.Convert(drawn.At(x, 0)); got == black {
			t.Errorf("the top row at %d is black: a picture is missing "+
				"or short", x)
		}
	}
	if _, err := os.Stat(strings.TrimSuffix(out, ".png") +
		".pfm"); err == nil {
		t.Error("a raw was written for a pair")
	}
}

// what cannot be drawn is refused, saying why: a file not there or not a
// set, a set with no world or no film, a world the tracer cannot hold, a
// camera the set has not or one off an asset not placed, a picture too
// big to draw, and a picture or raw that cannot be written.
func TestWhatCannotBeDrawnIsRefused(t *testing.T) {
	dir := t.TempDir()
	file := kept(t, "file", "")
	taken := filepath.Join(dir, "taken.png")
	if err := os.MkdirAll(filepath.Join(dir, "taken.pfm"),
		0o755); err != nil {
		t.Fatal(err)
	}
	pngTaken := filepath.Join(dir, "png.png")
	if err := os.MkdirAll(pngTaken, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, refusal := range []struct {
		set     string
		cameras []string
		at      places
		out     string
		said    string
	}{
		{filepath.Join(dir, "nowhere.yaml"), []string{"front"}, places{},
			"", "no such file"},
		{porch(t, "scene:", "scenery:"), []string{"front"}, places{}, "",
			"scenery is not a key here"},
		{kept(t, "bare.yaml", "scene:\n  film: {width: 4, height: 4, "+
			"rays: 1}\n"), []string{"front"}, places{}, "",
			"says no world to draw"},
		{kept(t, "dark.yaml", "scene:\n  world: {}\n"), []string{"front"},
			places{}, "", "says no film to draw on"},
		{porch(t, "temperature: 2700", "temperature: 10"),
			[]string{"front"}, places{}, "",
			"porch.yaml: lamp bulb: a black body at 10 K"},
		{porch(t), []string{"back"}, places{}, "",
			"the set has no camera back"},
		{porch(t), []string{"side"}, places{}, "",
			`camera side: stands off "the box", which is not on the set`},
		{porch(t, "width: 16", "width: 100000", "height: 9",
			"height: 100000", "rays: 1", "rays: 64"), []string{"front"},
			places{}, filepath.Join(dir, "big.png"), "trace: "},
		{porch(t, "width: 16", "width: 1"), []string{"front", "side"}, box,
			filepath.Join(dir, "thin.png"), "trace: a picture is at least"},
		{porch(t), []string{"front"}, places{},
			filepath.Join(file, "under.png"), "mkdir " + file},
		{porch(t), []string{"front"}, places{}, taken, "taken.pfm"},
		{porch(t), []string{"front"}, places{}, pngTaken, "png.png"},
		{porch(t), []string{"front", "side"}, box, pngTaken, "png.png"},
	} {
		out := refusal.out
		if out == "" {
			out = filepath.Join(dir, "out.png")
		}
		_, err := shoot(refusal.set, refusal.cameras, refusal.at, out)
		if err == nil || !strings.Contains(err.Error(), refusal.said) {
			t.Errorf("%s %v: %v, wanted %q", refusal.set, refusal.cameras,
				err, refusal.said)
		}
	}
}
