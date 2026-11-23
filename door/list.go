package door

// list says what there is to draw and to draw with. Each heading is
// read from the registry of the package that owns it, never from a list
// kept here, so a scene, smoke, preset, stock, tune or kind added there
// shows here with nothing else changed.

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/ninjafilm"
	namedscene "github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/scene/smoke"
)

// listHelp is list's page.
const listHelp = `# list

slayer list says the headings there are, one a line. slayer list
HEADING lists one of them: each entry's name, one line of what it
is, and below it what it takes or holds.

Every entry is read from the registry of the package that owns it, so
the list cannot fall behind the code. --json writes the same as JSON
for a program: an object of headings, or one heading's entries.
`

// jsonFlag is the switch that asks for JSON in place of words.
const jsonFlag = "json"

// listed is one entry a heading lists: its name, a line saying what it
// is, and the lines below it, each one thing it takes or holds.
type listed struct {
	Name    string   `json:"name"`
	About   string   `json:"about,omitempty"`
	Details []string `json:"details,omitempty"`
}

// topic is one heading: a kind of thing there is, and how its entries
// are read from the registry that owns it.
type topic struct {
	name, about string
	read        func() ([]listed, error)
}

// topics is every topic, in the order list says them.
var topics = []topic{
	{"scenes", "the named scenes a job draws by name, and what each takes",
		sceneEntries},
	{"smokes", "the numbered smoke scenes, by number, with their group",
		smokeEntries},
	{"presets", "the film presets a job names for its size and rate",
		presetEntries},
	{"stocks", "the film stocks a job's stock develops through",
		stockEntries},
	{"tunes", "the built-in scores a job's sound plays", tuneEntries},
	{"voicings", "the built-in voicings a score is played with",
		voicingEntries},
	{"library", "the library tunes, each a score with its cut sheet",
		libraryEntries},
	{"kinds", "the kinds the manufactory makes, as the catalogue shows",
		kindEntries},
}

// topicNamed is the topic of a name, or a refusal that names them.
func topicNamed(name string) (topic, error) {
	at := slices.IndexFunc(topics, func(each topic) bool {
		return each.name == name
	})
	if at < nothing {
		headingNames := topicNames()
		return topic{}, fmt.Errorf("there is no heading %q; the "+
			"headings are %s", name, strings.Join(headingNames, ", "))
	}
	return topics[at], nil
}

// topicNames is every topic's name, in order.
func topicNames() []string {
	names := make([]string, nothing, len(topics))
	for _, each := range topics {
		names = append(names, each.name)
	}
	return names
}

// list is the verb: the topics, or one topic's entries.
func list(call pzm.Call) error {
	asJSON := call.On(jsonFlag)
	switch len(call.Args()) {
	case nothing:
		return sayHeadings(call.Out(), asJSON)
	case one:
		chosen, err := topicNamed(call.Args()[nothing])
		if err != nil {
			return err
		}
		return sayEntries(call.Out(), chosen, asJSON)
	}
	wordCount := len(call.Args())
	headingNames := topicNames()
	return fmt.Errorf("list takes one heading or none, not %d words: "+
		"slayer list [%s]", wordCount,
		strings.Join(headingNames, "|"))
}

// sayHeadings writes each topic and what it is, or as JSON every
// topic's entries.
func sayHeadings(out io.Writer, asJSON bool) error {
	if !asJSON {
		var page strings.Builder
		for _, each := range topics {
			fmt.Fprintf(&page, "%-9s %s\n", each.name, each.about)
		}
		page.WriteString("\nslayer list HEADING lists one of them.\n")
		_, err := io.WriteString(out, page.String())
		return err
	}
	all := map[string][]listed{}
	for _, each := range topics {
		read, err := each.read()
		if err != nil {
			return fmt.Errorf("%s: %w", each.name, err)
		}
		all[each.name] = read
	}
	return writeJSON(out, all)
}

