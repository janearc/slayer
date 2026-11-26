package workspace

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

type presetResult struct {
	presets                  []stagewire.FilmPreset
	err                      error
	authored, sampling       preview.Profile
	authoredErr, samplingErr error
	cameras                  []string
	cameraPath               string
	cameraHash               [32]byte
	cameraErr                error
}

// openRenderSetup reads producer choices in a managed child, off the UI
// thread. A missing helper leaves the authored-size option available.
func (work *Workspace) openRenderSetup() {
	if work.renderSetupOpen {
		return
	}
	work.renderSetupOpen, work.renderSetupLoading = true, true
	work.renderSetupChoice, work.renderSetupError = 0, ""
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
		authored, authoredErr := preview.InspectFull(path, source, preview.Profile{})
		sampling, samplingErr := preview.InspectFull(path, source, preview.Profile{Width: 1, Height: 1, SizePreset: "selected"})
		executable, err := os.Executable()
		var presets []stagewire.FilmPreset
		if err == nil {
			presets, err = stagewire.ReadFilmPresets(ctx, filepath.Join(filepath.Dir(executable), "ferruccio-stage"))
		}
		results <- presetResult{presets: presets, err: err, authored: authored, sampling: sampling, authoredErr: authoredErr, samplingErr: samplingErr, cameras: cameras, cameraPath: cameraPath, cameraHash: cameraHash, cameraErr: cameraErr}
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
		work.renderAuthored, work.renderSampling = result.authored, result.sampling
		work.renderAuthoredError, work.renderSamplingError = result.authoredErr, result.samplingErr
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
		work.refreshLayout = true
		work.invalidate()
	}))
	choice.SetID("Output size")
	cameraLabels := append([]string{"Use scene's camera"}, work.renderCameras...)
	camera := dropdown.New(dropdown.Items(cameraLabels...), dropdown.Selected(work.renderCameraChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		work.renderCameraChoice = index
	}))
	camera.SetID("Render camera")
	backend := dropdown.New(dropdown.Items("CPU", "GPU"), dropdown.Selected(work.renderBackendChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		work.renderBackendChoice = index
		work.refreshLayout = true
		work.invalidate()
	}))
	backend.SetID("Render backend")
	samplingLabels := []string{"Use scene's sampling"}
	for count := 1; count <= 64; count++ {
		samplingLabels = append(samplingLabels, fmt.Sprintf("%d samples per pixel", count))
	}
	sampling := dropdown.New(dropdown.Items(samplingLabels...), dropdown.Selected(work.renderSamplesChoice), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		work.renderSamplesChoice = index
		work.refreshLayout = true
		work.invalidate()
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
	render := work.action("Render", true, func() bool { return work.Running() || work.renderSetupLoading }, func() {
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
			work.renderSetupError = err.Error()
			work.refreshLayout = true
			work.invalidate()
			return
		}
		work.closeRenderSetup()
	})
	buttons := primitives.HBox(primitives.Expanded(primitives.Box()), work.action("Cancel", false, func() bool { return false }, work.closeRenderSetup), render).Gap(12)
	panel := primitives.VBox(work.text("Render setup").FontSize(work.size+settingsTitleExtra).Color(appearance.Pink), work.text("Output size").Color(appearance.Cyan), choice, work.text("Camera").Color(appearance.Cyan), camera, work.text("Samples per pixel").Color(appearance.Cyan), sampling, work.text("Backend").Color(appearance.Cyan), backend, work.paragraph(note, appearance.Ink), buttons).Gap(16).Padding(settingsSheetPadding).Width(settingsSheetWidth).Background(appearance.Panel)
	page := primitives.HBox(primitives.Expanded(primitives.Box()), panel, primitives.Expanded(primitives.Box())).CrossAlign(primitives.CrossAxisStretch).Background(appearance.Ground)
	return newSheet(page, work.closeRenderSetup)
}

// renderEstimate counts primary samples, not secondary lighting work.
func (work *Workspace) renderEstimate() string {
	profile, err := work.renderAuthored, work.renderAuthoredError
	if work.renderSetupChoice > 0 && work.renderSetupChoice <= len(work.renderPresets) {
		preset := work.renderPresets[work.renderSetupChoice-1]
		profile, err = work.renderSampling, work.renderSamplingError
		profile.Width, profile.Height = preset.Width, preset.Height
	}
	if work.renderSamplesChoice > 0 {
		profile.Samples = work.renderSamplesChoice
	}
	if err != nil || profile.Width <= 0 || profile.Height <= 0 || (profile.Rays <= 0 && profile.Samples <= 0) {
		return "Sampling follows the scene; the render checks its film settings before submission."
	}
	perPixel := new(big.Int).Mul(big.NewInt(int64(profile.Rays)), big.NewInt(int64(profile.Rays)))
	if profile.Samples > 0 {
		perPixel.SetInt64(int64(profile.Samples))
	}
	total := new(big.Int).Mul(big.NewInt(int64(profile.Width)), big.NewInt(int64(profile.Height)))
	total.Mul(total, perPixel)
	renderer := "CPU"
	if work.renderBackendChoice == 1 {
		renderer = "GPU"
	}
	return fmt.Sprintf("%d × %d · %s samples per pixel · %s primary samples · %s.", profile.Width, profile.Height, groupedNumber(perPixel.String()), groupedNumber(total.String()), renderer)
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
