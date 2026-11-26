package session

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func TestDeformingInstancesAreFreshPlacedAndReversible(t *testing.T) {
	world := thing.NewWorld()
	fixed := world.Add(thing.Thing{Shape: shape.Sphere{Radius: 7}, Material: &thing.Material{Name: "fixed"}})
	material := &thing.Material{Name: "skin"}
	build := func(at time.Duration) ([]thing.Thing, error) {
		return []thing.Thing{{Shape: shape.Sphere{Radius: 1 + at.Seconds()}, Material: material,
			Motion: thing.Still(maths.Subspace{Origin: maths.V(1, 0, 0)})}}, nil
	}
	nolamps := func(maths.Subspace) ([]emit.Lit, error) { return nil, nil }
	stage := staging{world: world, given: asked{}, have: stock{}}
	place := maths.Subspace{Origin: maths.V(10, 2, 3), Orientation: maths.About(maths.YAxis, math.Pi/2)}
	if err := stage.add(kept{deform: build, lamps: nolamps}, "test", place, 0, false); err != nil {
		t.Fatal(err)
	}
	live := stage.retain()
	first, _, err := live.frameAt(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	later, _, err := live.frameAt(2 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	again, _, err := live.frameAt(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Things) != 2 || first.Things[0] != fixed || later.Things[0] != fixed || again.Things[0] != fixed {
		t.Fatal("static world was changed or dynamic objects accumulated")
	}
	if first.Things[1] == later.Things[1] || first.Things[1] == again.Things[1] {
		t.Fatal("frame objects were reused")
	}
	if first.Things[1].Shape.(shape.Sphere).Radius != 2 || later.Things[1].Shape.(shape.Sphere).Radius != 3 || again.Things[1].Shape.(shape.Sphere).Radius != 2 {
		t.Fatal("frame time or reverse replay was lost")
	}
	want := place.Out(maths.V(1, 0, 0))
	got := first.Things[1].At(time.Second).Origin
	if got.Sub(want).Len() > 1e-9 {
		t.Fatalf("local placement was not composed: got %v want %v", got, want)
	}
	if world.Things[0] != fixed || len(world.Things) != 2 || world.Things[1].Shape.(shape.Sphere).Radius != 1 {
		t.Fatal("initial staged world was mutated")
	}
	still := stillFramer{world: world, given: asked{live: live}}
	stillAtTwo, err := still.worldAt(2 * time.Second)
	if err != nil || stillAtTwo.Things[1].Shape.(shape.Sphere).Radius != 3 {
		t.Fatalf("still path did not evaluate its requested time: %v", err)
	}
	film := timedFramer{given: asked{live: live}}
	filmAtOne, err := film.inspect(time.Second)
	if err != nil || filmAtOne.Things[1].Shape.(shape.Sphere).Radius != 2 {
		t.Fatalf("film path did not evaluate its requested time: %v", err)
	}
}

func TestDeformationRefusesExclusionsWithoutConservativeBounds(t *testing.T) {
	raw, err := os.ReadFile("../doc/observatory-ecosystem.yaml")
	if err != nil {
		t.Fatal(err)
	}
	withStand := string(raw)
	withoutStand := strings.Replace(withStand, "        exclude:\n          around: the observatory\n          clearance: {value: 40, unit: m}\n", "", 1)
	region := "  ecosystems:\n    mixed grove:\n      seed: 1\n      cell: {value: 100, unit: m}\n      density: {value: 1, unit: ha^-1}\n      variety: 2\n      bounds: {west: 0, east: 200, north: 0, south: -200}\n      mixture: {redwoods: 1}\n      exclude: {around: the observatory, clearance: {value: 30, unit: m}}\n"
	withRegion := strings.Replace(withoutStand, "  terrain:", region+"  terrain:", 1)
	if withStand == withoutStand || withoutStand == withRegion {
		t.Fatal("fixture insertion failed")
	}
	for _, tc := range []struct{ source, want string }{{withoutStand, ""}, {withStand, "stand"}, {withRegion, "ecosystem"}} {
		read, err := set.Read("../doc/observatory-ecosystem.yaml", []byte(tc.source))
		if err != nil {
			t.Fatal(err)
		}
		err = deformingExclusionRefusal(read)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("want %q refusal, got %v", tc.want, err)
		}
	}
}

func TestDeformationRefusesUnsupportedModes(t *testing.T) {
	for _, tc := range []struct {
		given   asked
		shutter bool
		want    string
	}{
		{given: asked{onGPU: true}, want: "CPU"},
		{shutter: true, want: "shutter"},
		{given: asked{cachedLighting: true}, want: "cached lighting"},
		{given: asked{closeGlow: true}, want: "close-glow"},
	} {
		if err := deformingRefusal(tc.given, tc.shutter); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("expected %q refusal, got %v", tc.want, err)
		}
	}
}
