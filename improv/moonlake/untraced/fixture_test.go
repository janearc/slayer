package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/explain"
	"github.com/janearc/slayer/improv/moonlake/rig"
)

// recipeSet is a walkthrough set with the two cameras the panes use, a
// north plan and a ridge, over three things; the film has no rig and
// sees 45 degrees across.
const recipeSet = `scene:
  name: moonlake
  renderer: gpu-native
  light: moon
  location:
    lat: 19.4165
    lon: -155.2485
  timeofday: 1900
  moon:
    azimuth: 109.65
    altitude: 2.3
    face: svs-moon-kit-color
  sun:
    azimuth: 304
    altitude: -12
  terrain:
    from: usgs-kilauea-2019-dem
    file: unpacked/kil_2019dem/kil_2019dem.tif
    west: 262900
    north: 2149000
    across: 2400
    down: 1700
    step: 4
  film:
    width: 640
    height: 360
    rays: 4
    wide: 45
  cameras:
    north plan:
      title: "straight down"
      orientation:
        point_to: the far tree
        alpha: -90
      location:
        loc_relative:
          object: the west rise
          standoff_x: 0
          standoff_y: 0
          standoff_z: 150
    ridge:
      title: "from the ridge"
      orientation:
        point_to: the far tree
        alpha: -6
      location:
        loc_relative:
          object: the west rise
          standoff_x: 0
          standoff_y: 0
          standoff_z: 20
  set:
    things:
      the west rise:
        kind: cobble
        seed: 6
        east: 263104.5
        north: 2148423.5
      the far tree:
        kind: cobble
        seed: 3
        east: 264832
        north: 2147806
`

// recipePlaces is the places table for the recipe set.
const recipePlaces = `of,kind,east,north,height
"the west rise",cobble,263104.5,2148423.5,1127.7
"the far tree",cobble,264832.0,2147806.0,1195.5
`

// recipe writes a walkthrough recipe folder, the cue and score files, a
// script and a places table into a directory, and says what to ask for.
func recipe(t *testing.T) asked {
	t.Helper()
	dir := t.TempDir()
	write := func(path, text string) string {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return full
	}
	write("sets/walk-north.yaml", recipeSet)
	write("sets/walk-ridge.yaml", recipeSet)
	still := image.NewRGBA(image.Rect(0, 0, 1620, 1620))
	for y := 0; y < 1620; y += 1 {
		for x := 0; x < 1620; x += 1 {
			still.SetRGBA(x, y, color.RGBA{uint8(x / 8), uint8(y / 8), 90, 255})
		}
	}
	for _, name := range []string{"north.png", "ridge.png"} {
		full := filepath.Join(dir, "stills", name)
		os.MkdirAll(filepath.Dir(full), 0o755)
		file, err := os.Create(full)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, still); err != nil {
			t.Fatal(err)
		}
		file.Close()
	}
	return asked{layout: "wide", stills: dir,
		score:  write("score.yaml", beatsEvery(400)),
		cues:   write("cues.yaml", "cues:\n  - {at_sec: 0, master: 1, moon: 1, moon_alt: 6}\n  - {at: \"1.1\", rig: 1, beams: 4}\n  - {at: \"3.1\", overhead: 1}\n"),
		places: write("places.csv", recipePlaces),
		script: testScript(t)}
}

