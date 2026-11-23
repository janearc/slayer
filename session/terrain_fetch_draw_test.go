//go:build draw

package session_test

import (
	"bytes"
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

	"github.com/janearc/slayer/session"
	"golang.org/x/image/tiff"
)

// the numbers the tests count and write with.
const (
	// fileMode is the mode of a file a test writes; defined: the owner
	// writes, all others read.
	fileMode = 0o644
	// dirMode is the mode of a folder a test makes; defined: the owner
	// writes, all others read.
	dirMode = 0o755
	// none is zero; defined: no requests, and the exit number of a run
	// that held.
	none = 0
	// once is one; defined: one request, and the exit number of a run
	// that failed.
	once = 1
	// twice is two; defined: two requests or datasets, and the exit
	// number of a flag a run could not take.
	twice = 2
)

// terrainFixture is terrain's placed fixture: 100 by 70 heights two
// metres apart, its first corner at 1000.5 e, 9000.25 n.
const terrainFixture = "../terrain/testdata/fixture-plain.tif"

// terrainScene is a set standing on the fixture, lit by a moon of a face
// named moon-face, with a stone to look at.
const terrainScene = `scene:
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
        east: 1050
        north: 8960
      the stone:
        kind: cobble
        seed: 4
        east: 1050
        north: 8990
`

// fakeTerrain is a server on localhost that holds a terrain and a moon
// face, and counts every request it is sent.
type fakeTerrain struct {
	*httptest.Server
	requests atomic.Int64
	held     map[string][]byte
}

// moonTiff is a small moon face, as a tiff.
func moonTiff(t *testing.T) []byte {
	t.Helper()
	const (
		// wide is the face's width in pixels; picked: a few.
		wide = 4
		// tall is the face's height in pixels; picked: two rows.
		tall = 2
		// bright is the one lit pixel's grey; picked: well above black.
		bright = 200
	)
	picture := image.NewGray(image.Rect(0, 0, wide, tall))
	picture.SetGray(1, 1, color.Gray{Y: bright})
	var held bytes.Buffer
	if err := tiff.Encode(&held, picture, nil); err != nil {
		t.Fatal(err)
	}
	return held.Bytes()
}

// terrainSet is a set file standing on terrain's fixture, a shelf that
// holds nothing, a sources.yaml pinning the terrain and the moon face to
// the bytes a fake server holds, and the server. the environment names
// the shelf. it gives the set's path and the shelf's.
func terrainSet(t *testing.T) (string, string, *fakeTerrain) {
	t.Helper()
	ground, err := os.ReadFile(terrainFixture)
	if err != nil {
		t.Fatal(err)
	}
	server := &fakeTerrain{held: map[string][]byte{
		"/fixture.tif": ground, "/moon.tif": moonTiff(t)}}
	server.Server = httptest.NewServer(http.HandlerFunc(server.serve))
	t.Cleanup(server.Close)
	root := t.TempDir()
	text := "sources:\n"
	for key, name := range map[string]string{"fixture": "/fixture.tif",
		"moon-face": "/moon.tif"} {
		digest := sha256.Sum256(server.held[name])
		pin := hex.EncodeToString(digest[:])
		size := strconv.Itoa(len(server.held[name]))
		text += "  - key: " + key + "\n    from: " + server.URL + name +
			"\n    licence: a test's own\n    sha256: " + pin +
			"\n    bytes: " + size + "\n"
	}
	write(t, filepath.Join(root, "sources.yaml"), text)
	if err := os.MkdirAll(filepath.Join(root, "sets"), dirMode); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "sets", "scene.yaml")
	write(t, path, terrainScene)
	shelf := filepath.Join(root, "shelf")
	t.Setenv("SLAYER_SHELF", shelf)
	return path, shelf, server
}

// write puts text in a file.
func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), fileMode); err != nil {
		t.Fatal(err)
	}
}

// serve answers a GET with the bytes held at the path, a HEAD with their
// length, and counts the request.
func (server *fakeTerrain) serve(answer http.ResponseWriter,
	request *http.Request) {
	server.requests.Add(once)
	body, found := server.held[request.URL.Path]
	if !found {
		http.NotFound(answer, request)
		return
	}
	length := strconv.Itoa(len(body))
	answer.Header().Set("Content-Length", length)
	if request.Method != http.MethodHead {
		answer.Write(body)
	}
}

