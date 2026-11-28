package workspace

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

func TestRenderSetupControlsFitSmallestWindow(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 22, 24} {
		work := settingsWorkspace(t, "")
		work.size, work.theme = size, appearance.Theme()
		work.renderSubmitError = strings.Repeat("could not create the render folder ", 100)
		ctx := uitest.NewMockContext()
		ctx.WindowSizeVal = smallestWindow
		root := work.renderSetupSheet()
		root.Layout(ctx, geometry.Loose(smallestWindow))
		for _, id := range []string{"Output size", "Render camera", "Samples per pixel", "Render backend", "Choose render destination", "Render details…", "Cancel", "Render"} {
			point, found := centreOf(root, id, geometry.Point{})
			if !found || point.X < 0 || point.Y < 0 || point.X > smallestWindow.Width || point.Y > smallestWindow.Height {
				t.Fatalf("size %v: %s found %v at %v", size, id, found, point)
			}
		}
		for _, id := range []string{"Output size", "Render backend"} {
			control := libraryWidget(root, id).(interface{ Bounds() geometry.Rect })
			if control.Bounds().Width() < 440 {
				t.Fatalf("size %v: %s has only %v points for its label", size, id, control.Bounds().Width())
			}
		}
	}
}

// TestRenderSetupOffscreenEvidence exports the actual sheet widget tree when
// a destination and a built helper are supplied. It starts no render.
func TestRenderSetupOffscreenEvidence(t *testing.T) {
	path, helper := os.Getenv("FERRUCCIO_RENDER_SETUP_IMAGE"), os.Getenv("FERRUCCIO_STAGE_HELPER")
	if path == "" || helper == "" {
		t.Skip("set an evidence image and built helper to export the setup sheet")
	}
	presets, err := stagewire.ReadFilmPresets(context.Background(), helper)
	if err != nil {
		t.Fatal(err)
	}
	examples, err := filepath.Abs("../../../examples")
	if err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{Composition: "play", Examples: examples, SuppressAutomaticPreview: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(work.state.path)) })
	work.ui.Window().HandleResize(1600, 1000)
	work.renderSetupOpen = true
	work.renderPresets = presets
	source, err := os.ReadFile(filepath.Join(examples, "full-render-check.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	work.renderFilm, err = preview.ReadFullFilm("set.yaml", string(source))
	if err != nil {
		t.Fatal(err)
	}
	work.renderCameras = []string{"arrival"}
	for i, preset := range presets {
		if preset.Name == "imax" {
			work.renderSetupChoice = i + 1
		}
	}
	if work.renderSetupChoice == 0 {
		t.Fatal("producer has no imax preset")
	}
	work.rebuild()
	picture, err := work.exportWith(nil)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, picture); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRenderEstimateUsesChosenSizeAndActualLegacySampling(t *testing.T) {
	source, err := os.ReadFile("../../../examples/full-render-check.yaml")
	if err != nil {
		t.Fatal(err)
	}
	film, err := preview.ReadFullFilm("set.yaml", string(source))
	if err != nil {
		t.Fatal(err)
	}
	work := &Workspace{
		renderSetupOpen: true,
		renderFilm:      film,
		renderPresets:   []stagewire.FilmPreset{{Name: "half-portrait", Width: 1080, Height: 960}},
	}
	if got := work.renderEstimate(); !strings.Contains(got, "4 samples per pixel") || !strings.Contains(got, "256,000 primary samples") {
		t.Fatalf("authored estimate %q", got)
	}
	work.renderSetupChoice = 1
	if got := work.renderEstimate(); !strings.Contains(got, "1080 × 960") || !strings.Contains(got, "4,147,200 primary samples") {
		t.Fatalf("preset estimate %q", got)
	}
	work.renderBackendChoice = 1
	if got := work.renderEstimate(); !strings.Contains(got, "GPU") {
		t.Fatalf("GPU estimate %q", got)
	}
	work.renderBackendChoice = 2
	if got := work.renderEstimate(); !strings.Contains(got, "Native mesh traversal") {
		t.Fatalf("native GPU estimate %q", got)
	}
	work.renderSamplesChoice = 7
	if got := work.renderEstimate(); !strings.Contains(got, "Native mesh traversal") ||
		!strings.Contains(got, "7 samples per pixel") || !strings.Contains(got, "7,257,600 primary samples") {
		t.Fatalf("native GPU exact sampling estimate %q", got)
	}
	work.renderBackendChoice = 0
	if got := work.renderEstimate(); !strings.Contains(got, "7 samples per pixel") || !strings.Contains(got, "7,257,600 primary samples") {
		t.Fatalf("exact estimate %q", got)
	}
}

func TestRenderEstimateResolvesPhysicalRowsAndCountsShutterGlimpses(t *testing.T) {
	source := strings.Replace(addRecipeFixture, "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}",
		"film: {width: 640, rays: 2, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}", 1)
	film, err := preview.ReadFullFilm("set.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	work := &Workspace{renderFilm: film, renderSetupOpen: true, renderSetupChoice: 1,
		renderPresets: []stagewire.FilmPreset{{Name: "matching", Width: 24, Height: 15}, {Name: "wrong", Width: 24, Height: 12}}}
	if got := work.renderEstimate(); !strings.Contains(got, "24 × 16 · 4 samples per pixel per glimpse · 2 shutter glimpses · 3,072 primary samples") {
		t.Fatalf("rig estimate %q", got)
	}
	work.renderSetupChoice = 2
	if got := work.renderEstimate(); !strings.Contains(got, "does not match the physical frame") {
		t.Fatalf("incompatible preset estimate %q", got)
	}
}

func TestReferencedJobSetCamerasAndChangedFileRefusal(t *testing.T) {
	dir := t.TempDir()
	setPath := filepath.Join(dir, "set.yaml")
	raw, err := os.ReadFile("../../../examples/full-render-check.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(setPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(dir, "job.yaml")
	cameras, path, digest, err := referencedCameras(jobPath, "set.yaml")
	if err != nil || path != setPath || len(cameras) == 0 || cameras[0] != "arrival" {
		t.Fatalf("cameras %v path %s: %v", cameras, path, err)
	}
	if _, _, _, err := referencedCameras(jobPath, "missing.yaml"); err == nil {
		t.Fatal("missing set silently accepted")
	}
	work := libraryEventWorkspace(t)
	work.renderSetupOpen = true
	work.renderSetupPath, work.renderSetupSource = work.state.path, work.state.document.Text()
	work.renderCameraSourcePath, work.renderCameraSourceHash = path, digest
	work.renderCameras = cameras
	work.rebuild()
	drawLibraryWindow(t, work)
	if err := os.WriteFile(setPath, append(raw, []byte("\n# changed\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	work.renderSubmitError = "Previous destination unavailable"
	control := libraryWidget(work.root, "Render")
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	if work.renderSubmitError != "" || !strings.Contains(work.renderSetupError, "referenced set changed") || work.Running() {
		t.Fatal("stale referenced set was submitted")
	}
}

func TestOpeningJobRenderSetupLoadsReferencedCameras(t *testing.T) {
	dir := t.TempDir()
	setRaw, err := os.ReadFile("../../../examples/full-render-check.yaml")
	if err != nil {
		t.Fatal(err)
	}
	setPath := filepath.Join(dir, "set.yaml")
	if err := os.WriteFile(setPath, setRaw, 0600); err != nil {
		t.Fatal(err)
	}
	jobRaw, err := os.ReadFile("../../../examples/first-picture.yaml")
	if err != nil {
		t.Fatal(err)
	}
	jobRaw = []byte(strings.Replace(string(jobRaw), "scene: {smoke: 1}", "scene: {set: set.yaml}", 1))
	jobPath := filepath.Join(dir, "job.yaml")
	if err := os.WriteFile(jobPath, jobRaw, 0600); err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	if err := work.state.load(jobPath); err != nil {
		t.Fatal(err)
	}
	work.openRenderSetup()
	select {
	case result := <-work.renderPresetResults:
		work.renderPresetResults <- result
		work.drainRenderPresets()
	case <-time.After(5 * time.Second):
		t.Fatal("render setup did not finish loading")
	}
	if len(work.renderCameras) == 0 || work.renderCameras[0] != "arrival" || work.renderCameraSourcePath != setPath || work.renderCameraError != "" {
		t.Fatalf("cameras %v path %s error %s", work.renderCameras, work.renderCameraSourcePath, work.renderCameraError)
	}
	work.closeRenderSetup()
}
