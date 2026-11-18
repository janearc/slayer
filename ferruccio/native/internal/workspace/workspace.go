package workspace

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/dialog"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/core/progress"
	"github.com/gogpu/ui/core/splitview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/theme/devtools"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/editor"
	"github.com/janearc/slayer/ferruccio/native/internal/picking"
	"github.com/janearc/slayer/internal/observability"
)

// Options identifies existing renderer executables and editable examples.
type Options struct {
	Tools                    preview.Tools
	Examples, File           string
	Composition              string
	Evidence                 string
	Starter                  string
	SuppressAutomaticPreview bool
}

// Workspace owns UI-thread state; background work returns only immutable results.
type Workspace struct {
	state                               documentState
	host                                *gogpu.App
	ui                                  *app.App
	options                             Options
	editor                              *editor.Widget
	viewer                              *imageView
	root                                widget.Widget
	theme                               *devtools.Theme
	size                                float32
	status, diagnostic, caption, output string
	results                             chan renderResult
	progress                            chan renderProgress
	queueState, queueHeld               string
	queueSequence                       uint64
	queueObserved                       bool
	acceptedGeneration                  uint64
	acceptedText, acceptedPath          string
	cancel                              context.CancelFunc
	cancelRequested                     bool
	log                                 *boundedLog
	closing, closeHook                  bool
	lastCheck                           time.Time
	needsCheck                          bool
	exampleDraft                        bool
	study                               string
	specimen                            string
	pending                             navigationQueue
	navigationTimer                     *time.Timer
	generation                          uint64
	refreshLayout                       bool
	composition                         string
	play                                playState
	working                             workingState
	library                             []sceneEntry
	autoPreview                         bool
	// drawn says what drew the most recent Export.
	drawn Drawn
}

type renderResult struct {
	full        bool
	observation preview.Update
	geometry    *picking.Scene
	plan        *preview.Plan
	err         error
	generation  uint64
}

// New prepares a real file and the initial image without starting a render.
func New(host *gogpu.App, ui *app.App, options Options) (*Workspace, error) {
	started := time.Now()
	defer func() { observability.Observe("ferruccio_workspace_init_seconds", time.Since(started).Seconds()) }()
	var err error
	options, err = locations(options)
	if err != nil {
		return nil, err
	}
	work := &Workspace{host: host, ui: ui, options: options, size: 16, theme: appearance.Theme(), results: make(chan renderResult, 1), progress: make(chan renderProgress, 32), log: &boundedLog{}}
	work.composition = "observe"
	if options.Composition != "" {
		if options.Composition != "play" && options.Composition != "observe" && options.Composition != "compose" {
			return nil, fmt.Errorf("composition must be play, observe or compose")
		}
		work.composition = options.Composition
	}
	if options.File != "" {
		if err := work.state.load(options.File); err != nil {
			return nil, err
		}
	} else if options.Examples != "" {
		example := "first-picture.yaml"
		if work.composition == "play" {
			example = "ferruccio-playground.yaml"
		}
		if err := work.example(example); err != nil {
			return nil, err
		}
	} else {
		document, err := draft.Open("Untitled.yaml", []byte("# Open an existing Slayer YAML file to begin.\nscene: {}\n"))
		if err != nil {
			return nil, err
		}
		work.state.document = document
	}
	work.selectSpecimen()
	work.editor = editor.New(work.state.document.Text(), func(text string) {
		work.stopPreview()
		work.state.document.Replace(text)
		work.needsCheck = true
		work.lastCheck = time.Now()
		if work.output != "" {
			work.caption = "Previous preview · draft has changed"
		}
		work.invalidate()
	})
	work.editor.SetEscapeHandler(work.returnToWorld)
	work.viewer = newImageView()
	work.viewer.SetNavigateHandler(func(yaw, pitch, zoom float64) {
		work.problem(work.Navigate(yaw, pitch, zoom))
	})
	work.viewer.SetUndoHandler(func(ctx widget.Context, redo bool) {
		work.undoCamera(ctx, redo)
	})
	if work.composition == "play" {
		work.setupPlay()
		work.autoPreview = !options.SuppressAutomaticPreview
		work.caption = "Grab the ship. Try something."
	} else {
		work.reference("venus")
	}
	work.validate()
	if work.composition == "play" && options.Starter != "" {
		if err := work.openStarter(options.Starter); err != nil {
			_, _ = work.log.Write([]byte("Starting view: " + err.Error() + "\n"))
		}
	}
	work.rebuild()
	return work, nil
}

