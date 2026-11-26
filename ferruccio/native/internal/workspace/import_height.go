package workspace

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// importHeightPanel edits an imported model's authored physical height.
func (work *Workspace) importHeightPanel() widget.Widget {
	object, ok := work.selectedObject()
	if !ok || object.Kind() != "import" {
		return work.text("Choose an imported model to change its height.")
	}
	if work.play.sizeObject != object.ID() {
		work.play.sizeObject, work.play.sizeText = object.ID(), ""
		if height, has := object.ImportHeight(); has {
			work.play.sizeText = strconv.FormatFloat(height, 'g', -1, 64)
		}
	}
	field := textfield.New(
		textfield.InitialValue(work.play.sizeText),
		textfield.Placeholder("e.g. 2.0"),
		textfield.A11yLabel("Model height in metres"),
		textfield.PainterOpt(appearance.TextFieldPainter{Size: work.size}),
		textfield.OnChange(func(text string) { work.play.sizeText = text }),
		textfield.OnSubmit(func(text string) { work.problem(work.submitImportHeight(text)) }),
	)
	field.SetID("Model height in metres")
	apply := work.action("Set height", true, func() bool { return false }, func() {
		work.problem(work.submitImportHeight(field.Text()))
	})
	return primitives.VBox(work.text("Model height (metres)"), field,
		work.text("Other dimensions scale with height."), apply).Gap(12)
}

// submitImportHeight checks the typed metre value before making the edit.
func (work *Workspace) submitImportHeight(text string) error {
	metres, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return fmt.Errorf("enter a height in metres")
	}
	if err := work.changeImportHeight(metres); err != nil {
		return err
	}
	work.play.sizeObject = ""
	work.refreshLayout = true
	return nil
}

// changeImportHeight records one undoable physical-size edit without rendering.
func (work *Workspace) changeImportHeight(metres float64) error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	if work.play.selected == "" {
		return fmt.Errorf("choose an imported model first")
	}
	before := work.state.document.Text()
	if err := work.state.document.SetImportHeight(work.play.selected, metres); err != nil {
		return err
	}
	if work.state.document.Text() == before {
		return nil
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.working.dirty = true
	work.caption = "Size changed · working view updating"
	work.status = "Size changed · ⌘Z to undo"
	work.diagnostic = ""
	work.invalidate()
	return nil
}
