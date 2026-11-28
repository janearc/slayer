//go:build cgo

package workspace

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

func TestSceneDetailSkipsUnrelatedManufactureAndUsesAuthoredAnchors(t *testing.T) {
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	if err := os.Mkdir(shelf, 0755); err != nil {
		t.Fatal(err)
	}
	terrain, err := os.ReadFile("../../../../terrain/testdata/fixture-plain.tif")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fixture.tif", "fixture-plain.tif"} {
		if err := os.WriteFile(filepath.Join(shelf, name), terrain, 0600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := fmt.Sprintf("sources:\n  - key: fixture\n    from: https://x.test/fixture.tif\n    licence: test\n    sha256: %x\n", sha256.Sum256(terrain))
	if err := os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_SHELF", shelf)
	t.Setenv("SLAYER_CACHE", t.TempDir())

	path := filepath.Join(root, "scene.yaml")
	source := groundEditFixture + "      unrelated: {kind: yurt, seed: 8, east: 1070, north: 8980, alight: true, scale: 2}\n"
	source += "      offscreen: {kind: yurt, seed: 8, east: 8, north: 8}\n"
	source += "    stands:\n      grove: {kind: yurt, count: 4, seed: 2, complexity: quick, within: {of: stone, from: 0, to: 360, near: 4, far: 8}}\n"
	ctx := context.Background()
	if _, err := StageSnapshot(ctx, path, source, "set"); err == nil || !strings.Contains(err.Error(), "unrelated: scale is not supported") {
		t.Fatalf("full staging did not check the unrelated placement: %v", err)
	}
	for _, id := range []string{"scene.set.things/stone", "scene.set.stands/grove"} {
		snapshot, err := StageDetailSnapshot(ctx, path, source, "set", stagewire.Camera{}, id)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, batch := range snapshot.Scene.Batches {
			if !batch.Authored {
				continue
			}
			if actual := batch.Author.Collection + "/" + batch.Author.Name; actual != id {
				t.Fatalf("detail %q manufactured unrelated geometry %q", id, actual)
			}
			found = true
		}
		if !found || snapshot.Scene.Focus == snapshot.Scene.Eye.Origin {
			t.Fatalf("detail %q did not frame the selection", id)
		}
		if anchor, ok := snapshot.Scene.Origins["scene.set.things/stone"]; !ok || anchor.X != 49 || anchor.Z != 9 {
			t.Fatalf("the camera and stand's reference anchor is %v (present %v)", anchor, ok)
		}
	}
}
