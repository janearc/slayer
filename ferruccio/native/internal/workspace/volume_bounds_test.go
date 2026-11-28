//go:build cgo

package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

func TestStageExistingFizzCarriesReadOnlyBounds(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path, err := filepath.Abs("../../../../sets/fizz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := StageSnapshot(context.Background(), path, string(source), "legacy-set")
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
	want := maths.NewBox(maths.V(-.8, 0, -.8), maths.V(.8, 2, .8))
	if len(scene.VolumeBounds) != 1 || scene.VolumeBounds[0].Box != want {
		t.Fatalf("native snapshot volume bounds = %+v", scene.VolumeBounds)
	}
	visible := visibleVolumeBounds(scene)
	if len(visible) != 1 || visible[0].label != "Fizz bounds" || len(visible[0].edges) == 0 {
		t.Fatalf("expected visible fizz edges, got %+v", visible)
	}
	if !strings.Contains(detached.Notice, volumeBoundsNotice) || len(scene.Batches) != 1 {
		t.Fatalf("expected one floor surface and a read-only volume notice: %q", detached.Notice)
	}
}

func TestLocalWorkingOpenReportsOmittedVolumeBounds(t *testing.T) {
	study, err := os.ReadFile("../../../../sets/guide/step-8.yaml")
	if err != nil {
		t.Fatal(err)
	}
	const cavities = "    vapor-cavities:\n"
	_, authored, ok := strings.Cut(string(study), cavities)
	if !ok {
		t.Fatal("existing producer fixture has no cavitation")
	}
	unsupported, _, ok := strings.Cut(authored, "  set:\n")
	if !ok {
		t.Fatal("existing producer fixture has no set boundary")
	}
	var many strings.Builder
	many.WriteString("    fizzes:\n")
	for i := range placement.VolumeBoundLimit + 1 {
		fmt.Fprintf(&many, "      fizz-%d: {colour: {swatch: paint}, foot: [0, 0, 0], radius: 0.4, height: 2, thick: 4, rise: 0.3, churn: 0.75, seed: 43, source: eye}\n", i)
	}
	for _, row := range []struct {
		name, volumes string
		shown         int
	}{
		{"unsupported only", cavities + unsupported, 0},
		{"over the limit", many.String(), placement.VolumeBoundLimit},
	} {
		t.Run(row.name, func(t *testing.T) {
			work := playWorkspace(t)
			work.state.document.Replace(checkingSource + "  world:\n" + row.volumes)
			work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
				return nil, fmt.Errorf("cached scene reached the helper")
			}
			defer work.closeStaged()
			work.updateWorking()
			waitStaged(t, work)
			scene := work.working.scene
			if scene == nil || len(scene.VolumeBounds) != row.shown || scene.OmittedVolumeBounds != 1 {
				t.Fatalf("cached volume coverage = %+v, diagnostic %q", scene, work.working.failure)
			}
			if !strings.Contains(work.working.notice, "Volume bounds omitted (1)") ||
				!strings.Contains(work.workingCaption(), "View details") {
				t.Fatalf("cached volume omission is missing from the details: %q", work.working.notice)
			}
		})
	}
}