// Tick drains worker results on GoGPU's UI thread and debounces schema checks.
func (work *Workspace) Tick() {
	if observability.Enabled() {
		started := time.Now()
		defer func() { observability.Observe("ferruccio_ui_tick_seconds", time.Since(started).Seconds()) }()
	}
	if work.autoPreview && work.cancel == nil {
		work.autoPreview = false
		work.startAutomaticPreview()
	}
	if !work.closeHook && work.host.PrimaryWindow() != nil {
		work.closeHook = true
		work.host.PrimaryWindow().SetOnClose(func() bool {
			if work.closing {
				return true
			}
			work.protect(func() {
				work.clearNavigation()
				if work.cancel != nil {
					work.closing = true
					work.cancel()
					work.status = "Stopping the preview before closing…"
					work.invalidate()
				} else {
					work.closing = true
					work.host.Quit()
				}
			})
			return false
		})
	}
	if work.needsCheck && time.Since(work.lastCheck) > 250*time.Millisecond {
		work.needsCheck = false
		work.validate()
		work.invalidate()
	}
	work.drainProgress()
	select {
	case result := <-work.results:
		work.finishPreview(result)
		work.refreshLayout = true
		work.invalidate()
	default:
	}
	if work.refreshLayout && !work.gesturing() {
		work.refreshLayout = false
		work.rebuild()
	}
	work.advanceNavigation()
	work.updateWorking()
}

// StartPreview freezes current text and invokes the same tested Slayer adapter.
func (work *Workspace) StartPreview() error { return work.startRender(false) }

// startRender shares cancellation, frozen input and acceptance across both modes.
func (work *Workspace) startRender(full bool) error {
	work.autoPreview = false
	if work.cancel != nil {
		return fmt.Errorf("a preview is already running")
	}
	if !work.rendererConfigured() {
		return fmt.Errorf("configure the preview queue, renderer and shared render destination before drawing")
	}
	request := work.previewRequest(time.Now())
	if full {
		request.Profile = preview.Profile{Full: true, CPURenderer: true}
		request.DirectoryPrefix = strings.Replace(request.DirectoryPrefix, "-preview-", "-full-", 1)
	}
	measured := observability.Enabled()
	var prepareStarted time.Time
	if measured {
		prepareStarted = time.Now()
	}
	plan, err := preview.Prepare(request)
	if measured {
		observability.Observe("ferruccio_preview_prepare_seconds", time.Since(prepareStarted).Seconds())
	}
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	work.clearNavigation()
	work.generation++
	generation := work.generation
	work.cancel = cancel
	work.cancelRequested = false
	work.diagnostic = ""
	work.log.reset()
	work.queueState, work.queueHeld = "", ""
	work.queueSequence, work.queueObserved = 0, false
	work.status = "Preparing preview"
	if full {
		work.status = "Preparing full render"
	}
	if work.options.Tools.Transport != "slayer" {
		work.status = "Submitting preview request"
		if full {
			work.status = "Submitting full render request"
		}
	}
	work.caption = work.retainedImageCaption(work.status)
	work.refreshLayout = true
	work.invalidate()
	play := work.composition == "play"
	go func() {
		var started time.Time
		if measured {
			started = time.Now()
		}
		var observation preview.Update
		err := preview.RunObserved(ctx, work.options.Tools, plan, work.log, func(update preview.Update) {
			observation = update
			select {
			case work.progress <- renderProgress{update: update, generation: generation}:
				work.host.RequestRedraw()
			case <-ctx.Done():
			}
		})
		if measured {
			observability.Observe("ferruccio_preview_roundtrip_seconds", time.Since(started).Seconds())
			if err != nil && ctx.Err() != nil {
				observability.Add("ferruccio_preview_cancelled_total", 1)
			} else if err != nil {
				observability.Add("ferruccio_preview_failed_total", 1)
			} else {
				observability.Add("ferruccio_preview_completed_total", 1)
			}
		}
		var geometry *picking.Scene
		if err == nil && play && ctx.Err() == nil {
			geometry, _ = picking.New(request.Text, float64(plan.Profile.Width)/float64(plan.Profile.Height))
		}
		closeErr := plan.Close()
		cancel()
		if err == nil {
			err = closeErr
		}
		work.results <- renderResult{full: full, plan: plan, err: err, generation: generation, geometry: geometry, observation: observation}
		work.host.RequestRedraw()
	}()
	return nil
}

