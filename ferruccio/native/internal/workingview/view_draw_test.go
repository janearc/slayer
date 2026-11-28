//go:build draw && darwin

package workingview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/gogpu/wgpu"
	"github.com/janearc/slayer/ferruccio/native/internal/headless"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestImmutableBufferIsUnmappedAndChecksDeviceLimit(t *testing.T) {
	view, _ := openView(t)
	limit := view.device.Limits().MaxBufferSize
	if limit < 4 || limit > ^uint64(0)-4 {
		t.Fatalf("invalid Metal buffer limit %d", limit)
	}
	if buffer, err := view.mappedMeshBuffer("oversized geometry", (limit&^3)+4,
		wgpu.BufferUsageVertex, func([]byte) { t.Fatal("oversized buffer was mapped") }); err == nil || buffer != nil {
		t.Fatalf("oversized immutable buffer was accepted: %v", err)
	}
	buffer, err := view.mappedMeshBuffer("small geometry", 4, wgpu.BufferUsageVertex,
		func(bytes []byte) { copy(bytes, []byte{1, 2, 3, 4}) })
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Release()
	if buffer.MapState() != wgpu.MapStateUnmapped || buffer.Usage() != wgpu.BufferUsageVertex {
		t.Fatal("immutable geometry retained a mapping or staging usage")
	}
}

func TestDrawModesKeepSharedCameraAcrossMeshes(t *testing.T) {
	view, scene := openView(t)
	if len(scene.Objects) < 2 || scene.Objects[0].Mesh == scene.Objects[1].Mesh {
		t.Fatal("expected multiple resident meshes")
	}
	for _, mode := range []string{"solid", "edges", "wire"} {
		if err := view.SetMode(mode); err != nil {
			t.Fatal(err)
		}
		frame, err := drawUntilReady(view, scene, 320, 200)
		if err != nil || oneColour(frame.Pix) {
			t.Fatalf("%s did not render the meshes: %v", mode, err)
		}
		first := sha256.Sum256(frame.Pix)
		shifted := *scene
		shifted.Eye.Origin.X += 1
		frame, err = drawUntilReady(view, &shifted, 320, 200)
		if err != nil || oneColour(frame.Pix) {
			t.Fatalf("%s did not render after a camera move: %v", mode, err)
		}
		second := sha256.Sum256(frame.Pix)
		if first == second {
			t.Fatalf("%s did not update the visible camera", mode)
		}
		t.Logf("%s pixels before and after camera move: %x %x", mode, first, second)
	}
}

func TestWorkingSkyMetalPreservesGeometryAndClearsBetweenScenes(t *testing.T) {
	view, scene := openView(t)
	const width, height = 320, 200
	withSky := *scene
	magenta := maths.RGB(1, 0, 1)
	withSky.SkyGradient = placement.WorkingSkyGradient(sky.Gradient{Horizon: magenta, Zenith: magenta, Ground: magenta})
	skyPixel := color.NRGBA{R: 255, B: 255, A: 255}

	for _, mode := range []string{"solid", "edges", "wire"} {
		if err := view.SetMode(mode); err != nil {
			t.Fatal(err)
		}
		base, err := drawUntilReady(view, scene, width, height)
		if err != nil {
			t.Fatal(err)
		}
		flat := base.NRGBAAt(0, 0)
		pixels := bytes.Clone(base.Pix)
		changed, err := drawUntilReady(view, &withSky, width, height)
		if err != nil {
			t.Fatal(err)
		}
		shown := 0
		for y := range height {
			for x := range width {
				i := base.PixOffset(x, y)
				old := color.NRGBA{R: pixels[i], G: pixels[i+1], B: pixels[i+2], A: pixels[i+3]}
				got := changed.NRGBAAt(x, y)
				if old != flat && got != old {
					t.Fatalf("%s foreground pixel %d,%d changed from %v to %v", mode, x, y, old, got)
				}
				if got == skyPixel {
					shown++
				}
			}
		}
		if shown == 0 || shown == width*height {
			t.Fatalf("%s should show both sky and geometry, saw %d sky pixels", mode, shown)
		}
	}
	empty := withSky
	empty.Objects, empty.Batches = nil, nil
	blank, err := drawUntilReady(view, &empty, width, height)
	if err != nil || blank.NRGBAAt(0, 0) != skyPixel || !oneColour(blank.Pix) {
		t.Fatalf("geometry-free view lost its gradient: %v", err)
	}
	empty.SkyGradient = nil
	blank, err = drawUntilReady(view, &empty, width, height)
	if err != nil || blank.NRGBAAt(0, 0) == skyPixel || !oneColour(blank.Pix) {
		t.Fatalf("previous gradient survived an empty no-sky frame: %v", err)
	}
}

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
	snapshot, err := stagewire.Run(context.Background(), helper, path, string(source), kind, cache, os.Getenv("FERRUCCIO_STAGE_SHELF"), os.Getenv("FERRUCCIO_STAGE_DATA"), 2)
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

