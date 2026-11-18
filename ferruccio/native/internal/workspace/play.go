package workspace

import (
	"fmt"
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/dropdown"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/editor"
	"github.com/janearc/slayer/ferruccio/native/internal/picking"
	"math"
	"strings"
	"time"
)

// playState retains the intention and selection independently of disclosure.
type playState struct {
	mode, selected, disclosure, renderedText string
	geometry                                 *picking.Scene
}

// setupPlay connects the world surface to real scene identity and source edits.
func (work *Workspace) setupPlay() {
	work.viewer.ball = &cameraBall{work: work}
	work.viewer.travel = &cameraBall{work: work, translation: true}
	work.play.mode = "move"
	work.viewer.mode = "move"
	work.viewer.selectionCurrent = func() bool {
		if work.workingCurrent() || work.play.renderedText == work.state.document.Text() {
			return true
		}
		work.caption = work.retainedImageCaption("new preview pending · choose Look to examine the view")
		work.invalidate()
		return false
	}
	work.viewer.hoverObject = func(x, y float64) string {
		if !work.workingCurrent() && (work.play.geometry == nil || work.play.renderedText != work.state.document.Text()) {
			return ""
		}
		id := work.workingPick(x, y)
		objects, err := work.state.document.Objects()
		if err != nil {
			return ""
		}
		for _, object := range objects {
			if object.ID() == id {
				return object.Name()
			}
		}
		return ""
	}
	work.viewer.selectObject = func(x, y float64) bool {
		if !work.workingCurrent() && (work.play.geometry == nil || work.play.renderedText != work.state.document.Text()) {
			return false
		}
		id := work.workingPick(x, y)
		if id == "" {
			return false
		}
		work.play.selected = id
		work.refreshLayout = true
		work.invalidate()
		return true
	}
	work.viewer.changeObject = func(x0, y0, x1, y1 float64) {
		if work.play.mode == "turn" {
			work.problem(work.changePose(0, 0, (x1-x0)*180))
			return
		}
		var east, north float64
		var err error
		if work.working.gesture != nil {
			east, north, err = work.working.gesture.Drag(work.play.selected, x0, y0, x1, y1)
		} else if work.play.geometry != nil {
			east, north, err = work.play.geometry.Drag(work.play.selected, x0, y0, x1, y1)
		} else {
			return
		}
		if err != nil {
			work.problem(err)
			return
		}
		work.problem(work.changePose(east, north, 0))
	}
	work.viewer.beginWorking = work.beginWorking
	work.viewer.moveWorking = work.moveWorking
	work.viewer.endWorking = work.endWorking
	work.viewer.playKey = work.playKey
}

// setPlayMode changes one explicit intention without changing the scene.
func (work *Workspace) setPlayMode(mode string) {
	work.play.mode = mode
	work.viewer.mode = mode
	work.refreshLayout = true
	work.invalidate()
	work.ui.Window().Context().RequestFocus(work.viewer)
}