// Running reports an outstanding client observation or local pending preview.
// A false result alone does not confirm that the external job has stopped.
func (work *Workspace) Running() bool { return work.cancel != nil || work.pending.document != nil }

// Output returns the accepted preview and its current diagnostic.
func (work *Workspace) Output() (string, string) { return work.output, work.diagnostic }

// ProofOutput returns only a preview accepted for this exact source and generation.
// Queue state is evidence of waiting, not evidence of a completed new image.
func (work *Workspace) ProofOutput() (string, string) {
	if work.diagnostic != "" {
		return "", work.diagnostic
	}
	if work.Running() || work.acceptedGeneration != work.generation ||
		work.acceptedPath != work.state.path || work.acceptedText != work.state.document.Text() {
		return "", work.status
	}
	return work.output, ""
}

// DocumentName exposes the active file for a capture receipt.
func (work *Workspace) DocumentName() string { return work.state.document.Name() }

// invalidate updates dynamic labels without replacing the focused text widget.
func (work *Workspace) invalidate() {
	if work.root != nil {
		if marker, ok := work.root.(interface{ MarkNeedsLayout() }); ok {
			marker.MarkNeedsLayout()
		}
		if marker, ok := work.root.(interface{ SetNeedsRedraw(bool) }); ok {
			marker.SetNeedsRedraw(true)
		}
	}
	work.ui.Window().Context().Invalidate()
	work.host.RequestRedraw()
}

// rebuild changes structural controls while preserving the editor and its text.
func (work *Workspace) rebuild() {
	work.working.dirty = true
	work.editor.SetFontSize(work.size)
	work.viewer.SetFontSize(work.size)
	work.theme = appearance.Theme()
	work.ui.SetTheme(appearance.ThemeForSize(work.size))
	work.root = work.layout()
	work.ui.SetRoot(work.root)
	work.invalidate()
}

// validate keeps invalid drafts editable and reports the schema's real error.
func (work *Workspace) validate() {
	if err := work.state.document.Check(); err != nil {
		work.status = "Draft needs attention"
		work.diagnostic = err.Error()
	} else {
		work.status = "YAML checked · " + work.state.document.Kind()
		work.diagnostic = ""
	}
}

// problem retains an actionable error in the visible status and detail dialog.
func (work *Workspace) problem(err error) {
	if err != nil {
		work.status = "Action could not finish"
		work.diagnostic = err.Error()
		work.invalidate()
		work.message("Could not finish", err.Error())
	}
}

// protect offers save, discard and cancel before replacing unsaved work.
func (work *Workspace) protect(action func()) {
	if !work.state.document.Dirty() {
		action()
		return
	}
	dialog.New(dialog.Title("Keep your changes?"), dialog.Content(work.text("Save this draft before continuing. Unfinished YAML can be saved.")), dialog.Actions(
		dialog.Action{Label: "Cancel", Variant: dialog.VariantOutlined},
		dialog.Action{Label: "Discard", Variant: dialog.VariantOutlined, OnClick: action},
		dialog.Action{Label: "Save", OnClick: func() {
			if work.save(false) {
				action()
			}
		}}), dialog.PainterOpt(appearance.DialogPainter{Theme: work.theme, Size: work.size})).Show(work.ui.Window().Context())
	work.host.RequestRedraw()
}

// open chooses an existing file through the native macOS picker.
func (work *Workspace) open() {
	work.protect(func() {
		paths, err := work.host.ShowOpenFileDialog(gogpu.FileDialogOptions{Title: "Open a Slayer YAML file"})
		if err != nil {
			work.problem(err)
			return
		}
		if len(paths) == 0 {
			return
		}
		if err := work.state.load(paths[0]); err != nil {
			work.problem(err)
			return
		}
		work.resetDocumentView()
		work.exampleDraft = false
		work.study = ""
		work.selectSpecimen()
		work.editor.SetText(work.state.document.Text())
		work.viewer.set(nil)
		work.output = ""
		work.caption = "Render this file to see its preview"
		if work.composition == "play" {
			work.play.geometry = nil
			work.play.selected = ""
			work.autoPreview = true
		}
		work.validate()
		work.rebuild()
	})
}

