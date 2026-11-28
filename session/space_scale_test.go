package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

const scaleScene = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Scale study
  setting: space
  seed: 19
  light: none
  set:
    things:
      object:
        kind: can
        seed: 7
        complexity: quick
        east: 10
        north: 5
        height: {value: 3, unit: m}
  film: {width: 32, height: 24, wide: 40, rays: 1, exposure: 1}
  cameras:
    view: {location: {at: [0, -60, 20]}, orientation: {look_at: [10, 5, 3]}}
`

func scaleSession(t *testing.T, path, source string) (*Session, job.Job) {
	t.Helper()
	wanted, err := job.StillFrom(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	return opened, wanted
}

func scaledTypedSpace(source string) string {
	return strings.Replace(source, "        east: 10\n        north: 5\n        height: {value: 3, unit: m}\n",
		"        position:\n          east: {value: 10, unit: m}\n          north: {value: 5, unit: m}\n          up: {value: 3, unit: m}\n        scale: 2\n", 1)
}

func TestSpaceScaleReachesStagedGeometryAndJobKey(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "scene.yaml")
	base, oldJob := scaleSession(t, path, scaleScene)
	scaled, newJob := scaleSession(t, path, scaledTypedSpace(scaleScene))
	if oldJob.Key() == newJob.Key() {
		t.Fatal("scale did not change the job key")
	}
	before, err := base.WorkingView(0)
	if err != nil {
		t.Fatal(err)
	}
	after, err := scaled.WorkingView(0)
	if err != nil {
		t.Fatal(err)
	}
	low, high, has := before.World.Mesh.Bounds()
	newLow, newHigh, hasNew := after.World.Mesh.Bounds()
	origin := maths.V(10, 3, -5)
	if !has || !hasNew || newLow.Sub(origin).Sub(low.Sub(origin).Scale(2)).Len() > 1e-8 || newHigh.Sub(origin).Sub(high.Sub(origin).Scale(2)).Len() > 1e-8 {
		t.Fatalf("scaled bounds %v %v, base %v %v", newLow, newHigh, low, high)
	}
}

func TestSpaceScaleMovesAttachedLightsButPreservesIntensity(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	source := strings.Replace(scaleScene, "kind: can", "kind: spotlight", 1)
	path := filepath.Join(t.TempDir(), "scene.yaml")
	base, _ := scaleSession(t, path, source)
	scaled, _ := scaleSession(t, path, scaledTypedSpace(source))
	oldWorld := base.source.(stillFramer).world
	newWorld := scaled.source.(stillFramer).world
	if len(oldWorld.Lamps) == 0 || len(oldWorld.Lamps) != len(newWorld.Lamps) {
		t.Fatal("attached lamp was missing")
	}
	before, ok := oldWorld.Lamps[0].(emit.Lit)
	after, newOK := newWorld.Lamps[0].(emit.Lit)
	if !ok || !newOK {
		t.Fatal("attached lamp did not retain its native type")
	}
	origin := maths.V(10, 3, -5)
	if after.At.Sub(origin).Sub(before.At.Sub(origin).Scale(2)).Len() > 1e-8 || after.Beyond != before.Beyond*2 || after.Strength != before.Strength {
		t.Fatalf("scaled light %+v, base %+v", after, before)
	}
}

func TestSpaceScaleRefusesDeformingProfiles(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	dir := t.TempDir()
	_, book, err := manufacture.EmbeddedAnimalBook("whale")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "whale.yaml"), book, 0600); err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(scaleScene, "  set:\n", "  recipes:\n    orca: {animal: {family: whale, file: whale.yaml, species: orca, pose: swim}}\n  set:\n", 1)
	source = strings.Replace(source, "kind: can", "recipe: orca", 1)
	source = strings.Replace(source, "        east: 10\n", "        scale: 2\n        east: 10\n", 1)
	read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := placement.CheckDisplay(read); err == nil || !strings.Contains(err.Error(), "deforming") {
		t.Fatalf("working display accepted scaled deforming profile: %v", err)
	}
	wanted, err := job.StillFrom(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(context.Background(), wanted, Stamp{})
	if opened != nil {
		_ = opened.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "deforming") {
		t.Fatalf("scaled deforming profile was not refused: %v", err)
	}
}

func TestSpaceScaleUpdatesStandExclusionRadius(t *testing.T) {
	source := strings.Replace(scaleScene, "  film:", `    stands:
      neighbors:
        kind: can
        seed: 3
        count: 1
        within: {of: object, from: 0, to: 360, near: 0, far: 1}
        exclude: {around: object, clearance: {value: 0, unit: m}}
  film:`, 1)
	var radii [2]float64
	var stages [2]*staging
	var stand set.Stand
	for index, scale := range []string{"1", "2"} {
		text := strings.Replace(source, "        east: 10\n", "        scale: "+scale+"\n        east: 10\n", 1)
		read, err := set.Read("scale.yaml", []byte(text))
		if err != nil {
			t.Fatal(err)
		}
		stage := testStaging()
		stage.given.seed, stage.recipes = 19, read
		if err := stage.craft(read.Things()[0]); err != nil {
			t.Fatal(err)
		}
		stages[index], radii[index], stand = stage, stage.radii["object"], read.Stands()[0]
	}
	if radii[0] <= 0 || radii[1] != radii[0]*2 {
		t.Fatalf("scaled radii = %v", radii)
	}
	point := stages[0].places["object"].Add(maths.V(radii[0]*1.5, 0, 0))
	if !stages[0].allowed(stand, point) || stages[1].allowed(stand, point) {
		t.Fatal("stand exclusion did not use the scaled radius")
	}
}
