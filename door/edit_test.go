package door

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/job"
	"go.uber.org/mock/gomock"
)

// The tests' tmux window and pane, picked by the author of this slice,
// 2026-11-12: a laptop's window, and a pane id as tmux writes one.
const (
	testWindowCols, testWindowRows = 120, 40
	testPane                       = "%3"
)

// laidPanes is three commands as a test lays them out.
func laidPanes() panes {
	return panes{watch: []string{"/bin/slayer", "watch", "/jobs/a.yaml"},
		editor:  []string{"/bin/nvim", "/jobs/a.yaml"},
		console: []string{"/bin/carrie", "chat", "--workspace", "/jobs"}}
}

// the console goes across the whole foot, ten rows; then the editor a
// third of the width on the right, from the pane edit runs in, last so
// that it has the focus.
func TestSplitsLayTheWindowOut(t *testing.T) {
	got := splits(window{cols: testWindowCols, rows: testWindowRows},
		testPane, "/jobs", laidPanes())
	want := [][]string{
		{"split-window", "-v", "-f", "-l", "10", "-t", testPane, "-c",
			"/jobs", "/bin/carrie", "chat", "--workspace", "/jobs"},
		{"split-window", "-h", "-l", "40", "-t", testPane, "-c", "/jobs",
			"/bin/nvim", "/jobs/a.yaml"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splits are\n%q\nwant\n%q", got, want)
	}
}

// a short window gives the console no more than half its
// rows, and a narrow one still gives the editor a column; neither split
// runs a shell.
// anti-test of TestSplitsLayTheWindowOut
func TestSplitsHoldASmallWindow(t *testing.T) {
	got := splits(window{cols: 2, rows: 8}, testPane, "/jobs", laidPanes())
	if got[0][4] != "4" || got[1][3] != "1" {
		t.Errorf("a small window splits as %q", got)
	}
	for _, args := range got {
		for _, word := range args {
			if word == "sh" || word == "-c" && args[len(args)-1] == "sh" {
				t.Errorf("a split runs a shell: %q", args)
			}
		}
	}
}

// a window's size is read as tmux prints it; anything else is refused.
func TestWindowOfReadsTmux(t *testing.T) {
	if got, err := windowOf("120 40\n"); err != nil ||
		got != (window{cols: 120, rows: 40}) {
		t.Errorf("120 40 read as %+v, %v", got, err)
	}
	// anti-test of TestWindowOfReadsTmux
	for _, said := range []string{"", "120", "120 forty", "0 40",
		"1 2 3"} {
		if _, err := windowOf(said); err == nil {
			t.Errorf("%q was read as a window", said)
		}
	}
}

// editingIn is edit with a mock machine, in a directory, its cache
// under it.
func editingIn(t *testing.T, directory string, machine programs,
	variables ...string) editing {
	t.Helper()
	env := envIn(t, context.Background(), directory,
		append([]string{"SLAYER_CACHE",
			filepath.Join(directory, "cache")}, variables...)...)
	return editing{env: env, machine: machine,
		watcher: watching{env: env}}
}

// finding is a mock machine that finds the programs named, at
// /bin/NAME, and no others, and is slayer at /bin/slayer.
func finding(control *gomock.Controller, names ...string) *Mockprograms {
	machine := NewMockprograms(control)
	found := map[string]bool{}
	for _, name := range names {
		found[name] = true
	}
	machine.EXPECT().Find(gomock.Any()).DoAndReturn(func(name string) (
		string, error) {
		if found[name] {
			return "/bin/" + name, nil
		}
		return "", errors.New(name + " not found")
	}).AnyTimes()
	machine.EXPECT().Self().Return("/bin/slayer", nil).AnyTimes()
	return machine
}

// outside tmux edit says so, prints the three commands and runs watch
// here, which a test's output, no terminal, refuses; no tmux is run.
func TestEditOutsideTmuxPrintsTheCommands(t *testing.T) {
	control := gomock.NewController(t)
	directory := t.TempDir()
	editor := editingIn(t, directory, finding(control, tmuxName,
		nvimName, serverName, carrieName))
	got := running(toolWith(editor), "edit", "a.yaml")
	for _, words := range []string{"not inside tmux", "/bin/slayer watch",
		"/bin/nvim", "luafile", "/bin/carrie chat --workspace " +
			directory, "slayer draw"} {
		if !strings.Contains(got.said+got.wrongSaid, words) {
			t.Errorf("edit outside tmux did not say %q:\n%s%s", words,
				got.said, got.wrongSaid)
		}
	}
}

// inside tmux edit asks the window's size and splits it, each command
// run with no shell; then it runs watch here.
// anti-test of TestEditOutsideTmuxPrintsTheCommands
func TestEditInsideTmuxSplitsTheWindow(t *testing.T) {
	control := gomock.NewController(t)
	directory := t.TempDir()
	machine := finding(control, tmuxName, nvimName, serverName,
		carrieName)
	var ran [][]string
	machine.EXPECT().Run(gomock.Any(), "/bin/tmux", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, args ...string) (
			string, error) {
			ran = append(ran, args)
			return fmt.Sprintf("%d %d\n", testWindowCols,
				testWindowRows), nil
		}).Times(3)
	editor := editingIn(t, directory, machine, tmuxVar, "/tmp/tmux",
		paneVar, testPane)
	got := running(toolWith(editor), "edit", "a.yaml")
	if len(ran) != 3 || ran[0][0] != "display-message" ||
		ran[1][0] != "split-window" || ran[2][0] != "split-window" {
		t.Fatalf("tmux ran %q", ran)
	}
	if strings.Contains(got.said, "not inside tmux") {
		t.Errorf("edit inside tmux said it was not:\n%s", got.said)
	}
}

