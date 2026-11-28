package workspace

import (
	"context"
	"errors"
	"github.com/gogpu/gg"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/editor"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// One placement release must be one editor history entry, including source undo.
func TestPlayPoseCreatesOneUndoAndRedoThroughWindow(t *testing.T) {
	work := playWorkspace(t)
	before := work.state.document.Text()
	if err := work.changePose(1.25, -.5, 15); err != nil {
		t.Fatal(err)
	}
	changed := work.state.document.Text()
	if changed == before || work.editor.Text() != changed {
		t.Fatal("placement did not update the authoritative source and editor")
	}
	if work.pending.text != changed || work.pending.document != work.state.document {
		t.Fatal("placement did not queue exactly the current authored source")
	}
	focusPlayViewer(work)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if work.editor.Text() != before || work.state.document.Text() != before {
		t.Fatal("one window Command-Z did not restore the whole placement")
	}
	if work.pending.text != before {
		t.Fatal("undo did not queue restored source")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', event.ModSuper))
	if work.state.document.Text() != before {
		t.Fatal("placement created an extra undo entry")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyZ, 'Z', event.ModSuper|event.ModShift))
	if work.editor.Text() != changed || work.state.document.Text() != changed {
		t.Fatal("window Command-Shift-Z did not restore the placement")
	}
	if work.pending.text != changed {
		t.Fatal("redo did not queue restored placement")
	}
}

// Old renderer completion cannot overwrite feedback or the queued new pose.
func TestPlayPoseCancelsAndRejectsStaleGeneration(t *testing.T) {
	work := playWorkspace(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	oldGeneration := work.generation
	if err := work.changePose(.25, 0, 0); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != context.Canceled || work.generation == oldGeneration {
		t.Fatal("placement did not cancel and invalidate the old renderer")
	}
	if work.cancel == nil {
		t.Fatal("in-flight renderer disappeared before completion")
	}
	pending := work.pending.text
	caption, status := work.caption, work.status
	work.finishPreview(renderResult{generation: oldGeneration, err: errors.New("obsolete pose")})
	if work.diagnostic != "" || work.caption != caption || work.status != status {
		t.Fatal("obsolete completion replaced current pose feedback")
	}
	if work.pending.text != pending || work.pending.document == nil {
		t.Fatal("obsolete completion discarded the replacement pose")
	}
	if !work.pending.take(work.state, work.specimen, time.Now().Add(time.Second), false) {
		t.Fatal("replacement pose could not run after obsolete worker exited")
	}
	if work.pending.take(work.state, work.specimen, time.Now().Add(time.Second), false) {
		t.Fatal("replacement pose was consumed twice")
	}
}

// Click-sized edits and missing selections must not cancel valid work.
func TestPlayPoseNoOpAndMissingSelectionDoNotQueue(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing selection", true: "zero delta"}[selected], func(t *testing.T) {
			work := playWorkspace(t)
			if !selected {
				work.play.selected = ""
			}
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			work.cancel = cancel
			before, generation := work.state.document.Text(), work.generation
			err := work.changePose(0, 0, 0)
			if selected && err != nil {
				t.Fatal(err)
			}
			if !selected && err == nil {
				t.Fatal("missing selection accepted")
			}
			if work.state.document.Text() != before || work.editor.Text() != before ||
				work.generation != generation || work.pending.document != nil || ctx.Err() != nil {
				t.Fatal("refused or zero placement altered source or render work")
			}
		})
	}
}

// Native window key dispatch selects modes without modifying source.
func TestPlayModeKeysAndPlacementUseWindowDispatch(t *testing.T) {
	work := playWorkspace(t)
	original := work.state.document.Text()
	for _, choice := range []struct {
		key  event.Key
		mode string
	}{
		{event.KeyT, "turn"}, {event.KeyL, "look"}, {event.KeyM, "move"},
	} {
		focusPlayViewer(work)
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, choice.key, 0, event.ModNone))
		if work.play.mode != choice.mode || work.viewer.mode != choice.mode {
			t.Fatalf("key %v mode=%q viewer=%q", choice.key, work.play.mode, work.viewer.mode)
		}
		if work.state.document.Text() != original {
			t.Fatal("mode selection edited source")
		}
	}
	focusPlayViewer(work)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModShift))
	object, ok := work.selectedObject()
	if !ok || object.Position()[0] != 1 {
		t.Fatalf("Shift-Right native east=%v, selected=%t", object.Position(), ok)
	}
	if work.editor.Text() != work.state.document.Text() || work.pending.text != work.editor.Text() {
		t.Fatal("window placement bypassed editor or preview queue")
	}
	focusPlayViewer(work)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyT, 't', event.ModNone))
	focusPlayViewer(work)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModNone))
	object, ok = work.selectedObject()
	heading, present := object.Heading()
	if !ok || !present || heading != 5 || object.Position()[0] != 1 {
		t.Fatalf("Turn Right pose=%v heading=%g present=%t", object.Position(), heading, present)
	}
}