// save preserves invalid work and uses the native overwrite confirmation.
func (work *Workspace) save(as bool) bool {
	path := work.state.path
	if as || path == "" || work.exampleDraft {
		initialDirectory := filepath.Dir(path)
		if work.exampleDraft {
			initialDirectory = ""
		}
		chosen, err := work.host.ShowSaveFileDialog(gogpu.FileDialogOptions{Title: "Save YAML draft", DefaultFilename: work.state.document.Name(), InitialDirectory: initialDirectory})
		if err != nil {
			work.problem(err)
			return false
		}
		if chosen == "" {
			return false
		}
		path = chosen
	}
	var err error
	if work.study != "" {
		err = saveBundledStudyDocument(work.state.document, path, filepath.Dir(work.state.path), work.study)
	} else {
		err = work.state.document.Save(path)
	}
	if err != nil {
		work.problem(err)
		return false
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		work.problem(err)
		return false
	}
	work.state.path = absolute
	work.exampleDraft = false
	work.status = "Saved · " + filepath.Base(path)
	work.invalidate()
	return true
}

// example copies a bundled example so app updates cannot overwrite user edits.
func (work *Workspace) example(name string) error {
	var path, study string
	var folder string
	if _, isStudy := studyForExample(name); isStudy {
		var err error
		path, study, err = copyBundledStudy(work.options.Examples, name)
		if err != nil {
			return err
		}
		folder = filepath.Dir(path)
	} else {
		clean := filepath.Clean(name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("example path escapes its bundled folder")
		}
		examples, err := filepath.Abs(work.options.Examples)
		if err != nil {
			return err
		}
		path = filepath.Join(examples, clean)
		relative, err := filepath.Rel(examples, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("example path escapes its bundled folder")
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		folder, err = os.MkdirTemp("", "atelier-example-")
		if err != nil {
			return err
		}
		path = filepath.Join(folder, clean)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			os.RemoveAll(folder)
			return err
		}
		if err := os.WriteFile(path, source, 0600); err != nil {
			os.RemoveAll(folder)
			return err
		}
		if err := preview.CopySourceManifest(filepath.Join(examples, clean), folder); err != nil {
			os.RemoveAll(folder)
			return err
		}
	}
	if err := work.state.load(path); err != nil {
		_ = os.RemoveAll(folder)
		return err
	}
	work.exampleDraft = true
	work.study = study
	work.selectSpecimen()
	return nil
}

// chooseExample protects the draft before opening an editable bundled example.
func (work *Workspace) chooseExample(name string) {
	work.chooseCatalogueExample(name, "")
}

// chooseCatalogueExample declares the identity of a bundled study explicitly.
func (work *Workspace) chooseCatalogueExample(name, specimen string) {
	work.protect(func() {
		if err := work.example(name); err != nil {
			work.problem(err)
			return
		}
		work.resetDocumentView()
		if specimen != "" {
			work.specimen = specimen
		}
		work.editor.SetText(work.state.document.Text())
		work.viewer.set(nil)
		work.output = ""
		work.caption = "Render this file to see its preview"
		if work.composition == "play" {
			work.play.geometry = nil
			work.play.selected = ""
			work.autoPreview = true
		}
		work.validate()
		work.rebuild()
	})
}

// reference displays credited mission imagery without pretending it is a render.
func (work *Workspace) reference(place string) {
	name, credit := "venus.jpg", "Venus · NASA / JPL Magellan · mission reference"
	if place == "titan" {
		name, credit = "titan-colour.jpg", "Titan · NASA / JPL / ESA / University of Arizona · mission reference"
	}
	raw, err := assets.Read(name)
	if err != nil {
		work.problem(err)
		return
	}
	picture, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		work.problem(err)
		return
	}
	work.stopPreview()
	work.viewer.SetNavigable(false)
	work.output = ""
	work.working.photograph = picture
	work.working.displayed = false
	work.working.dirty = true
	work.viewer.set(picture)
	work.caption = credit
	work.invalidate()
}

