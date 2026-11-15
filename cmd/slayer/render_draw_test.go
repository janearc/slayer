//go:build draw

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// the fixture is terrain's: 100 by 70 heights two metres apart, whose
// first corner is at 1000.5 e, 9000.25 n.
const fixture = "../../terrain/testdata/fixture-plain.tif"

// a scene whose things half stands a spruce alight on the fixture's ground, a
// stone to stand by and a marker to look at, all three on one row of the
// ground, which climbs 125 m from row to row and a sixteenth of a metre a
// metre along one; and whose world half puts a glowing box, six metres on
// a side, at the spruce's foot.
const treeAndBox = `scene:
  name: a box at a tree's foot
  light: moon
  timeofday: 2200
  terrain:
    from: fixture
    file: fixture-plain.tif
    west: 1001.5
    north: 8995.25
    across: 150
    down: 100
    step: 2
  film:
    width: 48
    height: 32
    rays: 1
    exposure: 100000
  cameras:
    stone:
      orientation:
        point_to: the marker
        alpha: 0
      location:
        loc_relative:
          object: the stone
          standoff_x: 0
          standoff_y: 0
          standoff_z: 2
  set:
    things:
      the tree:
        kind: spruce
        seed: 3
        east: 1051.5
        north: 8955.25
        alight: true
      the stone:
        kind: cobble
        seed: 4
        east: 1031.5
        north: 8955.25
      the marker:
        kind: cobble
        seed: 5
        east: 1071.5
        north: 8955.25
  world:
    materials:
      crate:
        glow: {temperature: 2700, luminance: 400}
        source: eye
    boxes:
      crate:
        material: crate
        middle: [1051.5, 8955.25, 2756.125]
        size: [6, 6, 6]
        source: eye
`

// the pieces of the scene a test takes out to see the picture change.
const (
	theTree = "      the tree:\n        kind: spruce\n        seed: 3\n" +
		"        east: 1051.5\n        north: 8955.25\n        alight: true\n"
	theWorld = "  world:"
)

// once is how many places of a text a test takes a piece out of.
const once = 1