// A working snapshot for older text is never selectable through setupPlay.
func TestPlaySelectionRequiresDisplayedSourceIdentity(t *testing.T) {
	work := playWorkspace(t)
	work.play.selected = ""
	installPlayWorkingView(t, work)
	if work.workingPick(.5, .5) == "" {
		t.Fatal("native scene center is not selectable")
	}
	work.working.source = work.state.document.Text() + "# older display\n"
	if work.viewer.selectObject(.5, .5) || work.play.selected != "" {
		t.Fatal("unmatched display selected current document geometry")
	}
	work.working.source = work.state.document.Text()
	if !work.viewer.selectObject(.5, .5) || work.play.selected != "scene.set.things/larson" {
		t.Fatal("current display did not select actual native Larson geometry")
	}
}

func installPlayWorkingView(t *testing.T, work *Workspace) {
	t.Helper()
	path, source := work.state.path, work.state.document.Text()
	read, err := set.Read(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, work.working.source, work.working.path, work.working.displayed = scene, source, path, true
}

func TestPlacementGestureCancelsOnSourceEdit(t *testing.T) {
	work := playWorkspace(t)
	installPlayWorkingView(t, work)
	work.viewer.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(800, 600)})
	start, finish := geometry.Pt(400, 300), geometry.Pt(430, 300)
	ctx := work.ui.Window().Context()
	work.viewer.Event(ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, event.ModNone))
	if !work.viewer.dragging || !work.viewer.dragObject || work.working.gesture == nil {
		t.Fatal("fixture did not start a placement drag")
	}
	edited := work.editor.Text() + "\n# Edit while the pointer is held.\n"
	work.editor.ApplyText(edited)
	if work.viewer.dragging || work.working.gesture != nil {
		t.Fatal("source edit left the previous gesture captured")
	}
	work.viewer.Event(ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, finish, finish, event.ModNone))
	if work.state.document.Text() != edited {
		t.Fatal("stale release changed the edited source")
	}
}

// playWorkspace uses the real native app, source document and editor callbacks.
func playWorkspace(t *testing.T) *Workspace {
	t.Helper()
	raw, err := os.ReadFile("../../../../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "play.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{File: path, Composition: "play"})
	if err != nil {
		t.Fatal(err)
	}
	work.autoPreview = false
	work.play.selected = "scene.set.things/larson"
	work.viewer.set(image.NewRGBA(image.Rect(0, 0, 640, 400)))
	work.viewer.SetNavigable(true)
	t.Cleanup(func() {
		work.clearNavigation()
		if work.cancel != nil {
			work.cancel()
		}
	})
	return work
}

func focusPlayViewer(work *Workspace) {
	work.viewer.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(200, 125)})
	work.ui.SetRoot(work.viewer)
	point := geometry.Pt(40, 40)
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone))
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone))
}

