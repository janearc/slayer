package live

import (
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
)

// mode is what the pane is showing: the preview ladder, the help, or a
// frame at the job's full size.
type mode int

// The modes.
const (
	previewing mode = iota
	helping
	full
)

// view is the state the keys change.
type view struct {
	index  int // the frame shown, from nought
	camera string
	// moment is a still's time as the keys moved it, and said whether
	// they did
	moment time.Duration
	said   bool
	thirds int // exposure, in thirds of a stop from the job's
	mode   mode
	// playing, and the frame and time play started from
	playing   bool
	playFrom  int
	playStart time.Time
}

// action is what a key does to the watcher, and what that costs.
type action func(watching *Watcher) Change

// byName is the action of each named key. KeyHost is the hosting
// program's, and does nothing here: it is in the table so that it is
// seen to be left alone.
var byName = map[terminal.KeyName]action{
	terminal.KeyLeft:  back,
	terminal.KeyRight: on,
	terminal.KeySpace: (*Watcher).play,
	terminal.KeyCtrlC: (*Watcher).quitting,
	terminal.KeyHost:  func(*Watcher) Change { return Unchanged },
}

// back and on step a film a frame back or on.
func back(watching *Watcher) Change { return watching.step(-one) }

// on steps the film one frame on.
func on(watching *Watcher) Change { return watching.step(one) }

// byRune is the action of each character key.
var byRune = map[rune]action{
	'[': func(watching *Watcher) Change { return watching.second(-one) },
	']': func(watching *Watcher) Change { return watching.second(one) },
	'c': (*Watcher).nextCamera,
	'e': func(watching *Watcher) Change { return watching.expose(-one) },
	'E': func(watching *Watcher) Change { return watching.expose(one) },
	'f': (*Watcher).fullSize,
	's': (*Watcher).save,
	'?': (*Watcher).help,
	'q': (*Watcher).quitting,
}

// actionOf is a key's action, and whether it has one.
func actionOf(key terminal.Key) (action, bool) {
	if key.Name() == terminal.KeyChar {
		act, known := byRune[key.Rune()]
		return act, known
	}
	act, known := byName[key.Name()]
	return act, known
}

// key is what a key does: in the help or a full frame, any key but the
// host's and quitting returns to the preview; otherwise its action.
func (watching *Watcher) key(key terminal.Key) Change {
	act, known := actionOf(key)
	if key.Name() == terminal.KeyHost {
		return Unchanged
	}
	if watching.view.mode != previewing && !watching.quits(key) {
		return watching.back()
	}
	if !known {
		return Unchanged
	}
	return act(watching)
}

// quits is whether a key quits.
func (watching *Watcher) quits(key terminal.Key) bool {
	return key.Name() == terminal.KeyCtrlC ||
		key.Name() == terminal.KeyChar && key.Rune() == 'q'
}

// back returns to the preview: from the help nothing is drawn again,
// from a full frame the ladder starts again.
func (watching *Watcher) back() Change {
	was := watching.view.mode
	watching.view.mode = previewing
	if was == full {
		return Redraw
	}
	return Unchanged
}

// quitting asks the loop to stop.
func (watching *Watcher) quitting() Change {
	watching.quit = true
	return Unchanged
}

// help shows the keys in the foot until any key.
func (watching *Watcher) help() Change {
	watching.view.mode = helping
	return Unchanged
}

// fullSize draws the frame at the job's own size.
func (watching *Watcher) fullSize() Change {
	watching.view.mode = full
	return Redraw
}