// a program not on PATH is said by name with what edit does
// without it, and nothing is installed or run in its place but a
// shell's editor and a shell.
// anti-test of TestEditInsideTmuxSplitsTheWindow
func TestEditSaysWhatIsMissing(t *testing.T) {
	control := gomock.NewController(t)
	directory := t.TempDir()
	editor := editingIn(t, directory, finding(control), shellVar,
		"/bin/zsh")
	got := running(toolWith(editor), "edit", "a.yaml")
	for _, words := range []string{"tmux is not on PATH",
		"nvim is not on PATH", "yaml-language-server is not on PATH",
		"carrie is not on PATH", "vi " + filepath.Join(directory,
			"a.yaml"), "/bin/zsh"} {
		if !strings.Contains(got.said, words) {
			t.Errorf("edit with nothing on PATH did not say %q:\n%s",
				words, got.said)
		}
	}
	if strings.Contains(got.said, "install") &&
		!strings.Contains(got.said, "npm install -g yaml-language-server") {
		t.Errorf("edit offered to install something:\n%s", got.said)
	}
}

// toolWith is the slayer tool with edit's machine the test's.
func toolWith(editor editing) *pzm.Tool {
	return tool(editor.env, pieces{}, editor.machine)
}

// a new job is started from the template, which reads as a job, names
// the schema's copy on its first line, and lists the presets; the copy
// is there and parses.
func TestTheTemplateIsAJob(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	schema, err := env.schemaCopy()
	if err != nil {
		t.Fatal(err)
	}
	path := written(t, directory, "new.yaml", string(template(schema)))
	if _, err := job.Read(path); err != nil {
		t.Fatalf("the template does not read: %v", err)
	}
	raw, _ := os.ReadFile(path)
	first := strings.SplitN(string(raw), "\n", pair)[nothing]
	where := strings.TrimPrefix(first,
		"# yaml-language-server: $schema=")
	if where == first || where != schema {
		t.Fatalf("the first line is %q", first)
	}
	named, err := os.ReadFile(where)
	var parsed map[string]any
	if err != nil || json.Unmarshal(named, &parsed) != nil ||
		parsed["$id"] == nil && parsed["title"] == nil {
		t.Errorf("the schema the modeline names does not parse: %v", err)
	}
	for _, preset := range job.Presets() {
		if !strings.Contains(string(raw), preset.Name()) {
			t.Errorf("the template does not list %s", preset.Name())
		}
	}
}

