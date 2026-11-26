package workspace

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

func TestImageViewDrawFitsSourceAndKeepsLetterbox(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 {
				source.SetRGBA(x, y, color.RGBA{G: 255, A: 255})
			} else {
				source.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
			}
		}
	}
	view := newImageView()
	view.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(24, 24)})
	view.set(source)

	context := gg.NewContext(24, 24)
	defer context.Close()
	view.Draw(widget.NewContext(), render.NewCanvas(context, 24, 24))
	if err := context.FlushGPU(); err != nil {
		t.Fatalf("flush real gg canvas: %v", err)
	}
	picture := context.Image()
	if picture.Bounds() != image.Rect(0, 0, 24, 24) {
		t.Fatalf("rendered bounds = %v", picture.Bounds())
	}
	assertImageColor(t, picture, 4, 10, color.RGBA{G: 255, A: 255}, "left source region")
	assertImageColor(t, picture, 20, 10, color.RGBA{R: 255, A: 255}, "right source region")
	red, green, blue, alpha := appearance.Panel.RGBA8()
	assertImageColor(t, picture, 12, 2, color.RGBA{R: red, G: green, B: blue, A: alpha}, "letterbox")
}

func TestImageViewOrbitDragCapturesAndNavigatesOnceOnRelease(t *testing.T) {
	view := configuredNavigableImage()
	ctx := widget.NewContext()
	var captured, released widget.Widget
	ctx.SetOnCapturePointer(func(w widget.Widget) { captured = w })
	ctx.SetOnReleasePointer(func(w widget.Widget) { released = w })
	var calls [][3]float64
	view.SetNavigateHandler(func(yaw, pitch, zoom float64) { calls = append(calls, [3]float64{yaw, pitch, zoom}) })

	if !view.Event(ctx, mouseCoordinates(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 20, 20, 220, 120)) {
		t.Fatal("in-bounds primary press was not handled")
	}
	if captured != view || !view.dragging || len(calls) != 0 {
		t.Fatalf("press capture=%T dragging=%t calls=%v", captured, view.dragging, calls)
	}
	if !view.Event(ctx, mouseCoordinates(event.MouseDrag, event.ButtonLeft, event.ButtonStateLeft, 320, 160, 320, 160)) {
		t.Fatal("captured drag outside bounds was not handled")
	}
	if !view.Event(ctx, mouseCoordinates(event.MouseRelease, event.ButtonLeft, 0, 320, 160, 320, 160)) {
		t.Fatal("captured release outside bounds was not handled")
	}
	if released != view || view.dragging || len(calls) != 1 {
		t.Fatalf("release capture=%T dragging=%t calls=%v", released, view.dragging, calls)
	}
	if calls[0] != [3]float64{-35, 14, 0} {
		t.Fatalf("navigate args = %v, want 100x40 screen drag (-35,14,0)", calls[0])
	}
}

func TestImageViewGestureThresholdWheelBoundsAndAccessibilityKeys(t *testing.T) {
	view := configuredNavigableImage()
	ctx := widget.NewContext()
	var captures, releases, calls int
	ctx.SetOnCapturePointer(func(widget.Widget) { captures++ })
	ctx.SetOnReleasePointer(func(widget.Widget) { releases++ })
	var args [][3]float64
	view.SetNavigateHandler(func(yaw, pitch, zoom float64) { calls++; args = append(args, [3]float64{yaw, pitch, zoom}) })

	view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 5, 5))
	view.Event(ctx, mouse(event.MouseRelease, event.ButtonLeft, 0, 6, 5))
	if calls != 0 || captures != 1 || releases != 1 {
		t.Fatalf("sub-threshold click calls=%d captures=%d releases=%d", calls, captures, releases)
	}

	if !view.Event(ctx, event.NewWheelEvent(geometry.Pt(0, 10), geometry.Pt(40, 40), geometry.Point{}, event.ModNone)) {
		t.Fatal("in-bounds wheel was not handled")
	}
	if calls != 1 || math.Abs(args[0][2]-math.Exp(0.06)) > 1e-6 {
		t.Fatalf("wheel calls=%d args=%v, want zoom exp(.06)", calls, args)
	}
	view.Event(ctx, event.NewWheelEvent(geometry.Pt(0, 10000), geometry.Pt(40, 40), geometry.Point{}, event.ModNone))
	if math.Abs(args[len(args)-1][2]-math.E) > 1e-6 {
		t.Fatalf("large wheel zoom = %v, want bounded exp(1)", args[len(args)-1][2])
	}
	before := calls
	view.Event(ctx, event.NewWheelEvent(geometry.Pt(0, 1), geometry.Pt(101, 40), geometry.Point{}, event.ModNone))
	view.Event(ctx, event.NewWheelEvent(geometry.Pt(0, float32(math.NaN())), geometry.Pt(40, 40), geometry.Point{}, event.ModNone))
	if calls != before {
		t.Fatalf("outside/non-finite wheel dispatched navigation: %v", args)
	}

	view.SetFocused(true)
	if !view.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyLeft, 0, event.ModNone)) ||
		!view.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUp, 0, event.ModNone)) ||
		!view.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyEqual, '+', event.ModShift)) {
		t.Fatal("focused orbit keys were not handled")
	}
	if calls != before+3 || args[len(args)-3] != [3]float64{0.35, 0, 0} || args[len(args)-2] != [3]float64{0, -0.35, 0} || args[len(args)-1] != [3]float64{0, 0, math.Exp(-0.1)} {
		t.Fatalf("keyboard navigation = %v", args[before:])
	}
	view.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyMinus, '-', event.ModNone))
	if args[len(args)-1] != [3]float64{0, 0, math.Exp(0.1)} {
		t.Fatalf("minus zoom factor = %v, want zoom out (>1)", args[len(args)-1])
	}
}