// pictureOf is where a set's test picture goes.
func pictureOf(path string) string {
	return filepath.Join(filepath.Dir(path), "picture.png")
}

// drawn is what Run says, rightly and wrongly, and the number it exits
// with, for the set drawn small to a picture in its own directory.
func drawn(t *testing.T, path string, flags ...string) (string, string, int) {
	t.Helper()
	arguments := append([]string{"-set", path, "-w", "8", "-h", "6", "-n",
		"1", "-out", pictureOf(path)}, flags...)
	var said, wrong bytes.Buffer
	stamp := session.Stamped("", "", "", "")
	code := session.Run(arguments, &said, &wrong, stamp)
	return said.String(), wrong.String(), code
}

// mustSay fails for each word the text lacks.
func mustSay(t *testing.T, text string, words ...string) {
	t.Helper()
	for _, word := range words {
		if !strings.Contains(text, word) {
			t.Errorf("the output lacks %q:\n%s", word, text)
		}
	}
}

// a set whose terrain is not on the shelf is not staged, and downloads
// nothing: the error names the dataset, the path it looked at, and the flag.
func TestStagingAnAbsentTerrainFailsNamingTheFlagAndDownloadsNothing(t *testing.T) {
	path, shelf, server := terrainSet(t)
	_, wrong, code := drawn(t, path)
	if code != once {
		t.Fatalf("a set whose terrain is not on the shelf: exit %d", code)
	}
	mustSay(t, wrong, "fixture", filepath.Join(shelf, "fixture.tif"),
		"--fetch-terrain")
	if got := server.requests.Load(); got != none {
		t.Errorf("staging made %d requests", got)
	}
}

// a set whose moon face is not on the shelf fails the same way.
func TestStagingAnAbsentMoonFaceFailsNamingTheFlagAndDownloadsNothing(t *testing.T) {
	path, shelf, server := terrainSet(t)
	if err := os.MkdirAll(shelf, dirMode); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(shelf, "fixture.tif"), string(server.held["/fixture.tif"]))
	_, wrong, code := drawn(t, path)
	if code != once {
		t.Fatalf("a set whose moon face is not on the shelf: exit %d", code)
	}
	mustSay(t, wrong, "moon-face", filepath.Join(shelf, "moon-face.tif"),
		"--fetch-terrain")
	if got := server.requests.Load(); got != none {
		t.Errorf("staging made %d requests", got)
	}
}

// with the flag the terrain and the moon face are fetched from the fake,
// the output says what, and the picture is drawn.
func TestFetchTerrainFetchesTerrainAndMoonFaceThenDraws(t *testing.T) {
	path, shelf, server := terrainSet(t)
	said, wrong, code := drawn(t, path, "-fetch-terrain")
	if code != none {
		t.Fatalf("exit %d: %s%s", code, said, wrong)
	}
	mustSay(t, said, "fixture: fetched", "moon-face: fetched",
		server.URL+"/fixture.tif", server.URL+"/moon.tif", "bytes", "sha256")
	for _, name := range []string{"fixture.tif", "moon-face.tif"} {
		if _, err := os.Stat(filepath.Join(shelf, name)); err != nil {
			t.Error(err)
		}
	}
	if _, err := os.Stat(pictureOf(path)); err != nil {
		t.Errorf("no picture was drawn: %v", err)
	}
}

// with the flag, a dataset already there and right is not fetched.
func TestFetchTerrainLeavesWhatIsOnTheShelfAndSaysSo(t *testing.T) {
	path, _, server := terrainSet(t)
	if _, wrong, code := drawn(t, path, "-fetch-terrain"); code != none {
		t.Fatalf("first fetch: exit %d: %s", code, wrong)
	}
	before := server.requests.Load()
	said, wrong, code := drawn(t, path, "-fetch-terrain")
	if code != none {
		t.Fatalf("exit %d: %s", code, wrong)
	}
	if strings.Count(said, "not fetched") != twice {
		t.Errorf("the output does not say both are not fetched:\n%s", said)
	}
	if got := server.requests.Load(); got != before {
		t.Errorf("a present terrain made %d requests", got-before)
	}
}

