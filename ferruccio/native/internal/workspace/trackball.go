package workspace

import (
	"fmt"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/maths"
	"math"
)

// ballLabelLineRatio is the height of a ball label as a multiple of the
// chosen text size, so the label grows with the size.
const ballLabelLineRatio = 1.5

// cameraBall rolls a virtual sphere, retaining the authored camera horizon.
// Banking is a separate camera capability, not discarded state hidden by the ball.
type cameraBall struct {
	work                   *Workspace
	translation            bool
	dragging, hover        bool
	start, current, center geometry.Point
	radius                 float64
	yaw, pitch             float64
	displacement           maths.Vec
}

// bounds returns the screen rectangle of the ball control inside view.
func (ball *cameraBall) bounds(view geometry.Rect) geometry.Rect {
	offset := float32(0)
	if ball.translation {
		offset = 184
	}
	return geometry.Rect{Min: geometry.Pt(view.Min.X+16+offset, view.Max.Y-204), Max: geometry.Pt(view.Min.X+192+offset, view.Max.Y-16)}
}

// disc returns the centre and radius of the ball's round area inside view.
func (ball *cameraBall) disc(view geometry.Rect) (geometry.Point, float32) {
	b := ball.bounds(view)
	return geometry.Pt((b.Min.X+b.Max.X)/2, b.Min.Y+86), 58
}

// hit reports whether point lies on the ball, with a small margin.
func (ball *cameraBall) hit(point geometry.Point) bool {
	c, r := ball.disc(ball.work.viewer.Bounds())
	return finitePoint(point) && math.Hypot(float64(point.X-c.X), float64(point.Y-c.Y)) <= float64(r+8)
}