func TestImageViewOverlayHonorsReadableFontFloor(t *testing.T) {
	view := newImageView()
	if view.fontSize != 16 {
		t.Fatalf("default font size = %v, want minimum 16", view.fontSize)
	}
	view.SetFontSize(8)
	if view.fontSize != 16 {
		t.Fatalf("font below the readable floor = %v", view.fontSize)
	}
	view.SetFontSize(20)
	if view.fontSize != 20 {
		t.Fatalf("user font size = %v, want 20", view.fontSize)
	}
}

func TestImageViewNavigableHoverUsesMoveCursor(t *testing.T) {
	view := configuredNavigableImage()
	view.SetNavigateHandler(func(float64, float64, float64) {})
	ctx := widget.NewContext()
	if view.Event(ctx, mouse(event.MouseMove, event.ButtonNone, 0, 40, 40)) {
		t.Fatal("hover was consumed")
	}
	if got := ctx.Cursor(); got != widget.CursorMove {
		t.Fatalf("navigable preview cursor = %v, want move cursor", got)
	}
	view.SetNavigable(false)
	ctx.SetCursor(widget.CursorDefault)
	view.Event(ctx, mouse(event.MouseMove, event.ButtonNone, 0, 40, 40))
	if got := ctx.Cursor(); got != widget.CursorDefault {
		t.Fatalf("non-navigable preview cursor = %v, want default", got)
	}
}

func TestImageViewRedrawsOnlyWhenNavigationChanges(t *testing.T) {
	view := newImageView()
	view.SetNeedsRedraw(false)
	view.SetNavigable(false)
	if view.NeedsRedraw() {
		t.Fatal("unchanged disabled navigation marked the view dirty")
	}
	view.SetNavigable(true)
	if !view.NeedsRedraw() {
		t.Fatal("enabling navigation did not mark the view dirty")
	}
	view.SetNeedsRedraw(false)
	view.SetNavigable(true)
	if view.NeedsRedraw() {
		t.Fatal("unchanged enabled navigation marked the view dirty")
	}
	view.SetNavigable(false)
	if !view.NeedsRedraw() {
		t.Fatal("disabling navigation did not mark the view dirty")
	}
}

func TestFocusedNavigablePreviewRoutesUndoAndRedoKeys(t *testing.T) {
	view := configuredNavigableImage()
	view.SetNavigateHandler(func(float64, float64, float64) {})
	ui := app.New()
	ui.SetRoot(view)
	center := geometry.Pt(50, 50)
	ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, center, center, event.ModNone))
	ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, center, center, event.ModNone))
	var redo []bool
	view.SetUndoHandler(func(_ widget.Context, isRedo bool) { redo = append(redo, isRedo) })
	if !view.IsFocused() {
		t.Fatal("real Window click did not focus the preview")
	}
	ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'Z', event.ModSuper|event.ModShift))
	if len(redo) == 0 {
		t.Fatal("focused preview did not consume Command-Z / Command-Shift-Z")
	}
	if len(redo) != 2 || redo[0] || !redo[1] {
		t.Fatalf("undo routing = %v, want [undo, redo]", redo)
	}
}

