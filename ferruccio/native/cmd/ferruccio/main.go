// Command ferruccio opens the separate GoGPU authoring workspace.
package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/gogpu/gg/gpu"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/desktop"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/ferruccio/internal/capture"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/processmetrics"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
	"github.com/janearc/slayer/ferruccio/native/internal/workspace"
	"github.com/janearc/slayer/internal/observability"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

var build, built, who, sessions string

// main delegates the public interface to Puzzlemouth.
func main() { os.Exit(execute()) }

// execute runs the command line and returns the exit status.
func execute() int {
	defer processmetrics.Register()()
	stop, err := observability.Start("ferruccio")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ferruccio observability:", err)
	} else {
		defer stop()
	}
	defer observability.StartSummary("ferruccio", os.Stderr)()
	return command().RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

// command declares every verb and flag before native initialization.
func command() *pzm.Tool {
	tool := pzm.New("ferruccio", "A place to build worlds for Slayer", pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("files explicitly saved and requested previews"), verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("open", verb.NoShort, "open the authoring window", "Open a GoGPU Metal workspace. Play opens the supplied starter image and uses the configured preview transport.", "Example_open", facts, func(call pzm.Call) error {
		if len(call.Args()) > 1 {
			return fmt.Errorf("open accepts at most one YAML file")
		}
		if err := validateDisplay(call.Value("display")); err != nil {
			return err
		}
		if err := validatePreviewTransport(call.Value("preview-transport")); err != nil {
			return err
		}
		if err := validateComposition(call.Value("composition")); err != nil {
			return err
		}
		if err := validateTheme(call.Value("theme")); err != nil {
			return err
		}
		if err := validateChoices(call); err != nil {
			return err
		}
		if call.DryRun() {
			_, err := fmt.Fprintln(call.Out(), "open Ferruccio on GoGPU Metal; dry-run opens no window and starts no preview")
			return err
		}
		file := ""
		if len(call.Args()) == 1 {
			file = call.Args()[0]
		}
		return run(call, file, false)
	}, runtimeFlags()...)
	tool.Bare("open")
	proofFacts := verb.NewFacts(verb.WritesTo("new widget and preview proof PNGs with receipt"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("proof", verb.NoShort, "verify the native workspace and export its widgets", "Open Metal, optionally render the loaded document, dispatch a gesture, export the widget tree and close. Run this command unwrapped. The configured preview transport owns finite scene rendering.", "Example_proof", proofFacts, func(call pzm.Call) error {
		if len(call.Args()) != 0 || call.Value("out") == "" {
			return fmt.Errorf("proof needs --out and no positional arguments")
		}
		gesture := call.Value("gesture")
		if workingGesture(gesture) && call.Value("composition") != "" && call.Value("composition") != "play" {
			return fmt.Errorf("held working gesture requires Play composition")
		}
		if gesture != "" && gesture != "orbit" && gesture != "zoom" && gesture != "undo" && gesture != "move" && gesture != "turn" && !workingGesture(gesture) {
			return fmt.Errorf("unknown proof gesture %q; choose orbit, zoom, move, turn, undo, working-move, working-look, trackball or travel", gesture)
		}
		if gesture != "" && !renderRequested(call) && !(workingGesture(gesture) && call.Value("starter") != "") {
			return fmt.Errorf("gesture proof needs --render, or --starter for a held working gesture")
		}
		if renderRequested(call) && call.Value("file") == "" {
			return fmt.Errorf("render proof needs --file")
		}
		if err := validateDisplay(call.Value("display")); err != nil {
			return err
		}
		if err := validatePreviewTransport(call.Value("preview-transport")); err != nil {
			return err
		}
		if err := validateComposition(call.Value("composition")); err != nil {
			return err
		}
		if gesture == "undo" && call.Value("composition") == "compose" {
			return fmt.Errorf("undo gesture proof requires --composition observe so the YAML editor can receive Command-Z")
		}
		if err := validateTheme(call.Value("theme")); err != nil {
			return err
		}
		if err := validateChoices(call); err != nil {
			return err
		}
		if call.DryRun() {
			_, err := fmt.Fprintln(call.Out(), "open native Metal Ferruccio unwrapped; configured preview transport owns scene rendering; dispatch requested UI gesture, export its own widget tree, close only Ferruccio")
			return err
		}
		if renderRequested(call) {
			if err := validateSlayerRoad(call); err != nil {
				return err
			}
		}
		return run(call, call.Value("file"), true)
	}, append([]pzm.Flag{pzm.Value("out", verb.NoShort, "new widget PNG path"), pzm.Value("file", verb.NoShort, "existing YAML file"), pzm.Switch("render", verb.NoShort, "render the file before export"), pzm.Switch("full", verb.NoShort, "render the authored film with finite CPU Slayer before export"), pzm.Value("gesture", verb.NoShort, "orbit, zoom, move, turn, undo, working-move/look, trackball or travel"), pzm.Value("late-readbacks", verb.NoShort, "make the first N working-view readbacks late, to show recovery"), pzm.Switch("hover", verb.NoShort, "rest the pointer on the middle of the viewport before export"), pzm.Switch("settings", verb.NoShort, "open the Settings sheet before the export"), pzm.Value("settings-click", verb.NoShort, "with --settings, press this control of the sheet first"), pzm.Value("size", verb.NoShort, "window size WIDTHxHEIGHT, at least 1280x800; default 1600x1000")}, runtimeFlags()...)...)
	registerRegression(tool)
	registerCheck(tool)
	registerLook(tool)
	return tool
}

// validateComposition keeps CLI-selected layouts within the supported pair.
func validateComposition(composition string) error {
	if composition != "" && composition != "play" && composition != "observe" && composition != "compose" {
		return fmt.Errorf("unknown composition %q; choose play, observe or compose", composition)
	}
	return nil
}

// validateTheme selects a checked colourway or restores the default palette.
func validateTheme(name string) error {
	if name == "" {
		name = "vaporwave"
	}
	return appearance.SelectTheme(name)
}

// runtimeFlags keeps renderer location and scoped configuration explicit.
func runtimeFlags() []pzm.Flag {
	return []pzm.Flag{pzm.Switch("background", verb.NoShort, "proof only: nonactivating native startup on macOS"), pzm.Value("display", verb.NoShort, "rendered, solid, edges or wire"), pzm.Value("preview-transport", verb.NoShort, "slayer or azazel; default slayer"), pzm.Value("starter", verb.NoShort, "directory containing source.yaml, preview.png and receipt.json"), pzm.Value("azazel", verb.NoShort, "operator Azazel executable"), pzm.Value("azazel-queue", verb.NoShort, "operator Azazel queue path"), pzm.Value("agent", verb.NoShort, "submitted agent identity, default ferruccio"), pzm.Value("evidence", verb.NoShort, "existing parent directory for new preview evidence"), pzm.Value("slayer", verb.NoShort, "Slayer executable"), pzm.Value("catalogue", verb.NoShort, "catalogue specimen renderer executable"), pzm.Value("game", verb.NoShort, "Game5 executable"), pzm.Value("game-home", verb.NoShort, "scoped Game configuration home"), pzm.Value("shelf", verb.NoShort, "optional Slayer source shelf"), pzm.Value("cache", verb.NoShort, "shared Slayer cache directory"), pzm.Value("examples", verb.NoShort, "directory of existing YAML examples"), pzm.Value("composition", verb.NoShort, "play, observe or compose"), pzm.Value("theme", verb.NoShort, "checked Chroma colourway"), pzm.Value("renderer", verb.NoShort, "cpu or gpu for your own renders; default cpu"), pzm.Value("processors", verb.NoShort, "2, 4, 8 or all for your own renders; default 2")}
}

// settingsSource says where this launch reads and saves its settings: the
// file under the person's home folder, with this call's flags and this
// process's environment above it. With no home folder there is no file.
func settingsSource(call pzm.Call) workspace.SettingsSource {
	path := ""
	if home, err := os.UserHomeDir(); err == nil {
		path = settings.Path(home)
	}
	return workspace.SettingsSource{Path: path, Flag: call.Value, Environment: os.Getenv}
}

// validateSlayerRoad refuses a proof that renders and has no cache folder
// from any of the places a setting can come from. The window itself does not
// refuse to open without one: it opens Settings and says so. The Azazel road
// gives its worker its own settings and is not asked.
func validateSlayerRoad(call pzm.Call) error {
	if previewTransport(call.Value("preview-transport")) != "slayer" {
		return nil
	}
	source := settingsSource(call)
	resolved := settings.Resolve(source.Path, source.Flag, source.Environment)
	return preview.RequireCache(preview.Tools{Cache: resolved.Value(settings.Cache)})
}

// The window's size when it opens, and the smallest the person can make it.
const (
	defaultWindowWidth  = 1600
	defaultWindowHeight = 1000
	minimumWindowWidth  = 1280
	minimumWindowHeight = 800
)

// parseWindowSize reads WIDTHxHEIGHT, which a proof may give to open the
// window at a size other than the default. It refuses a size under the
// smallest the window may be.
func parseWindowSize(text string) (int, int, error) {
	first, second, found := strings.Cut(text, "x")
	width, widthError := strconv.Atoi(first)
	height, heightError := strconv.Atoi(second)
	if !found || widthError != nil || heightError != nil || width < minimumWindowWidth || height < minimumWindowHeight {
		return 0, 0, fmt.Errorf("size is WIDTHxHEIGHT, at least %dx%d, not %q", minimumWindowWidth, minimumWindowHeight, text)
	}
	return width, height, nil
}

// validateChoices refuses a flag that gives a setting with its own list of
// values one that is not on it, at launch, in the words the sheet uses.
func validateChoices(call pzm.Call) error {
	for _, key := range settings.Keys {
		value := ""
		if key.Flag != "" {
			value = call.Value(key.Flag)
		}
		if value != "" && key.Choices != nil && !slices.Contains(key.Choices, value) {
			return fmt.Errorf("unknown %s %q; choose %s", strings.ToLower(key.Label), value, strings.Join(key.Choices, ", "))
		}
	}
	return nil
}

// validateDisplay keeps the working-view selector within its explicit modes.
func validateDisplay(mode string) error {
	if mode != "" && mode != "rendered" && mode != "solid" && mode != "edges" && mode != "wire" {
		return fmt.Errorf("unknown display %q", mode)
	}
	return nil
}

// lateReadbacks reads the proof's count of readbacks to make late. Only a
// proof takes it; open has no such flag.
func lateReadbacks(call pzm.Call, proof bool) (int, error) {
	if !proof || call.Value("late-readbacks") == "" {
		return 0, nil
	}
	value := call.Value("late-readbacks")
	count, err := strconv.Atoi(value)
	if err != nil || count < 0 {
		return 0, fmt.Errorf("late-readbacks must be a whole number, not %q", value)
	}
	return count, nil
}

// renderRequested reports whether the call asks for a render.
func renderRequested(call pzm.Call) bool { return call.On("render") || call.On("full") }

// workingGesture reports whether gesture is one of the proof gestures that
// move the working view.
func workingGesture(gesture string) bool {
	return gesture == "working-move" || gesture == "working-look" || gesture == "trackball" || gesture == "travel"
}

// heldProofPress starts a held proof gesture on work.
func heldProofPress(work *workspace.Workspace, gesture string) error {
	if gesture == "travel" {
		return work.ProofTravelPress()
	}
	if gesture == "trackball" {
		return work.ProofTrackballPress()
	}
	return work.ProofWorkingPress(gesture)
}

// heldProofMove moves a held proof gesture on work.
func heldProofMove(work *workspace.Workspace, gesture string) error {
	if gesture == "travel" {
		return work.ProofTravelMove()
	}
	if gesture == "trackball" {
		return work.ProofTrackballMove()
	}
	return work.ProofWorkingMove(gesture)
}

// heldProofRelease ends a held proof gesture on work.
func heldProofRelease(work *workspace.Workspace, gesture string) error {
	if gesture == "travel" {
		return work.ProofTravelRelease()
	}
	if gesture == "trackball" {
		return work.ProofTrackballRelease()
	}
	return work.ProofWorkingRelease()
}

type heldWorkingProof struct {
	stage, ticks           int
	source                 string
	before, held           any
	picture, beforePicture image.Image
	// drawn and beforeDrawn say what drew each capture.
	drawn, beforeDrawn workspace.Drawn
}

// drawnFields records in a receipt what drew its capture: the backend, the
// adapter, and whether the export handed the window's device to gg itself.
func drawnFields(receipt map[string]any, drawn workspace.Drawn) {
	receipt["drawn_backend"] = drawn.Backend
	receipt["drawn_adapter"] = drawn.Adapter
	receipt["export_shared_device"] = drawn.Shared
	if drawn.Problem != "" {
		receipt["device_share_problem"] = drawn.Problem
	}
}

// step dispatches held input across actual UI ticks, then waits for owned work.
func (proof *heldWorkingProof) step(work *workspace.Workspace, gesture string) (bool, error) {
	switch proof.stage {
	case 0:
		if !work.WorkingReady() {
			diagnostic := work.WorkingDiagnostic()
			if diagnostic != "" {
				return false, fmt.Errorf("working viewport: %s", diagnostic)
			}
			return false, nil
		}
		proof.source = work.ProofText()
		proof.before = work.WorkingStats()
		var err error
		proof.beforePicture, err = work.Export()
		if err != nil {
			return false, err
		}
		proof.beforeDrawn = work.ExportDrawn()
		if err := heldProofPress(work, gesture); err != nil {
			return false, err
		}
		proof.stage = 1
		return false, nil
	case 1:
		if work.ProofText() != proof.source {
			return false, fmt.Errorf("proof interrupted before planned release: source changed after press")
		}
		if err := heldProofMove(work, gesture); err != nil {
			return false, err
		}
		proof.stage = 2
		return false, nil
	case 2:
		proof.ticks++
		if proof.ticks < 2 {
			return false, nil
		}
		if !work.WorkingReady() {
			diagnostic := work.WorkingDiagnostic()
			if diagnostic != "" {
				return false, fmt.Errorf("held working viewport: %s", diagnostic)
			}
			return false, nil
		}
		if work.ProofText() != proof.source {
			return false, fmt.Errorf("proof interrupted before planned release: source changed while held")
		}
		var err error
		proof.picture, err = work.Export()
		if err != nil {
			return false, err
		}
		proof.drawn = work.ExportDrawn()
		proof.held = work.WorkingStats()
		if err = heldProofRelease(work, gesture); err != nil {
			return false, err
		}
		proof.stage = 3
		return false, nil
	default:
		if work.Running() {
			return false, nil
		}
		if work.ProofText() == proof.source {
			return false, fmt.Errorf("working gesture release did not commit source")
		}
		return true, nil
	}
}

// previewTransport keeps the app on the shared renderer unless explicitly selected otherwise.
func previewTransport(name string) string {
	if name == "" {
		return "slayer"
	}
	return name
}

// validatePreviewTransport rejects unsupported routes before native startup.
func validatePreviewTransport(name string) error {
	selected := previewTransport(name)
	if selected != "slayer" && selected != "azazel" {
		return fmt.Errorf("unknown preview transport %q; choose slayer or azazel", name)
	}
	return nil
}

// queueAgent defaults only an absent name; cancellation retains explicit identity.
func queueAgent(name string) string {
	if name == "" {
		return "ferruccio"
	}
	return name
}

// evidenceDirectory selects the shared destination without creating it.
func evidenceDirectory(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if configured := os.Getenv("FERRUCCIO_EVIDENCE"); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "mesh", "the-vorrh", "renders", "frames")
}

// run shares one real workspace between normal use and the native proof.
func run(call pzm.Call, file string, proof bool) error {
	composition := call.Value("composition")
	if composition == "" {
		composition = "play"
	}
	starter := call.Value("starter")
	if proof && renderRequested(call) {
		starter = ""
	}
	late, err := lateReadbacks(call, proof)
	if err != nil {
		return err
	}
	source := settingsSource(call)
	resolved := settings.Resolve(source.Path, source.Flag, source.Environment)
	if theme := resolved.Value(settings.Theme); theme != "" && call.Value("theme") == "" {
		// A theme the file names that Chroma does not have is left unused.
		_ = appearance.SelectTheme(theme)
	}
	width, height := defaultWindowWidth, defaultWindowHeight
	if proof && call.Value("size") != "" {
		if width, height, err = parseWindowSize(call.Value("size")); err != nil {
			return err
		}
	}
	host := gogpu.NewApp(gogpu.DefaultConfig().WithTitle("Ferruccio · Slayer").WithSize(width, height).WithMinSize(minimumWindowWidth, minimumWindowHeight).WithGraphicsAPI(gogpu.GraphicsAPIMetal).WithBackground(proof && call.On("background")))
	ui := app.New(app.WithWindowProvider(host), app.WithPlatformProvider(host), app.WithEventSource(host.EventSource()), app.WithTheme(appearance.ThemeForSize(16)))
	work, err := workspace.New(host, ui, workspace.Options{Starter: starter, SuppressAutomaticPreview: proof, Evidence: evidenceDirectory(call.Value("evidence")), File: file, Examples: resolved.Value(settings.Examples), Settings: source, Composition: composition, Tools: preview.Tools{Transport: previewTransport(call.Value("preview-transport")), Azazel: call.Value("azazel"), Queue: call.Value("azazel-queue"), Agent: queueAgent(call.Value("agent")), Slayer: resolved.Value(settings.Slayer), Catalogue: call.Value("catalogue"), Game: call.Value("game"), GameHome: call.Value("game-home"), Shelf: resolved.Value(settings.Shelf), Cache: resolved.Value(settings.Cache)}})
	if err != nil {
		return err
	}
	if !proof {
		work.InstallMenu()
	}
	display := call.Value("display")
	if display == "" {
		display = "rendered"
	}
	if proof && workingGesture(call.Value("gesture")) && display == "rendered" {
		display = "edges"
	}
	if err := work.SetWorkingMode(display); err != nil {
		return err
	}
	work.ForceLateReadbacks(late)
	held := &heldWorkingProof{}
	started, finished := false, false
	beginning := time.Now()
	ready := time.Time{}
	var proofError error
	gesture := ""
	if proof {
		gesture = call.Value("gesture")
	}
	gestureStage := 0
	var initialPreview, gesturePreview, initialText, gestureText string
	var undoImageMatches bool
	backgroundApplied := proof && call.On("background")
	animationStarted, settingsAsked, settingsSeen, settingsClicked := false, false, false, false
	var watchdog *time.Timer
	timeout := &proofStop{}
	if proof {
		watchdog = time.AfterFunc(4*time.Minute, func() { timeout.expire(); host.RequestRedraw() })
		defer watchdog.Stop()
	}
	host.OnUpdate(func(_ float64) {
		if proof && !animationStarted {
			animationStarted = true
			host.StartAnimation()
		}
		if backgroundApplied {
			ui.Window().Frame()
		}
		work.Tick()
		if proof && !finished {
			if expired, stopped := timeout.poll(work); expired {
				if stopped {
					_, diagnostic := work.ProofOutput()
					proofError = fmt.Errorf("Ferruccio proof exceeded four minutes: %s", diagnostic)
					finished = true
					host.Quit()
				}
				return
			}
		}
		if !proof || finished || time.Since(beginning) < time.Second {
			return
		}
		if !started {
			started = true
			if renderRequested(call) {
				current, _ := work.ProofOutput()
				if current == "" && !work.Running() {
					startRender := work.StartPreview
					if call.On("full") {
						startRender = work.StartFullRender
					}
					if err := startRender(); err != nil {
						proofError = err
						finished = true
						host.Quit()
						return
					}
				}
			}
		}
		if work.Running() {
			return
		}
		if call.On("settings") && !settingsAsked {
			settingsAsked = true
			work.OpenSettings()
		}
		if control := call.Value("settings-click"); control != "" && !settingsClicked {
			if !work.SettingsOpen() {
				return
			}
			if !settingsSeen {
				settingsSeen, ready = true, time.Now()
				return
			}
			if time.Since(ready) < 500*time.Millisecond {
				return
			}
			settingsClicked = true
			if err := work.ProofSettingsClick(control); err != nil {
				proofError = err
				finished = true
				host.Quit()
				return
			}
			ready = time.Time{}
			return
		}
		if ready.IsZero() {
			ready = time.Now()
			return
		}
		if time.Since(ready) < 500*time.Millisecond {
			return
		}
		if display != "rendered" && !work.WorkingReady() {
			diagnostic := work.WorkingDiagnostic()
			if diagnostic == "" && time.Since(ready) < 3*time.Second {
				return
			}
			if diagnostic == "" {
				diagnostic = "working viewport did not become ready after warming"
			}
			proofError = fmt.Errorf("working viewport: %s", diagnostic)
			finished = true
			host.Quit()
			return
		}
		if workingGesture(gesture) {
			done, err := held.step(work, gesture)
			if err != nil {
				proofError = err
				finished = true
				host.Quit()
				return
			}
			if !done {
				return
			}
			output, diagnostic := work.ProofOutput()
			if renderRequested(call) && output == "" {
				proofError = fmt.Errorf("post-release refinement missing: %s", diagnostic)
				finished = true
				host.Quit()
				return
			}
			receipt := map[string]any{
				"build": build, "native_window_opened": !backgroundApplied, "native_window_created": true, "window_hidden_at_capture": backgroundApplied, "requested_backend": "metal",
				"capture": "native widget-tree export captured while pointer held; not OS screenshot",
				"gesture": gesture, "display": display, "held_source_unchanged": true,
				"source_text_before": held.source, "source_text_held": held.source,
				"source_text_after_release": work.ProofText(), "working_before": held.before,
				"working_held": held.held, "working_after_release": work.WorkingStats(),
				"preview_after_release": output, "refinement_diagnostic": diagnostic,
			}
			out := call.Value("out")
			extension := filepath.Ext(out)
			beforePath := out[:len(out)-len(extension)] + "-before" + extension
			beforeReceipt := map[string]any{
				"build": build, "native_window_opened": !backgroundApplied, "native_window_created": true, "window_hidden_at_capture": backgroundApplied, "requested_backend": "metal",
				"capture": "native widget-tree export before held gesture; not OS screenshot",
				"gesture": gesture, "display": display, "source_text": held.source, "working": held.before,
			}
			receipt["before_capture"] = beforePath
			drawnFields(receipt, held.drawn)
			drawnFields(beforeReceipt, held.beforeDrawn)
			proofError = capture.Write(beforePath, held.beforePicture, beforeReceipt)
			if proofError == nil {
				proofError = capture.Write(out, held.picture, receipt)
			}
			if proofError == nil {
				proofError = json.NewEncoder(call.Out()).Encode(receipt)
			}
			finished = true
			host.Quit()
			return
		}
		if gesture != "" {
			if gestureStage == 0 {
				var initialDiagnostic string
				initialPreview, initialDiagnostic = work.ProofOutput()
				if initialPreview == "" {
					proofError = fmt.Errorf("gesture proof has no accepted initial preview: %s", initialDiagnostic)
					finished = true
					host.Quit()
					return
				}
				initialText = work.ProofText()
				action := gesture
				if action == "undo" {
					action = "orbit"
					if composition == "play" {
						action = "move"
					}
				}
				if err := work.ProofGesture(action); err != nil {
					proofError = err
					finished = true
					host.Quit()
					return
				}
				if work.ProofText() == initialText {
					proofError = fmt.Errorf("proof gesture did not change the YAML draft")
					finished = true
					host.Quit()
					return
				}
				gestureText = work.ProofText()
				gestureStage = 1
				ready = time.Time{}
				return
			}
			if gesture == "undo" && gestureStage == 1 {
				gesturePreview, _ = work.ProofOutput()
				if err := work.ProofUndo(initialText); err != nil {
					proofError = err
					finished = true
					host.Quit()
					return
				}
				gestureStage = 2
				ready = time.Time{}
				return
			}
			if gesture == "undo" {
				output, diagnostic := work.ProofOutput()
				if output == "" {
					proofError = fmt.Errorf("undo proof produced no restored preview: %s", diagnostic)
					finished = true
					host.Quit()
					return
				}
				matches, err := workspace.ProofPreviewsEqual(initialPreview, output)
				if err != nil || !matches {
					proofError = fmt.Errorf("undo proof preview does not match the initial render: matches=%t: %v", matches, err)
					finished = true
					host.Quit()
					return
				}
				undoImageMatches = true
			} else {
				gesturePreview, _ = work.ProofOutput()
				if gesturePreview == "" || work.ProofText() == initialText {
					proofError = fmt.Errorf("camera gesture proof produced no changed rendered preview")
					finished = true
					host.Quit()
					return
				}
			}
		}
		finished = true
		output, diagnostic := work.ProofOutput()
		if (renderRequested(call) || call.Value("starter") != "") && output == "" {
			proofError = fmt.Errorf("render proof produced no current preview: %s", diagnostic)
			host.Quit()
			return
		}
		hovered := ""
		if call.On("hover") {
			if hovered, proofError = work.ProofHover(); proofError != nil {
				host.Quit()
				return
			}
		}
		picture, err := work.Export()
		if err != nil || picture == nil {
			proofError = fmt.Errorf("widget export failed: %v", err)
			host.Quit()
			return
		}
		receipt := map[string]any{"build": build, "native_window_opened": !backgroundApplied, "native_window_created": true, "window_hidden_at_capture": backgroundApplied, "requested_backend": "metal", "composition": composition, "theme": appearance.SelectedTheme(), "capture": "GoGPU UI widget-tree export; not an OS window screenshot", "width": picture.Bounds().Dx(), "height": picture.Bounds().Dy(), "source_file": work.DocumentName(), "preview_file": output, "full_render": call.On("full"), "display": display, "working": work.WorkingStats()}
		drawnFields(receipt, work.ExportDrawn())
		if hovered != "" {
			receipt["hover"] = hovered
		}
		if gesture != "" {
			receipt["gesture"] = gesture
			receipt["initial_preview_file"] = initialPreview
			receipt["gesture_preview_file"] = gesturePreview
			receipt["source_text_before"] = initialText
			receipt["source_text_after_gesture"] = gestureText
			receipt["source_text_after"] = work.ProofText()
			if gesture == "undo" {
				receipt["restored_preview_matches_initial"] = undoImageMatches
			}
		}
		proofError = capture.Write(call.Value("out"), picture, receipt)
		if proofError == nil {
			proofError = json.NewEncoder(call.Out()).Encode(receipt)
		}
		host.Quit()
	})
	if err := desktop.Run(host, ui); err != nil {
		return err
	}
	if proof && !finished {
		return fmt.Errorf("window closed before the proof finished")
	}
	return proofError
}