// TestRecipeLoads reads the recipe into the parts of a frame: the wide
// layout's size, the film read at the pane's width with its shape kept,
// each drawn pane's four moves and the stills' parts, and the tall
// layout's own parts.
func TestRecipeLoads(t *testing.T) {
	given := recipe(t)
	made, err := partsOf(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.layout.Size() != image.Pt(1920, 1080) {
		t.Errorf("the layout is %v", made.layout.Size())
	}
	// the viewer pane is 960 by 540; the film's shape, 4096 by 3008, at
	// 960 across is 705 down
	if made.viewerSize != image.Pt(960, 705) {
		t.Errorf("the film is read at %v, want 960 by 705", made.viewerSize)
	}
	if len(made.above.moves) != 4 || len(made.moon.moves) != 4 {
		t.Errorf("moves: %d above, %d on the ridge", len(made.above.moves),
			len(made.moon.moves))
	}
	if made.above.part != stillParts["wide"][0] || made.moon.part != stillParts["wide"][1] {
		t.Errorf("parts %v and %v", made.above.part, made.moon.part)
	}
	if made.rigLayout != rig.NewLayout(eyeOverLake, towerDeck) {
		t.Error("the rig's layout is not the show's eye and decks")
	}
	// a plain grade passes a mid grey toward the blueprint's tint
	if got := made.above.grade(color.RGBA{255, 255, 255, 255}); got != (color.RGBA{70, 90, 130, 255}) {
		t.Errorf("the north plan grades white to %v", got)
	}
	if got := made.moon.grade(color.RGBA{255, 255, 255, 255}); got != (color.RGBA{8, 12, 26, 255}) {
		t.Errorf("the ridge grades white to %v", got)
	}
	given.layout = "tall"
	tall, err := partsOf(given)
	if err != nil {
		t.Fatal(err)
	}
	if tall.layout.Size() != image.Pt(1080, 1920) || tall.above.part != stillParts["tall"][0] {
		t.Errorf("the tall layout is %v with part %v", tall.layout.Size(),
			tall.above.part)
	}
	given.passes = filepath.Join(t.TempDir(), "none")
	if _, err := partsOf(given); err == nil {
		t.Error("passes that are not there were taken")
	}
}

// TestRecipeRefusals: a set whose cameras are not the panes', and a
// still that is not a picture, each stop partsOf.
func TestRecipeRefusals(t *testing.T) {
	given := recipe(t)
	if err := os.WriteFile(filepath.Join(given.stills, "stills", "ridge.png"),
		[]byte("not a picture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := partsOf(given); err == nil || !strings.Contains(err.Error(), "ridge.png") {
		t.Errorf("a broken still gave %v", err)
	}
	given = recipe(t)
	ridgePath := filepath.Join(given.stills, "sets", "walk-ridge.yaml")
	crested := strings.Replace(recipeSet, "    ridge:", "    crest:", 1)
	if err := os.WriteFile(ridgePath,
		[]byte(crested), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := partsOf(given); err == nil || !strings.Contains(err.Error(), "crest") {
		t.Errorf("a set with no ridge camera gave %v", err)
	}
	given = recipe(t)
	if err := os.WriteFile(filepath.Join(given.stills, "sets", "walk-north.yaml"),
		[]byte("scene: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := partsOf(given); err == nil {
		t.Error("a set that is not yaml was read")
	}
	given = recipe(t)
	scoreText := beatsEvery(20)
	if err := os.WriteFile(given.score, []byte(scoreText), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := partsOf(given); err == nil || !strings.Contains(err.Error(), "bar 9 beat 1") {
		t.Errorf("a score too short for the shots gave %v", err)
	}
	given = recipe(t)
	os.Remove(filepath.Join(given.stills, "stills", "north.png"))
	if _, err := partsOf(given); err == nil {
		t.Error("a missing still was read")
	}
}

// solid is a picture of a size, every pixel one colour.
func solid(width, height int, colour color.RGBA) *image.RGBA {
	picture := blank(width, height)
	for i := 0; i < len(picture.Pix); i += 4 {
		picture.Pix[i], picture.Pix[i+1], picture.Pix[i+2], picture.Pix[i+3] =
			colour.R, colour.G, colour.B, colour.A
	}
	return picture
}

// TestComposeLaysFourPanes: the film fills its pane, the panel is on the
// ground, the drawn panes carry the rig, and the pane of the whole frame
// is the layout's.
func TestComposeLaysFourPanes(t *testing.T) {
	made, err := partsOf(recipe(t))
	if err != nil {
		t.Fatal(err)
	}
	red := color.RGBA{200, 10, 10, 255}
	frame, err := made.compose(4, solid(960, 705, red))
	if err != nil {
		t.Fatal(err)
	}
	if frame.Rect.Size() != image.Pt(1920, 1080) {
		t.Fatalf("the frame is %v", frame.Rect)
	}
	if got := frame.RGBAAt(480, 300); got != red {
		t.Errorf("the film pane is %v, want %v", got, red)
	}
	ground, _ := explain.ThemeColour("ground")
	if got := frame.RGBAAt(1919, 539); got != ground {
		t.Errorf("the panel's corner is %v, want %v", got, ground)
	}
	for name, area := range map[string]image.Rectangle{
		"above": image.Rect(0, 540, 960, 1080),
		"moon":  image.Rect(960, 540, 1920, 1080)} {
		bright := 0
		for y := area.Min.Y; y < area.Max.Y; y++ {
			for x := area.Min.X; x < area.Max.X; x++ {
				if c := frame.RGBAAt(x, y); c.R > 120 || c.G > 120 {
					bright++
				}
			}
		}
		if bright == 0 {
			t.Errorf("the %s pane shows no drawn light", name)
		}
	}
	// the rig is cued on at bar one: the same moment before it differs
	quiet, err := made.compose(0.5, solid(960, 705, red))
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for i := range quiet.Pix {
		if quiet.Pix[i] != frame.Pix[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("the frame before the cues is the frame after them")
	}
}

// TestComposeShowsTheTracedPaneWhenTheLightFalls: with traced passes and
// the ship's light on, the above pane is the rendered set, and with the
// passes too small for its window the frame is refused.
func TestComposeShowsTheTracedPaneWhenTheLightFalls(t *testing.T) {
	made, err := partsOf(recipe(t))
	if err != nil {
		t.Fatal(err)
	}
	size := made.above.still.Rect.Size()
	plate := make([]float64, size.X*size.Y*tracedChannel)
	for i := range plate {
		plate[i] = 1e-9
	}
	mine := &traced{size: size, plate: plate, passes: map[string][]float64{
		"back-left": plate, "back-right": plate, "overhead": plate}}
	red := solid(960, 705, color.RGBA{200, 10, 10, 255})
	// the ship's light falls from bar three, 5 s
	drawn, err := made.compose(6, red)
	if err != nil {
		t.Fatal(err)
	}
	made.aboveTraced = mine
	rendered, err := made.compose(6, red)
	if err != nil {
		t.Fatal(err)
	}
	differ := 0
	for y := 540; y < 1080; y++ {
		for x := 0; x < 960; x++ {
			if drawn.RGBAAt(x, y) != rendered.RGBAAt(x, y) {
				differ++
			}
		}
	}
	if differ < 1000 {
		t.Errorf("the traced pane differs from the drawn in only %d pixels", differ)
	}
	if drawn.RGBAAt(480, 300) != rendered.RGBAAt(480, 300) {
		t.Error("the film pane changed with the traced pane")
	}
	made.aboveTraced = &traced{size: image.Pt(10, 10), plate: plate[:300],
		passes: mine.passes}
	if _, err := made.compose(6, red); err == nil {
		t.Error("passes too small for the window were taken")
	}
}

// TestRunRefusesWhatIsMissing: a run with no film or no song stops at the
// source that is missing, one way or the other.
func TestRunRefusesWhatIsMissing(t *testing.T) {
	given := recipe(t)
	given.film = filepath.Join(t.TempDir(), "none.mp4")
	given.png = filepath.Join(t.TempDir(), "frame.png")
	given.at = 4
	if err := run(given); err == nil {
		t.Error("a still from a missing film was made")
	}
	if _, err := os.Stat(given.png); err == nil {
		t.Error("a still was written for a film that is not there")
	}
	given.png = ""
	given.from, given.until = 1, 3
	given.song = filepath.Join(t.TempDir(), "none.wav")
	given.mp4 = filepath.Join(t.TempDir(), "out.mp4")
	if err := run(given); err == nil {
		t.Error("a film from a missing song was made")
	}
	given.layout = "round"
	if err := run(given); err == nil {
		t.Error("a layout that is not one was run")
	}
}
