package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/hud"
)

// withHud is the fixture's set with a hud on it: a frame round the
// picture and a label pinned a metre over the stone, in a theme in the
// set's own directory; and each pair of changes after it made to the
// file, the first of each pair replaced by the second.
func withHud(t *testing.T, changes ...string) string {
	t.Helper()
	path := onFixture(t, "  set:\n", "  hud:\n"+
		"    theme: themes/dusk.css\n    rows: 4\n    elements:\n"+
		"      edge:\n        kind: frame\n        left: 0\n"+
		"        top: 0\n        right: 1\n        bottom: 1\n"+
		"        ink: ink\n"+
		"      stone:\n        kind: label\n        of: the stone\n"+
		"        above: 1\n        words: s\n        ink: ink\n"+
		"        ground: ground\n"+
		"      fire:\n        kind: bar\n        left: 0.1\n"+
		"        top: 0.75\n        right: 0.6\n        bottom: 1\n"+
		"        value:\n          from: 0\n          to: 1\n"+
		"          over: 4\n"+
		"        fill: ink\n        empty: ground\n"+
		"      clock:\n        kind: text\n        count: 3\n"+
		"        format: \"%.0f\"\n"+
		"        across: 0.9\n        down: 0\n"+
		"        align: right\n        ink: ink\n"+
		"  set:\n")
	themes := filepath.Join(filepath.Dir(path), "themes")
	os.MkdirAll(themes, 0o755)
	os.WriteFile(filepath.Join(themes, "dusk.css"), []byte(":root {\n"+
		"  --ink: #f0e0c0;\n  --ground: #ff00ff;\n}\n"), 0o644)
	return changed(t, path, changes...)
}

// changed is the set file at path with each pair of changes made to it,
// the first of each pair replaced by the second; and its path.
func changed(t *testing.T, path string, changes ...string) string {
	t.Helper()
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scene := string(written)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(scene, changes[i]) {
			t.Fatalf("the scene has no %q to change", changes[i])
		}
		scene = strings.Replace(scene, changes[i], changes[i+1], 1)
	}
	os.WriteFile(path, []byte(scene), 0o644)
	return path
}

// a set's hud is staged as written: its theme read from beside the set,
// its rows, its four elements, the label pinned a metre over the stone,
// the bar filling over four seconds and the count saying 3.
func TestASetsHudIsStaged(t *testing.T) {
	given := mustParse(t)
	given.set = withHud(t)
	_, _, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	made := given.overlay
	if made == nil || made.rows != 4 || len(made.made.Elements) != 4 {
		t.Fatalf("the hud staged is %+v", made)
	}
	label, ok := made.made.Elements[1].(hud.Label)
	if !ok || label.Point.Y < 1 || !strings.Contains(label.Words.At(0),
		"s") {
		t.Errorf("the label is %+v", made.made.Elements[1])
	}
	bar, ok := made.made.Elements[2].(hud.Bar)
	if !ok || bar.Value.At(2*time.Second) != 0.5 {
		t.Errorf("the bar is %+v", made.made.Elements[2])
	}
	clock, ok := made.made.Elements[3].(hud.Text)
	if !ok || clock.Words.At(0) != "3" || clock.Align != hud.Right {
		t.Errorf("the count is %+v", made.made.Elements[3])
	}
	if colour, err := made.theme.Colour("ground"); err != nil ||
		colour.G != 0 || colour.R != 255 {
		t.Errorf("the theme's ground is %v, %v", colour, err)
	}
}

// a hud whose theme is not there, whose label is pinned to a name not
// stood on the ground, or in a role the theme has not, is refused by
// name.
func TestASetsHudThatCannotBeStagedIsRefused(t *testing.T) {
	given := mustParse(t)
	given.set = withHud(t, "themes/dusk.css", "themes/gone.css")
	if _, _, _, err := staged(given); !errors.Is(err, hud.ErrHud) {
		t.Errorf("a theme not there: %v", err)
	}
	given = mustParse(t)
	given.set = withHud(t, "of: the stone", "of: the box", "  set:\n",
		"  set:\n    assets:\n      house:\n        name: the box\n")
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "not stood") {
		t.Errorf("a label pinned to an asset never stood: %v", err)
	}
	given = mustParse(t)
	given.set = withHud(t, "        ink: ink\n        ground: ground\n",
		"        ink: mauve\n        ground: ground\n")
	path := filepath.Join(t.TempDir(), "out.png")
	var said, wrong bytes.Buffer
	code := run([]string{"-set", given.set, "-out", path}, &said, &wrong)
	if code != 1 || !strings.Contains(wrong.String(), "mauve") {
		t.Errorf("a role the theme has not: %d, %s",
			code, wrong.String())
	}
}

// drawn with its hud, a set's picture is the picture without it in
// every cell the hud does not cover, to the byte; the label's magenta
// ground is drawn where the camera, ten metres off, sees the stone, and
// the frame's ink in its corner.
func TestASetsPictureCarriesItsHud(t *testing.T) {
	aimed := []string{"point_to: the tree", "point_to: the stone",
		"standoff_y: 0", "standoff_y: -10"}
	path := withHud(t, aimed...)
	with := printed(t, path)
	without := printed(t, changed(t, onFixture(t, "", ""), aimed...))
	cover := laid(t, path)
	magenta, inked := 0, 0
	for y := range with.Bounds().Dy() {
		for x := range with.Bounds().Dx() {
			got := with.RGBAAt(x, y)
			_, ok := cover.At(x/2, y/4)
			if !ok && got != without.RGBAAt(x, y) {
				t.Fatalf("pixel %d, %d moved, in no hud cell",
					x, y)
			}
			if got == (color.RGBA{R: 255, B: 255, A: 255}) {
				magenta++
			}
			if x < 2 && y < 4 && got == (color.RGBA{R: 240, G: 224,
				B: 192, A: 255}) {
				inked++
			}
		}
	}
	if magenta == 0 || inked == 0 {
		t.Errorf("%d magenta pixels and %d inked in the corner",
			magenta, inked)
	}
}

// laid is a set's hud laid out as hillside lays it on the set's
// picture, twelve cells by four for the fixture's 24 by 16 pixels.
func laid(t *testing.T, path string) hud.Cover {
	t.Helper()
	given := mustParse(t)
	given.set = path
	_, _, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	view := hud.View{Eye: *given.eye, Sensor: lensFor(given),
		Width: given.width, Height: given.height}
	cover, err := given.overlay.made.Cover(image.Rect(0, 0, 12, 4), 0,
		given.overlay.theme, view)
	if err != nil {
		t.Fatal(err)
	}
	return cover
}

// printed draws a set file to a png and reads it back.
func printed(t *testing.T, path string) *image.RGBA {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.png")
	var said, wrong bytes.Buffer
	if code := run([]string{"-set", path, "-out", out}, &said,
		&wrong); code != 0 {
		t.Fatalf("drawing gave %d: %s", code, wrong.String())
	}
	file, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(decoded.Bounds())
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			img.Set(x, y, decoded.At(x, y))
		}
	}
	return img
}
