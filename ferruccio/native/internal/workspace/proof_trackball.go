package workspace

import (
	"fmt"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
)

// trackballProofPoints maps the actual ball rectangle into native screen coordinates.
func (work *Workspace) trackballProofPoints() (geometry.Point, geometry.Point, error) {
	return work.cameraBallProofPoints(work.viewer.ball)
}

// cameraBallProofPoints returns the centre of the camera ball on screen and
// a point to drag to. It fails when the control has no native layout.
func (work *Workspace) cameraBallProofPoints(instrument *cameraBall) (geometry.Point, geometry.Point, error) {
	if work.composition != "play" || work.ui == nil || work.viewer == nil || instrument == nil || !work.viewer.IsScreenOriginValid() {
		return geometry.Point{}, geometry.Point{}, fmt.Errorf("trackball proof requires a displayed native camera control")
	}
	bounds := work.viewer.Bounds()
	ball := instrument.bounds(bounds)
	screen := work.viewer.ScreenBounds()
	center := geometry.Pt((ball.Min.X+ball.Max.X)/2+screen.Min.X-bounds.Min.X, (ball.Min.Y+ball.Max.Y)/2+screen.Min.Y-bounds.Min.Y)
	if ball.IsEmpty() || !finitePoint(center) {
		return geometry.Point{}, geometry.Point{}, fmt.Errorf("trackball proof control has no native layout")
	}
	return center, geometry.Pt(center.X+34, center.Y-20), nil
}

// ProofTrackballPress reaches the camera ball through the normal window dispatcher.
func (work *Workspace) ProofTrackballPress() error { return work.proofBallPress(work.viewer.ball) }

// ProofTravelPress dispatches native input to the adjacent camera movement ball.
func (work *Workspace) ProofTravelPress() error { return work.proofBallPress(work.viewer.travel) }

// proofBallPress sends a left press at the ball centre and checks that the
// ball captured it.
func (work *Workspace) proofBallPress(instrument *cameraBall) error {
	center, _, err := work.cameraBallProofPoints(instrument)
	if err != nil {
		return err
	}
	if instrument.dragging || work.viewer.dragging {
		return fmt.Errorf("trackball proof interrupted: another pointer gesture is active")
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, center, center, event.ModNone))
	if !instrument.dragging {
		return fmt.Errorf("trackball proof press did not capture the camera control")
	}
	return nil
}

// ProofTrackballMove advances captured input on a later UI tick, without callbacks.
func (work *Workspace) ProofTrackballMove() error { return work.proofBallMove(work.viewer.ball) }

// ProofTravelMove dispatches native input to the adjacent camera movement ball.
func (work *Workspace) ProofTravelMove() error { return work.proofBallMove(work.viewer.travel) }

// proofBallMove sends a motion to the drag point while the ball holds the
// press.
func (work *Workspace) proofBallMove(instrument *cameraBall) error {
	_, end, err := work.cameraBallProofPoints(instrument)
	if err != nil {
		return err
	}
	if !instrument.dragging {
		return fmt.Errorf("trackball proof interrupted before motion: pointer capture is no longer held")
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
	if !instrument.dragging {
		return fmt.Errorf("trackball proof interrupted during held motion")
	}
	return nil
}

// ProofTrackballRelease commits the camera edit only after the held frame capture.
func (work *Workspace) ProofTrackballRelease() error { return work.proofBallRelease(work.viewer.ball) }

// ProofTravelRelease dispatches native input to the adjacent camera movement ball.
func (work *Workspace) ProofTravelRelease() error { return work.proofBallRelease(work.viewer.travel) }

// proofBallRelease sends the release at the drag point and ends the held
// gesture.
func (work *Workspace) proofBallRelease(instrument *cameraBall) error {
	_, end, err := work.cameraBallProofPoints(instrument)
	if err != nil {
		return err
	}
	if !instrument.dragging {
		return fmt.Errorf("trackball proof interrupted before planned release")
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
	if instrument.dragging {
		return fmt.Errorf("trackball proof release did not end pointer capture")
	}
	return nil
}
