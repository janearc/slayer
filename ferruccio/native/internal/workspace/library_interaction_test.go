package workspace

import (
	"github.com/gogpu/gg"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScenePickerNamesOpenedFileOutsideLibrary(t *testing.T) {
	work := libraryEventWorkspace(t)
	path := filepath.Join(t.TempDir(), "job.yaml")
	if err := os.WriteFile(path, []byte("scene: {set: example.yaml}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := work.state.load(path); err != nil {
		t.Fatal(err)
	}
	work.exampleDraft = false
	control := work.scenePicker().(*dropdown.Widget)
	if got := control.SelectedValue(); got != "job.yaml" {
		t.Fatalf("opened file label = %q, want job.yaml", got)
	}
	// An unrelated file with the same basename as an example is still its own file.
	name := filepath.Base(work.library[0].name)
	other := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(other, []byte("scene: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := work.state.load(other); err != nil {
		t.Fatal(err)
	}
	if got := work.scenePicker().(*dropdown.Widget).SelectedValue(); got != name {
		t.Fatalf("unrelated file label = %q, want %q", got, name)
	}
}

func libraryEventWorkspace(t *testing.T) *Workspace {
	t.Helper()
	examples, err := filepath.Abs("../../../examples")
	if err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{Composition: "play", Examples: examples, SuppressAutomaticPreview: true})
	if err != nil {
		t.Fatal(err)
	}
	temporary := filepath.Dir(work.state.path)
	t.Cleanup(func() { _ = os.RemoveAll(temporary) })
	work.ui.Window().HandleResize(1600, 1000)
	drawLibraryWindow(t, work)
	return work
}
func drawLibraryWindow(t *testing.T, work *Workspace) {
	t.Helper()
	work.ui.Window().Frame()
	context := gg.NewContext(1600, 1000)
	defer context.Close()
	work.ui.Window().DrawTo(render.NewCanvas(context, 1600, 1000))
	if err := context.FlushGPU(); err != nil {
		t.Fatal(err)
	}
}
func libraryWidget(root widget.Widget, id string) widget.Widget {
	if named, ok := root.(interface{ ID() string }); ok && named.ID() == id {
		return root
	}
	for _, child := range root.Children() {
		if found := libraryWidget(child, id); found != nil {
			return found
		}
	}
	return nil
}
func clickLibraryPoint(work *Workspace, point geometry.Point) {
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone))
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone))
}

// TestSceneLibraryPointerSelectionChangesDocument reaches the actual menu through window events.
func TestSceneLibraryPointerSelectionChangesDocument(t *testing.T) {
	work := libraryEventWorkspace(t)
	control := libraryWidget(work.root, "Scene library")
	if control == nil {
		t.Fatal("missing library")
	}
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("menu not opened: %d overlays", len(overlays))
	}
	menu := overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	want := work.library[1].name
	if filepath.Base(want) == work.DocumentName() {
		t.Fatal("fixture first entry must differ")
	}
	point := geometry.Pt(menu.Min.X+20, menu.Min.Y+60)
	clickLibraryPoint(work, point)
	if work.DocumentName() != filepath.Base(want) {
		t.Fatalf("selection stayed %s; wanted %s; diagnostic=%s overlays=%d", work.DocumentName(), want, work.diagnostic, len(work.ui.Window().OverlayContentWidgets()))
	}
	if work.working.photograph != nil || work.output != "" {
		t.Fatal("scene selection retained previous photograph")
	}
}

// TestSceneLibraryDirtySelectionKeepsProtectionAcrossFrames reproduces selecting after a pose edit.
func TestSceneLibraryDirtySelectionKeepsProtectionAcrossFrames(t *testing.T) {
	work := libraryEventWorkspace(t)
	before := work.state.document.Text()
	work.state.document.Replace(before + "\n# unsaved camera edit\n")
	work.editor.SetText(work.state.document.Text())
	control := libraryWidget(work.root, "Scene library")
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	work.Tick()
	drawLibraryWindow(t, work)
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatal("scene menu disappeared between frames")
	}
	menu := overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	want := work.library[1].name
	clickLibraryPoint(work, geometry.Pt(menu.Min.X+20, menu.Min.Y+60))
	work.Tick()
	drawLibraryWindow(t, work)
	if work.DocumentName() == filepath.Base(want) {
		t.Fatal("unsaved draft replaced before confirmation")
	}
	if len(work.ui.Window().OverlayContentWidgets()) != 1 {
		t.Fatal("save/discard prompt disappeared when scene menu closed")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModShift))
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if work.DocumentName() != filepath.Base(want) {
		t.Fatalf("Discard did not load selected scene: %s / %s", work.DocumentName(), work.diagnostic)
	}
}

