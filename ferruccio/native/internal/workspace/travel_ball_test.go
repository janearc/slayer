package workspace

import (
	"math"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/hillside/placement"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

func travelWorkspace(t *testing.T) *Workspace {
	t.Helper()
	work := playWorkspace(t)
	read, err := set.Read(work.state.path, []byte(work.state.document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene = scene
	work.working.source = work.state.document.Text()
	work.working.path = work.state.path
	work.working.displayed = true
	drawWholePlayRoot(t, work)
	return work
}

func travelInput(t *testing.T, work *Workspace, kind event.MouseEventType, buttons event.ButtonState, delta geometry.Point, mods event.Modifiers) {
	t.Helper()
	ball := work.viewer.travel
	center, _ := ball.disc(work.viewer.Bounds())
	bounds, screen := work.viewer.Bounds(), work.viewer.ScreenBounds()
	point := geometry.Pt(center.X+screen.Min.X-bounds.Min.X+delta.X, center.Y+screen.Min.Y-bounds.Min.Y+delta.Y)
	button := event.ButtonNone
	if kind == event.MousePress || kind == event.MouseRelease {
		button = event.ButtonLeft
	}
	work.ui.HandleEvent(event.NewMouseEvent(kind, button, buttons, point, point, mods))
}

// The second ball must move a real scene, commit once and undo through window input.
func TestTravelBallWindowDragCommitsOneCameraTranslation(t *testing.T) {
	work := travelWorkspace(t)
	before := work.state.document.Text()
	base := *work.working.scene
	selected, mode := work.play.selected, work.play.mode
	travelInput(t, work, event.MousePress, event.ButtonStateLeft, geometry.Pt(0, 0), event.ModNone)
	if !work.viewer.travel.dragging || work.viewer.ball.dragging {
		t.Fatal("second ball did not capture its own gesture")
	}
	travelInput(t, work, event.MouseMove, event.ButtonStateLeft, geometry.Pt(29, -29), event.ModNone)
	held := *work.working.scene
	displacement := held.Eye.Origin.Sub(base.Eye.Origin)
	if displacement.Len() == 0 {
		t.Fatal("held camera did not move")
	}
	if held.Focus.Sub(base.Focus).Sub(displacement).Len() > 1e-10 {
		t.Fatal("camera movement changed view direction")
	}
	if work.state.document.Text() != before {
		t.Fatal("held movement edited source")
	}
	if work.play.selected != selected || work.play.mode != mode {
		t.Fatal("camera ball changed object selection or manipulation mode")
	}
	travelInput(t, work, event.MouseRelease, 0, geometry.Pt(29, -29), event.ModNone)
	changed := work.state.document.Text()
	if work.viewer.travel.dragging || changed == before || work.pending.text != changed {
		t.Fatal("release did not commit and queue current camera")
	}
	read, err := set.Read(work.state.path, []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Eye.Origin.Sub(held.Eye.Origin).Len() > 1e-9 || saved.Focus.Sub(held.Focus).Len() > 1e-9 {
		t.Fatal("saved pose differs from held working pose")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if work.state.document.Text() != before {
		t.Fatal("one undo did not restore entire camera movement")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if work.state.document.Text() != before {
		t.Fatal("movement created more than one undo entry")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'Z', event.ModSuper|event.ModShift))
	if work.state.document.Text() != changed {
		t.Fatal("redo did not restore movement")
	}
}

func TestTravelBallShiftRaisesAndEscapeRestoresWorkingPose(t *testing.T) {
	work := travelWorkspace(t)
	before := work.state.document.Text()
	at, focus := work.working.scene.Eye.Origin, work.working.scene.Focus
	travelInput(t, work, event.MousePress, event.ButtonStateLeft, geometry.Pt(0, 0), event.ModNone)
	travelInput(t, work, event.MouseMove, event.ButtonStateLeft, geometry.Pt(0, -29), event.ModShift)
	delta := work.working.scene.Eye.Origin.Sub(at)
	if !(delta.Y > 0) || math.Abs(delta.X)+math.Abs(delta.Z) > 1e-10 {
		t.Fatal("Shift vertical movement did not raise camera in world space")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, event.ModNone))
	if work.viewer.travel.dragging || work.state.document.Text() != before || work.pending.document != nil {
		t.Fatal("cancel committed source or queued a render")
	}
	if work.working.scene.Eye.Origin.Sub(at).Len() != 0 || work.working.scene.Focus.Sub(focus).Len() != 0 {
		t.Fatal("cancel did not restore exact working pose")
	}
}

// Known camera geometry establishes translation sign and scale without using the implementation as oracle.
func TestTravelBallMovesRightAndForwardWithoutChangingLens(t *testing.T) {
	work := travelWorkspace(t)
	eye, err := maths.View(maths.V(0, 0, 10), maths.V(0, 0, 0), maths.YAxis)
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene.Eye = *eye
	work.working.scene.Focus = maths.V(0, 0, 0)
	lens := work.working.scene.Lens
	travelInput(t, work, event.MousePress, event.ButtonStateLeft, geometry.Pt(0, 0), event.ModNone)
	travelInput(t, work, event.MouseMove, event.ButtonStateLeft, geometry.Pt(29, -29), event.ModNone)
	at := work.working.scene.Eye.Origin
	if math.Abs(at.X-1.25) > 1e-10 || math.Abs(at.Y) > 1e-10 || math.Abs(at.Z-8.75) > 1e-10 {
		t.Fatalf("right/forward camera = %v, want [1.25 0 8.75]", at)
	}
	if work.working.scene.Lens != lens {
		t.Fatal("translation changed lens")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, event.ModNone))
}
