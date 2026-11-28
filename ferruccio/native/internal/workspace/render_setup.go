package workspace

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

var renderBackends = []string{"CPU", "GPU · Software traversal", "GPU · Native mesh traversal"}

type presetResult struct {
	presets    []stagewire.FilmPreset
	err        error
	film       preview.FullFilm
	filmErr    error
	cameras    []string
	cameraPath string
	cameraHash [32]byte
	cameraErr  error
}

// openRenderSetup reads producer choices in a managed child, off the UI
// thread. A missing helper leaves the authored-size option available.
func (work *Workspace) openRenderSetup() {
	if work.renderSetupOpen {
		return
	}
	work.renderSetupOpen, work.renderSetupLoading = true, true
	work.renderSetupChoice, work.renderSetupError = 0, ""
	work.renderSubmitError = ""
	work.renderSetupFolder = ""
	work.renderCameraChoice = 0
	work.renderSamplesChoice = 0
	work.renderBackendChoice = 0
	if work.ownRender(preview.Profile{}).GPU {
		work.renderBackendChoice = 1
	}
	work.renderCameras = work.state.document.Cameras()
	work.renderCameraSourcePath, work.renderCameraSourceHash, work.renderCameraError = "", [32]byte{}, ""
	reference := work.state.document.ReferencedSet()
	work.renderPresets = nil
	path, source := work.state.path, work.state.document.Text()
	work.renderSetupPath, work.renderSetupSource = path, source
	results := make(chan presetResult, 1)
	work.renderPresetResults = results
	ctx, cancel := context.WithCancel(context.Background())
	work.renderPresetCancel = cancel
	go func() {
		cameras, cameraPath, cameraHash, cameraErr := referencedCameras(path, reference)
		film, filmErr := preview.ReadFullFilm(path, source)
		executable, err := os.Executable()
		var presets []stagewire.FilmPreset
		if err == nil {
			presets, err = stagewire.ReadFilmPresets(ctx, filepath.Join(filepath.Dir(executable), "ferruccio-stage"))
		}
		results <- presetResult{presets: presets, err: err, film: film, filmErr: filmErr, cameras: cameras,
			cameraPath: cameraPath, cameraHash: cameraHash, cameraErr: cameraErr}
		if work.host != nil {
			work.host.RequestRedraw()
		}
	}()
	work.refreshLayout = true
	work.invalidate()
}

// closeRenderSetup cancels any pending catalogue read and dismisses the sheet.
func (work *Workspace) closeRenderSetup() {
	if work.renderPresetCancel != nil {
		work.renderPresetCancel()
		work.renderPresetCancel = nil
	}
	work.renderPresetResults = nil
	work.renderSetupFolder = ""
	if work.renderSetupOpen {
		work.renderSetupOpen = false
		work.refreshLayout = true
		work.invalidate()
	}
}

// drainRenderPresets applies a completed catalogue read on the UI thread.
func (work *Workspace) drainRenderPresets() {
	if work.renderPresetResults == nil {
		return
	}
	select {
	case result := <-work.renderPresetResults:
		work.renderPresetResults = nil
		work.renderPresetCancel = nil
		work.renderSetupLoading = false
		if result.cameraPath != "" {
			work.renderCameras = result.cameras
			work.renderCameraSourcePath = result.cameraPath
			work.renderCameraSourceHash = result.cameraHash
		}
		if result.cameraErr != nil {
			work.renderCameraError = "Referenced set cameras unavailable: " + result.cameraErr.Error()
		}
		work.renderFilm, work.renderFilmError = result.film, result.filmErr
		if result.err != nil {
			work.renderSetupError = "Output presets unavailable: " + result.err.Error()
		} else {
			work.renderPresets = result.presets
		}
		work.refreshLayout = true
		work.invalidate()
	default:
	}
}

