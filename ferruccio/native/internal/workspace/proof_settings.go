package workspace

import (
	"fmt"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// ProofSettingsClick presses the control of the open sheet with the given id,
// through the same event path a pointer takes: a move onto it, a press and a
// release at its centre. A proof uses it to show what a press does without a
// person.
func (work *Workspace) ProofSettingsClick(id string) error {
	if !work.settingsOpen {
		return fmt.Errorf("the settings sheet is not open")
	}
	centre, found := centreOf(work.root, id, geometry.Point{})
	if !found {
		return fmt.Errorf("the sheet has no control %q", id)
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0, centre, centre, event.ModNone))
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, centre, centre, event.ModNone))
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, centre, centre, event.ModNone))
	return nil
}

// centreOf is where the widget with the id has its centre in the window,
// found by walking from w, whose parent starts at origin. A widget's bounds
// are in its parent's space, so each step adds the parent's own corner.
func centreOf(w widget.Widget, id string, origin geometry.Point) (geometry.Point, bool) {
	if w == nil {
		return geometry.Point{}, false
	}
	bounds := geometry.Rect{}
	if placed, ok := w.(interface{ Bounds() geometry.Rect }); ok {
		bounds = placed.Bounds()
	}
	corner := geometry.Pt(origin.X+bounds.Min.X, origin.Y+bounds.Min.Y)
	if named, ok := w.(interface{ ID() string }); ok && named.ID() == id {
		return geometry.Pt(corner.X+bounds.Width()/2, corner.Y+bounds.Height()/2), true
	}
	for _, child := range w.Children() {
		if at, found := centreOf(child, id, corner); found {
			return at, true
		}
	}
	return geometry.Point{}, false
}
