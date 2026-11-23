package door

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/ninjafilm"
	namedscene "github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/scene/smoke"
	"go.uber.org/mock/gomock"
)

// registered is every name each heading's registry holds, read from the
// registry itself, so a test against it holds list to the code.
func registered(t *testing.T) map[string][]string {
	t.Helper()
	stocks, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	scores, voicings := cheeptune.Builtin()
	all := map[string][]string{"stocks": stocks, "tunes": scores,
		"voicings": voicings, "kinds": manufacture.KindNames()}
	for _, scene := range namedscene.All() {
		all["scenes"] = append(all["scenes"], scene.Name())
	}
	for _, made := range smoke.All() {
		all["smokes"] = append(all["smokes"], strconv.Itoa(made.Number))
	}
	for _, preset := range job.Presets() {
		all["presets"] = append(all["presets"], preset.Name())
	}
	for _, name := range cheeptune.Library() {
		all["library"] = append(all["library"], "@"+name)
	}
	return all
}

// listing runs list with the args given.
func listing(t *testing.T, args ...string) ran {
	t.Helper()
	env := envIn(t, context.Background(), t.TempDir())
	return running(Tool(env), append([]string{"list"}, args...)...)
}

// every registry's every entry is in its heading's words, each at the
// start of a line, and in its JSON, in the registry's order.
func TestListSaysEveryEntryOfEveryRegistry(t *testing.T) {
	for name, names := range registered(t) {
		if len(names) == nothing {
			t.Errorf("%s: the registry holds nothing", name)
		}
		said := listing(t, name)
		if said.code != nothing {
			t.Fatalf("list %s gave %d: %s", name, said.code,
				said.wrongSaid)
		}
		for _, each := range names {
			if !strings.Contains(said.said, "\n"+each+"\n") &&
				!strings.Contains(said.said, "\n"+each+"  ") {
				t.Errorf("list %s does not say %q", name, each)
			}
		}
		var read []listed
		asJSON := listing(t, name, "--json")
		if err := json.Unmarshal([]byte(asJSON.said), &read); err != nil {
			t.Fatalf("list %s --json: %v", name, err)
		}
		if len(read) != len(names) {
			t.Fatalf("list %s --json holds %d, the registry %d", name,
				len(read), len(names))
		}
		for i, each := range read {
			if each.Name != names[i] {
				t.Errorf("list %s --json[%d] is %q, not %q", name, i,
					each.Name, names[i])
			}
		}
	}
}

// a named scene's parameters are said below it, each with its kind,
// default and help.
func TestListSaysWhatASceneTakes(t *testing.T) {
	said := listing(t, "scenes").said
	for _, scene := range namedscene.All() {
		for _, taken := range scene.Parameters() {
			if !strings.Contains(said, "    "+parameterLine(taken)+"\n") {
				t.Errorf("%s's %s is not said", scene.Name(),
					taken.Name())
			}
		}
	}
	for _, scene := range namedscene.All() {
		for _, taken := range scene.Parameters() {
			if len(taken.Choices()) > nothing {
				line := parameterLine(taken)
				if !strings.Contains(line, " of "+taken.Choices()[nothing]) {
					t.Errorf("%q does not say its choices", line)
				}
				return
			}
		}
	}
	t.Error("no named scene has a parameter of choices to see said")
}

// a preset with a rate says it, and one without says none.
func TestListSaysAPresetsRate(t *testing.T) {
	read, err := presetEntries()
	if err != nil {
		t.Fatal(err)
	}
	for i, preset := range job.Presets() {
		_, has := preset.Rate()
		says := strings.Contains(read[i].About, "frames a second")
		// anti-test of TestListSaysAPresetsRate
		if has != says {
			t.Errorf("%s has a rate %v, says one %v", preset.Name(), has,
				says)
		}
	}
}

// with no heading list says every heading, and with --json every
// heading's entries, which parse.
func TestListSaysItsHeadings(t *testing.T) {
	said := listing(t)
	for _, name := range topicNames() {
		if !strings.Contains(said.said, name+" ") {
			t.Errorf("list does not say the heading %s", name)
		}
	}
	all := map[string][]listed{}
	asJSON := listing(t, "--json")
	if err := json.Unmarshal([]byte(asJSON.said), &all); err != nil {
		t.Fatalf("list --json: %v: %s", err, asJSON.said)
	}
	if len(all) != len(topics) {
		t.Errorf("list --json holds %d headings, not %d", len(all),
			len(topics))
	}
}

