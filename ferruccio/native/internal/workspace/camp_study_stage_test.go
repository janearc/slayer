//go:build cgo

package workspace

import (
	"bytes"
	"context"
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

func TestBundledVisitorStagesAnEditableSelection(t *testing.T) {
	for _, name := range []string{campExample, playgroundExample} {
		t.Run(name, func(t *testing.T) {
			checkBundledVisitorSelection(t, name)
		})
	}
}

func checkBundledVisitorSelection(t *testing.T, name string) {
	t.Helper()
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path, study, err := copyBundledStudy("../../../examples", name)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path))
	document, _, err := ReadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := StageSnapshot(context.Background(), path, document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	before = campWireRoundTrip(t, before)
	first := campVisitorOrigin(t, before)
	work := playWorkspace(t)
	work.state.document, work.state.path = document, path
	work.editor.SetText(document.Text())
	work.play.selected = ""
	work.working.scene, work.working.path = &before.Scene, path
	work.working.source, work.working.displayed = document.Text(), true
	width, height := before.Scene.Width, before.Scene.Height
	work.viewer.set(image.NewNRGBA(image.Rect(0, 0, width, height)))
	work.viewer.SetBounds(geometry.Rect{Max: geometry.Pt(float32(width), float32(height))})
	point := campPickPoint(t, work)
	ctx := work.ui.Window().Context()
	press := event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone)
	release := event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone)
	if !work.viewer.Event(ctx, press) {
		t.Fatal("camp pointer press was not handled")
	}
	work.viewer.Event(ctx, release)
	if work.play.selected != campVisitorID {
		t.Fatalf("camp pointer selected %q", work.play.selected)
	}
	if err := work.changePose(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(t.TempDir(), "camp.yaml")
	if err := saveBundledStudyDocument(document, saved, filepath.Dir(path), study); err != nil {
		t.Fatal(err)
	}
	after, err := StageSnapshot(context.Background(), saved, document.Text(), "set")
	if err != nil {
		t.Fatal(err)
	}
	delta := campVisitorOrigin(t, after).Sub(first)
	if !delta.Near(maths.V(1, 0, 0), 1e-9) || math.Abs(first.Y) > 1e-9 {
		t.Fatalf("grounded visitor delta = %v from %v", delta, first)
	}
}

func campVisitorOrigin(t *testing.T, staged stagewire.Snapshot) maths.Vec {
	t.Helper()
	for _, batch := range staged.Scene.Batches {
		if batch.Authored && batch.Author.Collection == "scene.world.imports" &&
			batch.Author.Name == "visitor" && len(batch.Places) == 1 {
			return batch.Places[0].Origin
		}
	}
	t.Fatal("stage has no single authored visitor to select")
	return maths.Vec{}
}

const campVisitorID = "scene.world.imports/visitor"

func campWireRoundTrip(t *testing.T, snapshot stagewire.Snapshot) stagewire.Snapshot {
	t.Helper()
	var wire bytes.Buffer
	if err := stagewire.Write(&wire, snapshot); err != nil {
		t.Fatal(err)
	}
	decoded, err := stagewire.Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func campPickPoint(t *testing.T, work *Workspace) geometry.Point {
	t.Helper()
	scene := work.working.scene
	for _, batch := range scene.Batches {
		if !batch.Authored || batch.Author.Collection+"/"+batch.Author.Name != campVisitorID {
			continue
		}
		box := projectedCampBounds(t, scene, batch)
		for y := int(math.Floor(float64(box.Min.Y))); y <= int(math.Ceil(float64(box.Max.Y))); y++ {
			for x := int(math.Floor(float64(box.Min.X))); x <= int(math.Ceil(float64(box.Max.X))); x++ {
				if work.workingPick((float64(x)+.5)/float64(scene.Width),
					(float64(y)+.5)/float64(scene.Height)) == campVisitorID {
					t.Logf("visitor projected bounds %v, picked at %d,%d in %d×%d",
						box, x, y, scene.Width, scene.Height)
					return geometry.Pt(float32(x)+.5, float32(y)+.5)
				}
			}
		}
		t.Fatalf("no pixel in staged visitor bounds %v selected %q", box, campVisitorID)
	}
	t.Fatal("the staged scene contained no authored visitor")
	return geometry.Point{}
}

func projectedCampBounds(t *testing.T, scene *placement.DisplayScene, batch placement.StagedBatch) geometry.Rect {
	t.Helper()
	low, high, ok := batch.Mesh.Bounds()
	if !ok || len(batch.Places) != 1 {
		t.Fatal("expected one bounded visitor instance")
	}
	width, height := float64(scene.Width), float64(scene.Height)
	half := math.Tan(scene.Lens.Wide / 2)
	aspect := width / height
	minimum, maximum := geometry.Pt(float32(width), float32(height)), geometry.Point{}
	for _, x := range []float64{low.X, high.X} {
		for _, y := range []float64{low.Y, high.Y} {
			for _, z := range []float64{low.Z, high.Z} {
				eye := scene.Eye.In(batch.Places[0].Out(maths.V(x, y, z)))
				if eye.Z >= 0 {
					t.Fatal("visitor intersects or lies behind the camera")
				}
				across := float32(width * (1 + eye.X/(-eye.Z*half)) / 2)
				down := float32(height * (1 - eye.Y*aspect/(-eye.Z*half)) / 2)
				minimum = geometry.Pt(min(minimum.X, across), min(minimum.Y, down))
				maximum = geometry.Pt(max(maximum.X, across), max(maximum.Y, down))
			}
		}
	}
	return geometry.Rect{Min: minimum, Max: maximum}
}
