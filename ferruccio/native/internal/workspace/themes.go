package workspace

import (
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"slices"
)

// themePicker changes the live window using only Chroma's named colourways.
func (work *Workspace) themePicker() widget.Widget {
	names := appearance.AvailableThemes()
	choice := dropdown.New(dropdown.Items(names...),
		dropdown.Selected(slices.Index(names, appearance.SelectedTheme())),
		dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}),
		dropdown.OnChange(func(_ int, name string) {
			if err := appearance.SelectTheme(name); err != nil {
				work.problem(err)
				return
			}
			work.working.dirty = true
			work.rebuild()
		}))
	choice.SetID("Chroma theme")
	return choice
}
