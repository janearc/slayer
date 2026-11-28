//go:build cgo

package workspace

import (
	"context"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/maths"
)

func TestSaveAsJobStagesTheOriginalReferencedScene(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	work, _, target := jobSaveFixture(t)
	if !work.saveAt(target) {
		t.Fatal(work.diagnostic)
	}
	reopened, path, err := ReadDocument(target)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := StageSnapshot(context.Background(), path, reopened.Text(), reopened.Kind())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := staged.Scene.Origins["scene.set.things/visitor"]; !ok {
		t.Fatal("reopened job did not stage its authored placement")
	}
}

func TestSaveAsImportStagesTheEditedPlacement(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	work, original, target := importedSaveFixture(t)
	source := work.state.document.Text()
	first, err := StageSnapshot(context.Background(), original, source, "set")
	if err != nil {
		t.Fatal(err)
	}
	work.editor.ApplyText(strings.Replace(source, "at: [7, 3, 5]", "at: [5, 3, 5]", 1))
	if !work.saveAt(target) {
		t.Fatal(work.diagnostic)
	}
	reopened, path, err := ReadDocument(target)
	if err != nil {
		t.Fatal(err)
	}
	second, err := StageSnapshot(context.Background(), path, reopened.Text(), reopened.Kind())
	if err != nil {
		t.Fatal(err)
	}
	delta := stagedImportPosition(t, second).Sub(stagedImportPosition(t, first))
	if !delta.Near(maths.V(-2, 0, 0), 1e-9) {
		t.Fatalf("reopened import moved %v, want two metres west", delta)
	}
}

func stagedImportPosition(t *testing.T, staged stagewire.Snapshot) maths.Vec {
	t.Helper()
	for _, batch := range staged.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.world.imports" &&
			batch.Author.Name == "figure" && len(batch.Places) != 0 {
			return batch.Places[0].Origin
		}
	}
	t.Fatal("reopened set did not stage its imported figure")
	return maths.Vec{}
}
