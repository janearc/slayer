// Package pages keeps what the operator sees of a render while it
// runs and when it ends: the render folder's own page, the film so
// far, the gallery pages under the renders root, and barkeep's
// progress line. One coordinator does all of it, for every job the
// worker runs.
package pages

import (
	"path/filepath"
	"sync"
	"time"
)

// The two shelves under the renders root that hold render folders.
const (
	framesShelf     = "frames"     // defined: the shared stills
	animationsShelf = "animations" // defined: the shared films
)

// The names pages keeps under the renders root and in a render folder.
const (
	pageName      = "index.html"               // defined: a folder's page
	handPageName  = "index-before-azazel.html" // defined: a hand page set aside
	latestName    = "latest.html"              // defined: the one page
	progressName  = "render-progress.txt"      // defined: barkeep's line
	partialName   = "partial.mp4"              // defined: the film so far
	partialRecord = "partial.json"             // defined: what it holds
	notesName     = "latest-notes.txt"         // defined: a team's notes
)

// Settings say where the renders are, what makes the films and how the
// coordinator keeps time.
type Settings struct {
	// Renders is the absolute renders root, which holds frames and
	// animations.
	Renders string
	// Encoder makes the partial films.
	Encoder Encoder
	// Now is the clock.
	Now func() time.Time
	// Location is the zone the page times are shown in.
	Location *time.Location
	// Every is how often the coordinator publishes while a job is live.
	Every time.Duration
}

// Folder is one render's folder: its shelf, frames or animations, and
// its name there.
type Folder struct {
	Shelf string
	Name  string
}

// path is the folder on disk under the renders root.
func (folder Folder) path(root string) string {
	return filepath.Join(root, folder.Shelf, folder.Name)
}

// publisher is the part of the work a coordinator and the publish verb
// share: what to publish where, and with what.
type publisher struct {
	root     string
	encoder  Encoder
	location *time.Location
	now      func() time.Time
	// digests keeps each music file's sha256 by its path, size and time,
	// so a long piece of music is read whole once, not at every tick.
	digests *sync.Map
}

// publisherOf is the publisher the settings describe.
func publisherOf(settings Settings) publisher {
	return publisher{root: filepath.Clean(settings.Renders),
		encoder: settings.Encoder, location: settings.Location,
		now: settings.Now, digests: &sync.Map{}}
}
