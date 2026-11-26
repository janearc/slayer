package workspace

import (
	"fmt"

	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/film"
)

// physicalGauges takes the named dimensions from the film producer.
func physicalGauges() []film.Gauge {
	return []film.Gauge{film.Super8, film.Sixteen, film.Academy, film.Super35, film.FullFrame, film.Film65, film.IMAX}
}

// physicalFilmPanel edits an existing physical rig's frame.
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
	return primitives.VBox(work.text("Physical frame"), choice, work.paragraph("Changing the frame changes the field of view and image shape. The existing lens, focus and shutter stay as authored. The working view shows a pinhole approximation.", appearance.Ink), set).Gap(12)
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
	if work.state.document.Text() == before {
		return nil
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.working.dirty = true
	work.play.gaugeChoice = 0
	work.caption = "Physical frame changed · working view updating"
	work.status = "Physical frame changed · ⌘Z to undo"
	work.diagnostic = ""
	work.refreshLayout = true
	work.invalidate()
	return nil
}