// renderSetupSheet presents the frozen still's available choices and cost.
func (work *Workspace) renderSetupSheet() widget.Widget {
	labels := []string{"Use scene's authored size"}
	for _, preset := range work.renderPresets {
		labels = append(labels, fmt.Sprintf("%s · %d × %d", preset.Name, preset.Width, preset.Height))
	}
	choice := dropdown.New(dropdown.Items(labels...), dropdown.Selected(work.renderSetupChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		work.renderSetupChoice = index
		work.refreshRenderChoices()
	}))
	choice.SetID("Output size")
	cameraLabel := "Use scene's camera"
	if work.working.cameraName != "" {
		cameraLabel = fmt.Sprintf("Use job camera %q", work.working.cameraName)
	}
	if work.working.cameraPath != "" {
		cameraLabel = "Use job camera path"
	}
	if work.working.cameraWide != 0 {
		cameraLabel += fmt.Sprintf(" · %g°", work.working.cameraWide)
	}
	cameraLabels := append([]string{cameraLabel}, work.renderCameras...)
	camera := dropdown.New(dropdown.Items(cameraLabels...), dropdown.Selected(work.renderCameraChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		work.renderCameraChoice = index
		work.refreshRenderChoices()
	}))
	camera.SetID("Render camera")
	backend := dropdown.New(dropdown.Items(renderBackends...), dropdown.Selected(work.renderBackendChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		work.renderBackendChoice = index
		work.refreshRenderChoices()
	}))
	backend.SetID("Render backend")
	samplingLabels := []string{"Use scene's sampling"}
	if work.renderFilm.HasPhysicalRig() {
		samplingLabels[0] = "Use the physical rig's ray grid"
	} else {
		for count := 1; count <= 64; count++ {
			samplingLabels = append(samplingLabels, sampleCountLabel(fmt.Sprint(count)))
		}
	}
	sampling := dropdown.New(dropdown.Items(samplingLabels...), dropdown.Selected(work.renderSamplesChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		work.renderSamplesChoice = index
		work.refreshRenderChoices()
	}))
	sampling.SetID("Samples per pixel")
	note := work.renderEstimate()
	if work.renderSetupLoading {
		note = "Loading output sizes…"
	} else if work.renderSetupError != "" {
		note = work.renderSetupError
	}
	if work.renderCameraError != "" {
		note += "  " + work.renderCameraError
	}
	if work.renderSubmitError != "" {
		note = work.renderSubmitError
	}
	render := work.action("Render", true, func() bool { return work.Running() || work.renderSetupLoading }, func() {
		work.renderSubmitError = ""
		if work.renderSetupPath != "" && (work.state.path != work.renderSetupPath || work.state.document.Text() != work.renderSetupSource) {
			work.renderSetupError = "The scene changed; close and reopen render setup to refresh its choices."
			work.refreshLayout = true
			work.invalidate()
			return
		}
		if work.renderCameraSourcePath != "" {
			current, err := os.ReadFile(work.renderCameraSourcePath)
			if err != nil || sha256.Sum256(current) != work.renderCameraSourceHash {
				work.renderSetupError = "The referenced set changed; close and reopen render setup to refresh its choices."
				work.refreshLayout = true
				work.invalidate()
				return
			}
		}
		var preset *stagewire.FilmPreset
		if work.renderSetupChoice > 0 && work.renderSetupChoice <= len(work.renderPresets) {
			preset = &work.renderPresets[work.renderSetupChoice-1]
		}
		if err := work.startRenderWithPreset(true, preset); err != nil {
			work.renderSubmitError = err.Error()
			work.refreshLayout = true
			work.invalidate()
			return
		}
		work.closeRenderSetup()
	})
	buttons := primitives.HBox(primitives.Expanded(primitives.Box()), work.action("Cancel", false, func() bool { return false }, work.closeRenderSetup), render).Gap(12)
	fields := primitives.VBox(
		work.text("Output size").Color(appearance.Cyan), choice,
		work.text("Camera").Color(appearance.Cyan), camera,
		work.text("Samples per pixel").Color(appearance.Cyan), sampling,
		work.text("Backend").Color(appearance.Cyan), backend,
	).Gap(8).Width(settingsSheetWidth - 2*settingsSheetPadding).CrossAlign(primitives.CrossAxisStretch)
	panel := primitives.VBox(
		work.text("Render setup").FontSize(work.size+settingsTitleExtra).Color(appearance.Pink),
		fields,
		work.renderFolderRow(),
		work.renderSetupNote(note),
		buttons,
	).Gap(12).Padding(settingsSheetPadding).Width(settingsSheetWidth).Background(appearance.Panel)
	page := primitives.HBox(primitives.Expanded(primitives.Box()), panel, primitives.Expanded(primitives.Box())).CrossAlign(primitives.CrossAxisStretch).Background(appearance.Ground)
	return newSheet(page, work.closeRenderSetup)
}

// renderSetupNote keeps long diagnostics accessible without pushing actions away.
func (work *Workspace) renderSetupNote(note string) widget.Widget {
	lines := wrapMessageLines(note, work.runesIn(settingsSheetWidth-2*settingsSheetPadding))
	if len(lines) <= 3 {
		return work.paragraph(note, appearance.Ink)
	}
	details := work.action("Render details…", false, func() bool { return false }, func() {
		work.message("Render setup details", note)
	})
	width := float32(settingsSheetWidth - 2*settingsSheetPadding)
	limit := max(1, int(width/(max(work.size, 16)*messageGlyphWidthRatio))-1)
	summary := wrapMessageLines(note, limit)[0] + "…"
	return primitives.VBox(work.text(summary), primitives.HBox(details)).Gap(4)
}

// renderFolderRow shows where this still's image and receipt will be saved.
func (work *Workspace) renderFolderRow() widget.Widget {
	folder := work.fullRenderDestination()
	if folder == "" {
		folder = "No output folder configured"
	}
	room := float32(settingsSheetWidth-2*settingsSheetPadding) - settingsLabelWidth - settingsControlWidth
	path := work.text(shorten(folder, work.runesIn(room))).MaxLines(1)
	pick := work.action("Choose…", false, func() bool { return false }, work.chooseRenderFolder)
	pick.SetID("Choose render destination")
	return primitives.HBox(work.text("Destination").Color(appearance.Cyan),
		primitives.Expanded(path), pick).Gap(12)
}