// This exports the actual native widget tree with existing Slayer pixels.
// It is a headless composition capture, not an OS window or a scene render.
func TestExportPlayWorkspaceWithActualSlayerPreview(t *testing.T) {
	directory := os.Getenv("FERRUCCIO_UI_EVIDENCE_DIR")
	if directory == "" {
		t.Skip("set FERRUCCIO_UI_EVIDENCE_DIR to export UI evidence")
	}
	work := playWorkspace(t)
	before := work.state.document.Text()
	previewPath := filepath.Join(directory, "preview-640.png")
	if err := work.showImage(previewPath); err != nil {
		t.Fatal(err)
	}
	if work.viewer.source.Bounds().Dx() != 640 || work.viewer.source.Bounds().Dy() != 400 {
		t.Fatal("capture requires the actual matched 640 by 400 Slayer preview")
	}
	work.viewer.SetNavigable(true)
	work.play.renderedText = before
	work.output = previewPath
	work.caption = "Native Slayer preview · 640 × 400 · one ray per side"
	work.status = "Preview ready"
	captureStamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	for _, capture := range []struct{ name, disclosure string }{
		{"ui-initial.png", ""},
		{"ui-construction.png", "construction"},
		{"ui-source.png", "source"},
	} {
		work.play.disclosure = capture.disclosure
		work.rebuild()
		name := strings.TrimSuffix(capture.name, ".png") + "-" + captureStamp + ".png"
		exportPlayWidget(t, work, filepath.Join(directory, name))
		t.Log("preserved native UI iteration", name)
	}
	raw, err := os.ReadFile(work.state.path)
	if err != nil || string(raw) != before || work.state.document.Text() != before {
		t.Fatal("headless widget capture changed the scene source", err)
	}
	t.Log("exported actual native workspace tree with existing Slayer pixels; no OS window claim")
}

// The bundled application opens a portable copy of the playable set.
func TestPlayBundledSceneOpensAsDefaultAndSavesTheVisitor(t *testing.T) {
	bundled, err := os.ReadFile("../../../examples/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{Examples: "../../../examples", Composition: "play"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { work.clearNavigation(); work.releaseExample() })
	if work.DocumentName() != "ferruccio-playground.yaml" || work.editor.Text() != string(bundled) || work.autoPreview || work.play.mode != "move" {
		t.Fatal("play opening did not load the real native set without an automatic preview")
	}
	original, err := filepath.Abs("../../../examples/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if work.state.path == original {
		t.Fatal("editable play scene points into bundled assets")
	}
	saved := filepath.Join(t.TempDir(), "playground.yaml")
	if !work.saveAt(saved) {
		t.Fatal("Save As failed")
	}
	savedSource, err := os.ReadFile(saved)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(saved, savedSource)
	if err != nil {
		t.Fatal(err)
	}
	world, ok := read.World()
	if !ok || len(world.Imports()) != 1 || world.Imports()[0].Name() != "visitor" {
		t.Fatal("saved playground lost its posed visitor")
	}
	parts := world.Imports()[0].Asset().Instances()
	if len(parts) != 1 {
		t.Fatalf("posed visitor has %d parts, want one", len(parts))
	}
	box := parts[0].Geometry().Bounds()
	at := parts[0].Placement().Within(world.Imports()[0].Placement(world.Unit()))
	if at.Origin.X != 7 || at.Origin.Z != -1 ||
		math.Abs(at.Out(box.Min).Y) > 1e-9 || math.Abs(at.Out(box.Max).Y-2) > 1e-9 {
		t.Fatalf("posed visitor changed position or scale: %v, %v", at.Origin, box)
	}
	for _, resource := range larsonPlaygroundStudy.files {
		if _, err := os.Stat(filepath.Join(filepath.Dir(saved), resource)); err != nil {
			t.Fatal(err)
		}
	}
	work.openPath(saved)
	second := filepath.Join(t.TempDir(), "reopened.yaml")
	if !work.saveAt(second) {
		t.Fatal("Save As after reopening failed")
	}
	for _, resource := range larsonPlaygroundStudy.files {
		first, err := os.ReadFile(filepath.Join(filepath.Dir(saved), resource))
		if err != nil {
			t.Fatal(err)
		}
		again, err := os.ReadFile(filepath.Join(filepath.Dir(second), resource))
		if err != nil || string(again) != string(first) {
			t.Fatalf("reopened save lost %s: %v", resource, err)
		}
	}
}

func TestPlaygroundSaveProtectsVisitorAssetNames(t *testing.T) {
	work := libraryEventWorkspace(t)
	source := filepath.Dir(work.state.path)
	for _, name := range larsonPlaygroundStudy.files {
		resource := filepath.Join(source, name)
		original, err := os.ReadFile(resource)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := work.savePlayground(resource); !errors.Is(err, ErrStudyAssetConflict) {
			t.Fatalf("source resource %q was not protected: %v", name, err)
		}
		current, err := os.ReadFile(resource)
		if err != nil || string(current) != string(original) {
			t.Fatalf("source resource %q changed: %v", name, err)
		}
	}
	work.state.document.Replace("invalid: [")
	for _, name := range larsonPlaygroundStudy.files {
		destination := t.TempDir()
		path := filepath.Join(destination, strings.ToUpper(name))
		if _, err := work.savePlayground(path); !errors.Is(err, ErrStudyAssetConflict) {
			t.Fatalf("destination resource %q was not protected: %v", name, err)
		}
		entries, err := os.ReadDir(destination)
		if err != nil || len(entries) != 0 {
			t.Fatalf("rejected save installed resources: %v, %v", entries, err)
		}
	}
}

func TestReopenedVisitorSaveTreatsTheCreditNoticeAsOptional(t *testing.T) {
	for _, name := range []string{"missing notice", "different notice", "different attribution"} {
		t.Run(name, func(t *testing.T) {
			work := libraryEventWorkspace(t)
			source := filepath.Join(t.TempDir(), "scene.yaml")
			if !work.saveAt(source) {
				t.Fatal(work.diagnostic)
			}
			notice := filepath.Join(filepath.Dir(source), larsonPlaygroundStudy.files[1])
			switch name {
			case "missing notice":
				if err := os.Remove(notice); err != nil {
					t.Fatal(err)
				}
			case "different notice":
				if err := os.WriteFile(notice, []byte("Unrelated notice"), 0600); err != nil {
					t.Fatal(err)
				}
			case "different attribution":
				text := strings.Replace(work.state.document.Text(), posedVisitorCredit, "Other attribution", 1)
				if err := os.WriteFile(source, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			work.openPath(source)
			if !work.saveAt(source) {
				t.Fatal(work.diagnostic)
			}
			saved := filepath.Join(t.TempDir(), "copy.yaml")
			if !work.saveAt(saved) {
				t.Fatal(work.diagnostic)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(saved), larsonPlaygroundStudy.files[0])); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(saved), filepath.Base(notice))); !os.IsNotExist(err) {
				t.Fatalf("the ordinary save carried an unrecognized notice: %v", err)
			}
		})
	}
}

