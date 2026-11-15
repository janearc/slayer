package door

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/contrib/nvim"
	"github.com/janearc/slayer/job"
	yaml "gopkg.in/yaml.v3"
)

// editHelp is edit's page.
const editHelp = `# edit

slayer edit JOB lays out three panes in this tmux window: slayer watch
JOB on the left, nvim on JOB on the right with the job schema on hover,
and carrie, the agent console, across the foot, working in the job's
folder. They share nothing but the file: the editor and the agent change
it, and the preview draws it again when it is saved. Focus is in the
editor.

A JOB that does not exist is started from a template: a commented job
with the presets listed. Beside it, edit writes SLAYER-JOB.md (the job's
keys) and SCORING.md (how to write a score) for the agent, where they
are absent.

Outside tmux it says so, prints the three commands, and runs watch in
this terminal. Where nvim, yaml-language-server or carrie is not on
PATH it says which, and what it does without it. It installs nothing.

Bare slayer at a terminal is slayer edit on the newest job in this
folder, or a new job.yaml. --dry-run prints what it would write and run.
`

// The programs edit lays out, found on PATH.
const (
	tmuxName   = "tmux"
	nvimName   = "nvim"
	serverName = "yaml-language-server"
	carrieName = "carrie"
	tmuxVar    = "TMUX"
	paneVar    = "TMUX_PANE"
	shellVar   = "SHELL"
	plainShell = "/bin/sh"
	newJobName = "job.yaml"
	// shortExtension is YAML's other extension, which a job may have
	shortExtension = ".yml"
	guideName      = "SLAYER-JOB.md"
	scoringName    = "SCORING.md"
)

// footRows is the console's height in rows. Picked by the author of this
// slice, 2026-11-12: a prompt and a few lines of answer, leaving the rest
// of a laptop's window to the picture.
const footRows = 10

// editorShare is the editor's share of the window's width, as a divisor:
// a third.
const editorShare = 3

// machine is the real programs: PATH, os.Executable and os/exec.
type machine struct{}

// Find is a program's path on PATH.
func (machine) Find(name string) (string, error) {
	return exec.LookPath(name)
}

// Self is the slayer binary running.
func (machine) Self() (string, error) { return os.Executable() }

// Run runs a program, no shell, and gives what it printed.
func (machine) Run(ctx context.Context, path string, args ...string) (
	string, error) {
	said, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", filepath.Base(path),
			strings.Join(args, " "), err, strings.TrimSpace(string(said)))
	}
	return string(said), nil
}

// editing is the edit verb, with what it runs under.
type editing struct {
	env     Env
	machine programs
	watcher watching
}

// panes is the three commands edit runs, each a program and its
// arguments, and what was missing in finding them.
type panes struct {
	watch, editor, console []string
	missing                []string
}

// edit is the verb: the job found or started, the agent's files beside
// it, the programs found, then the panes laid out and watch run here.
func (editor editing) edit(call pzm.Call) error {
	if len(call.Args()) > one {
		return errors.New("edit takes one file at most: slayer edit [JOB]")
	}
	given, err := editor.jobPath(call.Args())
	if err != nil {
		return err
	}
	if call.DryRun() {
		return editor.dry(call.Out(), given)
	}
	if err := editor.prepare(call.Out(), given); err != nil {
		return err
	}
	laid, err := editor.panesFor(given)
	if err != nil {
		return err
	}
	for _, line := range laid.missing {
		if _, err := fmt.Fprintln(call.Out(), line); err != nil {
			return err
		}
	}
	if err := editor.layOut(call.Out(), laid, given); err != nil {
		return err
	}
	read, err := readerOf(call, editor.env.directory)
	if err != nil {
		return err
	}
	return editor.watcher.live(call, given, read)
}

// jobPath is the job a call names, or with none the newest job in the
// working directory, or a new job.yaml there.
func (editor editing) jobPath(args []string) (string, error) {
	if len(args) == one {
		return absolute(args[nothing], editor.env.directory), nil
	}
	newest, found, err := newestJob(editor.env.directory)
	if err != nil || found {
		return newest, err
	}
	return filepath.Join(editor.env.directory, newJobName), nil
}

// prepare starts the job from the template where it is absent, and
// writes the agent's files beside it where they are absent.
func (editor editing) prepare(out io.Writer, given string) error {
	schema, err := editor.env.schemaCopy()
	if err != nil {
		return err
	}
	if _, err := os.Stat(given); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(given, template(schema),
			fileMode); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "started %s from the template\n",
			given); err != nil {
			return err
		}
	}
	guide, err := agentGuide()
	if err != nil {
		return err
	}
	directory := filepath.Dir(given)
	for name, holds := range map[string][]byte{guideName: guide,
		scoringName: cheeptune.Scoring()} {
		if err := absentWritten(filepath.Join(directory, name),
			holds); err != nil {
			return err
		}
	}
	return nil
}