// TestFullRenderButtonFreezesAuthoredRequestAndKeepsAcceptedPhotograph uses real button dispatch.
// The intentionally unsupported transport prevents external rendering; a PNG fixture exercises acceptance.
func TestFullRenderButtonFreezesAuthoredRequestAndKeepsAcceptedPhotograph(t *testing.T) {
	work := libraryEventWorkspace(t)
	source, err := filepath.Abs("../../../examples/full-render-check.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := work.state.load(source); err != nil {
		t.Fatal(err)
	}
	work.editor.SetText(work.state.document.Text())
	work.options.Evidence = t.TempDir()
	work.options.Tools = preview.Tools{Transport: "disabled-for-unit-test", Azazel: "unused", Queue: "unused", Slayer: "unused"}
	if err := work.SetWorkingMode("wire"); err != nil {
		t.Fatal(err)
	}
	work.rebuild()
	drawLibraryWindow(t, work)
	control := libraryWidget(work.root, "Render fully")
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	work.Tick()
	drawLibraryWindow(t, work)
	if !work.renderSetupOpen {
		t.Fatal("full-render button did not open setup")
	}
	select {
	case result := <-work.renderPresetResults:
		work.renderPresetResults <- result
		work.Tick()
		drawLibraryWindow(t, work)
	case <-time.After(5 * time.Second):
		t.Fatal("render setup did not finish loading")
	}
	control = libraryWidget(work.root, "Render")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	var result renderResult
	select {
	case result = <-work.results:
	case <-time.After(5 * time.Second):
		t.Fatal("button did not freeze a full request")
	}
	defer result.plan.Close()
	if result.err == nil || result.err.Error() != "unknown preview transport \"disabled-for-unit-test\"" {
		t.Fatalf("test unexpectedly attempted a transport: %v", result.err)
	}
	if !result.full || !result.plan.Profile.Full || result.plan.Profile.Width != 320 || result.plan.Profile.Height != 200 || result.plan.Profile.Rays != 2 {
		t.Fatalf("button did not preserve authored full request: %+v", result.plan.Profile)
	}
	// Represent an already displayed working frame before the current accepted renderer result.
	work.working.displayed = true
	work.viewer.setWorking(image.NewRGBA(image.Rect(0, 0, 100, 60)), 1.6)
	writeProofPNG(t, result.plan.Output, color.RGBA{R: 47, G: 93, B: 121, A: 255})
	result.err = nil
	work.refreshLayout = false
	work.finishPreview(result)
	if work.working.photograph == nil || work.working.photograph != work.viewer.source {
		t.Fatal("accepted full pixels not retained as photograph; next working tick would replace them")
	}
	if work.working.displayed || work.working.mode != "rendered" {
		t.Fatal("full completion left wire display active")
	}
	if work.output != result.plan.Output {
		t.Fatal("accepted full output was not recorded")
	}
	if !work.refreshLayout {
		t.Fatal("full completion did not refresh display picker")
	}
}

