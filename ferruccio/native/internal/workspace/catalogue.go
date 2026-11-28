package workspace

import (
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"slices"
)

// selectSpecimen resets document context and uses a receipt only when present.
func (work *Workspace) selectSpecimen() {
	work.stopPreview()
	if work.viewer != nil {
		work.viewer.SetNavigable(false)
	}
	work.specimen = ""
	if work.state.document.Kind() == "catalogue" {
		// Missing or obsolete metadata leaves the explicit chooser unselected.
		work.specimen, _ = preview.ResolveSpecimen(preview.Request{Path: work.state.path})
	}
}

// rendererConfigured checks the executable that owns the active format.
func (work *Workspace) rendererConfigured() bool {
	return work.rendererConfiguredAt(work.renderTools(), work.evidenceFolder())
}

func (work *Workspace) rendererConfiguredAt(tools preview.Tools, folder string) bool {
	if folder == "" {
		return false
	}
	if work.options.Tools.Transport == "slayer" {
		return tools.Slayer != "" && work.state.document.Kind() != "catalogue"
	}
	if tools.Azazel == "" || tools.Queue == "" {
		return false
	}
	if work.state.document.Kind() == "catalogue" {
		return tools.Catalogue != ""
	}
	return tools.Slayer != ""
}

// catalogueControl keeps specimen identity separate from unmodified plan YAML.
func (work *Workspace) catalogueControl() widget.Widget {
	if work.state.document.Kind() != "catalogue" {
		return primitives.Box().Height(0)
	}
	names, err := preview.Specimens()
	if err != nil {
		return work.text("Cannot read the catalogue menu")
	}
	choice := dropdown.New(dropdown.Items(names...), dropdown.Selected(slices.Index(names, work.specimen)), dropdown.Placeholder("Choose a specimen…"), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(_ int, name string) {
		if work.refuseWhileFull("choosing another specimen") {
			return
		}
		work.stopPreview()
		work.viewer.SetNavigable(false)
		work.specimen = name
		if work.output != "" {
			work.caption = "Previous preview · specimen has changed"
		}
		work.invalidate()
	}))
	choice.SetID("Catalogue specimen")
	return primitives.VBox(work.text("Specimen").Color(appearance.Cyan), choice).Gap(6)
}