// changePose creates one source-history entry and one debounced native draw.
func (work *Workspace) changePose(east, north, heading float64) error {
	if work.play.selected == "" {
		return fmt.Errorf("choose an object in the world first")
	}
	if heading != 0 {
		object, ok := work.selectedObject()
		if !ok {
			return fmt.Errorf("selected object is no longer present")
		}
		current, _ := object.Heading()
		next := math.Mod(current+heading, 360)
		if next < 0 {
			next += 360
		}
		heading = next - current
	}
	before := work.state.document.Text()
	err := work.state.document.ChangePose(draft.PoseChange{ID: work.play.selected, East: east, North: north, Heading: heading, LengthUnit: "m", AngleUnit: "deg"})
	if err != nil {
		return err
	}
	if before == work.state.document.Text() {
		return nil
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.pending.remember(work.state, work.specimen, time.Now())
	work.caption = "Previous render · your change is being drawn"
	work.status = "Change made · ⌘Z to undo"
	work.diagnostic = ""
	work.wakeNavigation(navigationDelay)
	work.invalidate()
	return nil
}

// playKey gives labelled modes and placement the same keyboard path as pointer edits.
func (work *Workspace) playKey(key *event.KeyEvent) bool {
	if key.Modifiers().HasAny(event.ModSuper | event.ModCtrl | event.ModAlt) {
		return false
	}
	switch key.Key {
	case event.KeyM:
		work.setPlayMode("move")
		return true
	case event.KeyT:
		work.setPlayMode("turn")
		return true
	case event.KeyL:
		work.setPlayMode("look")
		return true
	case event.KeyEscape:
		work.play.disclosure = ""
		work.rebuild()
		return true
	}
	if work.play.mode == "look" || work.play.selected == "" {
		return false
	}
	step := .25 // Authored keyboard placement step, metres; Shift uses one metre.
	if key.Modifiers().Has(event.ModShift) {
		step = 1
	}
	east, north, turn := 0.0, 0.0, 0.0
	switch key.Key {
	case event.KeyLeft:
		east = -step
		turn = -5
	case event.KeyRight:
		east = step
		turn = 5
	case event.KeyUp:
		north = step
		turn = 5
	case event.KeyDown:
		north = -step
		turn = -5
	default:
		return false
	}
	if work.play.mode == "turn" {
		work.problem(work.changePose(0, 0, turn))
	} else {
		work.problem(work.changePose(east, north, 0))
	}
	return true
}

// selectedObject returns the current authored values, never an invented readout.
func (work *Workspace) selectedObject() (draft.Object, bool) {
	objects, err := work.state.document.Objects()
	if err != nil {
		return draft.Object{}, false
	}
	for _, object := range objects {
		if object.ID() == work.play.selected {
			return object, true
		}
	}
	return draft.Object{}, false
}

// playLayout makes the world the main surface and discloses machinery on request.
func (work *Workspace) playLayout() widget.Widget {
	idle := func() bool { return false }
	top := primitives.HBox(
		primitives.Text("ferruccio").FontSize(28).Color(appearance.Pink),
		primitives.Expanded(work.scenePicker()),
		work.action("Open…", false, idle, work.open),
		work.action("Save…", false, idle, func() { work.save(false) }),
		work.drawControl(),
		work.action("Render fully", true, func() bool { return work.Running() }, func() { work.problem(work.StartFullRender()) }),
		work.action("Source", work.play.disclosure == "source", idle, func() { work.toggleDisclosure("source") }),
	).Gap(18).PaddingXY(24, 12).Background(appearance.Ground)
	world := primitives.VBox(primitives.Expanded(work.viewer), work.live(func() string { return work.caption })).Gap(8)
	var center widget.Widget = world
	if work.play.disclosure != "" {
		var detail widget.Widget
		if work.play.disclosure == "source" {
			detail = work.editor
		} else {
			detail = work.construction()
		}
		panel := primitives.VBox(primitives.HBox(work.text(strings.Title(work.play.disclosure)).Color(appearance.Pink), primitives.Expanded(primitives.Box()), work.action("Return to world", false, idle, func() { work.play.disclosure = ""; work.rebuild() })).Gap(12), primitives.Expanded(detail)).Width(490).Gap(16).Padding(20).Background(appearance.Panel)
		center = primitives.HBox(primitives.Expanded(world), panel).Gap(12)
	}
	rail := primitives.HBox(
		work.intent("move", "Move · M"),
		work.intent("turn", "Turn · T"),
		work.intent("look", "Look · L"),
		primitives.Expanded(work.objectPicker()),
		work.workingPicker(),
		work.action("Undo", false, idle, func() { work.undoCamera(work.ui.Window().Context(), false) }),
		work.action("Redo", false, idle, func() { work.undoCamera(work.ui.Window().Context(), true) }),
		work.action("Made from", work.play.disclosure == "construction", func() bool { return work.play.selected == "" }, func() { work.toggleDisclosure("construction") }),
	).Gap(12).PaddingXY(24, 14).Background(appearance.Panel)
	readout := primitives.HBox(primitives.Expanded(work.live(work.playReadout)), work.themePicker(), work.action("Type size", false, idle, func() {
		if work.size >= 24 {
			work.size = 16
		} else {
			work.size += 2
		}
		work.rebuild()
	})).Gap(18).PaddingXY(24, 8).Background(appearance.Ground)
	return primitives.VBox(top, primitives.Expanded(center), rail, readout).Gap(8).Background(appearance.Ground)
}

// toggleDisclosure preserves selection, view and source history on return.
func (work *Workspace) toggleDisclosure(name string) {
	if work.play.disclosure == name {
		work.play.disclosure = ""
	} else {
		work.play.disclosure = name
	}
	work.rebuild()
}

// objectPicker exposes the same native selection as an accessible named control.
func (work *Workspace) objectPicker() widget.Widget {
	objects, err := work.state.document.Objects()
	if err != nil || len(objects) == 0 {
		return work.text("Open a scene to explore")
	}
	labels := make([]string, len(objects))
	selected := -1
	for index, object := range objects {
		labels[index] = object.Name()
		if object.ID() == work.play.selected {
			selected = index
		}
	}
	choice := dropdown.New(dropdown.Items(labels...), dropdown.Selected(selected), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		if index >= 0 && index < len(objects) {
			work.play.selected = objects[index].ID()
			work.rebuild()
		}
	}))
	choice.SetID("Choose an object")
	return choice
}

