package workspace

import (
	"context"
	"image"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/widget"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/session/placement"
)

func testFilmFrame(number int) *stagedView {
	return &stagedView{
		scene: &placement.DisplayScene{Width: 32, Height: 24},
		frame: &stagewire.FilmFrame{Number: number, Count: 3, Time: time.Duration(number-1) * time.Second},
	}
}

func TestWorkingFrameLatestResultKeepsDraftAndSubmissionsUntouched(t *testing.T) {
	work, path, target := jobSaveFixture(t)
	defer work.closeStaged()
	const source = "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: location.yaml}\nfps: 1\nduration: {value: 3, unit: s}\n"
	work.editor.ApplyText(source)
	if !work.saveAt(path) {
		t.Fatal(work.diagnostic)
	}
	work.working.path, work.working.source = path, source
	work.working.frame = testFilmFrame(1).frame
	if work.workingFramePicker() == nil {
		t.Fatal("multi-frame job has no picker")
	}
	if err := work.chooseWorkingFrame(4); err == nil {
		t.Fatal("accepted an out-of-range frame")
	}
	firstStarted := make(chan struct{})
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(firstStarted)
		<-ctx.Done()
		return testFilmFrame(2), nil
	}
	if err := work.chooseWorkingFrame(2); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("first frame request did not start")
	}
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return testFilmFrame(3), nil
	}
	if err := work.chooseWorkingFrame(3); err != nil {
		t.Fatal(err)
	}
	waitStaged(t, work)
	if accepted := work.working.frame; accepted == nil || accepted.Number != 3 || accepted.Time != 2*time.Second {
		t.Fatalf("accepted stale frame: %+v", accepted)
	}
	work.working.cameraName, work.working.cameraWide = "other", 40
	if request := work.workingStageCamera(); request.Frame != 3 || request.Name != "other" || request.Wide != 40 {
		t.Fatalf("camera restaging lost the selected frame: %+v", request)
	}
	if work.Running() || work.state.document.Text() != source {
		t.Fatal("frame selection changed the job or started a submission")
	}
	if !work.saveAt(path) || work.workingStageCamera().Frame != 3 {
		t.Fatal("saving unchanged source lost the selected frame")
	}
	disk, err := os.ReadFile(path)
	if err != nil || string(disk) != source {
		t.Fatalf("saved source gained working-frame data: %v %s", err, disk)
	}
	if !work.saveAt(target) || work.workingStageCamera().Frame != 0 {
		t.Fatal("Save As reused the previous path's frame request")
	}
}

func TestWorkingFrameRefusesAHelperCardForTheWrongFrame(t *testing.T) {
	work := playWorkspace(t)
	defer work.closeStaged()
	work.working.path, work.working.source = work.state.path, work.state.document.Text()
	work.working.frame = testFilmFrame(1).frame
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return testFilmFrame(1), nil
	}
	if err := work.chooseWorkingFrame(2); err != nil {
		t.Fatal(err)
	}
	waitStaged(t, work)
	if work.working.scene != nil || !strings.Contains(work.working.failure, "wrong film frame") {
		t.Fatalf("same-source result bypassed frame validation: %q", work.working.failure)
	}
	work.working.frame = &stagewire.FilmFrame{Number: 1, Count: 1}
	if work.workingFramePicker() != nil {
		t.Fatal("single-frame film should not show navigation")
	}
}

func TestWorkingFrameMountedControlsDisableWhenDraftChanges(t *testing.T) {
	work, _, _ := jobSaveFixture(t)
	defer work.closeStaged()
	work.working.path, work.working.source = work.state.path, work.state.document.Text()
	work.working.frame = testFilmFrame(2).frame
	picker := work.workingFramePicker()
	next := libraryWidget(picker, "Next frame").(widget.Focusable)
	previous := libraryWidget(picker, "Previous frame").(widget.Focusable)
	if !next.IsFocusable() || !previous.IsFocusable() {
		t.Fatal("interior frame navigation is disabled")
	}
	photo := image.NewRGBA(image.Rect(0, 0, 32, 24))
	work.working.photograph = photo
	work.viewer.set(photo)
	started := make(chan struct{})
	work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
		close(started)
		<-ctx.Done()
		return testFilmFrame(3), nil
	}
	if err := work.chooseWorkingFrame(3); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("edited frame request did not start")
	}
	work.state.document.Replace(work.state.document.Text() + "\n# edit")
	if next.IsFocusable() || previous.IsFocusable() {
		t.Fatal("mounted controls used the previous draft's frame count")
	}
	work.refreshLayout = false
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return testFilmFrame(1), nil
	}
	work.updateWorking()
	if work.working.frame != nil || !work.refreshLayout || next.IsFocusable() || previous.IsFocusable() {
		t.Fatal("clearing a frame left mounted controls enabled or missed their layout refresh")
	}
	waitStaged(t, work)
	if frame := work.working.frame; frame == nil || frame.Number != 1 {
		t.Fatalf("source edit accepted the previous draft's frame: %+v, %q", frame, work.working.failure)
	}
	if work.working.photograph != photo || work.viewer.source != photo {
		t.Fatal("inspection dropped the retained photograph")
	}
}
