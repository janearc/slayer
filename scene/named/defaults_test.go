package named_test

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/named"
)

// commanded is what a command's flags defaulted to before its scene was
// moved here: its picture's size and rays a side, its lens's width, its
// time and its tone. Each is copied from the command, as it stood at the
// commit this package was made from.
type commanded struct {
	width, height, rays int
	wide                float64
	at                  time.Duration
	tone                colour.Tone
}

// commands are each scene's command's defaults.
var commands = map[string]commanded{
	"swarm": {width: 1600, height: 900, rays: 2,
		wide: 45 * maths.Degree, tone: colour.Tone{Knee: 0.7, Top: 8}},
	"orbital": {width: 1600, height: 900, rays: 2,
		wide: 90 * maths.Degree, at: 500 * time.Millisecond,
		tone: colour.Tone{Exposure: 1, Knee: 0.7, Top: 8}},
	"ring": {width: 960, height: 540, rays: 2, wide: 55 * maths.Degree,
		at:   1200 * time.Millisecond,
		tone: colour.Tone{Knee: 0.7, Top: 8}},
	"zappy": {width: 1080, height: 1920, rays: 2,
		wide: 40 * maths.Degree, at: 500 * time.Millisecond,
		tone: colour.Tone{Exposure: 1, Knee: 0.7, Top: 8}},
	"moonrise": {width: 1430, height: 1000, rays: 2,
		wide: 2 * maths.Degree},
	"magnetar": {width: 1080, height: 1920, rays: 8,
		wide: 40 * maths.Degree,
		tone: colour.Tone{Exposure: 1, Knee: 0.7, Top: 8}},
}

// Each scene's defaults are its command's.
func TestDefaultsAreTheCommands(t *testing.T) {
	for _, scene := range named.Traced() {
		want, found := commands[scene.Name()]
		if !found {
			t.Errorf("%s has no command's defaults to hold it to",
				scene.Name())
			continue
		}
		if got := heldOf(scene.Defaults()); got != want {
			t.Errorf("%s's defaults are %+v, want %+v", scene.Name(),
				got, want)
		}
	}
}

// anti-tests: the defaults of one scene are not another's. No two scenes
// share a lens, a time and a picture size at once.
func TestDefaultsAreEachScenesOwn(t *testing.T) {
	seen := map[commanded]string{}
	for _, scene := range named.Traced() {
		held := heldOf(scene.Defaults())
		if other, found := seen[held]; found {
			t.Errorf("%s has %s's defaults", scene.Name(), other)
		}
		seen[held] = scene.Name()
	}
}

// heldOf is a scene's defaults as a command's.
func heldOf(made named.Defaults) commanded {
	return commanded{width: made.Width(), height: made.Height(),
		rays: made.Rays(), wide: made.Wide(),
		at: made.At(), tone: made.Tone()}
}