// playReadout keeps computation state, exact units and first-touch cues visible.
func (work *Workspace) playReadout() string {
	if work.gesturing() && work.working.gesture != nil {
		if work.viewer.travel != nil && work.viewer.travel.dragging {
			return "Camera travel · drag left/right or approach/retreat · Shift to rise/fall · Escape cancels"
		}
		if work.viewer.ball != nil && work.viewer.ball.dragging {
			return "Camera trackball · Shift constrains an axis · Escape cancels"
		}
		if !work.viewer.dragObject {
			return "Working camera · release to commit · Escape to cancel"
		}
		for _, object := range work.working.scene.Objects {
			if object.ID == work.play.selected {
				return fmt.Sprintf("%s · working E %.2f m   N %.2f m   Up %.2f m · release to commit", object.Name, object.At.Origin.X, -object.At.Origin.Z, object.At.Origin.Y)
			}
		}
	}
	if work.diagnostic != "" {
		return work.diagnostic
	}
	if work.cancel != nil || work.pending.document != nil {
		return work.previewProgressText()
	}
	object, ok := work.selectedObject()
	if !ok {
		return "Grab the ship · scroll to approach · ⌘Z undo"
	}
	at := object.Position()
	heading, _ := object.Heading()
	return fmt.Sprintf("%s · E %.2f m   N %.2f m   Up %.2f m   Heading %.1f°", object.Name(), at[0], at[1], at[2], heading)
}

// construction reveals the actual producer recipe alongside the selected subject.
func (work *Workspace) construction() widget.Widget {
	object, ok := work.selectedObject()
	if !ok {
		return work.text("Choose something in the world.")
	}
	name := ""
	switch object.Kind() {
	case "mcrn-larson-landed":
		name = "native-larson.yaml"
	case "human-scale-2m":
		name = "native-human.yaml"
	}
	raw, err := assets.Read(name)
	text := object.Kind() + "\n\nNative manufacture recipe. Full scene placement is available in Source."
	if err == nil {
		text += "\n\n" + string(raw)
	}
	if object.Kind() == "mcrn-larson-landed" {
		if gear, err := assets.Read("native-larson-landed.yaml"); err == nil {
			text += "\n\n" + string(gear)
		}
	}
	// A read-only editor supplies existing wheel and keyboard scrolling.
	source := editor.NewReadOnly(text)
	source.SetFontSize(work.size)
	source.SetEscapeHandler(work.returnToWorld)
	return source
}

// returnToWorld dismisses disclosure and restores a useful keyboard focus.
func (work *Workspace) returnToWorld() {
	if work.composition != "play" {
		return
	}
	work.play.disclosure = ""
	work.rebuild()
	work.ui.Window().Context().RequestFocus(work.viewer)
}

// drawControl keeps a queued or running computation cancellable in the world view.
func (work *Workspace) drawControl() widget.Widget {
	if work.Running() {
		return work.action("Stop", false, func() bool { return false }, func() {
			work.stopPreview()
			work.caption = work.retainedImageCaption(work.status)
			work.rebuild()
		})
	}
	return work.action("Preview", false, func() bool { return !work.rendererConfigured() }, func() { work.problem(work.StartPreview()) })
}

// intent is one accessible button with a bespoke native instrument painter.
func (work *Workspace) intent(mode, label string) widget.Widget {
	control := button.New(button.TextOpt(label), button.OnClick(func() { work.setPlayMode(mode) }), button.PainterOpt(appearance.InstrumentPainter{ButtonPainter: appearance.ButtonPainter{Size: work.size, Primary: work.play.mode == mode}, Kind: mode}))
	control.SetID(label)
	return control
}