// absentWritten writes a file where none is; one that is there is left
// as it is, whoever wrote it.
func absentWritten(path string, holds []byte) error {
	written, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		fileMode)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = written.Write(holds)
	return errors.Join(err, written.Close())
}

// panesFor is the three commands for a job, each program found on PATH,
// and a line for each that was not, saying what edit does without it.
func (editor editing) panesFor(given string) (panes, error) {
	self, err := editor.machine.Self()
	if err != nil {
		return panes{}, fmt.Errorf("cannot tell where slayer is to run "+
			"watch: %w", err)
	}
	lua, err := editor.env.keep(luaName, nvim.Script())
	if err != nil {
		return panes{}, err
	}
	schema, err := editor.env.schemaCopy()
	if err != nil {
		return panes{}, err
	}
	laid := panes{watch: []string{self, "watch", given}}
	laid.editor = editor.editorFor(&laid, given, lua, schema)
	laid.console = editor.consoleFor(&laid, filepath.Dir(given))
	return laid, nil
}

// editorFor is nvim on the job with slayer.lua loaded, or, where nvim is
// not on PATH, the shell's editor. Where the YAML language server is not
// on PATH, it says the editor has no hover.
func (editor editing) editorFor(laid *panes, given, lua,
	schema string) []string {
	if _, err := editor.machine.Find(serverName); err != nil {
		laid.missing = append(laid.missing, serverName+" is not on "+
			"PATH: the editor opens the job with no hover or completion; "+
			"npm install -g yaml-language-server adds it")
	}
	found, err := editor.machine.Find(nvimName)
	if err != nil {
		laid.missing = append(laid.missing, nvimName+" is not on PATH: "+
			"the editor pane runs vi on the job, with no hover")
		return []string{"vi", given}
	}
	return []string{found, "-c", "lua vim.g.slayer_schema = [[" + schema +
		"]]", "-c", "luafile " + lua, given}
}

// consoleFor is carrie working in the job's folder, or, where carrie is
// not on PATH, a shell there.
func (editor editing) consoleFor(laid *panes, directory string) []string {
	found, err := editor.machine.Find(carrieName)
	if err == nil {
		return []string{found, "chat", "--workspace", directory}
	}
	shell, set := editor.env.lookup(shellVar)
	if !set {
		shell = plainShell
	}
	laid.missing = append(laid.missing, carrieName+" is not on PATH: "+
		"the console pane is a shell in "+directory+"; put carrie on "+
		"PATH to have the agent there")
	return []string{shell}
}

// layOut splits this tmux window into the editor and the console, or,
// outside tmux or with no tmux, says so and prints the three commands.
func (editor editing) layOut(out io.Writer, laid panes,
	given string) error {
	tmux, err := editor.machine.Find(tmuxName)
	_, inside := editor.env.lookup(tmuxVar)
	pane, hasPane := editor.env.lookup(paneVar)
	if err != nil || !inside || !hasPane {
		return outside(out, laid, err != nil)
	}
	sized, err := editor.machine.Run(editor.env.ctx, tmux, "display-message",
		"-p", "-t", pane, "#{window_width} #{window_height}")
	if err != nil {
		return err
	}
	window, err := windowOf(sized)
	if err != nil {
		return err
	}
	for _, args := range splits(window, pane, filepath.Dir(given), laid) {
		if _, err := editor.machine.Run(editor.env.ctx, tmux,
			args...); err != nil {
			return err
		}
	}
	return nil
}

// outside says why the panes are not laid out and prints the three
// commands to run by hand; watch then runs in this terminal.
func outside(out io.Writer, laid panes, noTmux bool) error {
	why := "not inside tmux"
	if noTmux {
		why = tmuxName + " is not on PATH"
	}
	_, err := fmt.Fprintf(out, "%s: run these in three panes; watch "+
		"runs here now\n  %s\n  %s\n  %s\n", why, quoted(laid.watch),
		quoted(laid.editor), quoted(laid.console))
	return err
}

