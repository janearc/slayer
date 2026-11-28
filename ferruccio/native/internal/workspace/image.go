package workspace

import (
	"fmt"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"golang.org/x/image/draw"
	"image"
	"math"
)

// imageView draws actual pixels; GoGPU UI's Image primitive is a placeholder.
type imageView struct {
	widget.WidgetBase
	source           image.Image
	aspect           float64
	scaled           *image.RGBA
	navigable        bool
	dragging         bool
	navigate         func(float64, float64, float64)
	undoHandler      func(widget.Context, bool)
	dragStart        geometry.Point
	dragCurrent      geometry.Point
	fontSize         float32
	selectObject     func(float64, float64) bool
	selectionCurrent func() bool
	refuseCamera     func() bool
	changeObject     func(float64, float64, float64, float64)
	playKey          func(*event.KeyEvent) bool
	mode             string
	dragObject       bool
	dragImage        geometry.Point
	hoverObject      func(float64, float64) string
	hoverName        string
	hoverPoint       geometry.Point
	beginWorking     func()
	moveWorking      func()
	endWorking       func(bool)
	ball             *cameraBall
	travel           *cameraBall
	lightMarkers     []lightMarker
	volumeBounds     []volumeOutline
	// settingsHandler opens the settings sheet on command and comma.
	settingsHandler func()
}

// newImageView creates a visible, noninteractive still-image surface.
func newImageView() *imageView {
	view := &imageView{}
	view.SetVisible(true)
	view.SetEnabled(true)
	view.fontSize = 16
	return view
}

// set invalidates resized pixels whenever the actual source image changes.
func (view *imageView) set(source image.Image) {
	view.source = source
	view.aspect = 0
	if source != nil && source.Bounds().Dy() > 0 {
		view.aspect = float64(source.Bounds().Dx()) / float64(source.Bounds().Dy())
	}
	view.hoverName = ""
	view.lightMarkers = nil
	view.volumeBounds = nil
	view.scaled = nil
	view.SetNeedsRedraw(true)
}

// setWorking keeps the authored image aspect independent of integer GPU target rounding.
func (view *imageView) setWorking(source image.Image, aspect float64) {
	view.set(source)
	if aspect > 0 && !math.IsNaN(aspect) && !math.IsInf(aspect, 0) {
		view.aspect = aspect
	}
}

// Layout uses the bounded area available beside the YAML editor.
func (view *imageView) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	return constraints.BiggestFinite(640, 400)
}

// Children reports that image pixels are painted by this leaf widget.
func (view *imageView) Children() []widget.Widget { return nil }

