//go:build cgo

package workspace

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestGeneratedGiantNamesWorkingDepthLimit(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())

	const path = "../../../../doc/examples/giant-controls/giant-amber-v7.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Retain the producer's giant and disc camera. Voyager needs a source
	// shelf, so replace it with a small native foreground marker.
	before, after, found := strings.Cut(string(raw), "  set:\n")
	if !found {
		t.Fatal("expected the Voyager set in the giant control")
	}
	_, film, found := strings.Cut(after, "  film:")
	if !found {
		t.Fatal("expected the authored giant film")
	}
	source := before + "  film:" + film + `    spheres:
      foreground:
        material: marker
        middle: [-318489795, 0, 115920805]
        radius: 50
        source: eye
      distant:
        material: marker
        middle: [0, 0, 0]
        radius: 50
        source: eye
    materials:
      marker: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
`
	scenePath := filepath.Join(t.TempDir(), "giant.yaml")
	read, err := set.Read(scenePath, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	local, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	if got := localWorkingNotice(read, local); !strings.Contains(got, "World surfaces entirely beyond the nominal 100000 scene-unit depth limit at the staged camera (1)") || !strings.Contains(got, omittedGiantNotice(1)) {
		t.Fatalf("cached giant scene did not report clipped solid and omitted atmosphere: %q", got)
	}
	snapshot, err := StageSnapshot(t.Context(), scenePath, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Notice; !strings.Contains(got, "Geometry groups entirely beyond the nominal 100000 scene-unit depth limit at the staged camera (1)") || !strings.Contains(got, omittedGiantNotice(1)) {
		t.Fatalf("staged giant scene did not report clipped solid and omitted atmosphere: %q", got)
	}
	if len(snapshot.Scene.Batches) < 2 {
		t.Fatal("working staging discarded the foreground or distant solid")
	}
	if !strings.Contains(snapshot.Notice, giantBoundaryNotice) || !strings.Contains(localWorkingNotice(read, local), giantBoundaryNotice) {
		t.Fatal("stage or cache omitted the read-only overlay limits")
	}
	var wire bytes.Buffer
	if err := stagewire.Write(&wire, snapshot); err != nil {
		t.Fatal(err)
	}
	detached, err := stagewire.Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if len(local.GiantBoundaries) != 1 || len(detached.Scene.GiantBoundaries) != 1 ||
		local.GiantBoundaries[0] != detached.Scene.GiantBoundaries[0] {
		t.Fatal("cached and detached producer boundaries disagree")
	}
	for _, scene := range []*placement.DisplayScene{local, &detached.Scene} {
		boundary := scene.GiantBoundaries[0]
		edges := scene.ProjectGiantBoundary(boundary)
		outlines := visibleVolumeBounds(scene)
		if len(edges) < 90 || len(outlines) != 1 || outlines[0].label != "Gas giant boundary overlay" {
			t.Fatal("the producer giant did not project a labeled disc outline")
		}
		foreground, hit := scene.PickAt(.5, .5)
		if !hit || foreground.Distance > 1500 {
			t.Fatalf("lost the small foreground sphere: %+v", foreground)
		}
		if far, hit := scene.PickAt(.6, .5); hit {
			t.Fatalf("the empty giant disc became pickable: %+v", far)
		}
	}
}

func TestFlowingGiantShowsProducerGlobeOutline(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	raw, err := os.ReadFile("../../../../doc/examples/flowing-giant/first.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Replace the shelf-backed Voyager with a native placement. Keeping a
	// set retains the session route used by this flowing giant example.
	source := strings.ReplaceAll(string(raw), "kind: voyager-1", "kind: yurt") + `    spheres:
      foreground:
        material: marker
        middle: [-318489795, 0, 115920805]
        radius: 50
        source: eye
    materials:
      marker: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
`
	source = strings.Replace(source, "middle: [0, 0, 0]", "middle: [17000000, 12000000, 0]", 1)
	shelf := t.TempDir()
	t.Setenv("SLAYER_SHELF", shelf)
	table := []byte("300.4 300.31 .0000 .5\n619.2 619.01 .7132 .1\n")
	manifest := fmt.Sprintf("sources:\n  - key: karkoschka-1994-methane\n    from: https://example.test/table.txt\n    licence: fixture\n    sha256: %x\n", sha256.Sum256(table))
	if err := os.WriteFile(filepath.Join(shelf, "karkoschka-1994-methane.txt"), table, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shelf, "sources.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(shelf, "flowing.yaml")
	read, err := set.Read(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := placement.NewDisplayCache().Scene(read); err == nil || !strings.Contains(err.Error(), "flowing globes require producer staging") {
		t.Fatalf("lightweight cache did not defer to producer staging: %v", err)
	}
	world, _ := read.World()
	amber := world.Bodies()[0]
	input, _ := amber.Generated()
	derived, err := generate.Derive(input)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := StageSnapshot(t.Context(), path, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := stagewire.Write(&wire, snapshot); err != nil {
		t.Fatal(err)
	}
	detached, err := stagewire.Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	scene := &detached.Scene
	want := placement.DisplaySphereOutline{Centre: maths.V(17000000, 0, -12000000), Radius: derived.Radius.InKilometres() * 1000}
	if len(scene.GiantBoundaries) != 0 || len(scene.GlobeOutlines) != 1 || scene.GlobeOutlines[0] != want {
		t.Fatalf("unexpected outline metadata: %+v, want %+v", scene.GlobeOutlines, want)
	}
	outlines := visibleVolumeBounds(scene)
	if len(outlines) != 1 || len(outlines[0].edges) < 90 || outlines[0].label != "Globe outline overlay" || !strings.Contains(detached.Notice, globeOutlineNotice) {
		t.Fatalf("working view omitted the globe or its limits: %q, %d overlays", detached.Notice, len(outlines))
	}
	if foreground, hit := scene.PickAt(.5, .5); !hit || foreground.Distance > 1500 {
		t.Fatalf("lost nearby sphere: %+v", foreground)
	}
	if far, hit := scene.PickAt(.6, .5); hit {
		t.Fatalf("globe overlay blocked a pick: %+v", far)
	}
}