// quoted is a command as a shell would take it, each word quoted that
// needs it.
func quoted(command []string) string {
	words := make([]string, nothing, len(command))
	for _, word := range command {
		if strings.ContainsAny(word, " '\"$[]#") {
			word = "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
		}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

// window is a tmux window's size in cells.
type window struct{ cols, rows int }

// windowOf is a window's size as tmux prints it: two whole numbers.
func windowOf(said string) (window, error) {
	parts := strings.Fields(said)
	if len(parts) != pair {
		return window{}, fmt.Errorf("tmux said the window is %q, not "+
			"its width and height", said)
	}
	cols, colsErr := strconv.Atoi(parts[nothing])
	rows, rowsErr := strconv.Atoi(parts[one])
	if colsErr != nil || rowsErr != nil || cols < one || rows < one {
		return window{}, fmt.Errorf("tmux said the window is %q, not "+
			"its width and height", said)
	}
	return window{cols: cols, rows: rows}, nil
}

// splits is the tmux arguments that lay the window out from the pane
// edit runs in: the console across the whole foot, then the editor a
// third of the width on the right, which takes the focus. The pane edit
// runs in is left the rest, for watch. A console taller than half the
// window is held to half.
func splits(at window, pane, directory string, laid panes) [][]string {
	foot := min(footRows, at.rows/pair)
	right := max(one, at.cols/editorShare)
	console := append([]string{"split-window", "-v", "-f", "-l",
		strconv.Itoa(foot), "-t", pane, "-c", directory}, laid.console...)
	editor := append([]string{"split-window", "-h", "-l",
		strconv.Itoa(right), "-t", pane, "-c", directory}, laid.editor...)
	return [][]string{console, editor}
}

// dry says what edit would write and run, and writes nothing.
func (editor editing) dry(out io.Writer, given string) error {
	lines := []string{}
	if _, err := os.Stat(given); errors.Is(err, os.ErrNotExist) {
		lines = append(lines, "would start "+given+" from the template")
	}
	schema, err := editor.env.cached(schemaName)
	if err != nil {
		return err
	}
	lines = append(lines, "would keep "+schema)
	for _, name := range []string{guideName, scoringName} {
		path := filepath.Join(filepath.Dir(given), name)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			lines = append(lines, "would write "+path)
		}
	}
	lines = append(lines, "would lay out watch, the editor and the "+
		"console, and watch "+given)
	_, err = fmt.Fprintln(out, strings.Join(lines, "\n"))
	return err
}

// newestJob is the newest job file in a directory, and whether there is
// one: a YAML file whose $schema is job-v1.
func newestJob(directory string) (string, bool, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", false, err
	}
	var newest string
	var when int64
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		info, err := entry.Info()
		if err != nil || entry.IsDir() || !isJobFile(path) {
			continue
		}
		if stamp := info.ModTime().UnixNano(); newest == "" ||
			stamp > when {
			newest, when = path, stamp
		}
	}
	return newest, newest != "", nil
}

// isJobFile is whether a file is YAML that names the job schema.
func isJobFile(path string) bool {
	if ext := filepath.Ext(path); ext != jobExtension &&
		ext != shortExtension {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var said heading
	return yaml.Unmarshal(raw, &said) == nil && said.Schema == jobSchema
}

// template is a new job's text: the modeline that names the schema's
// copy for the YAML language server, then a commented job of a smoke in
// portrait that reads as it is.
func template(schema string) []byte {
	var presets []string
	for _, each := range job.Presets() {
		presets = append(presets, each.Name())
	}
	return []byte(fmt.Sprintf(templateText, schema, jobSchema,
		strings.Join(presets, ", ")))
}

// templateText is the template, with the schema's path, the schema's
// name and the presets to fill in.
const templateText = `# yaml-language-server: $schema=%s
# a new job. hover a key to see what it says; slayer watch draws it as
# you save, and slayer draw draws it to files.
$schema: %s
# the scene: {set: PATH}, {smoke: N}, or {named: NAME}
scene: {smoke: 1}
# the film's size: a preset, or width and height. the presets are
# %s
film: {preset: portrait}
# a still is drawn at one moment; say fps and duration for a film
at: {value: 0, unit: s}
`

// agentGuide is SLAYER-JOB.md: the job's keys as the schema describes
// them, and two short examples.
func agentGuide() ([]byte, error) {
	lines, err := keyLines()
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf(guideText, strings.Join(lines, "\n"),
		jobSchema, jobSchema)), nil
}

// guideText is the agent's guide, with the keys and the schema's name
// twice to fill in.
const guideText = "# a slayer job\n\n" +
	"A job file says how to draw a scene: which scene, the film, the " +
	"time, the camera, the development, the titles and the sound. " +
	"slayer watch draws it again whenever it is saved. To make music for " +
	"it, write a score as SCORING.md says, save it beside the job as " +
	"NAME.json, and set `sound: {tune: NAME.json, voicing: \"@leap-dmg\"}`" +
	" in the job.\n\nQuantities are written `{value: 20, unit: s}`. " +
	"Paths are relative to the job file.\n\n## The keys\n\n%s\n\n" +
	"## A still\n\n```yaml\n$schema: %s\nscene: {smoke: 1}\n" +
	"film: {preset: portrait}\nat: {value: 0, unit: s}\n```\n\n" +
	"## A film with music\n\n```yaml\n$schema: %s\n" +
	"scene: {set: ../doc/fire-night.yaml}\nfilm: {preset: portrait, " +
	"rays: 1}\nfps: 24\nduration: {value: 10, unit: s}\n" +
	"stock: {name: lantern, rated: 3}\n" +
	"sound: {tune: moody.json, voicing: \"@leap-dmg\"}\n```\n"