// a download that is not what sources.yaml pins fails the render, leaves
// nothing on the shelf and draws no picture.
func TestFetchTerrainRefusesACorruptDownloadAndDrawsNothing(t *testing.T) {
	path, shelf, server := terrainSet(t)
	server.held["/fixture.tif"] = []byte("not the terrain")
	_, wrong, code := drawn(t, path, "-fetch-terrain")
	if code != once || !strings.Contains(wrong, "fixture") {
		t.Errorf("exit %d: %s", code, wrong)
	}
	if _, err := os.Stat(filepath.Join(shelf, "fixture.tif")); err == nil {
		t.Error("the refused terrain is on the shelf")
	}
	if _, err := os.Stat(pictureOf(path)); err == nil {
		t.Error("a picture was drawn")
	}
}

// refresh fetches again and says the bytes are the same, or what changed.
func TestRefreshTerrainFetchesAgainAndSaysWhatChanged(t *testing.T) {
	path, shelf, server := terrainSet(t)
	if _, wrong, code := drawn(t, path, "-fetch-terrain"); code != none {
		t.Fatalf("first fetch: exit %d: %s", code, wrong)
	}
	said, wrong, code := drawn(t, path, "-refresh-terrain")
	if code != none || strings.Count(said, "same bytes") != twice {
		t.Fatalf("exit %d: %s%s", code, said, wrong)
	}
	write(t, filepath.Join(shelf, "fixture.tif"), "an old terrain")
	said, wrong, code = drawn(t, path, "-refresh-terrain")
	if code != none {
		t.Fatalf("exit %d: %s%s", code, said, wrong)
	}
	mustSay(t, said, "changed", "sha256 was")
	got, err := os.ReadFile(filepath.Join(shelf, "fixture.tif"))
	if err != nil || !bytes.Equal(got, server.held["/fixture.tif"]) {
		t.Errorf("the shelf holds %d bytes, %v", len(got), err)
	}
}

// check downloads nothing and draws nothing: it exits 0 when the shelf is
// the source's, and 1 when it differs or is missing.
func TestCheckTerrainComparesAndDrawsNothing(t *testing.T) {
	path, shelf, server := terrainSet(t)
	_, wrong, code := drawn(t, path, "-check-terrain")
	if code != once || !strings.Contains(wrong, "differ") {
		t.Errorf("a missing shelf: exit %d: %s", code, wrong)
	}
	if _, wrong, code := drawn(t, path, "-fetch-terrain"); code != none {
		t.Fatalf("fetch: exit %d: %s", code, wrong)
	}
	os.Remove(pictureOf(path))
	before := server.requests.Load()
	said, wrong, code := drawn(t, path, "-check-terrain")
	if code != none || strings.Count(said, "same as sources.yaml") != twice {
		t.Errorf("a good shelf: exit %d: %s%s", code, said, wrong)
	}
	if got := server.requests.Load() - before; got != twice {
		t.Errorf("check made %d requests, want 2 HEADs", got)
	}
	write(t, filepath.Join(shelf, "moon-face.tif"), "another face")
	said, wrong, code = drawn(t, path, "-check-terrain")
	if code != once || !strings.Contains(said, "sha256 differs") {
		t.Errorf("a bent face: exit %d: %s%s", code, said, wrong)
	}
	if _, err := os.Stat(pictureOf(path)); err == nil {
		t.Error("check drew a picture")
	}
}

// a refusal is a run of the command that must fail with a number and say
// a thing.
type refusal struct {
	name      string
	arguments []string
	says      string
}

// run is the command run on the arguments: exit 2, and the words said.
func (refused refusal) run(t *testing.T) {
	var said, wrong bytes.Buffer
	stamp := session.Stamped("", "", "", "")
	code := session.Run(refused.arguments, &said, &wrong, stamp)
	if code != twice || !strings.Contains(wrong.String(), refused.says) {
		t.Errorf("%s: exit %d: %s", refused.name, code, wrong.String())
	}
}

// the terrain flags are for a set: with none, or two at once, they are
// refused by name, exit 2.
func TestTerrainFlagsAreRefusedWithoutASetOrTogether(t *testing.T) {
	path, _, _ := terrainSet(t)
	for _, refused := range []refusal{
		{"no set", []string{"-fetch-terrain"}, "need -set"},
		{"two flags", []string{"-set", path, "-fetch-terrain",
			"-check-terrain"}, "only one of"},
	} {
		t.Run(refused.name, refused.run)
	}
}
