package workspace

import (
	"fmt"
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
)

func TestPlacementSearchSelectsNamedStandAtSharedAnchor(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	marks := []draft.MapMark{
		{ID: "scene.set.things/anchor", Name: "anchor"},
		{ID: "scene.set.stands/near ferries", Name: "near ferries", Group: true},
		{ID: "scene.set.stands/near sampans", Name: "near sampans", Group: true},
	}
	for i := range 40 {
		name := fmt.Sprintf("near ferries %d", i)
		marks = append(marks, draft.MapMark{ID: "scene.set.things/" + name, Name: name})
	}
	view := makeAuthoredMap(marks, true, nil)
	work.working.plan, work.working.scene = view.plan, view.scene
	work.rebuild()
	drawLibraryWindow(t, work)
	open := libraryWidget(work.root, "Find placement…")
	if open == nil {
		t.Fatal("the large map had no placement search")
	}
	bounds := open.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)

	source := work.state.document.Text()
	search, ok := libraryWidget(work.root, "Search placements").(*textfield.Widget)
	if !ok || work.ui.Window().Context().FocusedWidget() != search {
		t.Fatal("opening search did not focus the name field")
	}
	submitPlacementSearch(t, work, "near")
	if work.play.selected != "" || work.play.disclosure == "" {
		t.Fatal("ambiguous search changed the selection")
	}
	search, _ = libraryWidget(work.root, "Search placements").(*textfield.Widget)
	if search == nil || work.ui.Window().Context().FocusedWidget() != search {
		t.Fatal("ambiguous search lost keyboard focus")
	}
	for _, letter := range " FERRIES" {
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, letter, event.ModNone))
	}
	work.rebuild()
	search, _ = libraryWidget(work.root, "Search placements").(*textfield.Widget)
	if search == nil || search.Text() != "near FERRIES" || work.ui.Window().Context().FocusedWidget() != search {
		t.Fatal("rebuild discarded unsubmitted text or keyboard focus")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if work.play.selected != marks[1].ID || work.play.disclosure != "" || !work.canShowSceneDetail() {
		t.Fatal("the named stand was not selected for detail")
	}
	if work.state.document.Text() != source || work.pending.document != nil {
		t.Fatal("placement search edited the document or queued a render")
	}
}

func submitPlacementSearch(t *testing.T, work *Workspace, query string) {
	t.Helper()
	search, ok := libraryWidget(work.root, "Search placements").(*textfield.Widget)
	if !ok {
		t.Fatal("placement search field missing")
	}
	search.SetText(query)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
}

func TestPlacementSearchDistinguishesExactThingAndGroupNames(t *testing.T) {
	labels := []string{"oak", "oak · group", "oak grove"}
	ids := []string{"scene.set.things/oak", "scene.set.stands/oak", "scene.set.stands/oak grove"}
	matches := placementMatches("Oak", labels, ids)
	if len(matches) != 2 || matches[0] != 0 || matches[1] != 1 {
		t.Fatalf("exact same-named thing and group = %v", matches)
	}
}