// Event handles preview gestures and focused keyboard navigation.
func (view *imageView) Event(ctx widget.Context, e event.Event) bool {
	if view.ball != nil && view.ball.Event(ctx, e) {
		return true
	}
	if view.travel != nil && view.travel.Event(ctx, e) {
		return true
	}
	if key, ok := e.(*event.KeyEvent); ok && key.KeyType == event.KeyPress && key.Key == event.KeyComma && key.Modifiers().HasAny(event.ModSuper|event.ModCtrl) && view.settingsHandler != nil {
		view.settingsHandler()
		return true
	}
	if mouse, ok := e.(*event.MouseEvent); ok {
		if !view.dragging && mouse.MouseType == event.MouseMove {
			view.updateHover(mouse.Position)
		}
		if mouse.MouseType == event.MouseLeave {
			view.hoverName = ""
			view.SetNeedsRedraw(true)
		}
		if view.dragging {
			switch mouse.MouseType {
			case event.MouseDrag, event.MouseMove:
				if !mouse.Buttons.Has(event.ButtonStateLeft) {
					view.cancelGesture(ctx)
					return true
				}
				ctx.SetCursor(widget.CursorMove)
				if finitePoint(mouse.GlobalPosition) {
					view.dragCurrent = mouse.GlobalPosition
					if view.moveWorking != nil {
						view.moveWorking()
					}
					view.SetNeedsRedraw(true)
				}
				return true
			case event.MouseRelease:
				if mouse.Button != event.ButtonLeft {
					return true
				}
				releaseValid := finitePoint(mouse.GlobalPosition)
				if releaseValid {
					view.dragCurrent = mouse.GlobalPosition
				}
				start, current := view.dragStart, view.dragCurrent
				isObject, imageStart := view.dragObject, view.dragImage
				imageBounds := view.imageBounds()
				navigable := view.navigable && view.source != nil
				view.dragging = false
				if capturer, ok := ctx.(widget.PointerCapturer); ok {
					capturer.ReleasePointer(view)
				}
				committed := false
				dx, dy := float64(current.X-start.X), float64(current.Y-start.Y)
				if releaseValid && navigable && view.navigate != nil && math.Hypot(dx, dy) > 2 {
					committed = true
					if view.moveWorking != nil {
						view.moveWorking()
					}
					if isObject && view.changeObject != nil && !imageBounds.IsEmpty() {
						view.changeObject(float64(imageStart.X), float64(imageStart.Y), float64(imageStart.X)+dx/float64(imageBounds.Width()), float64(imageStart.Y)+dy/float64(imageBounds.Height()))
					} else {
						view.navigate(-dx*0.35, dy*0.35, 0)
					}
				}
				if view.endWorking != nil {
					view.endWorking(committed)
				}
				view.SetNeedsRedraw(true)
				return true
			case event.MouseLeave:
				// Pointer capture keeps the gesture active outside the preview.
				return true
			}
			return true
		}
		if mouse.MouseType == event.MouseMove && view.navigable && view.source != nil && view.navigate != nil &&
			finitePoint(mouse.Position) && view.Bounds().Contains(mouse.Position) {
			ctx.SetCursor(widget.CursorMove)
			return false
		}
		if !view.navigable && view.source != nil && mouse.MouseType == event.MousePress && mouse.Button == event.ButtonLeft &&
			finitePoint(mouse.Position) && view.Bounds().Contains(mouse.Position) {
			ctx.RequestFocus(view)
			return true
		}
		if !view.navigable || view.source == nil || view.navigate == nil || !finitePoint(mouse.Position) || !view.Bounds().Contains(mouse.Position) {
			return false
		}
		if mouse.MouseType == event.MousePress && mouse.Button == event.ButtonLeft && mouse.Buttons.Has(event.ButtonStateLeft) {
			if !finitePoint(mouse.GlobalPosition) {
				return false
			}
			point, inImage := view.imagePoint(mouse.Position)
			if !inImage {
				return false
			}
			if (view.mode == "move" || view.mode == "turn") && view.selectionCurrent != nil && !view.selectionCurrent() {
				ctx.RequestFocus(view)
				return true
			}
			view.dragObject = false
			if view.selectObject != nil {
				selected := view.selectObject(float64(point.X), float64(point.Y))
				view.dragObject = selected && view.mode != "look"
			}
			if !view.dragObject && view.refuseCamera != nil && view.refuseCamera() {
				ctx.RequestFocus(view)
				return true
			}
			view.dragImage = point
			view.dragStart, view.dragCurrent = mouse.GlobalPosition, mouse.GlobalPosition
			view.dragging = true
			if view.beginWorking != nil {
				view.beginWorking()
			}
			ctx.RequestFocus(view)
			if capturer, ok := ctx.(widget.PointerCapturer); ok {
				capturer.CapturePointer(view)
			}
			view.SetNeedsRedraw(true)
			return true
		}
		return false
	}
	if wheel, ok := e.(*event.WheelEvent); ok {
		if view.dragging {
			return true
		}
		if !view.navigable || view.source == nil || view.navigate == nil || !finitePoint(wheel.Position) ||
			!finite32(wheel.Delta.Y) || !view.Bounds().Contains(wheel.Position) {
			return false
		}
		exponent := math.Max(-1, math.Min(1, float64(wheel.Delta.Y)*0.006))
		view.navigate(0, 0, math.Exp(exponent))
		return true
	}
	if key, ok := e.(*event.KeyEvent); ok && key.KeyType == event.KeyPress && view.IsFocused() {
		if view.dragging {
			if key.Key == event.KeyEscape {
				view.cancelGesture(ctx)
			}
			return true
		}
		if key.Key == event.KeyZ && key.Modifiers().HasAny(event.ModSuper|event.ModCtrl) {
			if view.undoHandler == nil {
				return false
			}
			view.undoHandler(ctx, key.Modifiers().Has(event.ModShift))
			return true
		}
		// Staging suspends navigation, but source history is still available.
		if !view.navigable || view.source == nil || view.navigate == nil {
			return false
		}
		if view.playKey != nil && view.playKey(key) {
			return true
		}
		switch key.Key {
		case event.KeyLeft:
			view.navigate(0.35, 0, 0)
		case event.KeyRight:
			view.navigate(-0.35, 0, 0)
		case event.KeyUp:
			view.navigate(0, -0.35, 0)
		case event.KeyDown:
			view.navigate(0, 0.35, 0)
		case event.KeyEqual:
			if key.Rune != '+' && !key.Modifiers().Has(event.ModShift) {
				return false
			}
			view.navigate(0, 0, math.Exp(-0.1))
		case event.KeyMinus:
			view.navigate(0, 0, math.Exp(0.1))
		default:
			return false
		}
		return true
	}
	return false
}

// IsFocusable makes the preview keyboard-navigable after pointer focus.
func (view *imageView) IsFocusable() bool { return view.IsEnabled() && view.IsVisible() }