// showImage decodes a bounded real image, including a renderer's PNG output.
func (work *Workspace) showImage(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	configuration, _, err := image.DecodeConfig(file)
	if err != nil {
		return err
	}
	if configuration.Width <= 0 || configuration.Height <= 0 || int64(configuration.Width)*int64(configuration.Height) > 64*1024*1024 {
		return fmt.Errorf("image exceeds the 64 megapixel viewer limit")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	picture, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	work.working.photograph = picture
	work.working.displayed = false
	work.viewer.set(picture)
	work.viewer.SetNavigable(false)
	return nil
}

// boundedLog accepts concurrent child stdout/stderr without growing indefinitely.
type boundedLog struct {
	mutex sync.Mutex
	text  string
}

// Write serializes child output and retains only its most recent 64 KiB.
func (log *boundedLog) Write(raw []byte) (int, error) {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	log.text += string(raw)
	if len(log.text) > 65536 {
		log.text = log.text[len(log.text)-65536:]
	}
	return len(raw), nil
}

// reset starts an independent transcript for the next preview.
func (log *boundedLog) reset() { log.mutex.Lock(); defer log.mutex.Unlock(); log.text = "" }

// read copies the transcript while the renderer may still be writing it.
func (log *boundedLog) read() string { log.mutex.Lock(); defer log.mutex.Unlock(); return log.text }

// text applies the user-selected reading size consistently.
func (work *Workspace) text(text string) *primitives.TextWidget {
	return primitives.Text(text).FontSize(work.size).Color(appearance.Ink)
}

// live reads current UI-thread state when a label is laid out or painted.
func (work *Workspace) live(fn func() string) *primitives.TextWidget {
	return primitives.TextFn(fn).FontSize(work.size).Color(appearance.Ink).MaxLines(2)
}

// action gives each working command a visible and size-aware button.
func (work *Workspace) action(label string, primary bool, disabled func() bool, action func()) *button.Widget {
	control := button.New(button.TextOpt(label), button.OnClick(action), button.DisabledFn(disabled), button.PainterOpt(appearance.ButtonPainter{Size: work.size, Primary: primary}))
	control.SetID(label)
	return control
}

// layout gives the real image most of the window and keeps YAML always visible.
func (work *Workspace) layout() widget.Widget {
	if work.composition == "play" {
		return work.playLayout()
	}
	idle := func() bool { return false }
	busy := func() bool { return work.cancel != nil }
	top := primitives.HBox(
		primitives.Text("ferruccio").FontSize(30).Color(appearance.Pink),
		work.action("Observe", work.composition == "observe", idle, func() { work.composition = "observe"; work.rebuild() }),
		work.action("Compose", work.composition == "compose", idle, func() { work.composition = "compose"; work.rebuild() }),
		primitives.Expanded(primitives.Box()),
		work.action("Open…", false, idle, work.open), work.action("Save", false, idle, func() { work.save(false) }), work.action("Save as…", false, idle, func() { work.save(true) }),
		work.action("Render preview", true, func() bool {
			return busy() || work.state.path == "" || !work.rendererConfigured() || work.state.document.Check() != nil
		}, func() { work.problem(work.StartPreview()) }),
		work.action("Stop", false, func() bool { return !work.Running() }, func() {
			work.stopPreview()
			work.invalidate()
		}),
	).Gap(10).PaddingXY(20, 14).Background(appearance.Panel)
	navigation := primitives.VBox(work.text("Studies").Color(appearance.Cyan),
		work.action("Larson + visitor", false, idle, func() { work.chooseExample("ferruccio-playground.yaml") }),
		work.action("Subaru", false, idle, func() { work.chooseCatalogueExample("subaru-settings.yaml", "subaru") }),
		work.action("First picture", false, idle, func() { work.chooseExample("first-picture.yaml") }), work.action("Webb at L2", false, idle, func() { work.chooseExample("webb-at-l2.yaml") }), work.action("Fizz", false, idle, func() { work.chooseExample("fizz.yaml") }),
		primitives.Expanded(primitives.Box()), work.text("World references").Color(appearance.Cyan),
		work.action("Venus", false, idle, func() { work.reference("venus") }), work.action("Titan", false, idle, func() { work.reference("titan") }),
	).Width(180).Padding(16).Gap(14).Background(appearance.Panel)
	picture := primitives.VBox(work.live(func() string {
		return strings.TrimSuffix(work.state.document.Name(), filepath.Ext(work.state.document.Name()))
	}).FontSize(28), primitives.Expanded(work.viewer), work.live(func() string { return work.caption }), work.action("Open image…", false, idle, func() {
		paths, err := work.host.ShowOpenFileDialog(gogpu.FileDialogOptions{Title: "View a PNG or JPEG"})
		if err != nil {
			work.problem(err)
			return
		}
		if len(paths) > 0 {
			if err := work.showImage(paths[0]); err != nil {
				work.problem(err)
				return
			}
			work.stopPreview()
			work.output = ""
			work.caption = "Image · " + filepath.Base(paths[0])
			work.invalidate()
		}
	})).Padding(18).Gap(12).Background(appearance.Ground)
	fontIndex := 0
	for index, size := range []float32{16, 18, 20, 24} {
		if work.size == size {
			fontIndex = index
		}
	}
	sizePicker := dropdown.New(dropdown.Items("16 pt", "18 pt", "20 pt", "24 pt"), dropdown.Selected(fontIndex), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) { work.size = []float32{16, 18, 20, 24}[index]; work.rebuild() }))
	document := primitives.VBox(primitives.HBox(work.text("YAML").Color(appearance.Cyan), primitives.Expanded(work.live(func() string {
		if work.state.document.Dirty() {
			return "Unsaved changes"
		}
		return work.state.document.Kind()
	}))).Gap(12), work.catalogueControl(), primitives.Expanded(work.editor), primitives.HBox(work.action("Check", false, idle, func() {
		work.validate()
		work.invalidate()
		if work.diagnostic != "" {
			work.message("YAML check", work.diagnostic)
		}
	}), work.action("Schema", false, idle, func() {
		work.message("Document schema", work.schemaName()+"\n\nEdit the YAML alongside the preview. Unknown future camera, sequence and sound fields remain in the document. Set files use their pinned ecosystem schema; legacy scenes receive syntax checks before Slayer validates them.")
	})).Gap(8), work.shortDiagnostic()).Width(480).Padding(18).Gap(12).Background(appearance.Panel)
	split := splitview.New(splitview.First(picture), splitview.Second(document), splitview.InitialRatio(.60), splitview.MinFirst(300), splitview.MinSecond(320), splitview.PainterOpt(devtools.SplitViewPainter{Theme: work.theme}))
	content := primitives.HBox(navigation, primitives.Expanded(split)).Gap(1)
	if work.composition == "compose" {
		content = primitives.HBox(primitives.Expanded(primitives.VBox(
			primitives.Expanded(picture), work.cameraControls(),
		).Gap(8)))
	}
	var footerChildren []widget.Widget
	if work.cancel != nil {
		footerChildren = append(footerChildren, progress.New(progress.Indeterminate(true), progress.Size(24), progress.TrackColor(appearance.Raised), progress.IndicatorColor(appearance.Cyan)))
	}
	footerChildren = append(footerChildren, primitives.Expanded(work.live(func() string { return work.status })))
	footerChildren = append(footerChildren, sizePicker)
	footerChildren = append(footerChildren, work.themePicker())
	footerChildren = append(footerChildren, work.action("Render log", false, idle, func() {
		text := work.log.read()
		if text == "" {
			text = "No preview has run in this window yet."
		}
		work.message("Render log", text)
	}))
	footer := primitives.HBox(footerChildren...).PaddingXY(20, 10).Gap(12).Background(appearance.Panel)
	return primitives.VBox(top, primitives.Expanded(content), footer).Gap(1).Background(appearance.Ground)
}

// schemaName identifies the actual local checker without claiming a legacy schema.
func (work *Workspace) schemaName() string {
	name, err := work.state.document.Value("$schema")
	if err == nil {
		return name
	}
	if work.state.document.Kind() == "legacy-set" {
		return "Legacy scene: YAML syntax checks; Slayer checks scene semantics"
	}
	if work.state.document.Kind() == "catalogue" {
		return "Catalogue plan: cmd/catalogue/plan.schema.json at 0b0d207e. Choose a specimen above the editor; the catalogue renderer checks camera and allocation limits."
	}
	return "https://github.com/janearc/slayer/schemas/job-v1.json"
}
