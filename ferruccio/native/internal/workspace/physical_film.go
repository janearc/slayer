package workspace

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/film"
)

// physicalGauges takes the named dimensions from the film producer.
func physicalGauges() []film.Gauge {
	return []film.Gauge{film.Super8, film.Sixteen, film.Academy, film.Super35, film.FullFrame, film.Film65, film.IMAX}
}

// physicalFilmPanel edits the frame and lens of an existing physical rig.
func (work *Workspace) physicalFilmPanel() widget.Widget {
	frame, ok := work.state.document.PhysicalFrame()
	if !ok {
		return work.text("This set does not have a complete physical film rig.")
	}
	gauges := physicalGauges()
	labels := []string{fmt.Sprintf("Current · %.2f × %.2f mm", frame[0], frame[1])}
	for _, g := range gauges {
		labels = append(labels, fmt.Sprintf("%s · %.2f × %.2f mm", g.Name, g.Width, g.Height))
	}
	if work.play.gaugeChoice < 0 || work.play.gaugeChoice >= len(labels) {
		work.play.gaugeChoice = 0
	}
	choice := dropdown.New(dropdown.Items(labels...), dropdown.Selected(work.play.gaugeChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) { work.play.gaugeChoice = index }))
	choice.SetID("Physical frame")
	set := work.action("Set frame", true, func() bool { return false }, func() { work.problem(work.changePhysicalFrame(work.play.gaugeChoice)) })
	return primitives.VBox(work.text("Physical frame"), primitives.HBox(primitives.Expanded(choice)), set, work.physicalLensPanel(),
		work.paragraphAtWidth("The working view shows a pinhole approximation. The f-stop affects the full render.", appearance.Ink, playDrawerContentWidth)).Gap(12)
}

// physicalLensPanel keeps typed optics while the same authored source is open.
func (work *Workspace) physicalLensPanel() widget.Widget {
	document := work.state.document
	if !document.Matches(work.play.lensSource) {
		lens, ok := document.PhysicalLens()
		if !ok {
			return work.text("The physical lens could not be read.")
		}
		work.play.lensSource = document.Text()
		work.play.lensText = [2]string{
			strconv.FormatFloat(lens.FocalMM, 'f', -1, 64),
			strconv.FormatFloat(lens.FStop, 'f', -1, 64),
		}
	}
	columns := make([]widget.Widget, 0, len(work.play.lensText))
	for index, label := range []string{"Focal length (mm)", "F-stop"} {
		field := textfield.New(textfield.InitialValue(work.play.lensText[index]), textfield.A11yLabel(label),
			textfield.PainterOpt(appearance.TextFieldPainter{Size: work.size}),
			textfield.OnChange(func(text string) { work.play.lensText[index] = text }))
		field.SetID(label)
		columns = append(columns, primitives.Expanded(primitives.VBox(work.text(label), field).Gap(8)))
	}
	set := work.action("Set lens", true, func() bool { return false }, func() {
		work.problem(work.changePhysicalLens(work.play.lensText))
	})
	return primitives.VBox(primitives.HBox(columns...).Gap(12), set).Gap(12)
}

// changePhysicalLens applies the two typed values as a single source edit.
func (work *Workspace) changePhysicalLens(text [2]string) error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	focal, err := strconv.ParseFloat(strings.TrimSpace(text[0]), 64)
	if err != nil {
		return fmt.Errorf("enter a focal length in millimetres")
	}
	stop, err := strconv.ParseFloat(strings.TrimSpace(text[1]), 64)
	if err != nil {
		return fmt.Errorf("enter a numeric f-stop")
	}
	before := work.state.document.Text()
	lens := draft.PhysicalLens{FocalMM: focal, FStop: stop}
	if err := work.state.document.SetPhysicalLens(lens); err != nil {
		return err
	}
	work.finishPhysicalEdit(before, "Physical lens changed")
	return nil
}

// changePhysicalFrame makes one undoable source edit without starting a render.
func (work *Workspace) changePhysicalFrame(index int) error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	gauges := physicalGauges()
	if index <= 0 || index > len(gauges) {
		return fmt.Errorf("choose a physical frame")
	}
	before := work.state.document.Text()
	g := gauges[index-1]
	if err := work.state.document.SetPhysicalFrame([2]float64{g.Width, g.Height}); err != nil {
		return err
	}
	work.finishPhysicalEdit(before, "Physical frame changed")
	return nil
}

// finishPhysicalEdit records one editor history entry for a changed rig.
func (work *Workspace) finishPhysicalEdit(before, status string) {
	if work.state.document.Matches(before) {
		return
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.working.dirty = true
	work.play.gaugeChoice = 0
	work.caption = status + " · working view updating"
	work.status = status + " · ⌘Z to undo"
	work.diagnostic = ""
	work.refreshLayout = true
	work.invalidate()
}