// lodge writes the scene onto a shelf with the fixture, pinned in a
// sources.yaml, and tells the process where the shelf is; it says where
// the set file is. the scene is changed as asked: a piece taken out, or
// the world half cut off.
func lodge(t *testing.T, without ...string) string {
	t.Helper()
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	model, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(shelf, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fixture.tif", "fixture-plain.tif"} {
		if err := os.WriteFile(filepath.Join(shelf, name), model,
			0o644); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(model)
	pinned := "sources:\n  - key: fixture\n    from: https://x.test/f.tif\n" +
		"    licence: a test's own\n    sha256: " +
		hex.EncodeToString(sum[:]) + "\n"
	if err := os.WriteFile(filepath.Join(root, "sources.yaml"),
		[]byte(pinned), 0o644); err != nil {
		t.Fatal(err)
	}
	scene := treeAndBox
	for _, piece := range without {
		scene = strings.Replace(scene, piece, "", once)
		if piece == theWorld {
			scene = scene[:strings.Index(treeAndBox, theWorld)]
		}
	}
	path := filepath.Join(root, "scene.yaml")
	if err := os.WriteFile(path, []byte(scene), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_SHELF", shelf)
	return path
}

// render of a set that says only a world writes the picture hillside
// writes for the same set and settings, byte for byte.
func TestRenderDrawsAWorldOnlySetAsHillsideDoes(t *testing.T) {
	got, code := renders(t, widow, longSettings)
	want, wanted := hillsides(t, widow, small)
	if code != nothing || wanted != nothing {
		t.Fatalf("render exited %d and hillside %d", code, wanted)
	}
	picture, err := png.Decode(bytes.NewReader(got))
	if err != nil || strconv.Itoa(picture.Bounds().Dx()) != smallWide ||
		strconv.Itoa(picture.Bounds().Dy()) != smallTall {
		t.Fatalf("render wrote %v, %v", picture, err)
	}
	if !bytes.Equal(got, want) {
		t.Error("render and hillside wrote different pictures")
	}
}

// render of a set with a box from its world half and a tree from its
// things half draws both: its picture is not the set's with the box
// taken out, nor the set's with the tree taken out.
func TestRenderDrawsTheBoxAndTheTree(t *testing.T) {
	whole, code := renders(t, lodge(t), "")
	if code != nothing {
		t.Fatalf("render exited %d", code)
	}
	bare, _ := renders(t, lodge(t, theWorld), "")
	felled, _ := renders(t, lodge(t, theTree), "")
	if bytes.Equal(whole, bare) {
		t.Error("the box does not show in the picture")
	}
	if bytes.Equal(whole, felled) {
		t.Error("the tree does not show in the picture")
	}
	if bytes.Equal(bare, felled) {
		t.Error("the picture with no box is the picture with no tree")
	}
	again, _ := renders(t, lodge(t), "")
	if !bytes.Equal(whole, again) {
		t.Error("the same set drew two pictures")
	}
}

// the size and rays render takes by name, as puzzlemouth's long flags
// say them: a value is the next word or follows an equals sign.
const (
	longSettings  = "--width " + smallWide + " --height " + smallTall + " --rays 1"
	equalSettings = "--width=" + smallWide + " --height=" + smallTall + " --rays=1"
)

// render's --width, --height and --rays set the size and rays, and draw the
// picture hillside draws for -w, -h and -n, byte for byte.
func TestRenderTakesTheSizeByLongName(t *testing.T) {
	want, wanted := hillsides(t, widow, small)
	if wanted != nothing {
		t.Fatalf("hillside exited %d", wanted)
	}
	for _, settings := range []string{longSettings, equalSettings} {
		got, code := renders(t, widow, settings)
		picture, err := png.Decode(bytes.NewReader(got))
		if code != nothing || err != nil ||
			strconv.Itoa(picture.Bounds().Dx()) != smallWide ||
			strconv.Itoa(picture.Bounds().Dy()) != smallTall {
			t.Fatalf("%q: exit %d, %v, %v", settings, code, picture, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%q drew a different picture from %q", settings, small)
		}
	}
}

// a modelServer sends a model to a GET and its length to a HEAD.
type modelServer struct{ model []byte }

// emptyShelfSet is the tree-and-box set on the fixture's ground, with a
// shelf that holds nothing and a sources.yaml pinning the fixture to a
// server on localhost that holds it. the environment names the shelf.
func emptyShelfSet(t *testing.T) (string, string) {
	t.Helper()
	model, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(modelServer{model: model})
	t.Cleanup(server.Close)
	root := t.TempDir()
	sum := sha256.Sum256(model)
	pinned := "sources:\n  - key: fixture\n    from: " + server.URL +
		"/f.tif\n    licence: a test's own\n    sha256: " +
		hex.EncodeToString(sum[:]) + "\n"
	if err := os.WriteFile(filepath.Join(root, "sources.yaml"),
		[]byte(pinned), 0o644); err != nil {
		t.Fatal(err)
	}
	scene := strings.Replace(treeAndBox, "file: fixture-plain.tif",
		"file: fixture.tif", once)
	path := filepath.Join(root, "scene.yaml")
	if err := os.WriteFile(path, []byte(scene), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_SHELF", filepath.Join(root, "shelf"))
	return path, filepath.Join(root, "picture.png")
}

// render takes --check-terrain, --refresh-terrain and --fetch-terrain by
// long name: with a shelf that lacks the terrain, a check fails and draws
// nothing, and a fetch fetches, says so and draws; staging alone, with
// none of them, downloads nothing and names the flag.
func TestRenderTakesTheTerrainFlagsByLongName(t *testing.T) {
	path, picture := emptyShelfSet(t)
	told, code := rendered(t, path, "-out "+picture)
	if code != exitFailed || !strings.Contains(told, "--fetch-terrain") {
		t.Errorf("no flag: exit %d: %s", code, told)
	}
	told, code = rendered(t, path, "--check-terrain -out "+picture)
	if code != exitFailed || !strings.Contains(told, "missing") {
		t.Errorf("check: exit %d: %s", code, told)
	}
	if _, err := os.Stat(picture); err == nil {
		t.Error("a check drew a picture")
	}
	told, code = rendered(t, path, "--fetch-terrain -out "+picture+" "+longSettings)
	if code != nothing || !strings.Contains(told, "fixture: fetched") {
		t.Fatalf("fetch: exit %d: %s", code, told)
	}
	if _, err := os.Stat(picture); err != nil {
		t.Errorf("a fetch drew no picture: %v", err)
	}
	told, code = rendered(t, path, "--refresh-terrain -out "+picture+" "+longSettings)
	if code != nothing || !strings.Contains(told, "same bytes") {
		t.Errorf("refresh: exit %d: %s", code, told)
	}
}

// ServeHTTP sends the model, or only its length for a HEAD.
func (held modelServer) ServeHTTP(answer http.ResponseWriter,
	request *http.Request) {
	length := strconv.Itoa(len(held.model))
	answer.Header().Set("Content-Length", length)
	if request.Method != http.MethodHead {
		answer.Write(held.model)
	}
}
