//go:build draw && darwin

package workingview

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/native/internal/headless"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestRealSceneSharedBatchDraw(t *testing.T) {
	path, helper, cache := os.Getenv("FERRUCCIO_STAGE_SCENE"), os.Getenv("FERRUCCIO_STAGE_HELPER"), os.Getenv("FERRUCCIO_STAGE_CACHE")
	if path == "" || helper == "" || cache == "" {
		t.Skip("set the scene, helper and cache for a real shared-scene draw")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	kind := os.Getenv("FERRUCCIO_STAGE_KIND")
	if kind == "" {
		kind = "set"
	}
	snapshot, err := stagewire.Run(context.Background(), helper, path, string(source), kind, cache, os.Getenv("FERRUCCIO_STAGE_SHELF"), 2)
	if err != nil {
		t.Fatal(err)
	}
	device, err := headless.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	view, err := New(device)
	if err != nil {
		t.Fatal(err)
	}
	defer view.Close()
	picture, err := drawUntilReady(view, &snapshot.Scene, 320, 180)
	if err != nil {
		t.Fatal(err)
	}
	if oneColour(picture.Pix) {
		t.Fatal("shared scene drew no visible geometry")
	}
	if path := os.Getenv("FERRUCCIO_STAGE_IMAGE"); path != "" {
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, picture); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("shared GPU draw: %+v; %s", view.Stats(), snapshot.Notice)
	started := time.Now()
	id := snapshot.Scene.Pick(.5, .5)
	t.Logf("centre selection %q in %s", id, time.Since(started))
	if os.Getenv("FERRUCCIO_STAGE_SECOND_FRAME") == "1" {
		changed := snapshot.Scene
		changed.Eye.Origin.X += 1
		started = time.Now()
		if _, err := drawUntilReady(view, &changed, 320, 180); err != nil {
			t.Fatalf("second working frame: %v", err)
		}
		t.Logf("one-metre camera move drew in %s: %+v", time.Since(started), view.Stats())
	}
}

// playground is the scene the workspace tests also draw.
const playground = "../../../../sets/ferruccio-playground.yaml"

// openView gives a test a working view on a real Metal device and the
// playground scene, both released when the test ends.
func openView(t *testing.T) (*View, *placement.DisplayScene) {
	t.Helper()
	device, err := headless.Open()
	if err != nil {
		t.Fatalf("open Metal: %v", err)
	}
	t.Cleanup(device.Close)
	view, err := New(device)
	if err != nil {
		t.Fatalf("working view: %v", err)
	}
	t.Cleanup(func() { view.Close() })
	raw, err := os.ReadFile(playground)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(playground, raw)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	return view, scene
}

// TestLateReadbackRecoversWithTheSameFrame checks a late readback is
// counted and abandoned, the view draws again on the next call, and the
// frame it then hands back is the frame it drew before the misses.
func TestLateReadbackRecoversWithTheSameFrame(t *testing.T) {
	view, scene := openView(t)
	const width, height, misses = 320, 200, 2
	first, err := view.Draw(scene, width, height)
	if err != nil {
		t.Fatalf("first draw: %v", err)
	}
	reference := bytes.Clone(first.Pix)
	if oneColour(reference) {
		t.Fatal("first frame is one colour; the scene did not draw")
	}
	view.ForceLateReadbacks(misses)
	for miss := range misses {
		if _, err = view.Draw(scene, width, height); !errors.Is(err, ErrReadbackLate) {
			t.Fatalf("forced late draw %d: %v; want ErrReadbackLate", miss, err)
		}
	}
	again, err := view.Draw(scene, width, height)
	if err != nil {
		t.Fatalf("draw after the misses: %v", err)
	}
	if !bytes.Equal(again.Pix, reference) {
		t.Fatal("the frame after recovery differs from the frame before")
	}
	stats := view.Stats()
	if stats.Misses != misses || stats.Draws != 2 {
		t.Fatalf("misses %d, draws %d; want %d and 2", stats.Misses, stats.Draws, misses)
	}
}

// oneColour reports whether every pixel of an RGBA buffer is the first.
func oneColour(pixels []byte) bool {
	for at := 4; at < len(pixels); at += 4 {
		if !bytes.Equal(pixels[at:at+4], pixels[:4]) {
			return false
		}
	}
	return true
}

