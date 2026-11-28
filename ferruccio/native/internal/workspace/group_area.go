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

// groupAreaPanel edits the actual region a terrain stand uses for placement.
func (work *Workspace) groupAreaPanel() widget.Widget {
	group, ok := work.selectedGroup()
	if !ok {
		return work.text("Choose a generated group.")
	}
	area, ok := work.state.document.GroupArea(group.ID)
	if !ok {
		return work.text("This group has no editable terrain region.")
	}
	if work.play.groupAreaID != group.ID || work.play.groupAreaSource != work.state.document.Text() {
		work.play.groupAreaID, work.play.groupAreaSource = group.ID, work.state.document.Text()
		for i, n := range area {
			work.play.groupAreaText[i] = strconv.FormatFloat(n, 'g', -1, 64)
		}
	}
	guidance := "Metres east and north of the terrain's southwest corner. Edits regenerate the group. Off-terrain bounds may yield fewer members."
	items := []widget.Widget{work.paragraphAtWidth(guidance, appearance.Ink, playDrawerContentWidth)}
	var fields [4]*textfield.Widget
	for i, name := range []string{"West", "South", "East", "North"} {
		index := i
		field := textfield.New(textfield.InitialValue(work.play.groupAreaText[index]), textfield.A11yLabel(name+" metres"), textfield.PainterOpt(appearance.TextFieldPainter{Size: work.size}), textfield.OnChange(func(text string) { work.play.groupAreaText[index] = text }))
		field.SetID(name + " metres")
		fields[index] = field
		label := primitives.VBox(work.text(name)).Width(84)
		row := primitives.HBox(label, primitives.Expanded(field)).Gap(12).
			CrossAlign(primitives.CrossAxisCenter)
		items = append(items, row)
	}
	items = append(items, work.action("Set region", true, func() bool { return false }, func() {
		for i, field := range fields {
			work.play.groupAreaText[i] = field.Text()
		}
		work.problem(work.submitGroupArea())
	}))
	return primitives.VBox(items...).Gap(8)
}

// submitGroupArea makes one undoable edit of the authored stand region.
func (work *Workspace) submitGroupArea() error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	group, ok := work.selectedGroup()
	if !ok {
		return fmt.Errorf("choose a generated group")
	}
	var area [4]float64
	for i, text := range work.play.groupAreaText {
		number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return fmt.Errorf("enter four region coordinates in metres")
		}
		area[i] = number
	}
	return work.applyGroupArea(func() error {
		return work.state.document.SetGroupArea(group.ID, area)
	})
}

// shiftGroupArea moves the authored region without resizing it.
func (work *Workspace) shiftGroupArea(id string, east, north float64) error {
	return work.applyGroupArea(func() error {
		return work.state.document.ShiftGroupArea(id, east, north)
	})
}

func (work *Workspace) applyGroupArea(edit func() error) error {
	before := work.state.document.Text()
	if err := edit(); err != nil {
		return err
	}
	if work.state.document.Text() == before {
		return nil
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.needsCheck = false
	work.working.dirty = true
	work.play.groupAreaID = ""
	work.caption = "Group region changed · working view updating"
	work.status = "Group region changed · ⌘Z to undo"
	work.diagnostic = ""
	work.refreshLayout = true
	work.invalidate()
	return nil
}
