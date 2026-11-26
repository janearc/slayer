package workspace

import (
	"fmt"

	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ninjafilm"
)

// stockPanel offers the producer's development stocks for a job.
func (work *Workspace) stockPanel() widget.Widget {
	current, ok := work.state.document.StockChoice()
	if !ok {
		return work.text("This job has an authored print curve; stock cannot be used beside it.")
	}
	stocks, err := ninjafilm.Stocks()
	if err != nil {
		return work.text("Film stock catalogue unavailable: " + err.Error())
	}
	description := "Current · scene development"
	if current != "" {
		description = "Current · " + current
	}
	labels := append([]string{description}, stocks...)
	if work.play.stockChoice < 0 || work.play.stockChoice >= len(labels) {
		work.play.stockChoice = 0
	}
	choice := dropdown.New(dropdown.Items(labels...), dropdown.Selected(work.play.stockChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) { work.play.stockChoice = index }))
	choice.SetID("Development stock")
	set := work.action("Set stock", true, func() bool { return false }, func() { work.problem(work.changeStock(work.play.stockChoice)) })
	return primitives.VBox(work.text("Development stock"), choice, work.paragraph("Stock changes development and grain. An existing stock's seed and rating are retained.", appearance.Ink), set).Gap(12)
}

// changeStock records one undoable job edit without starting a render.
func (work *Workspace) changeStock(index int) error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	stocks, err := ninjafilm.Stocks()
	if err != nil {
		return err
	}
	if index <= 0 || index > len(stocks) {
		return fmt.Errorf("choose a development stock")
	}
	current := work.workingCurrent()
	before := work.state.document.Text()
	if err := work.state.document.SetStock(stocks[index-1]); err != nil {
		return err
	}
	if work.state.document.Text() == before {
		return nil
	}
	if current {
		// Stock only develops the finished light; it does not change the
		// staged geometry or camera. Preserve its exact old source for undo.
		work.rememberWorking(work.working.scene)
		work.working.source = work.state.document.Text()
		work.working.waiting = true
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.working.dirty = true
	work.play.stockChoice = 0
	work.caption = "Stock changed · working view unchanged"
	work.status = "Stock changed · ⌘Z to undo"
	work.diagnostic = ""
	work.refreshLayout = true
	work.invalidate()
	return nil
}
