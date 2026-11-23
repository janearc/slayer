//go:build draw

package door

// The terrain flags against a set that stands on terrain: a shelf of
// the test's own, a sources.yaml pinning two datasets to the bytes a
// server of the test's own holds, and no other network. No test here
// names the real shelf. stood_draw_test.go uses its set and its shelf.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/image/tiff"
)

// tendFixture is terrain's placed fixture: 100 by 70 heights two metres
// apart, its first corner at 1000.5 east, 9000.25 north, the height of
// the sample at a column and a row (column + 1000 row) / 8
// (terrain/raster_test.go).
const tendFixture = "../terrain/testdata/fixture-plain.tif"

// datasets is how many datasets tendScene reads from the shelf: its
// terrain, fixture, and its moon's face, moon-face.
const datasets = 2

// tendScene is a set standing on the fixture, lit by a moon of a face
// named moon-face, with two stones to look at.
const tendScene = `scene:
  name: on the fixture
  light: moon
  timeofday: 2200
  moon: {azimuth: 180, altitude: 35, face: moon-face}
  terrain:
    from: fixture
    file: fixture.tif
    west: 1001
    north: 8999
    across: 150
    down: 100
    step: 2
  film:
    width: 24
    height: 16
    rays: 1
    exposure: 100000
  cameras:
    stone:
      orientation:
        point_to: the far stone
        alpha: 0
      location:
        loc_relative:
          object: the stone
          standoff_x: 0
          standoff_y: 0
          standoff_z: 2
  set:
    things:
      the far stone:
        kind: cobble
        seed: 3
        east: 1049
        north: 8961
      the stone:
        kind: cobble
        seed: 4
        east: 1049
        north: 8991
`

// tendServer is a server on localhost that holds a terrain and a moon
// face, and counts every request it is sent, and those that are gets.
type tendServer struct {
	*httptest.Server
	requests, gets atomic.Int64
	held           map[string][]byte
}

// serve answers a get with the bytes held at the path, a head with
// their length, and counts the request.
func (server *tendServer) serve(answer http.ResponseWriter,
	request *http.Request) {
	server.requests.Add(one)
	body, found := server.held[request.URL.Path]
	if !found {
		http.NotFound(answer, request)
		return
	}
	answer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if request.Method != http.MethodHead {
		server.gets.Add(one)
		answer.Write(body)
	}
}

// tendFace is a small moon face as a tiff: four pixels across, two
// down, one pixel lit.
func tendFace(t *testing.T) []byte {
	t.Helper()
	const (
		across, down = 4, 2 // picked: a few pixels
		lit          = 200  // picked: well above black
	)
	picture := image.NewGray(image.Rect(nothing, nothing, across, down))
	picture.SetGray(one, one, color.Gray{Y: lit})
	var held bytes.Buffer
	if err := tiff.Encode(&held, picture, nil); err != nil {
		t.Fatal(err)
	}
	return held.Bytes()
}

// tendSetup is a folder with a set standing on the fixture, a
// sources.yaml beside its sets folder that pins the terrain and the moon
// face to the server's bytes, and a shelf that holds nothing. It names
// the shelf in the environment, and gives the set's path, the shelf's
// and the server.
func tendSetup(t *testing.T) (string, string, *tendServer) {
	t.Helper()
	ground, err := os.ReadFile(tendFixture)
	if err != nil {
		t.Fatal(err)
	}
	server := &tendServer{held: map[string][]byte{
		"/fixture.tif": ground, "/moon.tif": tendFace(t)}}
	server.Server = httptest.NewServer(http.HandlerFunc(server.serve))
	t.Cleanup(server.Close)
	root := t.TempDir()
	list := "sources:\n"
	for key, name := range map[string]string{"fixture": "/fixture.tif",
		"moon-face": "/moon.tif"} {
		sum := sha256.Sum256(server.held[name])
		list += "  - key: " + key + "\n    from: " + server.URL + name +
			"\n    licence: a test's own\n    sha256: " +
			hex.EncodeToString(sum[:]) + "\n    bytes: " +
			strconv.Itoa(len(server.held[name])) + "\n"
	}
	written(t, root, "sources.yaml", list)
	if err := os.MkdirAll(filepath.Join(root, "sets"),
		folderMode); err != nil {
		t.Fatal(err)
	}
	path := written(t, filepath.Join(root, "sets"), "scene.yaml",
		tendScene)
	shelf := filepath.Join(root, "shelf")
	t.Setenv("SLAYER_SHELF", shelf)
	return path, shelf, server
}

