package workspace

import (
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"time"
)

// navigationDelay groups smooth trackpad input into one authored camera view.
const navigationDelay = 300 * time.Millisecond

// navigationQueue holds the latest camera request on the UI thread.
type navigationQueue struct {
	document             *draft.Document
	path, text, specimen string
	due                  time.Time
}

// remember replaces earlier requests with the latest document view.
func (queue *navigationQueue) remember(state documentState, specimen string, now time.Time) {
	*queue = navigationQueue{document: state.document, path: state.path,
		text: state.document.Text(), specimen: specimen, due: now.Add(navigationDelay)}
}

// take consumes a settled request only when its document is current and idle.
func (queue *navigationQueue) take(state documentState, specimen string, now time.Time, busy bool) bool {
	if queue.document == nil {
		return false
	}
	if state.document != queue.document || state.path != queue.path ||
		specimen != queue.specimen || state.document.Text() != queue.text {
		queue.clear()
		return false
	}
	if busy || now.Before(queue.due) {
		return false
	}
	queue.clear()
	return true
}

// clear forgets a view when its document or presentation changes.
func (queue *navigationQueue) clear() { *queue = navigationQueue{} }