func TestRenderSetupFreezesChosenProducerSize(t *testing.T) {
	work := libraryEventWorkspace(t)
	source, err := filepath.Abs("../../../examples/full-render-check.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := work.state.load(source); err != nil {
		t.Fatal(err)
	}
	work.editor.SetText(work.state.document.Text())
	work.options.Evidence = t.TempDir()
	work.options.Tools = preview.Tools{Transport: "disabled-for-unit-test", Azazel: "unused", Queue: "unused", Slayer: "unused"}
	work.renderSetupOpen = true
	work.renderSetupPath, work.renderSetupSource = work.state.path, work.state.document.Text()
	work.renderPresets = []stagewire.FilmPreset{{Name: "imax", Width: 4096, Height: 3002}}
	work.renderCameras = work.state.document.Cameras()
	work.rebuild()
	drawLibraryWindow(t, work)
	control := libraryWidget(work.root, "Output size")
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("output-size menu did not open: %d overlays", len(overlays))
	}
	menu := overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	clickLibraryPoint(work, geometry.Pt(menu.Min.X+20, menu.Min.Y+60))
	if work.renderSetupChoice != 1 {
		t.Fatalf("preset selection = %d, want imax", work.renderSetupChoice)
	}
	work.Tick()
	drawLibraryWindow(t, work)
	control = libraryWidget(work.root, "Render camera")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	overlays = work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("camera menu did not open: %d overlays", len(overlays))
	}
	menu = overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	clickLibraryPoint(work, geometry.Pt(menu.Min.X+20, menu.Min.Y+60))
	if work.renderCameraChoice != 1 {
		t.Fatalf("camera selection = %d, want arrival", work.renderCameraChoice)
	}
	control = libraryWidget(work.root, "Render backend")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	overlays = work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("backend menu did not open: %d overlays", len(overlays))
	}
	menu = overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	clickLibraryPoint(work, geometry.Pt(menu.Min.X+20, menu.Min.Y+60))
	if work.renderBackendChoice != 1 {
		t.Fatalf("backend selection = %d, want GPU", work.renderBackendChoice)
	}
	work.Tick()
	drawLibraryWindow(t, work)
	control = libraryWidget(work.root, "Render")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	work.state.document.Replace(work.renderSetupSource + "\n# changed while setup was open\n")
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	if !strings.Contains(work.renderSetupError, "scene changed") || work.Running() {
		t.Fatal("stale render setup was submitted")
	}
	work.state.document.Replace(work.renderSetupSource)
	work.Tick()
	drawLibraryWindow(t, work)
	control = libraryWidget(work.root, "Render")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	select {
	case result := <-work.results:
		if result.plan == nil {
			t.Fatalf("no frozen plan: %v", result.err)
		}
		defer result.plan.Close()
		profile := result.plan.Profile
		if profile.SizePreset != "imax" || profile.Width != 4096 || profile.Height != 3002 || profile.Rays != 2 || profile.Camera != "arrival" || !profile.GPU || profile.CPURenderer {
			t.Fatalf("wrong frozen preset or sampling: %+v", profile)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("preset selection did not start a frozen request")
	}
}

func TestRenderSetupFreezesSevenExactSamples(t *testing.T) {
	work := libraryEventWorkspace(t)
	source, err := filepath.Abs("../../../examples/full-render-check.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := work.state.load(source); err != nil {
		t.Fatal(err)
	}
	work.editor.SetText(work.state.document.Text())
	work.options.Evidence = t.TempDir()
	work.options.Tools = preview.Tools{Transport: "disabled-for-unit-test", Azazel: "unused", Queue: "unused", Slayer: "unused"}
	work.renderSetupOpen = true
	work.renderSetupPath, work.renderSetupSource = work.state.path, work.state.document.Text()
	work.renderBackendChoice = 1
	work.rebuild()
	drawLibraryWindow(t, work)
	control := libraryWidget(work.root, "Samples per pixel")
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("sampling menu did not open: %d overlays", len(overlays))
	}
	menu := overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	clickLibraryPoint(work, geometry.Pt(menu.Min.X+20, menu.Min.Y+300))
	if work.renderSamplesChoice != 7 || work.renderBackendChoice != 1 {
		t.Fatalf("selected samples=%d backend=%d", work.renderSamplesChoice, work.renderBackendChoice)
	}
	work.Tick()
	drawLibraryWindow(t, work)
	control = libraryWidget(work.root, "Render")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	select {
	case result := <-work.results:
		if result.plan == nil {
			t.Fatalf("no frozen plan: %v", result.err)
		}
		defer result.plan.Close()
		if result.plan.Profile.Samples != 7 || !result.plan.Profile.GPU || result.plan.Profile.CPURenderer {
			t.Fatalf("wrong frozen sampling: %+v", result.plan.Profile)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("sample selection did not start a frozen request")
	}
}