// shelved puts the server's two datasets on the shelf, as a fetch
// would, with no request made.
func shelved(t *testing.T, shelf string, server *tendServer) {
	t.Helper()
	if err := os.MkdirAll(shelf, folderMode); err != nil {
		t.Fatal(err)
	}
	for name, held := range map[string][]byte{
		"fixture.tif":   server.held["/fixture.tif"],
		"moon-face.tif": server.held["/moon.tif"]} {
		if err := os.WriteFile(filepath.Join(shelf, name), held,
			fileMode); err != nil {
			t.Fatal(err)
		}
	}
}

// tendRun runs a verb of the real tool on the set, in the set's own
// working folder, with the shelf named.
func tendRun(t *testing.T, verbName, path, shelf string,
	args ...string) ran {
	t.Helper()
	env := envIn(t, context.Background(), filepath.Dir(path),
		"SLAYER_SHELF", shelf)
	return running(Tool(env), append([]string{verbName, path}, args...)...)
}

// smallOnCPU is the flags that draw a set small and on the processor.
func smallOnCPU() []string {
	const (
		wide = "8" // picked: a few pixels, as the session's tests draw
		tall = "6" // picked: three quarters of the width
		rays = "1" // picked: one ray a pixel
	)
	return []string{"--width", wide, "--height", tall, "--rays", rays,
		"--renderer", "cpu"}
}

// pictured is where a draw of the set in a test puts its picture.
func pictured(path string) string {
	return filepath.Join(filepath.Dir(path), "picture.png")
}

// drawTending draws the set with the flags given, the picture to its
// own place.
func drawTending(t *testing.T, path, shelf string, flags ...string) ran {
	t.Helper()
	words := append(smallOnCPU(), "--out", pictured(path))
	return tendRun(t, "draw", path, shelf, append(words, flags...)...)
}

// mustHold fails the test for each word a run's output, its own or its
// error's, lacks.
func mustHold(t *testing.T, got ran, words ...string) {
	t.Helper()
	for _, word := range words {
		if !strings.Contains(got.said+got.wrongSaid, word) {
			t.Errorf("the output lacks %q: %q %q", word, got.said,
				got.wrongSaid)
		}
	}
}

// with --fetch-terrain the terrain and the moon face are fetched from
// the server, the output says what, the shelf holds the server's bytes
// and the picture is drawn.
func TestDrawFetchTerrainFetchesTheDatasetsThenDraws(t *testing.T) {
	path, shelf, server := tendSetup(t)
	got := drawTending(t, path, shelf, "--fetch-terrain")
	if got.code != nothing {
		t.Fatalf("draw gave %d: %s%s", got.code, got.said, got.wrongSaid)
	}
	mustHold(t, got, "fixture: fetched", "moon-face: fetched",
		server.URL+"/fixture.tif", server.URL+"/moon.tif")
	for name, key := range map[string]string{"fixture.tif": "/fixture.tif",
		"moon-face.tif": "/moon.tif"} {
		held, err := os.ReadFile(filepath.Join(shelf, name))
		if err != nil || !bytes.Equal(held, server.held[key]) {
			t.Errorf("the shelf's %s: %d bytes, %v", name, len(held), err)
		}
	}
	if _, err := os.Stat(pictured(path)); err != nil {
		t.Errorf("no picture was drawn: %v", err)
	}
	// each of the datasets takes at least one request
	if got := server.gets.Load(); got < datasets {
		t.Errorf("the fetch made %d gets for %d datasets", got, datasets)
	}
}