// chooseRenderFolder keeps the selection local to this render setup.
func (work *Workspace) chooseRenderFolder() {
	options := gogpu.FileDialogOptions{
		Title: "Choose render destination", Directory: true,
		InitialDirectory: work.fullRenderDestination(),
	}
	paths, err := work.host.ShowOpenFileDialog(options)
	if err != nil {
		work.message("Could not choose a render destination", err.Error())
		return
	}
	if len(paths) == 0 {
		return
	}
	work.setRenderFolder(paths[0])
}

func (work *Workspace) setRenderFolder(folder string) {
	work.renderSetupFolder = folder
	work.refreshRenderChoices()
}

func (work *Workspace) refreshRenderChoices() {
	work.renderSubmitError = ""
	work.refreshLayout = true
	work.invalidate()
}

// fullRenderDestination uses the sheet's choice only while it is open.
func (work *Workspace) fullRenderDestination() string {
	if work.renderSetupOpen && work.renderSetupFolder != "" {
		return work.renderSetupFolder
	}
	return work.evidenceFolder()
}

// renderEstimate shows the spatial count and total across shutter glimpses.
func (work *Workspace) renderEstimate() string {
	if work.renderFilmError != nil {
		return work.renderFilmError.Error()
	}
	var preset *stagewire.FilmPreset
	if work.renderSetupChoice > 0 && work.renderSetupChoice <= len(work.renderPresets) {
		preset = &work.renderPresets[work.renderSetupChoice-1]
	}
	profile, err := work.renderFilm.Resolve(work.fullRenderProfile(preset))
	if err != nil {
		return err.Error()
	}
	renderer := renderBackends[0]
	if profile.GPU {
		renderer = renderBackends[1]
	}
	if profile.NativeGPU {
		renderer = renderBackends[2]
	}
	total := groupedNumber(profile.PrimarySamples().String())
	width, height := profile.OutputSize()
	size := fmt.Sprintf("%d × %d", width, height)
	if width != profile.Width || height != profile.Height {
		size += fmt.Sprintf(" region of %d × %d", profile.Width, profile.Height)
	}
	estimate := fmt.Sprintf("%s · %s · %s primary samples · %s",
		size, renderSamplingLabel(profile), total, renderer)
	if profile.CameraWide != 0 {
		estimate += fmt.Sprintf(" · %s · %g° horizontal", profile.Camera, profile.CameraWide)
	}
	return estimate + "."
}

// fullRenderProfile collects the choices shared by the estimate and submission.
func (work *Workspace) fullRenderProfile(preset *stagewire.FilmPreset) preview.Profile {
	profile := work.ownRender(preview.Profile{Full: true, Camera: work.working.cameraName, CameraWide: work.working.cameraWide, CameraPathRecipe: work.working.cameraPath})
	if work.renderSetupOpen {
		profile.CPURenderer = work.renderBackendChoice == 0
		profile.GPU = work.renderBackendChoice == 1
		profile.NativeGPU = work.renderBackendChoice == 2
		profile.Samples = work.renderSamplesChoice
		if work.renderCameraChoice > 0 && work.renderCameraChoice <= len(work.renderCameras) {
			profile.Camera = work.renderCameras[work.renderCameraChoice-1]
			profile.CameraWide = 0
			profile.CameraPathRecipe = ""
			profile.StillAt = work.working.cameraPathAt
		}
	}
	if preset != nil {
		profile.SizePreset = preset.Name
		profile.Width, profile.Height = preset.Width, preset.Height
	}
	return profile
}

// renderSamplingLabel describes spatial sampling and its shutter timing.
func renderSamplingLabel(profile preview.Profile) string {
	samples := sampleCountLabel(profile.SamplesPerPixel().String())
	if profile.Glimpses() > 1 {
		samples += fmt.Sprintf(" per glimpse · %d shutter glimpses", profile.Glimpses())
	}
	return samples
}

// sampleCountLabel names a spatial sampling count.
func sampleCountLabel(digits string) string {
	if digits == "1" {
		return "1 sample per pixel"
	}
	return groupedNumber(digits) + " samples per pixel"
}

// groupedNumber makes a nonnegative whole sample count easier to read.
func groupedNumber(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	first := len(digits) % 3
	if first == 0 {
		first = 3
	}
	result := digits[:first]
	for at := first; at < len(digits); at += 3 {
		result += "," + digits[at:at+3]
	}
	return result
}

// referencedCameras reads job camera choices without mutating either file.
func referencedCameras(jobPath, reference string) ([]string, string, [32]byte, error) {
	var empty [32]byte
	if reference == "" {
		return nil, "", empty, nil
	}
	path := reference
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(jobPath), path)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, "", empty, err
	}
	document, err := draft.Open(path, source)
	if err != nil {
		return nil, "", empty, err
	}
	if _, err := document.Tree(); err != nil {
		return nil, "", empty, err
	}
	if document.Kind() != "set" && document.Kind() != "legacy-set" {
		return nil, "", empty, fmt.Errorf("%s is not a scene set", path)
	}
	return document.Cameras(), path, sha256.Sum256(source), nil
}