// SetNavigable controls whether this image accepts preview navigation gestures.
func (view *imageView) SetNavigable(enabled bool) {
	if view.navigable == enabled {
		return
	}
	view.navigable = enabled
	view.SetNeedsRedraw(true)
}

// SetNavigateHandler registers the preview navigation callback.
func (view *imageView) SetNavigateHandler(handler func(float64, float64, float64)) {
	view.navigate = handler
}

// SetUndoHandler routes focused preview undo and redo shortcuts to the draft editor.
func (view *imageView) SetUndoHandler(handler func(widget.Context, bool)) {
	view.undoHandler = handler
}

// SetFontSize updates the status overlay size while enforcing the 16-point floor.
func (view *imageView) SetFontSize(size float32) {
	if size < 16 {
		size = 16
	}
	if size > 36 {
		size = 36
	}
	if view.fontSize != size {
		view.fontSize = size
		view.SetNeedsRedraw(true)
	}
}

// Draw fits the actual image without cropping and caches the scaled pixels.
func (view *imageView) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := view.Bounds()
	canvas.DrawRoundRect(bounds, appearance.Panel, 8)
	canvas.PushClipRoundRect(bounds, 8)
	defer canvas.PopClip()
	if view.source == nil {
		canvas.DrawText("Your next world starts here", bounds, view.fontSize, appearance.Ink, false, widget.TextAlignCenter)
		return
	}
	source := view.source.Bounds()
	fitted := view.imageBounds()
	width, height := int(fitted.Width()), int(fitted.Height())
	var picture image.Image = view.source
	if source.Dx() != width || source.Dy() != height {
		if view.scaled == nil || view.scaled.Bounds().Dx() != width || view.scaled.Bounds().Dy() != height {
			view.scaled = image.NewRGBA(image.Rect(0, 0, width, height))
			draw.CatmullRom.Scale(view.scaled, view.scaled.Bounds(), view.source, source, draw.Src, nil)
		}
		picture = view.scaled
	}
	canvas.DrawImage(picture, fitted.Min)
	view.drawVolumeBounds(canvas, fitted)
	view.drawLightMarkers(canvas, fitted)
	view.drawHover(canvas)
	if view.ball != nil {
		view.ball.Draw(canvas, bounds)
	}
	if view.travel != nil {
		view.travel.Draw(canvas, bounds)
	}
	if view.dragging {
		dx, dy := view.dragCurrent.X-view.dragStart.X, view.dragCurrent.Y-view.dragStart.Y
		label := fmt.Sprintf("Look %+.1f° · %+.1f° · release to render", -dx*0.35, dy*0.35)
		if view.dragObject {
			label = "Move · release to render · ⌘Z undo"
			if view.mode == "turn" {
				label = fmt.Sprintf("Turn %+.1f° · release to render", float64(dx)/float64(view.imageBounds().Width())*180)
			}
		}
		fontSize := view.fontSize
		labelWidth := canvas.MeasureText(label, fontSize, false)
		boxWidth := min(bounds.Width()-16, labelWidth+20)
		if boxWidth > 0 && bounds.Height() >= 32 {
			box := geometry.Rect{Min: geometry.Pt(bounds.Min.X+8, bounds.Max.Y-32), Max: geometry.Pt(bounds.Min.X+8+boxWidth, bounds.Max.Y-8)}
			if view.dragObject {
				fitted := view.imageBounds()
				start := geometry.Pt(fitted.Min.X+view.dragImage.X*fitted.Width(), fitted.Min.Y+view.dragImage.Y*fitted.Height())
				end := geometry.Pt(start.X+dx, start.Y+dy)
				canvas.StrokeRoundRect(geometry.Rect{Min: geometry.Pt(end.X-12, end.Y-12), Max: geometry.Pt(end.X+12, end.Y+12)}, appearance.Cyan, 12, 2)
			}
			canvas.DrawRoundRect(box, appearance.Panel, 5)
			canvas.DrawText(label, box, fontSize, appearance.Cyan, false, widget.TextAlignLeft)
		}
	}
}

// cancelGesture ends an orbit gesture and relinquishes any native pointer capture.
func (view *imageView) cancelGesture(ctx widget.Context) {
	if view.endWorking != nil {
		view.endWorking(false)
	}
	view.dragging = false
	view.SetNeedsRedraw(true)
	if capturer, ok := ctx.(widget.PointerCapturer); ok {
		capturer.ReleasePointer(view)
	}
}

// finite32 reports whether a pointer coordinate or wheel delta is finite.
// finite32 rejects NaN and infinite event values before geometry arithmetic.
func finite32(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

// finitePoint rejects coordinates that cannot safely be used for hit testing.
func finitePoint(point geometry.Point) bool { return finite32(point.X) && finite32(point.Y) }