// the anti-test: with no terrain flag a set whose datasets are not on
// the shelf is not drawn and nothing is fetched or written: the refusal
// names the dataset, the path and the flag draw takes.
func TestDrawWithNoTerrainFlagFetchesNothingAndDrawsNothing(t *testing.T) {
	path, shelf, server := tendSetup(t)
	got := drawTending(t, path, shelf)
	if got.code != one {
		t.Fatalf("an absent dataset: draw gave %d: %s", got.code, got.said)
	}
	mustHold(t, got, "fixture", filepath.Join(shelf, "fixture.tif"),
		"--fetch-terrain")
	if requests := server.requests.Load(); requests != nothing {
		t.Errorf("a draw with no flag made %d requests", requests)
	}
	if _, err := os.Stat(shelf); err == nil {
		t.Error("a draw with no flag made the shelf")
	}
	if _, err := os.Stat(pictured(path)); err == nil {
		t.Error("a picture was drawn")
	}
}

// the anti-test: two terrain flags together are refused and the server
// is not asked for anything; the shelf is not made.
func TestDrawRefusesTwoTerrainFlagsAndFetchesNothing(t *testing.T) {
	path, shelf, server := tendSetup(t)
	got := drawTending(t, path, shelf, "--fetch-terrain",
		"--refresh-terrain")
	if got.code != one {
		t.Fatalf("two terrain flags: draw gave %d", got.code)
	}
	mustHold(t, got, "only one of --fetch-terrain")
	if requests := server.requests.Load(); requests != nothing {
		t.Errorf("a refused draw made %d requests", requests)
	}
	if _, err := os.Stat(shelf); err == nil {
		t.Error("a refused draw made the shelf")
	}
}

// --refresh-terrain fetches the datasets again and replaces a file
// that is not the source's with the source's, says what changed, and
// draws.
func TestDrawRefreshTerrainReplacesAWrongFileThenDraws(t *testing.T) {
	path, shelf, server := tendSetup(t)
	shelved(t, shelf, server)
	bent := filepath.Join(shelf, "fixture.tif")
	if err := os.WriteFile(bent, []byte("an old terrain"),
		fileMode); err != nil {
		t.Fatal(err)
	}
	got := drawTending(t, path, shelf, "--refresh-terrain")
	if got.code != nothing {
		t.Fatalf("draw gave %d: %s%s", got.code, got.said, got.wrongSaid)
	}
	mustHold(t, got, "changed", "sha256 was", "same bytes")
	held, err := os.ReadFile(bent)
	if err != nil || !bytes.Equal(held, server.held["/fixture.tif"]) {
		t.Errorf("the shelf holds %d bytes, %v", len(held), err)
	}
	if _, err := os.Stat(pictured(path)); err != nil {
		t.Errorf("no picture was drawn: %v", err)
	}
}

// --check-terrain draws nothing and downloads nothing: it exits 1 for a
// shelf with nothing on it, and 0 for one that is the source's, after
// one head request a dataset; it exits 1 again where one file is not
// the source's, saying so, and draws no picture throughout.
func TestAssessCheckTerrainSaysWhetherTheShelfAgrees(t *testing.T) {
	path, shelf, server := tendSetup(t)
	got := tendRun(t, "assess", path, shelf, "--check-terrain")
	if got.code != one {
		t.Errorf("an empty shelf: exit %d: %s%s", got.code, got.said,
			got.wrongSaid)
	}
	mustHold(t, got, "differ")
	shelved(t, shelf, server)
	before := server.requests.Load()
	got = tendRun(t, "assess", path, shelf, "--check-terrain")
	if got.code != nothing || strings.Count(got.said,
		"same as sources.yaml") != datasets {
		t.Errorf("a good shelf: exit %d: %s%s", got.code, got.said,
			got.wrongSaid)
	}
	// one head request for each of the two datasets, and no get
	if made := server.requests.Load() - before; made != datasets {
		t.Errorf("the check made %d requests, want %d heads", made,
			datasets)
	}
	if gets := server.gets.Load(); gets != nothing {
		t.Errorf("the check downloaded: %d gets", gets)
	}
	bent := filepath.Join(shelf, "moon-face.tif")
	if err := os.WriteFile(bent, []byte("another face"),
		fileMode); err != nil {
		t.Fatal(err)
	}
	got = tendRun(t, "assess", path, shelf, "--check-terrain")
	if got.code != one || !strings.Contains(got.said, "sha256 differs") {
		t.Errorf("a bent face: exit %d: %s%s", got.code, got.said,
			got.wrongSaid)
	}
	if _, err := os.Stat(pictured(path)); err == nil {
		t.Error("a check drew a picture")
	}
}
