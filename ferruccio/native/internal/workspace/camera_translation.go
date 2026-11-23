package workspace

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// moveCamera commits a single camera translation through the normal editor history.
func (work *Workspace) moveCamera(east, north, up float64) error {
	if err := work.fullRenderRefusal("moving the camera"); err != nil {
		return err
	}
	if !work.viewer.navigable || work.viewer.source == nil {
		return fmt.Errorf("a current working camera is required before moving it")
	}
	before := work.state.document.Text()
	if err := work.state.document.MoveCamera(east, north, up); err != nil {
		return err
	}
	if work.state.document.Text() == before {
		return nil
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.pending.remember(work.state, work.specimen, time.Now())
	work.caption = "Camera moved · preparing preview"
	work.status = "Camera moved · waiting for input to settle"
	work.diagnostic = ""
	work.wakeNavigation(navigationDelay)
	work.invalidate()
	return nil
}

// moveTravel retains orientation by translating both eye and focus in parallel.
func (ball *cameraBall) moveTravel(rise bool) {
	live := &ball.work.working
	base := live.gesture
	if base == nil || ball.radius <= 0 {
		return
	}
	offset := base.Focus.Sub(base.Eye.Origin)
	distance := math.Hypot(math.Hypot(offset.X, offset.Y), offset.Z)
	if distance <= 0 || math.IsInf(distance, 0) {
		return
	}
	dx, dy := float64(ball.current.X-ball.start.X), float64(ball.current.Y-ball.start.Y)
	scale := distance * .25 / ball.radius
	right := base.Eye.Orientation.Out(maths.XAxis)
	vertical := offset.Scale(1 / distance)
	if rise {
		vertical = maths.YAxis
	}
	delta := right.Scale(dx * scale).Add(vertical.Scale(-dy * scale))
	for _, v := range []float64{delta.X, delta.Y, delta.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return
		}
	}
	scene := copyDisplay(base)
	scene.Eye.Origin = scene.Eye.Origin.Add(delta)
	scene.Focus = scene.Focus.Add(delta)
	ball.displacement = delta
	live.scene, live.dirty = scene, true
	ball.work.invalidate()
}
