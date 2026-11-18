package workspace

import (
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"testing"
	"time"
)

// TestNavigationWaitsForQuietAndConsumesOnlyLatest protects render backpressure.
func TestNavigationWaitsForQuietAndConsumesOnlyLatest(t *testing.T) {
	state := navigationState(t)
	var queue navigationQueue
	now := time.Unix(100, 0)
	queue.remember(state, "subaru", now)
	if queue.take(state, "subaru", now.Add(299*time.Millisecond), false) {
		t.Fatal("camera rendered before input settled")
	}
	state.document.Replace("scene: {}\n# second view\n")
	queue.remember(state, "subaru", now.Add(200*time.Millisecond))
	if queue.take(state, "subaru", now.Add(400*time.Millisecond), false) {
		t.Fatal("second gesture inherited the first deadline")
	}
	if queue.take(state, "subaru", now.Add(time.Second), true) {
		t.Fatal("camera started while another render was running")
	}
	if !queue.take(state, "subaru", now.Add(time.Second), false) {
		t.Fatal("latest settled view did not start after the renderer became idle")
	}
	if queue.take(state, "subaru", now.Add(2*time.Second), false) {
		t.Fatal("consumed request rendered twice")
	}
}

// TestNavigationDiscardsChangedContext prevents edits or switches restarting work.
func TestNavigationDiscardsChangedContext(t *testing.T) {
	for _, change := range []string{"text", "path", "document", "specimen", "stop"} {
		t.Run(change, func(t *testing.T) {
			state := navigationState(t)
			original := state
			originalText := state.document.Text()
			var queue navigationQueue
			now := time.Unix(100, 0)
			queue.remember(state, "subaru", now)
			specimen := "subaru"
			switch change {
			case "text":
				state.document.Replace("scene: {}\n# manual edit\n")
			case "path":
				state.path = "/second.yaml"
			case "document":
				state = navigationState(t)
			case "specimen":
				specimen = "keck"
			case "stop":
				queue.clear()
			}
			if queue.take(state, specimen, now.Add(time.Second), true) {
				t.Fatal("changed context started while busy")
			}
			original.document.Replace(originalText)
			if queue.take(original, "subaru", now.Add(2*time.Second), false) {
				t.Fatal("stale request returned after changing back to the old context")
			}
		})
	}
}

// navigationState supplies a real editable document without renderer substitutes.
func navigationState(t *testing.T) documentState {
	t.Helper()
	document, err := draft.Open("view.yaml", []byte("scene: {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	return documentState{document: document, path: "/view.yaml"}
}