// Changing intentions must preserve keyboard focus for the next gesture.
func TestPlayModeKeysPreserveFocusWithoutAnotherClick(t *testing.T) {
	work := playWorkspace(t)
	focusPlayViewer(work)
	before := work.state.document.Text()
	for _, choice := range []struct {
		key  event.Key
		mode string
	}{
		{event.KeyT, "turn"}, {event.KeyL, "look"}, {event.KeyM, "move"},
	} {
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, choice.key, 0, event.ModNone))
		if work.play.mode != choice.mode || !work.viewer.IsFocused() {
			t.Fatalf("mode %s lost keyboard focus or dispatch: mode=%s focused=%t", choice.mode, work.play.mode, work.viewer.IsFocused())
		}
	}
	if work.state.document.Text() != before {
		t.Fatal("mode chain modified source")
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModNone))
	object, ok := work.selectedObject()
	if !ok || object.Position()[0] != .25 {
		t.Fatal("following arrow did not move the selected object without a second click")
	}
}

func exportPlayWidget(t *testing.T, work *Workspace, path string) {
	t.Helper()
	const width, height = 1440, 960
	ctx := work.ui.Window().Context()
	work.root.Layout(ctx, geometry.Tight(geometry.Sz(width, height)))
	canvasContext := gg.NewContext(width, height)
	defer canvasContext.Close()
	work.root.Draw(ctx, render.NewCanvas(canvasContext, width, height))
	if err := canvasContext.FlushGPU(); err != nil {
		t.Fatal(err)
	}
	picture := canvasContext.Image()
	if picture.Bounds().Dx() != width || picture.Bounds().Dy() != height {
		t.Fatal("native widget export has incorrect dimensions")
	}
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(output, picture); err != nil {
		output.Close()
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}

// Escape from either focused disclosure preserves source, view and selection.
func TestPlayDisclosureEscapeReturnsWithoutChangingWorld(t *testing.T) {
	for _, disclosure := range []string{"source", "construction"} {
		t.Run(disclosure, func(t *testing.T) {
			work := playWorkspace(t)
			before := work.state.document.Text()
			selected, mode := work.play.selected, work.play.mode
			pixels := work.viewer.source
			work.toggleDisclosure(disclosure)
			ctx := work.ui.Window().Context()
			work.root.Layout(ctx, geometry.Tight(geometry.Sz(1440, 960)))
			var detail *editor.Widget
			var visit func(widget.Widget)
			visit = func(current widget.Widget) {
				if candidate, ok := current.(*editor.Widget); ok {
					detail = candidate
				}
				for _, child := range current.Children() {
					visit(child)
				}
			}
			visit(work.root)
			if detail == nil {
				t.Fatal("disclosure has no canonical source editor")
			}
			ctx.RequestFocus(detail)
			if !detail.IsFocused() {
				t.Fatal("disclosure editor did not receive actual focus")
			}
			work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, event.ModNone))
			if work.play.disclosure != "" || !work.viewer.IsFocused() {
				t.Fatal("Escape did not return focus to the world")
			}
			if work.state.document.Text() != before || work.editor.Text() != before ||
				work.play.selected != selected || work.play.mode != mode || work.viewer.source != pixels {
				t.Fatal("disclosure return changed source, selection, intention or retained pixels")
			}
			if work.pending.document != nil {
				t.Fatal("disclosure return queued an unrequested scene draw")
			}
		})
	}
}