// TestDrawReusesThePictureWhileTheSizeHolds checks one picture serves every
// frame of a size, and a new size gets a new one.
func TestDrawReusesThePictureWhileTheSizeHolds(t *testing.T) {
	view, scene := openView(t)
	first, err := view.Draw(scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	second, err := view.Draw(scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("a second frame of the same size allocated a new picture")
	}
	resized, err := view.Draw(scene, 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	if resized == first || resized.Rect != image.Rect(0, 0, 300, 200) {
		t.Fatalf("resized picture %v, shared %t", resized.Rect, resized == first)
	}
}

func TestDrawBatchesSharedMeshPlacements(t *testing.T) {
	view, scene := openView(t)
	if len(scene.Objects) == 0 {
		t.Fatal("playground has no objects")
	}
	first, err := view.Draw(scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(first.Pix)
	uploads := view.Stats().Uploads
	copy := scene.Objects[0]
	copy.ID = "second-placement"
	copy.At.Origin.Y += 4
	scene.Batches = append(scene.Batches, placement.StagedBatch{Mesh: copy.Mesh, Places: []maths.Subspace{copy.At}})
	draws, err := view.place(scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	batched := false
	for _, draw := range draws {
		if draw.count >= 2 {
			batched = true
		}
	}
	if !batched || view.Stats().Uploads != uploads {
		t.Fatal("shared placement was not batched with its existing mesh")
	}
	after, err := drawUntilReady(view, scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after.Pix) {
		t.Fatal("the second placement did not change the GPU frame")
	}
}

func TestDrawBatchesKeepVisibleInstancesAcrossMaskWords(t *testing.T) {
	view, scene := openView(t)
	mesh := scene.Objects[0].Mesh
	places := make([]maths.Subspace, 130)
	want := 0
	for i := range places {
		local := maths.V(0, 0, -100)
		if i%3 == 0 {
			local.X = 1000000
		} else {
			want++
		}
		places[i] = maths.Subspace{Origin: scene.Eye.Out(local)}
	}
	scene.Objects = nil
	scene.Batches = []placement.StagedBatch{{Mesh: mesh, Places: places}}
	draws, err := view.place(scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(draws) != 1 || draws[0].count != uint32(want) {
		t.Fatalf("visible draw instances = %v; want one batch of %d", draws, want)
	}
}

func TestIndexedCullingPreservesTheWorkingFrame(t *testing.T) {
	view, scene := openView(t)
	mesh := scene.Objects[0].Mesh
	places := make([]maths.Subspace, 3*placement.PlacementBlockSize)
	for i := range places {
		local := maths.V(1000000, 0, -100)
		if i >= placement.PlacementBlockSize && i < 2*placement.PlacementBlockSize {
			local.X = float64(i%7) - 3
		}
		places[i] = maths.Subspace{Origin: scene.Eye.Out(local)}
	}
	scene.Objects = nil
	scene.Batches = []placement.StagedBatch{{Mesh: mesh, Places: places, HasBounds: true, Bounds: maths.Box{Min: maths.V(-2000000, -2000000, -2000000), Max: maths.V(2000000, 2000000, 2000000)}}}
	before, err := drawUntilReady(view, scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	reference := bytes.Clone(before.Pix)
	scene.IndexPicking()
	after, err := drawUntilReady(view, scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reference, after.Pix) {
		t.Fatal("indexed culling changed the displayed frame")
	}
}

func drawUntilReady(view *View, scene *placement.DisplayScene, width, height int) (*image.NRGBA, error) {
	deadline := time.Now().Add(3 * time.Minute)
	for {
		picture, err := view.Draw(scene, width, height)
		if !errors.Is(err, ErrReadbackLate) || time.Now().After(deadline) {
			return picture, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDrawReleasesMeshesNoLongerInScene(t *testing.T) {
	view, scene := openView(t)
	if _, err := view.Draw(scene, 320, 200); err != nil {
		t.Fatal(err)
	}
	if view.Stats().Meshes == 0 {
		t.Fatal("scene uploaded no meshes")
	}
	scene.Objects, scene.Batches = nil, nil
	if _, err := view.Draw(scene, 320, 200); err != nil {
		t.Fatal(err)
	}
	if view.Stats().Meshes != 0 || view.Stats().Triangles != 0 {
		t.Fatal("old scene meshes remain resident")
	}
}

func TestBatchReadbackIsRetainedAndStaleFrameCancelled(t *testing.T) {
	view, scene := openView(t)
	copy := scene.Objects[0]
	scene.Batches = []placement.StagedBatch{{Mesh: copy.Mesh, Places: []maths.Subspace{copy.At}}}
	view.ForceLateReadbacks(2)
	if _, err := view.Draw(scene, 320, 200); !errors.Is(err, ErrReadbackLate) || view.pending == nil {
		t.Fatalf("first pending draw: %v", err)
	}
	first := view.pending
	if _, err := view.Draw(scene, 320, 200); !errors.Is(err, ErrReadbackLate) || view.pending != first {
		t.Fatalf("pending frame was not retained: %v", err)
	}
	if _, err := drawUntilReady(view, scene, 320, 200); err != nil || view.pending != nil || view.Stats().Draws != 1 {
		t.Fatalf("pending frame did not complete once: %v", err)
	}
	view.ForceLateReadbacks(1)
	if _, err := view.Draw(scene, 320, 200); !errors.Is(err, ErrReadbackLate) || view.pending == nil {
		t.Fatalf("second pending draw: %v", err)
	}
	changed := *scene
	changed.Eye.Origin.X += 1
	if _, err := drawUntilReady(view, &changed, 320, 200); err != nil || view.pending != nil {
		t.Fatalf("changed view did not cancel its stale frame: %v", err)
	}
}

// TestDrawFitsAWindowPastTheCap checks a window wider than the cap draws,
// at the cap, rather than being refused.
func TestDrawFitsAWindowPastTheCap(t *testing.T) {
	view, scene := openView(t)
	picture, err := view.Draw(scene, 5000, 2500)
	if err != nil {
		t.Fatalf("draw past the cap: %v", err)
	}
	want := image.Rect(0, 0, largestSide, largestSide/2)
	if picture.Rect != want {
		t.Fatalf("picture %v; want %v", picture.Rect, want)
	}
}

// TestDrawRefusesAnEmptyWindowWithoutAMiss is the anti-test: a window with
// no area is an error of its own, never counted as a late frame.
func TestDrawRefusesAnEmptyWindowWithoutAMiss(t *testing.T) {
	view, scene := openView(t)
	_, err := view.Draw(scene, 0, 200)
	if err == nil || errors.Is(err, ErrReadbackLate) {
		t.Fatalf("empty window: %v; want a plain error", err)
	}
	if view.Stats().Misses != 0 {
		t.Fatal("an empty window was counted as a miss")
	}
}
