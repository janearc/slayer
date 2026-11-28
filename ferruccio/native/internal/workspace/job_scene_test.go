package workspace

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
)

func TestJobCanOpenItsReferencedSceneForEditing(t *testing.T) {
	root := t.TempDir()
	scene := filepath.Join(root, "city.yaml")
	job := filepath.Join(root, "job.yaml")
	jobText := "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: city.yaml}\ncamera: {name: view}\n"
	if err := os.WriteFile(scene, []byte(checkingSource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(job, []byte(jobText), 0600); err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	if err := work.state.load(job); err != nil {
		t.Fatal(err)
	}
	work.editor.SetText(work.state.document.Text())
	work.rebuild()
	if libraryWidget(work.root, "Open scene to edit") == nil {
		t.Fatal("job does not offer its referenced scene")
	}
	work.state.document.Replace(jobText + "# unsaved\n")
	work.openReferencedSet()
	if work.state.path != job || !work.protecting {
		t.Fatal("opening the referenced scene did not protect the job draft")
	}
	work.protecting = false
	if err := work.state.load(job); err != nil {
		t.Fatal(err)
	}
	work.openReferencedSet()
	if work.state.path != scene || work.state.document.Kind() != "set" || work.state.document.Text() != checkingSource {
		t.Fatalf("opened %q, kind %q", work.state.path, work.state.document.Kind())
	}
	if work.working.cameraName != "view" || !strings.Contains(work.workingCaption(), `job camera "view"`) {
		t.Fatalf("working camera was not retained or shown: %q", work.workingCaption())
	}
	if work.state.document.Dirty() {
		t.Fatal("opening the job camera changed the scene draft")
	}
	if got := work.previewRequest(time.Now()).Profile.Camera; got != "view" {
		t.Fatalf("Preview camera = %q", got)
	}
	work.renderSetupOpen = true
	if got := work.fullRenderProfile(nil).Camera; got != "view" {
		t.Fatalf("default full render camera = %q", got)
	}
	work.renderCameras = []string{"another"}
	work.renderCameraChoice = 1
	if got := work.fullRenderProfile(nil).Camera; got != "another" {
		t.Fatalf("explicit full render camera = %q", got)
	}
	work.renderSetupOpen = false
	work.viewer.SetNavigable(true)
	work.viewer.set(image.NewRGBA(image.Rect(0, 0, 640, 400)))
	if err := work.Navigate(1, 0, 0); err == nil || !strings.Contains(err.Error(), "in Source") {
		t.Fatalf("named camera gesture was not refused: %v", err)
	}
	if err := work.moveCamera(1, 0, 0); err == nil || !strings.Contains(err.Error(), "in Source") {
		t.Fatalf("named camera translation was not refused: %v", err)
	}
	work.viewer.mode = "look"
	work.viewer.SetBounds(geometry.Rect{Max: geometry.Pt(640, 400)})
	point := geometry.Pt(320, 100)
	press := event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone)
	work.diagnostic = ""
	handled := work.viewer.Event(work.ui.Window().Context(), press)
	if !handled || work.viewer.dragging || work.working.gesture != nil || !strings.Contains(work.diagnostic, "in Source") {
		t.Fatalf("press did not refuse the inherited camera before starting a gesture: %q", work.diagnostic)
	}
	if work.state.document.Text() != checkingSource {
		t.Fatal("refused gestures changed the source")
	}
	work.openPath(scene)
	if work.working.cameraName != "" {
		t.Fatal("opening a scene directly retained the prior job camera")
	}
}

func TestOpenReferencedSceneKeepsStillCameraZoom(t *testing.T) {
	root := t.TempDir()
	scene := filepath.Join(root, "city.yaml")
	job := filepath.Join(root, "job.yaml")
	text := `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: city.yaml}
at: {value: 20, unit: s}
camera:
  name: view
  zoom: {from: {value: 60, unit: deg}, to: {value: 20, unit: deg}, duration: {value: 8, unit: s}}
`
	if err := os.WriteFile(scene, []byte(checkingSource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(job, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	if err := work.state.load(job); err != nil {
		t.Fatal(err)
	}
	if err := work.state.document.Check(); err != nil {
		t.Fatalf("zoom fixture is not a valid draft: %v", err)
	}
	work.openReferencedSet()
	if work.state.path != scene || work.state.document.Text() != checkingSource || work.state.document.Dirty() {
		t.Fatal("opening the zoomed job changed the authored set")
	}
	if work.working.cameraName != "view" || work.working.cameraWide != 60 {
		t.Fatalf("working camera = %s, %g°; %s", work.working.cameraName, work.working.cameraWide, work.diagnostic)
	}
	if profile := work.previewRequest(time.Now()).Profile; profile.Camera != "view" || profile.CameraWide != 60 {
		t.Fatalf("Preview lost the still zoom: %+v", profile)
	}
	work.renderSetupOpen = true
	if profile := work.fullRenderProfile(nil); profile.Camera != "view" || profile.CameraWide != 60 {
		t.Fatalf("Full Render lost the still zoom: %+v", profile)
	}
	work.renderCameras = []string{"view", "another"}
	for index, camera := range work.renderCameras {
		work.renderCameraChoice = index + 1
		if profile := work.fullRenderProfile(nil); profile.Camera != camera || profile.CameraWide != 0 {
			t.Fatalf("explicit camera %q should use its authored lens: %+v", camera, profile)
		}
	}
	work.renderCameraChoice = 0
	if profile := work.fullRenderProfile(nil); profile.Camera != "view" || profile.CameraWide != 60 {
		t.Fatalf("returning to the inherited camera lost the still zoom: %+v", profile)
	}
	work.renderSetupOpen = false
	work.openPath(scene)
	if work.working.cameraWide != 0 {
		t.Fatal("opening the set directly retained a prior zoom")
	}
}

func TestOpenReferencedSceneKeepsAuthoredStillCameraPath(t *testing.T) {
	root := t.TempDir()
	scene := filepath.Join(root, "city.yaml")
	job := filepath.Join(root, "job.yaml")
	text := `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: city.yaml}
at: {value: 20.125, unit: s}
camera:
  unit: m
  look-at: {x: 0, y: 0, z: 0}
  moves:
    - from: {x: 0, y: 0, z: 6}
      to: {x: 0, y: 0, z: 2}
      speed: {value: 2, unit: m/s}
`
	if err := os.WriteFile(scene, []byte(checkingSource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(job, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	if err := work.state.load(job); err != nil {
		t.Fatal(err)
	}
	if err := work.state.document.Check(); err != nil {
		t.Fatalf("camera path fixture is not a valid draft: %v", err)
	}
	work.openReferencedSet()
	if work.state.path != scene || work.state.document.Text() != checkingSource ||
		work.working.cameraPath == "" || !strings.Contains(work.workingCaption(), "job camera path") {
		t.Fatalf("opening the scene lost its authored job camera path: %s", work.diagnostic)
	}
	recipe := work.working.cameraPath
	if work.previewRequest(time.Now()).Profile.CameraPathRecipe != recipe || work.fullRenderProfile(nil).CameraPathRecipe != recipe {
		t.Fatal("render requests did not keep the inherited path")
	}
	if err := work.cameraEditRefusal(); err == nil || !strings.Contains(err.Error(), "job YAML") {
		t.Fatalf("inherited camera editing was not refused: %v", err)
	}
	work.renderSetupOpen = true
	work.renderCameras = []string{"view"}
	work.renderCameraChoice = 1
	if profile := work.fullRenderProfile(nil); profile.Camera != "view" || profile.CameraPathRecipe != "" || profile.StillAt != 20*time.Second+125*time.Millisecond {
		t.Fatalf("explicit camera did not replace the path and preserve the still time: %+v", profile)
	}
	work.renderCameraChoice = 0
	if work.fullRenderProfile(nil).CameraPathRecipe != recipe {
		t.Fatal("returning to the inherited camera lost its path")
	}
	work.openPath(scene)
	if work.working.cameraPath != "" || work.working.cameraPathAt != 0 {
		t.Fatal("opening the set directly retained the job path or time")
	}
	for _, fixture := range []struct {
		name, yaml, diagnostic string
	}{
		{"film optics", "film: {wide: 40}", "set's own film settings"},
		{"film sampling", "film: {rays: 2}", "set's own film settings"},
		{"print", "print: {knee: 0.5, top: 8}", `job's "print" setting`},
		{"stock", "stock: {name: dusk, seed: 7}", `job's "stock" setting`},
		{"titles", "titles: {elements: {name: {kind: text, words: Moonlake, ink: ink, across: 0.5, down: 0.9}}}", `job's "titles" setting`},
		{"vibes", "vibes: on", `job's "vibes" setting`},
		{"fastlight", "fastlight: off", `job's "fastlight" setting`},
		{"shortcuts", "shortcuts: {lamp-store: on}", `job's "shortcuts" setting`},
		{"renderer", "renderer: gpu", `job's "renderer" setting`},
		{"quality", "quality: preview", `job's "quality" setting`},
		{"limits", "limits: {time: {value: 60, unit: s}}", `job's "limits" setting`},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			authored := text + fixture.yaml + "\n"
			if err := os.WriteFile(job, []byte(authored), 0600); err != nil {
				t.Fatal(err)
			}
			work.openPath(job)
			if err := work.state.document.Check(); err != nil {
				t.Fatalf("invalid fixture: %v", err)
			}
			work.openReferencedSet()
			if work.state.path != job || work.state.document.Text() != authored ||
				!strings.Contains(work.diagnostic, fixture.diagnostic) || !strings.Contains(work.diagnostic, "Render the job") {
				t.Fatalf("job override was not refused before switching documents: %s", work.diagnostic)
			}
		})
	}
	unrepresentable := strings.Replace(text, "20.125", "1.0000000071", 1)
	if err := os.WriteFile(job, []byte(unrepresentable), 0600); err != nil {
		t.Fatal(err)
	}
	work.openPath(job)
	work.openReferencedSet()
	if work.state.path != job || !strings.Contains(work.diagnostic, "cannot preserve this still time") {
		t.Fatalf("a lossy time override was not refused: %s", work.diagnostic)
	}
}