// a heading there is not is refused with the headings
// there are, and two words are refused; each exits one.
// anti-test of TestListSaysEveryEntryOfEveryRegistry
// anti-test of TestListSaysItsHeadings
func TestListRefusesWhatItHasNot(t *testing.T) {
	for _, refusal := range []struct {
		args []string
		said string
	}{
		{[]string{"lenses"}, `there is no heading "lenses"; the headings ` +
			"are scenes, smokes"},
		{[]string{"scenes", "smokes"}, "list takes one heading or none, " +
			"not 2 words"},
	} {
		got := listing(t, refusal.args...)
		if got.code != one || !strings.Contains(got.wrongSaid,
			refusal.said) {
			t.Errorf("list %v gave %d: %q", refusal.args, got.code,
				got.wrongSaid)
		}
	}
	if got := listing(t, "--dry-run"); got.code == nothing {
		t.Error("list, which writes nothing, took --dry-run")
	}
}

// a registry that cannot be read is said by its heading's name, in
// words and in JSON; a writer that will not take the words fails.
// anti-test of TestListSaysEveryEntryOfEveryRegistry
func TestListSaysARegistryThatFails(t *testing.T) {
	kept := topics
	t.Cleanup(func() { topics = kept })
	broken := errors.New("the shelf is gone")
	topics = []topic{{"broken", "nothing",
		func() ([]listed, error) { return nil, broken }}}
	if got := listing(t, "broken"); !strings.Contains(got.wrongSaid,
		"broken: the shelf is gone") {
		t.Errorf("a broken registry said %q", got.wrongSaid)
	}
	if got := listing(t, "--json"); !strings.Contains(got.wrongSaid,
		"broken: the shelf is gone") {
		t.Errorf("a broken registry in JSON said %q", got.wrongSaid)
	}
	topics = kept
	full := mocks.NewMockWriter(gomock.NewController(t))
	full.EXPECT().Write(gomock.Any()).Return(nothing, errors.New("full")).
		AnyTimes()
	if err := sayHeadings(full, false); err == nil {
		t.Error("the headings were said to a writer that took nothing")
	}
	if err := sayEntries(full, topics[nothing], true); err == nil {
		t.Error("JSON was said to a writer that took nothing")
	}
	if err := writeJSON(full, func() {}); err == nil {
		t.Error("a function was written as JSON")
	}
}

// list's example: the headings there are.
func Example_list() {
	here, _ := os.Getwd()
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(string) (string, bool) { return "", false },
		Now:         noon, Directory: here})
	Tool(env).Run([]string{"list"}, os.Stdout, os.Stdout)
	fmt.Println("done")
	// Output:
	// scenes    the named scenes a job draws by name, and what each takes
	// smokes    the numbered smoke scenes, by number, with their group
	// presets   the film presets a job names for its size and rate
	// stocks    the film stocks a job's stock develops through
	// tunes     the built-in scores a job's sound plays
	// voicings  the built-in voicings a score is played with
	// library   the library tunes, each a score with its cut sheet
	// kinds     the kinds the manufactory makes, as the catalogue shows
	//
	// slayer list HEADING lists one of them.
	// done
}

// a number's bounds are said in its line, and a switch's line says
// none: the anti-test.
// anti-test of TestListSaysWhatASceneTakes
func TestListSaysANumbersBounds(t *testing.T) {
	saw := map[namedscene.Kind]bool{}
	for _, scene := range namedscene.All() {
		for _, taken := range scene.Parameters() {
			line := parameterLine(taken)
			says := strings.Contains(line, taken.Bounds().String())
			if bounded[taken.Kind()] != says {
				t.Errorf("%q: a %s, bounds said %v", line, taken.Kind(),
					says)
			}
			saw[taken.Kind()] = true
		}
	}
	if !saw[namedscene.Whole] || !saw[namedscene.Switch] {
		t.Errorf("the scenes have no whole number or no switch: %v", saw)
	}
}
