package workspace

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

const placementResultLimit = 12

func (work *Workspace) objectPicker() widget.Widget {
	labels, ids := work.placementChoices()
	if len(ids) > placementResultLimit {
		return work.action("Find placement…", work.play.disclosure == "find placement", func() bool { return false }, func() {
			work.play.placementQuery, work.play.placementFilter = "", ""
			work.toggleDisclosure("find placement")
		})
	}
	if len(ids) > 0 {
		return work.placementPicker(labels, ids)
	}
	if work.state.document.ReferencedSet() != "" {
		return work.action("Open scene to edit", false, func() bool { return work.fullRunning }, work.openReferencedSet)
	}
	return work.text("Open a scene to explore")
}

func (work *Workspace) placementSearch() widget.Widget {
	labels, ids := work.placementChoices()
	submit := func(text string) {
		work.play.placementQuery, work.play.placementFilter = text, text
		matches := placementMatches(text, labels, ids)
		if len(matches) == 1 {
			work.chooseFoundPlacement(ids[matches[0]])
			return
		}
		work.rebuild()
		work.ui.Window().Context().RequestFocus(work.play.placementSearchField)
	}
	search := textfield.New(textfield.InitialValue(work.play.placementQuery),
		textfield.Placeholder("Search by placement or group name"),
		textfield.A11yLabel("Search placements"),
		textfield.PainterOpt(appearance.TextFieldPainter{Size: work.size}),
		textfield.OnChange(func(text string) { work.play.placementQuery = text }),
		textfield.OnSubmit(submit))
	search.SetID("Search placements")
	work.play.placementSearchField = search
	find := work.action("Find", false, func() bool { return false }, func() { submit(search.Text()) })
	row := primitives.HBox(primitives.Expanded(search), find).Gap(8)
	return primitives.VBox(row, work.placementResults(labels, ids)).Gap(8)
}

func placementMatches(query string, labels, ids []string) []int {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	var exact, partial []int
	for i, label := range labels {
		name := strings.ToLower(label)
		if strings.HasPrefix(ids[i], "scene.set.stands/") {
			name = strings.TrimSuffix(name, " · group")
		}
		if name == query {
			exact = append(exact, i)
		} else if strings.Contains(name, query) {
			partial = append(partial, i)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return partial
}

func (work *Workspace) placementResults(labels, ids []string) widget.Widget {
	query := work.play.placementFilter
	matches := placementMatches(query, labels, ids)
	switch {
	case strings.TrimSpace(query) == "":
		return work.text("Enter part of the authored name.")
	case len(matches) == 0:
		return work.text("No placements match that name.")
	case len(matches) > placementResultLimit:
		return work.text(fmt.Sprintf("%d matches. Enter more of the name.", len(matches)))
	}
	results := make([]widget.Widget, 0, len(matches))
	for _, index := range matches {
		results = append(results, work.action(labels[index], false, func() bool { return false }, func() {
			work.chooseFoundPlacement(ids[index])
		}))
	}
	return primitives.VBox(results...).Gap(4)
}

func (work *Workspace) chooseFoundPlacement(id string) {
	work.play.disclosure = ""
	work.selectWorkingObject(id)
	work.rebuild()
}
