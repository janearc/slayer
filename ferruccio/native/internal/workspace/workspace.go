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
	"sync/atomic"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/dialog"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/core/splitview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/theme/devtools"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/editor"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
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
	// PreferQueueClient allows the bundled client to defer to a queue's own
	// installed client. An explicitly selected client is never replaced.
	PreferQueueClient bool
	// PrepareQueue initializes the selected queue when the app owns its setup.
	PrepareQueue func(context.Context, string, string) error
	// Settings is where the person's settings are read and saved.
	Settings SettingsSource
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
	queueProgress                       preview.Progress
	queueSequence                       uint64
	queueObserved                       bool
	acceptedGeneration                  uint64
	acceptedText, acceptedPath          string
	cancel                              context.CancelFunc
	cancelRequested                     bool
	fullCancellationNote                string
	fullRunning                         bool
	unresolvedJob                       uint64
	unresolvedPlan                      *preview.Plan
	resolving                           bool
	lastResolve                         time.Time
	resolutions                         chan resolution
	exampleFolder, reducedNote          string
	renderBlocked                       bool
	log                                 *boundedLog
	closing, closeHook                  bool
	lastCheck                           time.Time
	needsCheck                          bool
	openCheck                           *draft.Document
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
	// settings says where settings are read and saved. wantSettings asks the
	// next tick to open the sheet, from any thread; sheet is the one open.
	settings               SettingsSource
	wantSettings           atomic.Bool
	wantQuit               atomic.Bool
	wantOpen               atomic.Bool
	wantSave               atomic.Bool
	protecting             bool
	settingsOpen           bool
	renderSetupOpen        bool
	renderSetupLoading     bool
	renderSetupError       string
	renderSubmitError      string
	renderSetupPath        string
	renderSetupSource      string
	renderSetupFolder      string
	renderCameraSourcePath string
	renderCameraSourceHash [32]byte
	renderCameraError      string
	renderSetupChoice      int
	renderCameraChoice     int
	renderBackendChoice    int
	renderSamplesChoice    int
	renderCameras          []string
	renderPresets          []stagewire.FilmPreset
	renderPresetResults    chan presetResult
	renderPresetCancel     context.CancelFunc
	renderFilm             preview.FullFilm
	renderFilmError        error
	// drawn says what drew the most recent Export.
	drawn Drawn
}

type renderResult struct {
	full        bool
	observation preview.Update
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
	work := &Workspace{host: host, ui: ui, options: options, settings: options.Settings, size: 16, theme: appearance.Theme(), results: make(chan renderResult, 1), resolutions: make(chan resolution, 1), progress: make(chan renderProgress, 32), log: &boundedLog{}}
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
			example = playgroundExample
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
		if work.viewer != nil && work.viewer.dragging {
			work.cancelViewportGesture()
		}
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
	work.viewer.settingsHandler = work.requestSettings
	work.viewer.refuseCamera = work.refuseCameraGesture
	work.editor.SetSettingsHandler(work.requestSettings)
	work.viewer.SetNavigateHandler(func(yaw, pitch, zoom float64) {
		work.problem(work.Navigate(yaw, pitch, zoom))
	})
	work.viewer.SetUndoHandler(func(ctx widget.Context, redo bool) {
		work.undoCamera(ctx, redo)
	})
	if work.composition == "play" {
		work.setupPlay()
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
	work.noteMissingCache()
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
			work.requestClose()
			return false
		})
	}
	if work.wantQuit.Swap(false) && !work.closing {
		work.requestClose()
	}
	if work.needsCheck && time.Since(work.lastCheck) > 250*time.Millisecond {
		work.needsCheck = false
		work.validate()
		work.invalidate()
	}
	work.openSettingsIfAsked()
	work.drainRenderPresets()
	if work.wantOpen.Swap(false) {
		work.open()
	}
	if work.wantSave.Swap(false) {
		work.save(false)
	}
	work.drainResolution()
	work.recheckUnresolved()
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

// requestClose uses the same draft protection and cleanup for the window
// close control and the application menu's Quit command.
func (work *Workspace) requestClose() {
	work.protect(func() {
		work.clearNavigation()
		work.closeStaged()
		work.closeRenderSetup()
		if work.cancel != nil {
			work.closing = true
			work.cancel()
			work.status = "Stopping the preview before closing…"
			work.invalidate()
		} else {
			work.closing = true
			work.releaseExample()
			work.host.Quit()
		}
	})
}

