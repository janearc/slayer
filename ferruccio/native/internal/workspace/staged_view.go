package workspace

import (
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/session/placement"
)

// stagedView holds a display and, for in-process staging, owns its session.
type stagedView struct {
	scene  *placement.DisplayScene
	notice string
	frame  *stagewire.FilmFrame
	plan   *authoredPlan
	owner  interface{ Close() error }
}

// close releases the owner. A view with an owner must not be used afterwards.
func (view *stagedView) close() error {
	if view == nil || view.owner == nil {
		return nil
	}
	err := view.owner.Close()
	view.owner = nil
	return err
}
