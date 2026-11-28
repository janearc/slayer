package workspace

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// enterPlay keeps the current draft and requests its native working view.
func (work *Workspace) enterPlay() {
	if work.state.document.Kind() == "catalogue" || work.refuseWhileFull("opening the working view") {
		return
	}
	work.composition = "play"
	work.setupPlay()
	if work.photographVisible() && work.acceptedPath == work.state.path {
		work.play.renderedText = work.acceptedText
	}
	_ = work.SetWorkingMode("solid")
	work.rebuild()
}

// compositionActions retains a source-editing route below the photograph.
func (work *Workspace) compositionActions() widget.Widget {
	return primitives.HBox(
		work.live(work.cameraEditingHint).Color(appearance.Cyan),
		primitives.Expanded(primitives.Box()),
		work.action("Edit YAML", false, func() bool { return false }, func() {
			work.composition = "observe"
			work.rebuild()
		}),
	).Gap(10).PaddingXY(20, 12).Background(appearance.Panel)
}
