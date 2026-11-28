package session

import (
	"flag"
	"fmt"
	"io"

	"github.com/janearc/slayer/set"
)

// toldOf says, before a picture is drawn, each line a staged set has to
// say: a departure of its fantasy block from what is measured or derived,
// and each "made up:" line of a number the set made up and drew anyway. The
// command's own draw said them; a job's drew in silence.
func toldOf(out io.Writer, given asked) {
	for _, line := range given.departures {
		fmt.Fprintln(out, line)
	}
}

// the scene flags a set's fantasy block may also say; the command line
// stands over the block
var fantasyFlags = map[string]bool{"lake": true, "lakeglow": true,
	"moon": true, "moonglow": true, "giants": true, "close": true,
	"closeglow": true}

// noteFantasy keeps a flag that was given, where it is one of the scene
// flags a fantasy block says.
func (sizes *sizing) noteFantasy(each *flag.Flag) {
	name := each.Name
	if !fantasyFlags[name] {
		return
	}
	if sizes.fantasy == nil {
		sizes.fantasy = map[string]bool{}
	}
	sizes.fantasy[name] = true
}

// fantasied is what was asked with the set's fantasy block put in it, a
// key where its flag was not given, and a line for each key: what it
// says and that it departs from the measured or derived value. Where the
// flag was given, it stands and the line says so.
func fantasied(read set.Set, given asked) (asked, []string) {
	var lines []string
	for _, each := range read.Fantasy() {
		name, value := each.Key(), each.Value()
		overridden := given.said.fantasy[name]
		if overridden {
			value = flagValue(given, name)
		} else {
			given = given.departed(each)
		}
		written := each.Value()
		line := departureLine(name, value, written, overridden)
		lines = append(lines, line)
	}
	return given, lines
}

// departed is what was asked with a departure of the set's taken as the
// flag's value.
func (given asked) departed(each set.Departure) asked {
	switch each.Key() {
	case "lake":
		given.lake = each.Number()
	case "lakeglow":
		given.lakeGlow = each.On()
	case "moon":
		given.moonScale = each.Number()
	case "moonglow":
		given.moonGlow = each.Number()
	case "giants":
		given.giants = each.Number()
	case "close":
		given.closeScale = each.Number()
	case "closeglow":
		given.closeGlow = each.On()
	}
	return given
}

// flagValue is what a scene flag was given, as text.
func flagValue(given asked, name string) string {
	switch name {
	case "lake":
		return fmt.Sprint(given.lake)
	case "lakeglow":
		return fmt.Sprint(given.lakeGlow)
	case "moon":
		return fmt.Sprint(given.moonScale)
	case "moonglow":
		return fmt.Sprint(given.moonGlow)
	case "giants":
		return fmt.Sprint(given.giants)
	case "close":
		return fmt.Sprint(given.closeScale)
	}
	return fmt.Sprint(given.closeGlow)
}

// departureLine is the line that says a key departs from the measured or
// derived value: the value that is used, and, where the flag stood over
// the set, the set's.
func departureLine(name, used, written string, overridden bool) string {
	said := fmt.Sprintf("fantasy: %s = %s", name, used)
	if overridden {
		said += fmt.Sprintf(" (the -%s flag overrides the set's %s)", name,
			written)
	}
	return said + " departs from the measured or derived value"
}
