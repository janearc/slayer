package workspace

import (
	"fmt"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

// workingStageCamera captures both camera and frame for the same draft.
func (work *Workspace) workingStageCamera() stagewire.Camera {
	live := &work.working
	camera := stagewire.Camera{Name: live.cameraName, Wide: live.cameraWide, PathRecipe: live.cameraPath}
	if live.path == work.state.path && work.state.document.Matches(live.source) {
		camera.Frame = live.requestedFrame
	}
	return camera
}

// workingFramePicker steps through the producer clock without writing source.
func (work *Workspace) workingFramePicker() widget.Widget {
	live := &work.working
	frame := work.currentWorkingFilm()
	if frame == nil || frame.Count < 2 {
		return nil
	}
	number := work.workingFrameNumber()
	label := fmt.Sprintf("Working frame %d of %d", number, live.frame.Count)
	if number == live.frame.Number && !live.staging && live.failure == "" {
		label += " · " + live.frame.Time.String()
	}
	return primitives.HBox(work.text(label), primitives.Expanded(primitives.Box()),
		work.action("Previous frame", false, func() bool { return !work.canStepWorkingFrame(-1) },
			func() { work.problem(work.chooseWorkingFrame(work.workingFrameNumber() - 1)) }),
		work.action("Next frame", false, func() bool { return !work.canStepWorkingFrame(1) },
			func() { work.problem(work.chooseWorkingFrame(work.workingFrameNumber() + 1)) }),
	).Gap(8).PaddingXY(16, 4).Background(appearance.Panel)
}

func (work *Workspace) currentWorkingFilm() *stagewire.FilmFrame {
	live := &work.working
	if live.path != work.state.path || !work.state.document.Matches(live.source) {
		return nil
	}
	return live.frame
}

func (work *Workspace) canStepWorkingFrame(step int) bool {
	frame := work.currentWorkingFilm()
	number := work.workingFrameNumber() + step
	return frame != nil && number >= 1 && number <= frame.Count
}

func (work *Workspace) resetWorkingFrame() {
	live := &work.working
	if live.frame != nil || live.requestedFrame != 0 {
		live.frame, live.requestedFrame = nil, 0
		work.refreshLayout = true
	}
}

func (work *Workspace) workingFrameNumber() int {
	if work.working.requestedFrame > 0 {
		return work.working.requestedFrame
	}
	if work.working.frame != nil {
		return work.working.frame.Number
	}
	return 0
}

// chooseWorkingFrame supersedes the previous display request for the same job.
func (work *Workspace) chooseWorkingFrame(number int) error {
	live := &work.working
	frame := work.currentWorkingFilm()
	if frame == nil {
		return fmt.Errorf("working frames are unavailable for this draft")
	}
	if number < 1 || number > frame.Count {
		return fmt.Errorf("working frame %d is outside the film", number)
	}
	if number == work.workingFrameNumber() {
		return nil
	}
	previous := live.requestedFrame
	live.requestedFrame = number
	build := work.stagedBuilder()
	if build == nil {
		live.requestedFrame = previous
		return fmt.Errorf("the staging helper is unavailable")
	}
	work.cancelViewportGesture()
	live.inspection = nil
	if live.mode == "" || live.mode == "rendered" {
		live.mode = "edges"
	}
	work.refreshLayout = true
	work.requestStaged(build, false)
	return nil
}
