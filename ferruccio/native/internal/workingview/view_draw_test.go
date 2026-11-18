//go:build draw && darwin

package workingview

import (
	"bytes"
	"errors"
	"image"
	"os"
	"testing"

	"github.com/janearc/slayer/ferruccio/native/internal/headless"
	"github.com/janearc/slayer/hillside/placement"
	"github.com/janearc/slayer/set"
)

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