// sayEntries writes one topic's entries, as words or as JSON.
func sayEntries(out io.Writer, chosen topic, asJSON bool) error {
	read, err := chosen.read()
	if err != nil {
		return fmt.Errorf("%s: %w", chosen.name, err)
	}
	if asJSON {
		return writeJSON(out, read)
	}
	var page strings.Builder
	fmt.Fprintf(&page, "%s: %s\n", chosen.name, chosen.about)
	for _, each := range read {
		page.WriteString("\n" + each.Name)
		if each.About != "" {
			page.WriteString("  " + each.About)
		}
		page.WriteString("\n")
		for _, detail := range each.Details {
			page.WriteString("    " + detail + "\n")
		}
	}
	_, err = io.WriteString(out, page.String())
	return err
}

// writeJSON writes a value as indented JSON and a line break.
func writeJSON(out io.Writer, value any) error {
	text, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.Write(append(text, '\n'))
	return err
}

// sceneEntries is every named scene, with what it takes.
func sceneEntries() ([]listed, error) {
	var read []listed
	for _, scene := range namedscene.All() {
		made := listed{Name: scene.Name(), About: scene.About()}
		for _, taken := range scene.Parameters() {
			made.Details = append(made.Details, parameterLine(taken))
		}
		read = append(read, made)
	}
	return read, nil
}

// parameterLine is one parameter in a line: its name, kind, default,
// bounds for a number, choices, and its help.
func parameterLine(taken namedscene.Parameter) string {
	line := fmt.Sprintf("%s (%s, %v", taken.Name(), taken.Kind(),
		taken.Default())
	if bounded[taken.Kind()] {
		line += ", " + taken.Bounds().String()
	}
	line += ")"
	if choices := taken.Choices(); len(choices) > nothing {
		line += " of " + strings.Join(choices, ", ")
	}
	return line + ": " + taken.Help()
}

// bounded are the kinds of parameter that have bounds.
var bounded = map[namedscene.Kind]bool{namedscene.Whole: true,
	namedscene.Number: true}

// smokeEntries is every smoke, by number.
func smokeEntries() ([]listed, error) {
	var read []listed
	for _, made := range smoke.All() {
		read = append(read, listed{Name: strconv.Itoa(made.Number),
			About: made.Name, Details: []string{"group " + made.Group()}})
	}
	return read, nil
}

// presetEntries is every film preset, with its size and rate.
func presetEntries() ([]listed, error) {
	var read []listed
	for _, preset := range job.Presets() {
		width, height := preset.Size()
		about := fmt.Sprintf("%d by %d", width, height)
		if rate, has := preset.Rate(); has {
			about += fmt.Sprintf(" at %d frames a second", rate)
		}
		read = append(read, listed{Name: preset.Name(), About: about})
	}
	return read, nil
}

// stockEntries is every stock on ninjafilm's shelf.
func stockEntries() ([]listed, error) {
	names, err := ninjafilm.Stocks()
	return namesOnly(names), err
}

// tuneEntries is every built-in score.
func tuneEntries() ([]listed, error) {
	scores, _ := cheeptune.Builtin()
	return namesOnly(scores), nil
}

// voicingEntries is every built-in voicing.
func voicingEntries() ([]listed, error) {
	_, voicings := cheeptune.Builtin()
	return namesOnly(voicings), nil
}

// libraryEntries is every library tune, named as a job names it.
func libraryEntries() ([]listed, error) {
	var called []string
	for _, name := range cheeptune.Library() {
		called = append(called, "@"+name)
	}
	return namesOnly(called), nil
}

// kindEntries is every kind the manufactory makes, with the details it
// may be made at.
func kindEntries() ([]listed, error) {
	levels := "made at " + strings.Join(manufacture.Complexities(), ", ")
	var read []listed
	for _, name := range manufacture.KindNames() {
		read = append(read, listed{Name: name, Details: []string{levels}})
	}
	return read, nil
}

// namesOnly is an listed for each name, with nothing more to say.
func namesOnly(names []string) []listed {
	read := make([]listed, nothing, len(names))
	for _, name := range names {
		read = append(read, listed{Name: name})
	}
	return read
}

// declareList adds list to the tool: it reads, and writes nothing.
func declareList(made *pzm.Tool) {
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	made.Verb("list", verb.NoShort, "say what there is to draw and to "+
		"draw with", listHelp, "Example_list", reads, list,
		pzm.Switch(jsonFlag, verb.NoShort, "write JSON for a program"))
}