func TestStableBatchEditMatchesAFullGpuDraw(t *testing.T) {
	view, scene := openView(t)
	for _, object := range scene.Objects {
		scene.Batches = append(scene.Batches, placement.StagedBatch{Mesh: object.Mesh, Places: []maths.Subspace{object.At}})
	}
	scene.Objects = nil
	drawStable := func(scene *placement.DisplayScene, width, height int) *image.NRGBA {
		t.Helper()
		deadline := time.Now().Add(30 * time.Second)
		for {
			picture, err := view.DrawStable(scene, width, height)
			if !errors.Is(err, ErrReadbackLate) {
				if err != nil {
					t.Fatal(err)
				}
				return picture
			}
			if time.Now().After(deadline) {
				t.Fatal("stable frame did not complete")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	drawStable(scene, 320, 200)
	resized := *scene
	actual := bytes.Clone(drawStable(&resized, 160, 100).Pix)
	if view.Stats().ReusedInstances == 0 {
		t.Fatal("same field of view did not reuse instances after resize")
	}
	full, err := drawUntilReady(view, &resized, 160, 100)
	if err != nil || !bytes.Equal(actual, full.Pix) {
		t.Fatalf("resized stable frame differed from a full GPU draw: %v", err)
	}
	drawStable(scene, 320, 200)
	changed := *scene
	changed.Batches = append([]placement.StagedBatch(nil), scene.Batches...)
	changed.Batches[0].Places = append([]maths.Subspace(nil), scene.Batches[0].Places...)
	changed.Batches[0].Places[0].Origin.X += 3
	actual = bytes.Clone(drawStable(&changed, 320, 200).Pix)
	if view.Stats().ReusedInstances == 0 {
		t.Fatal("unchanged visible placements were not reused")
	}
	full, err = drawUntilReady(view, &changed, 320, 200)
	if err != nil || !bytes.Equal(actual, full.Pix) {
		t.Fatalf("stable placement update differed from a full GPU draw: %v", err)
	}
	drawStable(&changed, 320, 200)
	changedCamera := changed
	changedCamera.Eye.Origin.X += 1
	actual = bytes.Clone(drawStable(&changedCamera, 320, 200).Pix)
	if view.Stats().ReusedInstances != 0 {
		t.Fatal("camera change reused placements culled for the previous view")
	}
	full, err = drawUntilReady(view, &changedCamera, 320, 200)
	if err != nil || !bytes.Equal(actual, full.Pix) {
		t.Fatalf("camera change reused stale visible instances: %v", err)
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

func TestSparseBatchDrawsTheVisiblePlacements(t *testing.T) {
	view, scene := openView(t)
	object := scene.Objects[0]
	places := make([]maths.Subspace, 130)
	for i := range places {
		places[i] = maths.Subspace{Origin: scene.Eye.Out(maths.V(1000000, 0, -100))}
	}
	for i, index := range []int{1, 64, 129} {
		places[index] = maths.Subspace{Origin: scene.Eye.Out(maths.V(float64(i-1)*12, 0, -100))}
	}
	scene.Objects = nil
	scene.Batches = []placement.StagedBatch{{Mesh: object.Mesh, Places: places}}
	batch, err := drawUntilReady(view, scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Clone(batch.Pix)
	scene.Batches = nil
	for _, index := range []int{1, 64, 129} {
		copy := object
		copy.At = places[index]
		scene.Objects = append(scene.Objects, copy)
	}
	reference, err := drawUntilReady(view, scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, reference.Pix) {
		t.Fatal("sparse batch did not draw the same placements as individual objects")
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
	hidden := scene.Objects[0]
	detached := *hidden.Mesh
	hidden.Mesh = &detached
	hidden.At.Origin = scene.Eye.Out(maths.V(1000000, 0, -10))
	scene.Objects = append(scene.Objects, hidden)
	picture, err := view.Draw(scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if oneColour(picture.Pix) {
		t.Fatal("populated frame did not draw geometry")
	}
	if view.Stats().Meshes == 0 || len(view.meshBounds) == 0 {
		t.Fatal("scene uploaded no meshes or kept no bounds")
	}
	if _, found := view.meshBounds[hidden.Mesh]; !found || view.meshes[hidden.Mesh] != nil {
		t.Fatal("culled mesh did not retain CPU bounds independently of upload")
	}
	scene.Objects, scene.Batches = nil, nil
	picture, err = view.Draw(scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if !oneColour(picture.Pix) {
		t.Fatal("empty frame did not clear prior geometry")
	}
	if view.Stats().Meshes != 0 || view.Stats().Triangles != 0 || len(view.meshBounds) != 0 {
		t.Fatal("old scene geometry remains resident")
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
	frame, err := drawUntilReady(view, scene, 320, 200)
	if err != nil || view.pending != nil || view.Stats().Draws != 1 {
		t.Fatalf("pending frame did not complete once: %v", err)
	}
	before := bytes.Clone(frame.Pix)
	view.ForceLateReadbacks(1)
	if _, err := view.Draw(scene, 320, 200); !errors.Is(err, ErrReadbackLate) || view.pending == nil {
		t.Fatalf("second pending draw: %v", err)
	}
	// Translate the same scene pointer without changing the projection.
	// Its pending frame still belongs to the old eye origin.
	scene.Eye.Origin.X++
	frame, err = drawUntilReady(view, scene, 320, 200)
	if err != nil || view.pending != nil || bytes.Equal(before, frame.Pix) {
		t.Fatalf("same-pointer camera translation retained stale pixels: %v", err)
	}
	after := bytes.Clone(frame.Pix)
	frame, err = drawUntilReady(view, scene, 320, 200)
	if err != nil || !bytes.Equal(after, frame.Pix) {
		t.Fatalf("frame after cancellation differed from a fresh draw: %v", err)
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

func TestDiscardSceneDropsPendingReadbackAndGeometry(t *testing.T) {
	view, scene := openView(t)
	for _, object := range scene.Objects {
		scene.Batches = append(scene.Batches, placement.StagedBatch{Mesh: object.Mesh, Places: []maths.Subspace{object.At}})
	}
	scene.Objects = nil
	before, err := drawUntilReady(view, scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Clone(before.Pix)
	view.ForceLateReadbacks(1)
	if _, err := view.DrawStable(scene, 320, 200); !errors.Is(err, ErrReadbackLate) || view.pending == nil || view.stable == nil {
		t.Fatalf("scene did not have a retained frame and geometry: %v", err)
	}
	if err := view.DiscardScene(); err != nil {
		t.Fatal(err)
	}
	if view.pending != nil || view.stable != nil || view.Stats().Meshes != 0 ||
		view.Stats().Triangles != 0 || len(view.meshBounds) != 0 {
		t.Fatalf("discard left the scene reachable: %+v", view.Stats())
	}
	after, err := drawUntilReady(view, scene, 320, 200)
	if err != nil || !bytes.Equal(want, after.Pix) {
		t.Fatalf("reupload after discarding a pending frame changed pixels: %v", err)
	}
}

func TestMeshUploadsKeepDistinctInstanceTransforms(t *testing.T) {
	view, scene := openView(t)
	object := scene.Objects[0]
	shifted := object.At
	shifted.Origin.Y += 4
	scene.Objects = nil
	scene.Batches = []placement.StagedBatch{{Mesh: object.Mesh, Places: []maths.Subspace{object.At}}}
	one, err := drawUntilReady(view, scene, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(one.Pix)
	scene.Batches[0].Places = []maths.Subspace{object.At, shifted}
	batched, err := drawUntilReady(view, scene, 320, 200)
	if err != nil || bytes.Equal(before, batched.Pix) {
		t.Fatalf("second placement did not draw: %v", err)
	}
	want := bytes.Clone(batched.Pix)
	duplicate := *object.Mesh
	scene.Batches[0].Places = []maths.Subspace{object.At}
	scene.Batches = append(scene.Batches, placement.StagedBatch{Mesh: &duplicate, Places: []maths.Subspace{shifted}})
	for range 2 {
		frame, err := drawUntilReady(view, scene, 320, 200)
		if err != nil || !bytes.Equal(want, frame.Pix) {
			t.Fatalf("two mesh uploads differed from two instances of one mesh: %v", err)
		}
	}
}

func TestWorkingViewMetalKeepsNearbyPixelsAtLargeCoordinates(t *testing.T) {
	view, base := openView(t)
	anchor := maths.V(-318490735, 115921147, 5e8)
	object := base.Objects[1]
	object.At.Origin = maths.V(3, 1, -30)
	large := object
	large.At.Origin = object.At.Origin.Add(anchor)
	const width, height = 320, 200
	base = &placement.DisplayScene{Eye: maths.Local(), Width: width, Height: height}
	distant := *base
	distant.Eye.Origin = anchor
	for _, batched := range []bool{false, true} {
		if batched {
			base.Objects, distant.Objects = nil, nil
			base.Batches = []placement.StagedBatch{{Mesh: object.Mesh, Places: []maths.Subspace{object.At}}}
			distant.Batches = []placement.StagedBatch{{Mesh: large.Mesh, Places: []maths.Subspace{large.At}}}
		} else {
			base.Objects, distant.Objects = []placement.DisplayObject{object}, []placement.DisplayObject{large}
		}
		first, err := drawUntilReady(view, base, width, height)
		if err != nil || oneColour(first.Pix) {
			t.Fatalf("reference with batched=%t did not draw: %v", batched, err)
		}
		want := bytes.Clone(first.Pix)
		second, err := drawUntilReady(view, &distant, width, height)
		if err != nil || !bytes.Equal(want, second.Pix) {
			t.Fatalf("translated scene with batched=%t changed pixels: %v", batched, err)
		}
	}
}

func TestCompactCachedWorldMetalMatchesLocalScene(t *testing.T) {
	view, _ := openView(t)
	cache := placement.NewDisplayCache()
	sceneAt := func(east, camera int) *placement.DisplayScene {
		t.Helper()
		source := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Working world precision
  setting: space
  seed: 3
  light: none
  film: {width: 640, height: 400, rays: 1, wide: 60}
  cameras:
    view:
      location: {at: [%d, 2, 0]}
      orientation: {look_at: [%d, 52, 0]}
  set:
    things:
      visitor: {kind: yurt, seed: 4, east: %d, north: 22}
  world:
    materials:
      white: {colour: {srgb_linear: [0.8, 0.6, 0.4]}, source: eye}
    boxes:
      marker: {middle: [%d.25, 31, 0], size: [2.5, 2.5, 2.5], material: white, source: eye}
`, camera, camera, east+7, east)
		read, err := set.Read("compact.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		scene, err := cache.Scene(read)
		if err != nil {
			t.Fatal(err)
		}
		return scene
	}
	draw := func(scene *placement.DisplayScene) []byte {
		t.Helper()
		for {
			frame, err := view.Draw(scene, 320, 200)
			if errors.Is(err, ErrReadbackLate) {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if oneColour(frame.Pix) {
				t.Fatal("working view drew no geometry")
			}
			return bytes.Clone(frame.Pix)
		}
	}
	const farEast = 400000000
	local, distant := sceneAt(0, 0), sceneAt(farEast, farEast)
	world := distant.Objects[len(distant.Objects)-1]
	if world.ID != "scene.world" || world.At.Origin.X != farEast+.25 {
		t.Fatalf("test did not use the anchored cached world: %+v", world.At)
	}
	before := draw(local)
	centre := (100*320 + 160) * 4
	if bytes.Equal(before[:4], before[centre:centre+4]) {
		t.Fatal("the compact world box did not paint the centre pixel")
	}
	if got := draw(distant); !bytes.Equal(before, got) {
		t.Fatal("anchored world changed the image when translated with the camera")
	}
	panned := sceneAt(farEast, farEast+1)
	if panned.Objects[len(panned.Objects)-1].Mesh != world.Mesh {
		t.Fatal("camera translation rebuilt the cached world mesh")
	}
	after := draw(panned)
	if bytes.Equal(before, after) {
		t.Fatal("camera translation left the prior image on screen")
	}
	if err := view.DiscardScene(); err != nil {
		t.Fatal(err)
	}
	if fresh := draw(panned); !bytes.Equal(after, fresh) {
		t.Fatal("camera translation with a cached mesh differed from a fresh draw")
	}
}
