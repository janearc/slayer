package job

import (
	"encoding/json"
	"slices"
	"strings"
)

// A sentence of a schema's description ends with a full stop and a
// space, or with the description. Defined.
const (
	fullStop    = "."
	sentenceEnd = ". "
)

// A job's flags say that its picture departs from the exact tracer's,
// on purpose, as a set's fantasy block says a scene departs from the
// physics. They are not a quality: exact and preview say how much of a
// frame is drawn, a flag says how its light is found.
//
//	vibes: on
//	fastlight: on
//	shortcuts:
//	  lamp-store: off
//
// A shortcut is one named approximation. fastlight is the flag for the
// light shortcuts; vibes is the flag for every flag. Each is on, off or
// unsaid, and the first of these that says anything decides a shortcut:
// its own word under shortcuts, then fastlight if it is a light
// shortcut, then vibes; with nothing said it is off.
//
// Shortcut works
// that out for the job alone. A renderer that also reads a set asks
// Said for each word and puts the set's flags after the job's.
//
// Everything here is read from the schema, job.schema.json: the names,
// and the one line each says of itself, which is also the editor's
// hover text and the door's help. A new shortcut is a property of the
// schema's shortcuts and, if it is a light shortcut, a name in
// lightShortcuts; nothing else in this package changes.

// The names of the flags, and the two words a flag or a shortcut takes.
const (
	FlagVibes     = "vibes"
	FlagFastlight = "fastlight"
	flagOn        = "on"
	flagOff       = "off"
	// shortcutsKey is the schema's key for the shortcuts.
	shortcutsKey = "shortcuts"
)

// The light shortcuts, which fastlight turns on.
const (
	ShortcutLampStore = "lamp-store"
	ShortcutSkyLooks  = "sky-looks"
	ShortcutRestir    = "restir"
)

// lightShortcuts is the shortcuts fastlight speaks for.
var lightShortcuts = []string{ShortcutLampStore, ShortcutSkyLooks,
	ShortcutRestir}

// Switch is one flag or shortcut as the schema declares it: its name,
// the line it says of itself, and whether it is a shortcut.
type Switch struct {
	name, says string
	shortcut   bool
}

// Name is the switch's key in a job.
func (made Switch) Name() string { return made.name }

// Says is everything the schema says of it: what it does to the
// picture, and what it is refused with.
func (made Switch) Says() string { return made.says }

// Line is the first sentence the schema says of it, what it does to
// the picture, which is a help line's seventy characters or fewer.
func (made Switch) Line() string {
	first, _, _ := strings.Cut(made.says, sentenceEnd)
	return strings.TrimSuffix(first, fullStop) + fullStop
}

// Shortcut is whether it is said under shortcuts, and not a flag at
// the job's top.
func (made Switch) Shortcut() bool { return made.shortcut }

// described is the part of a schema this package reads: each
// property's description, and a shortcut's properties.
type described struct {
	Description string               `json:"description"`
	Properties  map[string]described `json:"properties"`
}

// Switches is every flag and every shortcut the schema declares: vibes,
// fastlight, then the shortcuts by name.
func Switches() []Switch {
	var top described
	// the schema is embedded and compiled before any job is read, so
	// it is JSON
	_ = json.Unmarshal(schemaRaw, &top)
	made := []Switch{
		{name: FlagVibes, says: top.Properties[FlagVibes].Description},
		{name: FlagFastlight,
			says: top.Properties[FlagFastlight].Description},
	}
	parts := top.Properties[shortcutsKey].Properties
	names := make([]string, nothing, len(parts))
	for name := range parts {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		made = append(made, Switch{name: name,
			says: parts[name].Description, shortcut: true})
	}
	return made
}

// Said is a flag's or a shortcut's own word in the job, whether it is
// on, and whether the job says it at all. A name the schema has not is
// unsaid.
func (job Job) Said(name string) (on, said bool) {
	return job.wire.said(name)
}

// said is Said for the job as written.
func (wire document) said(name string) (on, said bool) {
	word := wire.Shortcuts[name]
	if name == FlagVibes {
		word = wire.Vibes
	}
	if name == FlagFastlight {
		word = wire.Fastlight
	}
	return word == flagOn, word != ""
}

// Shortcut is whether a shortcut is on by the job alone: the first said
// of its own word, fastlight for a light shortcut, and vibes.
func (job Job) Shortcut(name string) bool {
	return job.wire.shortcut(name)
}

// shortcut is Shortcut for the job as written.
func (wire document) shortcut(name string) bool {
	for _, each := range above(name) {
		if on, said := wire.said(each); said {
			return on
		}
	}
	return false
}

// above is the names that decide a shortcut, first to last: itself,
// fastlight if it is a light shortcut, and vibes.
func above(name string) []string {
	if slices.Contains(lightShortcuts, name) {
		return []string{name, FlagFastlight, FlagVibes}
	}
	return []string{name, FlagVibes}
}

// Above is the names that decide a shortcut, first to last, for a
// renderer that reads them from a job and then a set.
func Above(name string) []string { return above(name) }

// canonicalShortcuts is the shortcuts the canonical form holds: each
// word the job's own fastlight does not already say. The flags are
// held as said. A job that says none of them has the key it had before
// there were flags.
func canonicalShortcuts(wire document) map[string]string {
	kept := map[string]string{}
	for name, word := range wire.Shortcuts {
		light := slices.Contains(lightShortcuts, name)
		if light && wire.Fastlight == word {
			continue
		}
		kept[name] = word
	}
	if len(kept) == nothing {
		return nil
	}
	return kept
}

// Flagged is a flag or a shortcut turned on or off, over the job's. A
// name the schema has not is refused when the job is checked.
func Flagged(name string, on bool) Override {
	word := flagOff
	if on {
		word = flagOn
	}
	return Override{key: name, empty: name == "",
		apply: func(wire *document) error {
			switch name {
			case FlagVibes:
				wire.Vibes = word
			case FlagFastlight:
				wire.Fastlight = word
			default:
				shortcuts := map[string]string{name: word}
				for other, said := range wire.Shortcuts {
					if other != name {
						shortcuts[other] = said
					}
				}
				wire.Shortcuts = shortcuts
			}
			return nil
		}}
}