// Event retains capture outside the instrument and commits exactly once on release.
func (ball *cameraBall) Event(ctx widget.Context, e event.Event) bool {
	view := ball.work.viewer
	if !ball.dragging && (view.dragging || (ball.translation && view.ball != nil && view.ball.dragging) || (!ball.translation && view.travel != nil && view.travel.dragging)) {
		return false
	}
	if key, ok := e.(*event.KeyEvent); ok && ball.dragging {
		if key.KeyType == event.KeyPress && key.Key == event.KeyEscape {
			ball.finish(ctx, false)
		}
		return true
	}
	if _, ok := e.(*event.WheelEvent); ok && ball.dragging {
		return true
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if ball.dragging {
		switch mouse.MouseType {
		case event.MouseMove, event.MouseDrag:
			if !mouse.Buttons.Has(event.ButtonStateLeft) {
				ball.finish(ctx, false)
				return true
			}
			if finitePoint(mouse.GlobalPosition) {
				ball.current = mouse.GlobalPosition
				ball.move(mouse.Modifiers().Has(event.ModShift))
			}
		case event.MouseRelease:
			if mouse.Button != event.ButtonLeft {
				return true
			}
			valid := finitePoint(mouse.GlobalPosition)
			if valid {
				ball.current = mouse.GlobalPosition
				ball.move(mouse.Modifiers().Has(event.ModShift))
			}
			ball.finish(ctx, valid && math.Hypot(float64(ball.current.X-ball.start.X), float64(ball.current.Y-ball.start.Y)) > 2)
		}
		return true
	}
	if mouse.MouseType == event.MouseLeave {
		ball.hover = false
		return false
	}
	over := ball.hit(mouse.Position)
	if mouse.MouseType == event.MouseMove {
		if over != ball.hover {
			ball.hover = over
			view.SetNeedsRedraw(true)
		}
		if over {
			ctx.SetCursor(widget.CursorMove)
			return true
		}
	}
	if !over || mouse.MouseType != event.MousePress || mouse.Button != event.ButtonLeft || !mouse.Buttons.Has(event.ButtonStateLeft) {
		return false
	}
	if !finitePoint(mouse.GlobalPosition) || !ball.work.workingCurrent() {
		return true
	}
	if ball.translation && ball.work.state.document.Kind() != "set" {
		return true
	}
	ball.work.beginWorking()
	if ball.work.working.gesture == nil {
		return true
	}
	c, r := ball.disc(view.Bounds())
	ball.center = geometry.Pt(c.X+mouse.GlobalPosition.X-mouse.Position.X, c.Y+mouse.GlobalPosition.Y-mouse.Position.Y)
	ball.radius = float64(r)
	ball.start, ball.current = mouse.GlobalPosition, mouse.GlobalPosition
	ball.yaw, ball.pitch = 0, 0
	ball.displacement = maths.Vec{}
	ball.dragging = true
	ctx.RequestFocus(view)
	if capture, ok := ctx.(widget.PointerCapturer); ok {
		capture.CapturePointer(view)
	}
	ball.work.invalidate()
	return true
}

// spherePoint maps the screen point p onto a sphere of radius r centred on c.
func spherePoint(p, c geometry.Point, r float64) maths.Vec {
	x, y := float64(p.X-c.X)/r, -float64(p.Y-c.Y)/r
	d := x*x + y*y
	if d > 1 {
		s := math.Sqrt(d)
		return maths.V(x/s, y/s, 0)
	}
	return maths.V(x, y, math.Sqrt(1-d))
}

// move maps the sphere rotation into the canonical upright camera coordinates.
func (ball *cameraBall) move(constrain bool) {
	if ball.translation {
		ball.moveTravel(constrain)
		return
	}
	live := &ball.work.working
	base := live.gesture
	if base == nil {
		return
	}
	end := ball.current
	if constrain {
		if math.Abs(float64(end.X-ball.start.X)) >= math.Abs(float64(end.Y-ball.start.Y)) {
			end.Y = ball.start.Y
		} else {
			end.X = ball.start.X
		}
	}
	from, to := spherePoint(ball.start, ball.center, ball.radius), spherePoint(end, ball.center, ball.radius)
	axis := from.Cross(to)
	offset := base.Eye.Origin.Sub(base.Focus)
	moved := offset
	if axis.Len() <= 1e-12 && from.Dot(to) < 0 {
		axis = from.Cross(maths.YAxis)
		if axis.Len() <= 1e-12 {
			axis = from.Cross(maths.XAxis)
		}
	}
	if axis.Len() > 1e-12 {
		angle := math.Acos(math.Max(-1, math.Min(1, from.Dot(to))))
		turn := maths.Rotation(base.Eye.Orientation.Out(axis.Unit()), -angle)
		moved = turn.Out(offset)
	}
	oldYaw, newYaw := math.Atan2(-offset.Z, offset.X), math.Atan2(-moved.Z, moved.X)
	ball.yaw = math.Remainder((newYaw-oldYaw)/maths.Degree, 360)
	ball.pitch = (math.Atan2(moved.Y, math.Hypot(moved.X, moved.Z)) - math.Atan2(offset.Y, math.Hypot(offset.X, offset.Z))) / maths.Degree
	if constrain {
		if math.Abs(float64(ball.current.X-ball.start.X)) >= math.Abs(float64(ball.current.Y-ball.start.Y)) {
			ball.pitch = 0
		} else {
			ball.yaw = 0
		}
	}
	at, target := base.Eye.Origin, base.Focus
	next, err := draft.PreviewCamera([3]float64{at.X, -at.Z, at.Y}, [3]float64{target.X, -target.Z, target.Y}, ball.yaw, ball.pitch)
	if err != nil {
		return
	}
	eye, err := maths.View(maths.V(next[0], next[2], -next[1]), target, maths.YAxis)
	if err != nil {
		return
	}
	scene := copyDisplay(base)
	scene.Eye = *eye
	live.scene, live.dirty = scene, true
	ball.work.invalidate()
}

// finish ends the drag. It commits the camera edit when commit is set, and
// otherwise leaves the document as it was.
func (ball *cameraBall) finish(ctx widget.Context, commit bool) {
	before := ball.work.state.document.Text()
	if commit {
		if ball.translation {
			delta := ball.displacement
			ball.work.problem(ball.work.moveCamera(delta.X, -delta.Z, delta.Y))
		} else {
			ball.work.problem(ball.work.Navigate(ball.yaw, ball.pitch, 0))
		}
	}
	ball.dragging = false
	if capture, ok := ctx.(widget.PointerCapturer); ok {
		capture.ReleasePointer(ball.work.viewer)
	}
	ball.work.endWorking(before != ball.work.state.document.Text())
	ball.work.invalidate()
}

// Draw projects latitude/longitude and world axes through the actual camera pose.
func (ball *cameraBall) Draw(canvas widget.Canvas, view geometry.Rect) {
	b := ball.bounds(view)
	// Distinct instrument clips retain image-before-overlay order in gg's
	// GPU scissor groups, which otherwise bucket images after shape paths.
	canvas.PushClip(b.Expand(4))
	defer canvas.PopClip()
	c, r := ball.disc(view)
	canvas.DrawRoundRect(b, appearance.Ground, 18)
	canvas.DrawCircle(c, r+6, appearance.Panel)
	canvas.StrokeCircle(c, r+8, appearance.Border, 1)
	for i := 24; i >= 0; i-- {
		t := float32(i) / 24
		colour := blendToken(appearance.Raised, appearance.Panel, t)
		canvas.DrawCircle(geometry.Pt(c.X-r*.10*(1-t), c.Y-r*.12*(1-t)), r*(.88+.12*t), colour)
	}
	scene := ball.work.working.scene
	if scene != nil {
		project := func(p maths.Vec) (geometry.Point, bool) {
			q := scene.Eye.Orientation.In(p)
			return geometry.Pt(c.X+float32(q.X)*r, c.Y-float32(q.Y)*r), q.Z >= -.001
		}
		curve := func(point func(float64) maths.Vec, ink widget.Color) {
			previous, visible := project(point(0))
			for i := 1; i <= 72; i++ {
				p, front := project(point(float64(i) * 2 * math.Pi / 72))
				if visible && front {
					canvas.DrawLine(previous, p, ink, 1)
				}
				previous, visible = p, front
			}
		}
		for latitude := -60; latitude <= 60; latitude += 30 {
			a := float64(latitude) * maths.Degree
			curve(func(t float64) maths.Vec {
				return maths.V(math.Cos(a)*math.Cos(t), math.Sin(a), math.Cos(a)*math.Sin(t))
			}, appearance.Cyan)
		}
		for longitude := 0; longitude < 180; longitude += 30 {
			a := float64(longitude) * maths.Degree
			curve(func(t float64) maths.Vec {
				return maths.V(math.Cos(a)*math.Cos(t), math.Sin(t), math.Sin(a)*math.Cos(t))
			}, appearance.Cyan)
		}
		for _, axis := range []struct {
			p    maths.Vec
			name string
			ink  widget.Color
		}{{maths.XAxis, "E", appearance.Pink}, {maths.YAxis, "Up", appearance.Ink}, {maths.V(0, 0, -1), "N", appearance.Cyan}} {
			p, front := project(axis.p)
			if front {
				canvas.DrawLine(c, p, axis.ink, 2)
				canvas.DrawCircle(p, 3, axis.ink)
			}
		}
	}
	ink := appearance.Border
	if ball.hover || ball.dragging {
		ink = appearance.Pink
	}
	canvas.StrokeCircle(c, r, ink, 2)
	lineHeight := ball.work.size * ballLabelLineRatio
	label := geometry.Rect{Min: geometry.Pt(b.Min.X, b.Min.Y+3), Max: geometry.Pt(b.Max.X, b.Min.Y+3+lineHeight)}
	title := "Orbit"
	if ball.translation {
		title = "Travel"
	}
	canvas.DrawText(title, label, ball.work.size, appearance.Ink, false, widget.TextAlignCenter)
	label.Min.Y = b.Max.Y - 6 - lineHeight
	label.Max.Y = b.Max.Y - 6
	text := "Roll to orbit"
	if ball.translation {
		text = "Slide / approach"
	}
	if scene == nil {
		text = "Render only"
	}
	if ball.dragging {
		text = fmt.Sprintf("%+.0f°  %+.0f°", ball.yaw, ball.pitch)
		if ball.translation {
			text = fmt.Sprintf("%.2f m", ball.displacement.Len())
		}
	}
	canvas.DrawText(text, label, ball.work.size, appearance.Ink, false, widget.TextAlignCenter)
	if ball.work.ui.Window().Context().FocusedWidget() == ball.work.viewer && ball.work.play.mode == "look" {
		canvas.StrokeRoundRect(b, appearance.Pink, 18, 2)
	}
}

// blendToken returns the colour a fraction t of the way from a to b, opaque.
func blendToken(a, b widget.Color, t float32) widget.Color {
	return widget.Color{R: a.R + (b.R-a.R)*t, G: a.G + (b.G-a.G)*t, B: a.B + (b.B-a.B)*t, A: 1}
}

// gesturing reports whether a pointer drag is active on the viewer or on
// either camera ball.
func (work *Workspace) gesturing() bool {
	return work.viewer.dragging || (work.viewer.ball != nil && work.viewer.ball.dragging) || (work.viewer.travel != nil && work.viewer.travel.dragging)
}