func TestImageViewRejectsUnavailableAndLostButtonGestures(t *testing.T) {
	view := newImageView()
	view.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(100, 100)})
	ctx := widget.NewContext()
	captures, releases, calls := 0, 0, 0
	ctx.SetOnCapturePointer(func(widget.Widget) { captures++ })
	ctx.SetOnReleasePointer(func(widget.Widget) { releases++ })
	view.SetNavigateHandler(func(float64, float64, float64) { calls++ })
	if view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 20, 20)) {
		t.Fatal("image-less view consumed press")
	}
	view.SetNavigable(true)
	view.set(solidImage(2, 2, color.RGBA{A: 255}))
	if view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, float32(math.NaN()), 20)) {
		t.Fatal("non-finite pointer press was consumed")
	}
	if view.Event(ctx, mouse(event.MousePress, event.ButtonRight, event.ButtonStateRight, 20, 20)) ||
		view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 120, 20)) {
		t.Fatal("non-primary or out-of-bounds press was consumed")
	}
	view.SetNavigateHandler(nil)
	if view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 20, 20)) {
		t.Fatal("view without navigation handler consumed press")
	}
	view.SetNavigateHandler(func(float64, float64, float64) { calls++ })
	view.Event(ctx, mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 20, 20))
	if !view.Event(ctx, mouse(event.MouseDrag, event.ButtonLeft, 0, 30, 30)) {
		t.Fatal("lost-button captured drag was not canceled")
	}
	if view.dragging || captures != 1 || releases != 1 || calls != 0 {
		t.Fatalf("lost-button state dragging=%t captures=%d releases=%d calls=%d", view.dragging, captures, releases, calls)
	}
	view.SetNavigable(false)
	if view.Event(ctx, event.NewWheelEvent(geometry.Pt(0, 1), geometry.Pt(20, 20), geometry.Point{}, event.ModNone)) || calls != 0 {
		t.Fatal("disabled navigation consumed wheel")
	}
}

func TestImageViewDragDrawsStatusOverlayOnRealCanvas(t *testing.T) {
	view := configuredNavigableImage()
	view.SetNavigateHandler(func(float64, float64, float64) {})
	view.Event(widget.NewContext(), mouse(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, 20, 20))
	view.Event(widget.NewContext(), mouse(event.MouseDrag, event.ButtonLeft, event.ButtonStateLeft, 50, 35))
	context := gg.NewContext(100, 100)
	defer context.Close()
	view.Draw(widget.NewContext(), render.NewCanvas(context, 100, 100))
	if err := context.FlushGPU(); err != nil {
		t.Fatalf("flush real gg canvas: %v", err)
	}
	red, green, blue, alpha := appearance.Panel.RGBA8()
	assertImageColor(t, context.Image(), 20, 69, color.RGBA{R: red, G: green, B: blue, A: alpha}, "orbit status panel")
}

func configuredNavigableImage() *imageView {
	view := newImageView()
	view.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(100, 100)})
	view.set(solidImage(2, 2, color.RGBA{A: 255}))
	view.SetNavigable(true)
	return view
}

func mouse(kind event.MouseEventType, button event.Button, buttons event.ButtonState, x, y float32) *event.MouseEvent {
	point := geometry.Pt(x, y)
	return event.NewMouseEvent(kind, button, buttons, point, point, event.ModNone)
}

func mouseCoordinates(kind event.MouseEventType, button event.Button, buttons event.ButtonState, x, y, globalX, globalY float32) *event.MouseEvent {
	return event.NewMouseEvent(kind, button, buttons, geometry.Pt(x, y), geometry.Pt(globalX, globalY), event.ModNone)
}

func TestImageViewSetReplacesCachedScaledPixels(t *testing.T) {
	view := newImageView()
	view.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(24, 24)})
	view.set(solidImage(4, 2, color.RGBA{R: 255, A: 255}))
	first := renderImageView(t, view)
	assertImageColor(t, first, 12, 12, color.RGBA{R: 255, A: 255}, "first source")
	if view.scaled == nil {
		t.Fatal("Draw did not cache its scaled image")
	}

	view.set(solidImage(2, 2, color.RGBA{B: 255, A: 255}))
	if view.scaled != nil {
		t.Fatal("setting a new source retained scaled pixels from the old source")
	}
	second := renderImageView(t, view)
	assertImageColor(t, second, 12, 12, color.RGBA{B: 255, A: 255}, "replacement source")
}

func renderImageView(t *testing.T, view *imageView) image.Image {
	t.Helper()
	context := gg.NewContext(24, 24)
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Errorf("close gg context: %v", err)
		}
	})
	view.Draw(widget.NewContext(), render.NewCanvas(context, 24, 24))
	if err := context.FlushGPU(); err != nil {
		t.Fatalf("flush real gg canvas: %v", err)
	}
	return context.Image()
}

func solidImage(width, height int, pixel color.RGBA) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			result.SetRGBA(x, y, pixel)
		}
	}
	return result
}

func assertImageColor(t *testing.T, picture image.Image, x, y int, want color.RGBA, label string) {
	t.Helper()
	got := color.RGBAModel.Convert(picture.At(x, y)).(color.RGBA)
	const tolerance = 2
	if channelDelta(got.R, want.R) > tolerance || channelDelta(got.G, want.G) > tolerance ||
		channelDelta(got.B, want.B) > tolerance || channelDelta(got.A, want.A) > tolerance {
		t.Errorf("%s pixel (%d,%d) = %v, want near %v", label, x, y, got, want)
	}
}

func channelDelta(got, want uint8) int {
	if got > want {
		return int(got - want)
	}
	return int(want - got)
}