// the schema's copy is written again when it is stale, and left alone
// when it is this build's.
func TestTheSchemaCopyIsRefreshedWhenStale(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	where, err := env.schemaCopy()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(where, []byte("{}"), private); err != nil {
		t.Fatal(err)
	}
	if _, err := env.schemaCopy(); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(where); string(raw) != string(job.Schema()) {
		t.Error("a stale copy was not written again")
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(where, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := env.schemaCopy(); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(where); !info.ModTime().Equal(old) {
		t.Error("a copy that was this build's was written again")
	}
}

// unhomed is the door's surroundings with no SLAYER_CACHE, and with
// the renders home, the user's cache, HOME and the temporary folder
// each a folder of its own, so a test can see they stay empty. The
// variables given, as name, value, name, value, are laid over them; a
// value of nothing leaves the variable out. The folders are given back.
func unhomed(t *testing.T, variables ...string) (Env, map[string]string) {
	t.Helper()
	folders := map[string]string{"SLAYER_RENDERS": t.TempDir(),
		"HOME": t.TempDir(), "XDG_CACHE_HOME": t.TempDir(),
		"TMPDIR": t.TempDir()}
	said := map[string]string{}
	for name, folder := range folders {
		said[name] = folder
	}
	for i := nothing; i+one < len(variables); i += pair {
		said[variables[i]] = variables[i+1]
	}
	env, err := NewEnv(Given{Context: context.Background(), Now: noon,
		Directory: t.TempDir(),
		Environment: func(name string) (string, bool) {
			value, set := said[name]
			return value, set
		}})
	if err != nil {
		t.Fatal(err)
	}
	return env, folders
}

// untouched fails the test for each folder that has anything in it.
func untouched(t *testing.T, folders map[string]string) {
	t.Helper()
	for name, folder := range folders {
		if found, _ := os.ReadDir(folder); len(found) != nothing {
			t.Errorf("%s (%s) was written to: %v", name, folder, found)
		}
	}
}

// anti-test: with no SLAYER_CACHE there is no cache, that is said by
// name, and nothing is written to the other places it could have gone.
// anti-test of TestTheSchemaCopyIsRefreshedWhenStale
func TestNoHomeIsNoCache(t *testing.T) {
	env, folders := unhomed(t)
	if _, err := env.schemaCopy(); err == nil ||
		!strings.Contains(err.Error(), "SLAYER_CACHE") {
		t.Errorf("no SLAYER_CACHE gave %v", err)
	}
	untouched(t, folders)
}

// anti-test: a relative SLAYER_CACHE is refused by name.
func TestARelativeHomeIsNoCache(t *testing.T) {
	env := envIn(t, context.Background(), t.TempDir(), "SLAYER_CACHE",
		"relative")
	if _, err := env.schemaCopy(); err == nil ||
		!strings.Contains(err.Error(), "SLAYER_CACHE") {
		t.Errorf("a relative home gave %v", err)
	}
}

// edit writes the agent's two files beside the job where they are
// absent, and leaves one that is there as it is.
func TestEditWritesTheAgentsFilesOnce(t *testing.T) {
	directory := t.TempDir()
	written(t, directory, scoringName, "mine")
	env := envIn(t, context.Background(), directory)
	editor := editing{env: env}
	var said strings.Builder
	if err := editor.prepare(&said, filepath.Join(directory,
		"a.yaml")); err != nil {
		t.Fatal(err)
	}
	guide, err := os.ReadFile(filepath.Join(directory, guideName))
	if err != nil || !strings.Contains(string(guide), "`film`") ||
		!strings.Contains(string(guide), "SCORING.md") {
		t.Errorf("the guide is %q, %v", guide, err)
	}
	// anti-test of TestEditWritesTheAgentsFilesOnce
	if kept, _ := os.ReadFile(filepath.Join(directory,
		scoringName)); string(kept) != "mine" {
		t.Errorf("SCORING.md that was there was written over: %q", kept)
	}
	if !strings.Contains(said.String(), "from the template") {
		t.Errorf("edit did not say it started the job: %q", said.String())
	}
}

// the newest job in a folder is found among other YAML; with none, edit
// starts job.yaml.
func TestEditFindsTheNewestJob(t *testing.T) {
	directory := t.TempDir()
	older := written(t, directory, "older.yaml", jobLine+
		"scene: {smoke: 1}\n")
	newer := written(t, directory, "newer.yml", jobLine+
		"scene: {smoke: 1}\n")
	set := written(t, directory, "set.yaml", "name: not a job\n")
	for at, path := range []string{older, newer, set} {
		when := time.Now().Add(time.Duration(at) * time.Minute)
		if err := os.Chtimes(path, when, when); err != nil {
			t.Fatal(err)
		}
	}
	editor := editing{env: envIn(t, context.Background(), directory)}
	if got, err := editor.jobPath(nil); err != nil || got != newer {
		t.Errorf("the newest job is %q, %v; want %q", got, err, newer)
	}
	empty := editing{env: envIn(t, context.Background(), t.TempDir())}
	if got, _ := empty.jobPath(nil); filepath.Base(got) != newJobName {
		t.Errorf("with no job edit starts %q", got)
	}
}

// --dry-run says what edit would write and run, and writes nothing.
func TestEditDryRunWritesNothing(t *testing.T) {
	control := gomock.NewController(t)
	directory := t.TempDir()
	editor := editingIn(t, directory, NewMockprograms(control))
	got := running(toolWith(editor), "edit", "a.yaml", "--dry-run")
	if !strings.Contains(got.said, "would start") ||
		!strings.Contains(got.said, "would write") {
		t.Errorf("a dry run said:\n%s%s", got.said, got.wrongSaid)
	}
	if found := files(t, directory); len(found) != nothing {
		t.Errorf("a dry run wrote %v", found)
	}
}

// bare slayer at a terminal is edit, and starts job.yaml here; anywhere
// else it prints the help and writes nothing.
func TestBareSlayerAtATerminalEdits(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	made := Tool(env)
	made.AtTerminal(func(io.Reader, io.Writer) bool { return false })
	// anti-test of TestBareSlayerAtATerminalEdits
	if got := running(made); !strings.Contains(got.said, "watch") ||
		len(files(t, directory)) != nothing {
		t.Errorf("bare slayer not at a terminal said %q and wrote %v",
			got.said, files(t, directory))
	}
	made = Tool(env)
	made.AtTerminal(func(io.Reader, io.Writer) bool { return true })
	got := running(made)
	if !strings.Contains(got.said, "started "+filepath.Join(directory,
		newJobName)) {
		t.Errorf("bare slayer at a terminal said %q %q", got.said,
			got.wrongSaid)
	}
}

// the real machine runs a program with no shell and gives what it
// printed; one that fails is said by name with what it printed.
func TestMachineRunsAProgram(t *testing.T) {
	said, err := machine{}.Run(context.Background(), "/bin/echo", "hi")
	if err != nil || said != "hi\n" {
		t.Errorf("echo hi gave %q, %v", said, err)
	}
	// anti-test of TestMachineRunsAProgram
	if _, err := (machine{}).Run(context.Background(), "/bin/ls",
		"/nowhere/at/all"); err == nil || !strings.Contains(err.Error(),
		"ls /nowhere/at/all") {
		t.Errorf("a failing ls gave %v", err)
	}
	if found, err := (machine{}).Find("sh"); err != nil || found == "" {
		t.Errorf("sh is not found: %v", err)
	}
}

// schema prints the schema, and --where its copy's path; a file given
// to it, or no SLAYER_CACHE, is refused.
func TestSchemaVerb(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "schema", "--where")
	where := strings.TrimSpace(got.said)
	if got.code != nothing || !strings.HasSuffix(where, schemaName) {
		t.Fatalf("--where said %q %q", got.said, got.wrongSaid)
	}
	if _, err := os.Stat(where); err != nil {
		t.Errorf("--where named a copy that is not there: %v", err)
	}
	// anti-test of TestSchemaVerb
	if got := running(Tool(env), "schema", "a.yaml"); got.code == nothing {
		t.Error("schema took a file")
	}
	homeless, folders := unhomed(t)
	if got := running(Tool(homeless), "schema", "--where"); got.code ==
		nothing || !strings.Contains(got.wrongSaid, "SLAYER_CACHE") {
		t.Errorf("--where with no home gave %q", got.wrongSaid)
	}
	untouched(t, folders)
}

// a file that is not YAML, or not there, is no job; a dry
// watch of a job that does not read says so and still names the folder.
// anti-test of TestTheTemplateIsAJob
// anti-test of TestEditFindsTheNewestJob
func TestWhatIsNotAJob(t *testing.T) {
	directory := t.TempDir()
	if isJobFile(written(t, directory, "a.yaml", "a: [")) ||
		isJobFile(filepath.Join(directory, "none.yaml")) {
		t.Error("broken or absent YAML was taken as a job")
	}
	broken := written(t, directory, "b.yaml", jobLine+"scene: [\n")
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "watch", broken, "--dry-run")
	if !strings.Contains(got.said, "does not read yet") ||
		!strings.Contains(got.said, ".b.watch") {
		t.Errorf("a dry watch of a broken job said %q", got.said)
	}
	if got := running(Tool(env), "edit", "a", "b"); got.code == nothing {
		t.Error("edit took two files")
	}
	if got := running(Tool(env), "watch", broken, "--width",
		"x"); got.code == nothing {
		t.Error("watch took a width that is no number")
	}
}