// Proof gestures must reach native picking through the entire laid-out UI tree.
func TestPlayProofGestureUsesWholeWidgetTree(t *testing.T) {
	for _, gesture := range []string{"move", "turn"} {
		t.Run(gesture, func(t *testing.T) {
			work := wholePlayRoot(t)
			before := work.state.document.Text()
			if err := work.ProofGesture(gesture); err != nil {
				t.Fatal(err)
			}
			if work.state.document.Text() == before {
				t.Fatalf("%s proof dispatched through full widget tree without changing source; viewer=%v image=%v selected=%s", gesture, work.viewer.Bounds(), work.viewer.imageBounds(), work.play.selected)
			}
			if work.editor.Text() != work.state.document.Text() || work.pending.text != work.editor.Text() {
				t.Fatal("proof gesture bypassed source editor or latest preview queue")
			}
		})
	}
}

// Direct window events distinguish pointer coordinates from proof setup changes.
func TestPlayPointerUsesWholeWidgetTree(t *testing.T) {
	for _, gesture := range []string{"move", "turn"} {
		t.Run(gesture, func(t *testing.T) {
			work := wholePlayRoot(t)
			work.setPlayMode(gesture)
			// A real frame lays out the rebuilt full tree before pointer input.
			drawWholePlayRoot(t, work)
			bounds := work.viewer.ScreenBounds()
			start := geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
			localBounds := work.viewer.imageBounds()
			localCenter := geometry.Pt((localBounds.Min.X+localBounds.Max.X)/2, (localBounds.Min.Y+localBounds.Max.Y)/2)
			normalized, ok := work.viewer.imagePoint(localCenter)
			if !ok || work.workingPick(float64(normalized.X), float64(normalized.Y)) != "scene.set.things/larson" {
				t.Fatal("working view center does not address native Larson geometry")
			}
			before := work.state.document.Text()
			work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, event.ModNone))
			if !work.viewer.dragging || !work.viewer.dragObject {
				t.Fatalf("whole-tree press missed native object: dragging=%t object=%t viewer=%v pointer=%v", work.viewer.dragging, work.viewer.dragObject, work.viewer.Bounds(), start)
			}
			end := geometry.Pt(start.X+100, start.Y+40)
			work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
			if work.state.document.Text() != before {
				t.Fatal("pointer motion edited before release")
			}
			work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
			if work.state.document.Text() == before {
				t.Fatal("whole-tree native object release did not edit source")
			}
		})
	}
}

func wholePlayRoot(t *testing.T) *Workspace {
	t.Helper()
	work := playWorkspace(t)
	installPlayWorkingView(t, work)
	drawWholePlayRoot(t, work)
	return work
}

func drawWholePlayRoot(t *testing.T, work *Workspace) {
	t.Helper()
	ctx := work.ui.Window().Context()
	work.root.Layout(ctx, geometry.Tight(geometry.Sz(1440, 960)))
	canvasContext := gg.NewContext(1440, 960)
	defer canvasContext.Close()
	work.root.Draw(ctx, render.NewCanvas(canvasContext, 1440, 960))
	if err := canvasContext.FlushGPU(); err != nil {
		t.Fatal(err)
	}
	if !work.viewer.IsScreenOriginValid() {
		t.Fatal("real widget draw did not stamp viewer screen origin")
	}
}
