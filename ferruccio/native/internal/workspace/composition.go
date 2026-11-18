package workspace

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// cameraControls gives the composition view the same editable camera gestures.
func (work *Workspace) cameraControls() widget.Widget {
	disabled := func() bool { return !work.viewer.navigable || !work.state.document.CanNavigate() }
	move := func(yaw, pitch, zoom float64) func() {
		return func() { work.problem(work.Navigate(yaw, pitch, zoom)) }
	}
	return primitives.HBox(
		work.text("Camera").Color(appearance.Cyan),
		work.action("Left", false, disabled, move(5, 0, 0)),
		work.action("Right", false, disabled, move(-5, 0, 0)),
		work.action("Up", false, disabled, move(0, -5, 0)),
		work.action("Down", false, disabled, move(0, 5, 0)),
		work.action("Zoom +", false, disabled, move(0, 0, .9)),
		work.action("Zoom −", false, disabled, move(0, 0, 1/.9)),
		primitives.Expanded(primitives.Box()),
		work.action("Edit YAML", false, func() bool { return false }, func() {
			work.composition = "observe"
			work.rebuild()
		}),
	).Gap(10).PaddingXY(20, 12).Background(appearance.Panel)
}
