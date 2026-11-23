package session

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	milestone "github.com/janearc/slayer/scene/smoke"
)

// The numbers the parts tests pick, each by the session slice's author,
// 2026-11-12.
const (
	// partsWidth and partsHeight are a staged frame the mock draws, and
	// partsAskedWidth the width an ask stands over it with.
	partsWidth      = 4
	partsHeight     = 3
	partsAskedWidth = 6
	// partsBelow is a number below zero, which an ask refuses.
	partsBelow = -1
)

// partsOnce is defined as one: a sink told once, a still's one frame.
const partsOnce = 1

// partsFrameWidth and partsFrameHeight are the rig fixture's frame, in
// millimetres, as onFixtureRigged writes it: defined by that fixture.
const (
	partsFrameWidth  = 70.41
	partsFrameHeight = 51.61
)

// an ask refuses a number below zero by name, and takes zeros.
func TestNewAskRefusesANumberBelowZero(t *testing.T) {
	for _, each := range []struct {
		name                string
		width, height, rays int
	}{
		{"width", partsBelow, nothing, nothing},
		{"height", nothing, partsBelow, nothing},
		{"rays", nothing, nothing, partsBelow},
	} {
		_, err := NewAsk(each.width, each.height, each.rays, nil)
		if err == nil || !strings.Contains(err.Error(), each.name) {
			t.Errorf("an ask's %s below zero gave %v", each.name, err)
		}
	}
	if ask, err := NewAsk(nothing, nothing, nothing, nil); err != nil ||
		ask.width != nothing || ask.height != nothing ||
		ask.rays != nothing || ask.sink != nil {
		t.Errorf("an ask of zeros gave %v", err)
	}
}

// a smoke with bands, a gauge or a palette of its own is refused by
// name, saying what judges it with them; a smoke with none is not.
func TestPlainSmokeRefusesWhatASessionWouldDrawWithout(t *testing.T) {
	for _, each := range []struct {
		scene milestone.Smoke
		says  string
	}{
		{milestone.Smoke{Bands: []maths.Band{{}}}, "bands"},
		{milestone.Smoke{Gauge: film.Super8}, "a gauge"},
		{milestone.Smoke{Palette: []maths.Light{{}}}, "a palette"},
	} {
		err := plainSmoke(each.scene)
		if err == nil || !strings.Contains(err.Error(), each.says) ||
			!strings.Contains(err.Error(), "slayer assess --smokes "+
				"judges it with them") {
			t.Errorf("a smoke with %s gave %v", each.says, err)
		}
	}
	if err := plainSmoke(milestone.Smoke{}); err != nil {
		t.Errorf("a plain smoke gave %v", err)
	}
}

// a rig's frame says its rows from its width, and an asked height that
// is not them is refused; with no rig the height is left alone.
func TestRigSizedTakesTheRowsFromTheWidth(t *testing.T) {
	made, err := lens.FrameOf(partsFrameWidth, partsFrameHeight)
	if err != nil {
		t.Fatal(err)
	}
	given := asked{width: sessionWidth, height: partsHeight, frame: &made}
	rows := made.Rows(sessionWidth)
	for _, height := range []int{nothing, rows} {
		sized, err := given.rigSized(height)
		if err != nil || sized.height != rows {
			t.Errorf("height %d on a rig gave %d: %v", height,
				sized.height, err)
		}
	}
	if _, err := given.rigSized(rows + partsOnce); err == nil {
		t.Error("a height that is not the rig's rows was taken")
	}
	given.frame = nil
	if sized, err := given.rigSized(nothing); err != nil ||
		sized.height != partsHeight {
		t.Errorf("no rig changed the height to %d", sized.height)
	}
}

// legacyState is sequenceState as it was before a job's key: a plan's
// manifest is written from it byte for byte as from sequenceState.
type legacyState struct {
	Plan, Scene, Build, Built    string
	Executable, Sources, Terrain string
	Width, Height, FPS, Frames   int
	Complete                     map[int]sequenceEntry
}

// a plan's manifest is written as it was, without a key; a resume
// refuses a manifest of another key, and takes its own.
func TestSequenceManifestKeyRefusesAnotherJob(t *testing.T) {
	entries := map[int]sequenceEntry{nothing: {SHA256: "a"}}
	state := sequenceState{Plan: "p", Scene: "s", Width: partsWidth,
		Height: partsHeight, FPS: partsWidth, Frames: partsHeight,
		Complete: entries}
	before := legacyState{Plan: "p", Scene: "s", Width: partsWidth,
		Height: partsHeight, FPS: partsWidth, Frames: partsHeight,
		Complete: entries}
	now, _ := json.MarshalIndent(state, "", "  ")
	was, _ := json.MarshalIndent(before, "", "  ")
	if !bytes.Equal(now, was) {
		t.Errorf("a plan's manifest changed:\n%s\nwas\n%s", now, was)
	}
	dir := t.TempDir()
	ofJob := state
	ofJob.Key = "one"
	if err := writeSequenceState(dir, &ofJob); err != nil {
		t.Fatal(err)
	}
	other := state
	other.Key = "two"
	if _, err := sequenceFolder(dir, other, true); err == nil ||
		!strings.Contains(err.Error(), `"two"`) {
		t.Errorf("a manifest of another key gave %v", err)
	}
	if _, err := sequenceFolder(dir, ofJob, true); err != nil {
		t.Errorf("a manifest of its own key gave %v", err)
	}
	if _, err := sequenceFolder(dir, state, true); err == nil {
		t.Error("a plan resumed a job's manifest")
	}
}

// a job resuming a plan's manifest, which holds no key, is refused.
func TestKeyedRefusesAJobOnAPlansManifest(t *testing.T) {
	if keyed(sequenceState{}, "") != nil {
		t.Error("a plan's manifest refused a plan")
	}
	if keyed(sequenceState{}, "one") == nil {
		t.Error("a job resumed a plan's manifest")
	}
}

// a manifest's file is written where the folder says.
func TestSequenceStateWritesItsFile(t *testing.T) {
	dir := t.TempDir()
	if err := writeSequenceState(dir, &sequenceState{Key: "one"}); err !=
		nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "sequence.json"))
	if err != nil || !bytes.Contains(raw, []byte(`"Key": "one"`)) {
		t.Errorf("a job's manifest holds no key: %v", err)
	}
}
