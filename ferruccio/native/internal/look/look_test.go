package look

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
	"gopkg.in/yaml.v3"
)

// lookHead is a space set up to its things; a camera looks at the origin.
const lookHead = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Look
  setting: space
  seed: 19
  light: none
  set:
    things:
`

// lookTail is the set's film and its one fixed camera, 60 m south of the
// origin and 20 m up, looking at a point 1 m up.
const lookTail = `  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -60, 20]}
      orientation: {look_at: [0, 0, 1]}
`

// lookSource is a set of count visitors, two metres apart along east.
func lookSource(count int) string {
	var things strings.Builder
	for index := range count {
		fmt.Fprintf(&things, "      visitor%d: {kind: human-scale-2m, seed: %d, complexity: quick, east: %d, north: 0, height: {value: 0, unit: m}}\n", index, index+1, 2*index)
	}
	return lookHead + things.String() + lookTail
}

// lookInput reads source as the window would and gives Describe all of it.
func lookInput(t *testing.T, source string) Input {
	t.Helper()
	read, err := set.Read("look.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open("look.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	return Input{When: time.Date(2026, 11, 20, 1, 52, 10, 0, time.UTC), Text: source, File: "/sets/look.yaml", Display: "wire", Composition: "play", Scene: scene, Objects: objects, Things: read.Things()}
}

func TestDescribeHoldsTheDraftsHashAndTheWindowsSettings(t *testing.T) {
	input := lookInput(t, lookSource(1))
	described := Describe(input)
	sum := sha256.Sum256([]byte(input.Text))
	if described.DraftSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("the hash is %s", described.DraftSHA256)
	}
	if described.File != "/sets/look.yaml" || described.Display != "wire" || described.Composition != "play" || !described.When.Equal(input.When) {
		t.Fatalf("the settings were not kept: %+v", described)
	}
}

func TestDescribeGivesTheCameraInTheSetsOwnCoordinates(t *testing.T) {
	camera := Describe(lookInput(t, lookSource(1))).Camera
	want := [3]float64{0, -60, 20}
	for axis := range want {
		if math.Abs(camera.Eye[axis]-want[axis]) > 1e-6 {
			t.Fatalf("the eye is %v, not %v", camera.Eye, want)
		}
	}
	if math.Abs(camera.WideDegrees-40) > 1e-6 || camera.Width != 640 || camera.Height != 400 {
		t.Fatalf("the lens or picture is wrong: %+v", camera)
	}
}

func TestDescribeSaysWhatItDoesNotKnowAndWhy(t *testing.T) {
	described := Describe(Input{Text: "x", SceneWhy: "the working view has not been built", PointerWhy: "the pointer has not been over the picture"})
	for label, got := range map[string]string{"camera": described.Camera.Why, "pointer": described.Pointer.Why, "in view": described.SeenWhy} {
		if !strings.HasPrefix(got, "not known: ") {
			t.Fatalf("%s says %q", label, got)
		}
	}
	if !strings.Contains(described.Camera.Why, "has not been built") || !strings.Contains(described.Pointer.Why, "has not been over") {
		t.Fatalf("the reasons were lost: %+v", described)
	}
}

func TestDescribeNamesTheSelectedThingFromTheSetAndTheDisplay(t *testing.T) {
	input := lookInput(t, lookSource(2))
	input.SelectedID = input.Objects[1].ID()
	chosen := Describe(input).Selected
	if chosen == nil || chosen.Name != "visitor1" || chosen.Kind != "human-scale-2m" || !chosen.SetKnown || chosen.Seed != 2 || chosen.Detail != "quick" {
		t.Fatalf("the selection was %+v", chosen)
	}
	if !chosen.StandsKnown || math.Abs(chosen.Stands[0]-2) > 1e-6 {
		t.Fatalf("it stands at %v, not two metres east", chosen.Stands)
	}
	input.SelectedID = ""
	if Describe(input).Selected != nil {
		t.Fatal("something was selected when nothing was")
	}
	input.SelectedID = "scene.set.things/gone"
	if Describe(input).Selected != nil {
		t.Fatal("a placement that is not in the draft was described")
	}
}

func TestDescribeTakesTheDisplaysPositionOfADraggedThing(t *testing.T) {
	input := lookInput(t, lookSource(1))
	input.SelectedID = input.Objects[0].ID()
	input.Scene.Objects[0].At.Origin.X += 5
	if got := Describe(input).Selected.Stands[0]; math.Abs(got-5) > 1e-6 {
		t.Fatalf("the look stood the thing at %v, where the draft has it, not the display", got)
	}
}

func TestDescribeFindsWhatIsUnderThePointer(t *testing.T) {
	input := lookInput(t, lookSource(1))
	input.Pointer = &[2]float64{.5, .5}
	found := Describe(input).Pointer
	if found.Over != "visitor0" || found.Distance < 30 || found.Distance > 70 {
		t.Fatalf("under the picture's centre: %+v", found)
	}
	input.Pointer = &[2]float64{.01, .01}
	if empty := Describe(input).Pointer; empty.Over != "" || empty.Why != "" {
		t.Fatalf("the corner of the sky was over something: %+v", empty)
	}
}

func TestDescribeCapsWhatIsInViewAndCountsTheRest(t *testing.T) {
	count := MostSeen + 3
	source := strings.Replace(lookSource(count), "wide: 40", "wide: 90", 1)
	input := lookInput(t, source)
	if got := len(input.Scene.InView()); got != count {
		t.Fatalf("test scene has %d things in view, want %d", got, count)
	}
	described := Describe(input)
	if len(described.Seen) != MostSeen || described.More != 3 {
		t.Fatalf("listed %d and left out %d", len(described.Seen), described.More)
	}
	for index := 1; index < len(described.Seen); index++ {
		if described.Seen[index-1].Distance > described.Seen[index].Distance {
			t.Fatal("the things in view are not nearest first")
		}
	}
	if described.Seen[0].Kind != "human-scale-2m" {
		t.Fatalf("the kind was not taken from the set: %+v", described.Seen[0])
	}
}

func TestRenderIsOneYamlDocumentThatStartsWithTheDivider(t *testing.T) {
	input := lookInput(t, lookSource(3))
	input.SelectedID = input.Objects[0].ID()
	input.Pointer = &[2]float64{.5, .5}
	text := Render(Describe(input))
	if !strings.HasPrefix(text, Divider+"\nlook: 2026-11-20T01:52:10Z\n") || !strings.HasSuffix(text, "\n") {
		t.Fatalf("the block does not begin and end as a look does:\n%s", text)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("a look is not YAML: %v\n%s", err, text)
	}
	for _, key := range []string{"look", "draft_sha256", "file", "display", "composition", "camera", "selected", "pointer", "in_view", "more_in_view", "note"} {
		if _, found := parsed[key]; !found {
			t.Fatalf("the look has no %s:\n%s", key, text)
		}
	}
}

func TestRenderOfAnUnknownViewIsStillYamlAndSaysItIsNotKnown(t *testing.T) {
	text := Render(Describe(Input{Text: "x", SceneWhy: "working display: refused", PointerWhy: "none: yet"}))
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("a look with unknowns is not YAML: %v\n%s", err, text)
	}
	if !strings.Contains(text, `camera: "not known: working display: refused"`) {
		t.Fatalf("the camera does not say why it is unknown:\n%s", text)
	}
	if !strings.Contains(text, "selected: none") {
		t.Fatalf("nothing selected is not said:\n%s", text)
	}
}

func TestRenderDoesNotClaimANoteWasTaken(t *testing.T) {
	text := Render(Describe(Input{Text: "x"}))
	if !strings.Contains(text, "note: \"not recorded") {
		t.Fatalf("the note's absence is not said:\n%s", text)
	}
}

func TestQuoteKeepsOddNamesOnOneLine(t *testing.T) {
	for _, name := range []string{"tower: north", "line\nbreak", "#hash", "", "true", "it's", "two  spaces", "tab\there", "雪", "null\x00byte"} {
		var back map[string]string
		if err := yaml.Unmarshal([]byte("name: "+quote(name)+"\n"), &back); err != nil {
			t.Fatalf("%q does not read back: %v", name, err)
		}
		if strings.Contains(quote(name), "\n") {
			t.Fatalf("%q quotes to more than one line: %q", name, quote(name))
		}
		if back["name"] != name {
			t.Fatalf("%q reads back as %q", name, back["name"])
		}
	}
}

func TestPathForAndPlaceNeverNameTheSetItself(t *testing.T) {
	if got := PathFor("/a/sets/neon-street.yaml"); got != "/a/sets/neon-street.looks.txt" {
		t.Fatalf("got %s", got)
	}
	if strings.HasSuffix(PathFor("/a/neon-street.yml"), ".yaml") || strings.HasSuffix(PathFor("/a/neon-street.yml"), ".yml") {
		t.Fatal("a looks file has a yaml name, so a folder of sets would take it for one")
	}
	saved, err := Place("/a/sets/neon-street.yaml", "/evidence", false)
	if err != nil || saved != "/a/sets/neon-street.looks.txt" {
		t.Fatalf("a saved set's look went to %q, %v", saved, err)
	}
	example, err := Place("/tmp/atelier-example-1/fizz.yaml", "/evidence", true)
	if err != nil || example != "/evidence/fizz.looks.txt" {
		t.Fatalf("an example's look went to %q, %v", example, err)
	}
	if _, err := Place("/tmp/x/fizz.yaml", "", true); err == nil {
		t.Fatal("an example with no evidence directory was given a place")
	}
	if _, err := Place("", "/evidence", false); err == nil {
		t.Fatal("a look with no file was given a place")
	}
}

func TestAppendKeepsEarlierLooksAndStartsNewOnesOnALine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "set.looks.txt")
	first, second := Divider+"\nlook: one\n", Divider+"\nlook: two\n"
	if err := Append(path, first); err != nil {
		t.Fatal(err)
	}
	if err := Append(path, second); err != nil {
		t.Fatal(err)
	}
	blocks, err := Read(path)
	if err != nil || len(blocks) != 2 || blocks[0] != first || blocks[1] != second {
		t.Fatalf("the looks were %q, %v", blocks, err)
	}
	if err := os.WriteFile(path, []byte(Divider+"\nlook: cut off"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Append(path, second); err != nil {
		t.Fatal(err)
	}
	blocks, _ = Read(path)
	if len(blocks) != 2 || blocks[1] != second {
		t.Fatalf("a block ran into the line before it: %q", blocks)
	}
}

func TestReadNamesAFileThatIsNotThereAndAFileWithNoLooks(t *testing.T) {
	directory := t.TempDir()
	if _, err := Read(filepath.Join(directory, "none.looks.txt")); err == nil || !strings.Contains(err.Error(), "none.looks.txt") {
		t.Fatalf("a missing file was not named: %v", err)
	}
	empty := filepath.Join(directory, "empty.looks.txt")
	if err := os.WriteFile(empty, []byte("just words\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(empty); err == nil {
		t.Fatal("text with no divider was read as a look")
	}
}

func TestBlocksIgnoreTextBeforeTheFirstDivider(t *testing.T) {
	blocks := Blocks("preamble\n---\na: 1\n---\nb: 2\n")
	if len(blocks) != 2 || blocks[0] != "---\na: 1\n" || blocks[1] != "---\nb: 2\n" {
		t.Fatalf("got %q", blocks)
	}
}