// StartPreview freezes current text and invokes the same tested Slayer adapter.
func (work *Workspace) StartPreview() error { return work.startRender(false) }

// startRender shares cancellation, frozen input and acceptance across both modes.
func (work *Workspace) startRender(full bool) error {
	return work.startRenderWithPreset(full, nil)
}

// startRenderWithPreset freezes an optional size, camera and backend choice.
func (work *Workspace) startRenderWithPreset(full bool, preset *stagewire.FilmPreset) error {
	work.autoPreview = false
	if work.cancel != nil {
		return fmt.Errorf("a preview is already running")
	}
	if err := work.refuseWhileUnresolved(); err != nil {
		return err
	}
	request := work.previewRequest(time.Now())
	if full {
		request.Profile = work.fullRenderProfile(preset)
		request.OutputRoot = work.fullRenderDestination()
		request.DirectoryPrefix = strings.Replace(request.DirectoryPrefix, "-preview-", "-full-", 1)
	}
	tools := work.renderTools()
	if !work.rendererConfiguredAt(tools, request.OutputRoot) {
		return fmt.Errorf("configure the preview queue, renderer and shared render destination before drawing")
	}
	if err := work.refuseWithoutCache(tools); err != nil {
		return err
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
	work.fullRunning = full
	work.diagnostic = ""
	work.log.reset()
	work.queueState, work.queueHeld = "", ""
	work.queueProgress = preview.Progress{}
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
	go func() {
		var started time.Time
		if measured {
			started = time.Now()
		}
		var observation preview.Update
		err := runPreparedPreview(ctx, tools, plan, work.log, work.options.PrepareQueue, func(update preview.Update) {
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
		closeErr := plan.Close()
		cancel()
		if err == nil {
			err = closeErr
		}
		work.results <- renderResult{full: full, plan: plan, err: err, generation: generation, observation: observation}
		work.host.RequestRedraw()
	}()
	return nil
}

// runPreparedPreview provisions the app's own queue before its first submission.
func runPreparedPreview(ctx context.Context, tools preview.Tools, plan *preview.Plan, log io.Writer, prepare func(context.Context, string, string) error, observe func(preview.Update)) error {
	if prepare != nil {
		if err := prepare(ctx, tools.Azazel, tools.Queue); err != nil {
			return err
		}
	}
	return preview.RunObserved(ctx, tools, plan, log, observe)
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
	ctx := work.ui.Window().Context()
	focused, previousReturn := ctx.FocusedWidget(), work.play.sourceReturn
	previousPlacementSearch := work.play.placementSearchField
	work.play.sourceReturn = nil
	work.play.placementSearchField = nil
	work.editor.SetFontSize(work.size)
	work.viewer.SetFontSize(work.size)
	work.theme = appearance.Theme()
	work.ui.SetTheme(appearance.ThemeForSize(work.size))
	work.root = newShortcutRoot(work.layout(), work.requestOpen, work.requestSave)
	work.ui.SetRoot(work.root)
	if previousPlacementSearch != nil && focused == previousPlacementSearch {
		if work.play.placementSearchField != nil {
			ctx.RequestFocus(work.play.placementSearchField)
		} else {
			ctx.ReleaseFocus(focused)
		}
	}
	if previousReturn != nil && focused == previousReturn {
		if work.play.sourceReturn != nil {
			ctx.RequestFocus(work.play.sourceReturn)
		} else {
			ctx.ReleaseFocus(focused)
		}
	}
	if previousReturn != nil && focused == work.editor && work.play.sourceReturn == nil &&
		(work.composition == "play" || work.settingsOpen || work.renderSetupOpen) {
		ctx.ReleaseFocus(focused)
	}
	work.invalidate()
}

// validate keeps invalid drafts editable and reports the schema's real error.
func (work *Workspace) validate() {
	work.openCheck = nil
	err := work.state.document.Check()
	work.applyOpenCheck(err)
}

// applyOpenCheck reports a schema check without reading the document again.
func (work *Workspace) applyOpenCheck(err error) {
	if err != nil {
		work.status = "Draft needs attention"
		work.diagnostic = err.Error()
	} else {
		work.status = "YAML checked · " + work.state.document.Kind()
		work.diagnostic = ""
	}
	work.refreshRenderBlocked(err == nil)
}

// refreshRenderBlocked recomputes whether Render preview is unavailable for
// reasons other than a render being under way. It runs when the source, the
// path or the settings change, so painting never checks a schema or reads the
// settings file.
func (work *Workspace) refreshRenderBlocked(valid bool) {
	work.renderBlocked = work.state.path == "" || !work.rendererConfigured() || !valid
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
	if work.protecting {
		return
	}
	if !work.state.document.Dirty() {
		action()
		return
	}
	prompt := dialog.New(dialog.Title("Keep your changes?"), dialog.Content(work.text("Save this draft before continuing. Unfinished YAML can be saved.")), dialog.Actions(
		dialog.Action{Label: "Cancel", Variant: dialog.VariantOutlined},
		dialog.Action{Label: "Discard", Variant: dialog.VariantOutlined, OnClick: action},
		dialog.Action{Label: "Save", OnClick: func() {
			if work.save(false) {
				action()
			}
		}}), dialog.OnClose(func() { work.protecting = false }), dialog.PainterOpt(appearance.DialogPainter{Theme: work.theme, Size: work.size}))
	prompt.Show(work.ui.Window().Context())
	work.protecting = prompt.IsOpen()
	work.host.RequestRedraw()
}

// open chooses an existing file through the native macOS picker.
func (work *Workspace) open() {
	if work.refuseWhileFull("opening another file") {
		return
	}
	work.protect(func() {
		paths, err := work.host.ShowOpenFileDialog(gogpu.FileDialogOptions{Title: "Open a Slayer YAML file"})
		if err != nil {
			work.problem(err)
			return
		}
		if len(paths) == 0 {
			return
		}
		work.openPath(paths[0])
	})
}

// openReferencedSet takes a job's scene into the normal protected editor.
func (work *Workspace) openReferencedSet() {
	if work.refuseWhileFull("opening another file") {
		return
	}
	if work.state.document.ReferencedSet() == "" {
		return
	}
	work.protect(func() {
		reference := work.state.document.ReferencedSet()
		if reference == "" {
			return
		}
		if !filepath.IsAbs(reference) {
			reference = filepath.Join(filepath.Dir(work.state.path), reference)
		}
		cameraName, zoom, cameraPath := work.state.document.ReferencedCamera()
		if cameraPath {
			recipe, at, err := inheritedStillCameraPath(work.state.path, work.state.document.Text())
			if err != nil {
				work.problem(fmt.Errorf("%w. Render the job to keep its framing, or open its scene directly to edit it", err))
				return
			}
			work.openPathWithCamera(reference, "", 0, recipe, at)
			return
		}
		cameraWide := 0.0
		if zoom {
			var err error
			cameraWide, err = inheritedStillWide(work.state.path, work.state.document.Text(), cameraName)
			if err != nil {
				work.problem(err)
				return
			}
		}
		work.openPathWithCamera(reference, cameraName, cameraWide, "", 0)
	})
}

// openPath applies the same document and view reset for either open route.
func (work *Workspace) openPath(path string) {
	work.openPathWithCamera(path, "", 0, "", 0)
}

func (work *Workspace) openPathWithCamera(path, cameraName string, cameraWide float64, pathRecipe string, stillAt time.Duration) {
	next := work.state
	if err := next.load(path); err != nil {
		work.problem(err)
		return
	}
	if pathRecipe != "" {
		if _, err := preview.InspectFull(path, next.document.Text(), preview.Profile{Full: true, CameraPathRecipe: pathRecipe}); err != nil {
			work.problem(fmt.Errorf("%w. Render the job to keep its framing, or open its scene directly to edit it", err))
			return
		}
	}
	if cameraWide != 0 {
		if _, physical := next.document.PhysicalFrame(); physical {
			work.problem(fmt.Errorf("a physical camera rig cannot use an inherited job zoom"))
			return
		}
	}
	work.state = next
	work.needsCheck = false
	work.resetDocumentView()
	work.working.cameraName, work.working.cameraWide, work.working.cameraPath = cameraName, cameraWide, pathRecipe
	work.working.cameraPathAt = stillAt
	work.releaseExample()
	work.exampleDraft = false
	work.study = ""
	work.selectSpecimen()
	work.editor.SetText(work.state.document.Text())
	work.viewer.set(nil)
	work.output = ""
	work.caption = "Render this file to see its preview"
	if work.composition == "play" {
		work.play.selected = ""
	}
	if work.composition == "play" && work.state.document.Kind() == "set" && work.stagedBuilder() != nil {
		work.openCheck = work.state.document
		work.renderBlocked = true
		work.status = "Checking YAML…"
	} else {
		work.validate()
	}
	work.rebuild()
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
	return work.saveAt(path)
}

// saveAt writes the draft to the chosen path and updates the workspace.
func (work *Workspace) saveAt(path string) bool {
	var notice string
	var err error
	if work.study == larsonPlaygroundStudy.id || (work.study == "" && attributedVisitor(work.state.document)) {
		notice, err = work.savePlayground(path)
	} else if work.study != "" {
		err = saveBundledStudyDocument(work.state.document, path, filepath.Dir(work.state.path), work.study)
	} else if work.state.document.Kind() == "job" {
		err = work.saveJob(path)
	} else {
		notice, err = work.saveDraft(path)
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
	work.refreshRenderBlocked(work.state.document.Check() == nil)
	work.exampleDraft = false
	work.status = "Saved · " + filepath.Base(path)
	if notice != "" {
		work.status = "Draft saved with unchecked dependencies"
		work.diagnostic = notice
	}
	work.invalidate()
	return true
}

// example copies a bundled example so app updates cannot overwrite user edits.
func (work *Workspace) example(name string) error {
	var path, study string
	var folder string
	if _, isStudy := studyForExample(name); isStudy {
		var err error
		path, study, err = copyBundledStudy(work.examplesFolder(), name)
		if err != nil {
			return err
		}
		folder = filepath.Dir(path)
	} else {
		clean := filepath.Clean(name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("example path escapes its bundled folder")
		}
		examples, err := filepath.Abs(work.examplesFolder())
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
		source, err = builtinExampleProfiles(examples, path, source)
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
	work.releaseExample()
	work.exampleFolder = folder
	work.exampleDraft = true
	work.study = study
	work.selectSpecimen()
	return nil
}

// releaseExample removes the temporary folder holding the open example's
// copy. It runs when the document closes: another document replaces it, or
// the window quits.
func (work *Workspace) releaseExample() {
	if work.exampleFolder == "" {
		return
	}
	work.closeStaged()
	_ = os.RemoveAll(work.exampleFolder)
	work.exampleFolder = ""
}

// chooseExample protects the draft before opening an editable bundled example.
func (work *Workspace) chooseExample(name string) {
	work.chooseCatalogueExample(name, "")
}

// chooseCatalogueExample declares the identity of a bundled study explicitly.
func (work *Workspace) chooseCatalogueExample(name, specimen string) {
	if work.refuseWhileFull("choosing an example") {
		return
	}
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
			work.play.selected = ""
		}
		work.validate()
		work.rebuild()
	})
}

// reference displays credited mission imagery without pretending it is a render.
func (work *Workspace) reference(place string) {
	if work.refuseWhileFull("choosing a reference image") {
		return
	}
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
	work.cancelViewportGesture()
	work.viewer.SetNavigable(false)
	work.output = ""
	work.play.renderedText = ""
	work.working.photograph = picture
	work.working.displayed = false
	work.working.dirty = true
	work.viewer.set(picture)
	work.caption = credit
	if work.root != nil {
		work.refreshLayout = true
	}
	work.invalidate()
}

// showImage decodes a real image, including a renderer's PNG output. A
// picture over the viewer's limit is shown reduced and reducedNote says so.
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
	if configuration.Width <= 0 || configuration.Height <= 0 {
		return fmt.Errorf("image has no pixels")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	picture, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	picture, work.reducedNote = reduceForViewer(picture, viewerPixelLimit)
	work.cancelViewportGesture()
	work.play.renderedText = ""
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

const (
	// brandSizeRatio sizes the window's name as a multiple of the chosen text size.
	brandSizeRatio = 1.875
	// headingSizeRatio sizes a heading as a multiple of the chosen text size.
	headingSizeRatio = 1.75
)

// layout gives the real image most of the window and keeps YAML always visible.
func (work *Workspace) layout() widget.Widget {
	if work.settingsOpen {
		return work.settingsSheet()
	}
	if work.renderSetupOpen {
		return work.renderSetupSheet()
	}
	if work.composition == "play" {
		return work.playLayout()
	}
	idle := func() bool { return false }
	busy := func() bool { return work.cancel != nil }
	controls := []widget.Widget{primitives.Text("ferruccio").FontSize(work.size * brandSizeRatio).Color(appearance.Pink)}
	if work.state.document.Kind() != "catalogue" {
		controls = append(controls, work.action("Play", false, func() bool { return work.fullRunning }, work.enterPlay))
	}
	controls = append(controls,
		work.action("Observe", work.composition == "observe", idle, func() { work.composition = "observe"; work.rebuild() }),
		work.action("Compose", work.composition == "compose", idle, func() { work.composition = "compose"; work.rebuild() }),
		primitives.Expanded(primitives.Box()),
		work.action("Open…", false, idle, work.open), work.action("Save", false, idle, func() { work.save(false) }), work.action("Save as…", false, idle, func() { work.save(true) }),
		work.action("Render preview", true, func() bool {
			return busy() || work.renderBlocked
		}, func() { work.problem(work.StartPreview()) }),
		work.action("Stop", false, func() bool { return !work.Running() }, func() {
			work.cancelRender()
			work.invalidate()
		}),
	)
	top := primitives.HBox(controls...).Gap(10).PaddingXY(20, 14).Background(appearance.Panel)
	navigation := primitives.VBox(work.text("Studies").Color(appearance.Cyan),
		work.action("Larson + visitor", false, idle, func() { work.chooseExample(playgroundExample) }),
		work.action("Subaru", false, idle, func() { work.chooseCatalogueExample("subaru-settings.yaml", "subaru") }),
		work.action("First picture", false, idle, func() { work.chooseExample("first-picture.yaml") }), work.action("Webb at L2", false, idle, func() { work.chooseExample("webb-at-l2.yaml") }), work.action("Fizz", false, idle, func() { work.chooseExample("fizz.yaml") }),
		primitives.Expanded(primitives.Box()), work.text("World references").Color(appearance.Cyan),
		work.action("Venus", false, idle, func() { work.reference("venus") }), work.action("Titan", false, idle, func() { work.reference("titan") }),
	).Width(180).Padding(16).Gap(14).Background(appearance.Panel)
	picture := primitives.VBox(work.live(func() string {
		return strings.TrimSuffix(work.state.document.Name(), filepath.Ext(work.state.document.Name()))
	}).FontSize(work.size*headingSizeRatio), primitives.Expanded(work.viewer), work.imageCaption(), work.action("Open image…", false, idle, func() {
		if work.refuseWhileFull("opening an image") {
			return
		}
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
			work.caption = "Image · " + filepath.Base(paths[0]) + work.reducedNote
			work.refreshLayout = true
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
			primitives.Expanded(picture), work.compositionActions(),
		).Gap(8)))
	}
	var footerChildren []widget.Widget
	if bar := work.renderProgressBar(); bar != nil {
		footerChildren = append(footerChildren, bar)
	} else if work.cancel != nil {
		footerChildren = append(footerChildren, activeRenderMarker())
	}
	footerChildren = append(footerChildren, primitives.Expanded(work.statusLine(func() string { return work.status })))
	footerChildren = append(footerChildren, sizePicker)
	footerChildren = append(footerChildren, work.themePicker())
	footerChildren = append(footerChildren, work.action("Render log", false, idle, work.showRenderLog))
	footer := primitives.HBox(footerChildren...).PaddingXY(20, 10).Gap(12).Background(appearance.Panel)
	return primitives.VBox(top, primitives.Expanded(content), footer).Gap(1).Background(appearance.Ground)
}

// showRenderLog includes the complete footer status along with producer output.
func (work *Workspace) showRenderLog() {
	text := work.log.read()
	if text == "" {
		text = "No preview has run in this window yet."
	}
	if work.status != "" {
		text = work.status + "\n\n" + text
	}
	work.message("Render log", text)
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
