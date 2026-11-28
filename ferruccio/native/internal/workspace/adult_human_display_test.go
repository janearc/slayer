//go:build cgo

package workspace

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestAdultHumanExamplesStagePoseAndHairWithMaterialLimits(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	var bodies []maths.Box
	for _, name := range []string{"neutral", "reaching", "male"} {
		t.Run(name, func(t *testing.T) {
			path, raw := readAdultExample(t, name)
			start := time.Now()
			got, err := StageSnapshot(context.Background(), path, string(raw), "set")
			if err != nil {
				t.Fatal(err)
			}
			var hits []placement.StagedBatch
			for _, batch := range got.Scene.Batches {
				if batch.Authored && batch.Author.Name == "adult" {
					hits = append(hits, batch)
				}
			}
			if len(hits) != 4 {
				t.Fatalf("adult staged %d parts, want skin, two eyes and the scalp mesh", len(hits))
			}
			low, high, ok := hits[0].Mesh.Bounds()
			if !ok {
				t.Fatal("staged body has no bounds")
			}
			bodies = append(bodies, maths.NewBox(low, high))
			for _, batch := range hits {
				if batch.Mesh.Notice != "" || len(batch.Mesh.Indices) == 0 {
					t.Fatal("a body part lost its native triangles or its selectable disposition")
				}
			}
			checkAdultMaterialNotice(t, got.Notice, name == "male")
			t.Logf("CPU stage in %s, body %d and scalp %d triangles", time.Since(start),
				len(hits[0].Mesh.Indices)/3, len(hits[3].Mesh.Indices)/3)
		})
	}
	if len(bodies) == 3 && bodies[0] == bodies[1] {
		t.Fatal("reaching specimen lost its authored pose")
	}
}

func TestAdultHumanCachedDisplayKeepsMaterialLimits(t *testing.T) {
	path, raw := readAdultExample(t, "male")
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	cache := placement.NewDisplayCache()
	start := time.Now()
	first, err := cache.Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	cold := time.Since(start)
	start = time.Now()
	second, err := cache.Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	warm := time.Since(start)
	if len(first.Objects) != 2 || len(second.Objects) != 2 || first.Objects[0].Mesh != second.Objects[0].Mesh {
		t.Fatal("cached display lost the adult mesh or rebuilt it on the second open")
	}
	checkAdultMaterialNotice(t, localWorkingNotice(read, second), true)
	start = time.Now()
	hit, ok := second.PickAt(.5, .5)
	firstPick := time.Since(start)
	if !ok || hit.ID != "scene.set.things/adult" || hit.Part != "adult human skin" {
		t.Fatalf("cached centre pick lost the adult or skin identity: %+v", hit)
	}
	t.Logf("CPU cached build %s cold, %s warm, first centre pick %s, %d adult triangles", cold, warm,
		firstPick, len(first.Objects[0].Mesh.Indices)/3)
}

