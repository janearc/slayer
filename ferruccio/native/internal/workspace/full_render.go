package workspace

import "fmt"

// StartFullRender freezes the authored still dimensions and rays in exact quality.
// It uses the same cancellation, observation and verified-result acceptance as preview.
func (work *Workspace) StartFullRender() error { return work.startRender(true) }

// fullRenderBusyText begins the words shown when an action must wait for a
// full render. The action is named after it.
const fullRenderBusyText = "A full render is running. Stop it, or wait for it to finish, before "

// refuseWhileFull says in the status line that an action cannot proceed while
// a full render runs, and reports whether it was refused. A full render ends
// only when it finishes or the user asks to stop it.
func (work *Workspace) refuseWhileFull(action string) bool {
	if !work.fullRunning {
		return false
	}
	work.status = fullRenderBusyText + action
	work.invalidate()
	return true
}

// fullRenderRefusal is refuseWhileFull for callers that return an error.
func (work *Workspace) fullRenderRefusal(action string) error {
	if !work.refuseWhileFull(action) {
		return nil
	}
	return fmt.Errorf("%s%s", fullRenderBusyText, action)
}
