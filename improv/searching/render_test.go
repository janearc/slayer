package searching

import (
	"context"
	"errors"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/gpu"
)

// TestGradeMovesEachPixelTowardItsGrey checks worked values: a pixel of
// (200, 100, 0) has grey 100, so saturation one keeps it, zero makes it
// grey, one half halves its distance from grey, and two overshoots and
// clamps at both ends of the range. Alpha is untouched.
func TestGradeMovesEachPixelTowardItsGrey(t *testing.T) {
	for _, each := range []struct {
		saturation float64
		want       color.RGBA
	}{
		{1, color.RGBA{200, 100, 0, 255}},
		{0, color.RGBA{100, 100, 100, 255}},
		{0.5, color.RGBA{150, 100, 50, 255}},
		{2, color.RGBA{255, 100, 0, 255}},
		{-1, color.RGBA{0, 100, 200, 255}},
	} {
		picture := image.NewRGBA(image.Rect(0, 0, 2, 1))
		picture.SetRGBA(0, 0, color.RGBA{200, 100, 0, 255})
		picture.SetRGBA(1, 0, color.RGBA{30, 30, 30, 255})
		grade(picture, each.saturation)
		if got := picture.RGBAAt(0, 0); got != each.want {
			t.Errorf("saturation %g: %v, wanted %v", each.saturation, got, each.want)
		}
		if got := picture.RGBAAt(1, 0); got != (color.RGBA{30, 30, 30, 255}) {
			t.Errorf("saturation %g changed a grey pixel to %v", each.saturation, got)
		}
	}
}

// TestRendererFrameRefusesBeforeItOpensADevice checks the refusals that
// come before any scene or drawer is built: a context that has ended and
// a site the shoot does not have, from the edit or from an override.
func TestRendererFrameRefusesBeforeItOpensADevice(t *testing.T) {
	renderer := NewRenderer(halfSecondEdit())
	defer renderer.Close()
	if renderer.Edit == nil || len(renderer.sets) != 0 || len(renderer.drawers) != 0 {
		t.Fatalf("a new renderer holds %d sets and %d drawers", len(renderer.sets), len(renderer.drawers))
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if picture, _, err := renderer.Frame(cancelled, 1, ""); !errors.Is(err, context.Canceled) || picture != nil {
		t.Errorf("cancelled: %v, %v", picture, err)
	}
	picture, state, err := renderer.Frame(context.Background(), 1, "the moon")
	if err == nil || picture != nil || !strings.Contains(err.Error(), `unknown shoot site "the moon"`) {
		t.Errorf("override: %v, %v", picture, err)
	}
	if state.Site != "the moon" || state.At != 1 {
		t.Errorf("the refusal reported state %+v", state)
	}
	if len(renderer.sets) != 0 || len(renderer.drawers) != 0 {
		t.Errorf("a refused frame built %d sets and %d drawers", len(renderer.sets), len(renderer.drawers))
	}
}

// closedDrawer is a drawer that was never opened on a device and has been
// closed, so every draw on it is refused before any device is asked.
func closedDrawer() *gpu.Frames {
	drawer := &gpu.Frames{}
	drawer.Close()
	return drawer
}

// TestRendererCloseForgetsItsDrawers checks that Close empties the drawer
// table, leaves it usable, and may be repeated.
func TestRendererCloseForgetsItsDrawers(t *testing.T) {
	renderer := NewRenderer(halfSecondEdit())
	renderer.drawers["hawaii"], renderer.drawers["atacama"] = closedDrawer(), closedDrawer()
	renderer.Close()
	renderer.Close()
	if renderer.drawers == nil || len(renderer.drawers) != 0 {
		t.Fatalf("drawers %v", renderer.drawers)
	}
}

// TestRendererFrameReportsASceneThatCannotBeBuilt gives the renderer a plan
// whose observatory has no dome and expects the factory's refusal, with no
// set kept for the site.
func TestRendererFrameReportsASceneThatCannotBeBuilt(t *testing.T) {
	edit := halfSecondEdit()
	edit.Plan.Diameter = 0
	renderer := NewRenderer(edit)
	defer renderer.Close()
	picture, state, err := renderer.Frame(context.Background(), 1, "")
	if err == nil || picture != nil || state.Site != "hawaii" {
		t.Fatalf("got %v, %+v, %v", picture, state, err)
	}
	if len(renderer.sets) != 0 || len(renderer.drawers) != 0 {
		t.Fatalf("a failed scene left %d sets and %d drawers", len(renderer.sets), len(renderer.drawers))
	}
}

// TestRendererFrameStopsAtAClosedDrawer gives the renderer a set and a
// closed drawer, so the frame is posed and lit and then refused by the
// drawer before any device is used.
func TestRendererFrameStopsAtAClosedDrawer(t *testing.T) {
	edit := halfSecondEdit()
	edit.Plan.Diameter, edit.Plan.Aperture, edit.Plan.Altitude = 38, 8, 55
	edit.Plan.Width, edit.Plan.Height, edit.Plan.Rays, edit.Plan.Exposure = 160, 90, 1, 1.5
	renderer := NewRenderer(edit)
	defer renderer.Close()
	set, err := NewSet(edit.Plan, "blazar")
	if err != nil {
		t.Fatal(err)
	}
	renderer.sets["blazar"], renderer.drawers["blazar"] = set, closedDrawer()
	picture, state, err := renderer.Frame(context.Background(), 21, "")
	if !errors.Is(err, gpu.ErrFramesClosed) || picture != nil {
		t.Fatalf("got %v, %v", picture, err)
	}
	if state.Site != "blazar" || state.Spectrum != "radio" || set.JetField.Time != 21 {
		t.Fatalf("state %+v, jet time %g", state, set.JetField.Time)
	}
}

// TestRendererFrameReportsAScenePoseThatFails gives the renderer a set and
// a closed drawer but a plan that cannot place the beams, and expects the
// pose error before the drawer is asked.
func TestRendererFrameReportsAScenePoseThatFails(t *testing.T) {
	edit := halfSecondEdit()
	edit.Plan.Diameter, edit.Plan.Aperture, edit.Plan.Altitude = 38, 8, 55
	renderer := NewRenderer(edit)
	defer renderer.Close()
	set, err := NewSet(edit.Plan, "hawaii")
	if err != nil {
		t.Fatal(err)
	}
	edit.Plan.Diameter = math.NaN()
	renderer.sets["hawaii"], renderer.drawers["hawaii"] = set, closedDrawer()
	picture, _, err := renderer.Frame(context.Background(), 1, "")
	if err == nil || errors.Is(err, gpu.ErrFramesClosed) || picture != nil {
		t.Fatalf("got %v, %v", picture, err)
	}
}