func TestAdultHumanCachedSelectionEditsTypedPlacement(t *testing.T) {
	work := playWorkspace(t)
	_, raw := readAdultExample(t, "male")
	path := filepath.Join(t.TempDir(), "adult-human-male.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	work.state.document, work.state.path = document, path
	work.editor.SetText(source)
	work.working.scene, work.working.source, work.working.path, work.working.displayed = scene, source, path, true
	work.working.parsed = &read
	work.play.selected = ""
	work.setPlayMode("move")
	work.viewer.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(800, 600)})
	centre := geometry.Pt(400, 300)
	ctx := widget.NewContext()
	pressed := work.viewer.Event(ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, centre, centre, event.ModNone))
	work.viewer.Event(ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, centre, centre, event.ModNone))
	if !pressed || work.play.selected != "scene.set.things/adult" || document.Text() != source {
		t.Fatalf("centre press consumed %t, selection %q, working pick %q", pressed, work.play.selected, work.workingPick(.5, .5))
	}
	if err := work.changeThingScale(1.25); err != nil {
		t.Fatal(err)
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	scaledSource, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scaled, err := set.Read(path, scaledSource)
	if err != nil {
		t.Fatal(err)
	}
	positionBefore, _ := read.Things()[0].Position()
	positionAfter, absolute := scaled.Things()[0].Position()
	if scale, explicit := scaled.Things()[0].Scale(); !explicit || scale != 1.25 || !absolute || positionAfter != positionBefore {
		t.Fatal("size edit changed the typed position or lost its factor on save")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != source || work.editor.Text() != source {
		t.Fatal("size undo did not restore the exact source")
	}
	// Simulate the restored view's redraw without a GPU.
	work.working.displayed = true
	if err := work.changePose(-.25, .5, 15); err != nil {
		t.Fatal(err)
	}
	moved := document.Text()
	work.play.poseText[0], work.play.poseText[1], work.play.poseText[2] = "-.25", ".5", "-.125"
	if err := work.submitCoordinates(false); err != nil {
		t.Fatal(err)
	}
	lowered := document.Text()
	read, err = set.Read(path, []byte(lowered))
	if err != nil {
		t.Fatal(err)
	}
	scene, err = placement.NewDisplayCache().Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	work.working.scene, work.working.source = scene, lowered
	work.working.parsed = &read
	bounds := work.viewer.imageBounds()
	torso := geometry.Pt(bounds.Min.X+bounds.Width()*.32, bounds.Min.Y+bounds.Height()*.38)
	right := geometry.Pt(torso.X+bounds.Width()*.05, torso.Y)
	start, _ := work.viewer.imagePoint(torso)
	end, _ := work.viewer.imagePoint(right)
	if hit := scene.Pick(float64(start.X), float64(start.Y)); hit != "scene.set.things/adult" {
		t.Fatalf("test pointer missed the torso: %q", hit)
	}
	if _, _, err := scene.DragHeight(-.125, float64(start.X), float64(start.Y), float64(end.X), float64(end.Y)); err == nil {
		t.Fatal("test pointer unexpectedly reaches the old placement plane")
	}
	work.viewer.Event(ctx, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, torso, torso, event.ModNone))
	if !work.viewer.Event(ctx, event.NewWheelEvent(geometry.Pt(0, 20), torso, torso, event.ModNone)) ||
		document.Text() != lowered || work.working.gesture == nil || work.working.scene.Eye != scene.Eye {
		t.Fatal("wheel changed the camera or source during the captured drag")
	}
	work.viewer.Event(ctx, event.NewMouseEvent(event.MouseDrag, event.ButtonLeft, event.ButtonStateLeft, right, right, event.ModNone))
	previewEast := work.working.scene.Objects[0].At.Origin.X
	if previewEast <= scene.Objects[0].At.Origin.X || document.Text() != lowered {
		t.Fatal("torso drag did not move provisional geometry without editing source")
	}
	work.viewer.Event(ctx, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, right, right, event.ModNone))
	object, ok := work.selectedObject()
	if !ok || math.Abs(object.Position()[0]-previewEast) > 1e-9 || math.Abs(object.Position()[1]-.5) > 1e-9 || object.Position()[2] != -.125 {
		t.Fatalf("torso drag changed the wrong coordinates: %v", object.Position())
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != lowered || work.editor.Text() != lowered {
		t.Fatal("undo did not restore the exact pre-drag source")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := set.Read(path, saved)
	if err != nil {
		t.Fatal(err)
	}
	adult := reopened.Things()[0]
	position, absolute := adult.Position()
	heading, turned := adult.Heading()
	_, sized := adult.Scale()
	if string(saved) != lowered || !absolute || position.East() != -.25 || position.North() != .5 || position.Up() != -.125 ||
		!turned || heading != 15 || sized {
		t.Fatalf("saved adult position %v (%t), heading %g (%t), scale written %t", position, absolute, heading, turned, sized)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != moved || work.editor.Text() != moved {
		t.Fatal("undo did not restore the pre-coordinate pose and units")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != source || work.editor.Text() != source {
		t.Fatal("undo did not restore the exact original source")
	}
}

func readAdultExample(t *testing.T, name string) (string, []byte) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("../../../../sets/bioforms", "adult-human-"+name+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, raw
}

func checkAdultMaterialNotice(t *testing.T, notice string, fibre bool) {
	t.Helper()
	if !strings.Contains(notice, "Surface textures are omitted") ||
		!strings.Contains(notice, "Subsurface scattering is omitted") ||
		strings.Contains(notice, "fibre scattering is omitted") != fibre {
		t.Fatalf("incomplete adult material notice: %q", notice)
	}
}
