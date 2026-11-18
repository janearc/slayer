package workspace

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"image"
	"math"
	"testing"
)

func TestImagePointUsesActualFittedImage(t *testing.T) {
	view := newImageView()
	view.SetBounds(geometry.Rect{Min: geometry.Pt(20, 30), Max: geometry.Pt(220, 230)})
	view.set(image.NewRGBA(image.Rect(0, 0, 200, 100)))
	got, ok := view.imagePoint(geometry.Pt(70, 105))
	if !ok || got != geometry.Pt(.25, .25) {
		t.Fatalf("image point=%v,%t want .25,.25", got, ok)
	}
}
func TestImagePointRefusesLetterboxAndInvalidPointer(t *testing.T) {
	view := newImageView()
	view.SetBounds(geometry.Rect{Min: geometry.Pt(20, 30), Max: geometry.Pt(220, 230)})
	view.set(image.NewRGBA(image.Rect(0, 0, 200, 100)))
	for _, point := range []geometry.Point{geometry.Pt(70, 40), geometry.Pt(10, 100), geometry.Pt(float32(math.NaN()), 100)} {
		if _, ok := view.imagePoint(point); ok {
			t.Fatalf("nonimage point accepted %v", point)
		}
	}
	view.set(nil)
	if _, ok := view.imagePoint(geometry.Pt(100, 100)); ok {
		t.Fatal("absent image accepted")
	}
}

func TestObjectDragSelectsWithoutEditingUntilRelease(t *testing.T) {
	view := configuredNavigableImage()
	view.SetNavigateHandler(func(float64, float64, float64) { t.Error("object drag orbited view") })
	view.mode = "move"
	selected, changed := 0, 0
	view.selectObject = func(x, y float64) bool { selected++; return true }
	view.changeObject = func(x0, y0, x1, y1 float64) {
		changed++
		if x1 <= x0 {
			t.Error("drag direction reversed")
		}
	}
	ctx := widget.NewContext()
	view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 40, 40))
	view.Event(ctx, mouse(event.MouseDrag, event.ButtonLeft, event.ButtonStateLeft, 60, 40))
	if selected != 1 || changed != 0 {
		t.Fatalf("during drag selected=%d changed=%d", selected, changed)
	}
	view.Event(ctx, mouse(event.MouseRelease, event.ButtonLeft, 0, 60, 40))
	if changed != 1 {
		t.Fatalf("release changes=%d want1", changed)
	}
}
func TestObjectClickAndMissNeverMoveObject(t *testing.T) {
	view := configuredNavigableImage()
	view.mode = "move"
	view.SetNavigateHandler(func(float64, float64, float64) {})
	changed := 0
	view.changeObject = func(float64, float64, float64, float64) { changed++ }
	view.selectObject = func(float64, float64) bool { return true }
	ctx := widget.NewContext()
	view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 40, 40))
	view.Event(ctx, mouse(event.MouseRelease, event.ButtonLeft, 0, 40, 40))
	view.selectObject = func(float64, float64) bool { return false }
	view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 40, 40))
	view.Event(ctx, mouse(event.MouseRelease, event.ButtonLeft, 0, 60, 40))
	if changed != 0 {
		t.Fatalf("click/miss changedobject%d", changed)
	}
}

func TestNativeObjectHoverNamesOnlyImageHits(t *testing.T) {
	view := configuredNavigableImage()
	view.SetNavigateHandler(func(float64, float64, float64) {})
	view.hoverObject = func(x, y float64) string {
		if x > .5 {
			return "visitor"
		}
		return ""
	}
	ctx := widget.NewContext()
	view.Event(ctx, mouse(event.MouseMove, event.ButtonNone, 0, 70, 40))
	if view.hoverName != "visitor" {
		t.Fatalf("native hover=%q", view.hoverName)
	}
	view.Event(ctx, mouse(event.MouseMove, event.ButtonNone, 0, 30, 40))
	if view.hoverName != "" {
		t.Fatal("surface miss retained object cue")
	}
}
