package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
	"github.com/janearc/slayer/scene/thing"
)

// smallPlan is a scene plan with a small terrain budget over a level source,
// anchored where the test sphere is.
func smallPlan() *plan {
	p := &plan{Radius: 1e6, Model: "authored-radial-offsets"}
	p.Anchor = location{Longitude: 10, Latitude: 20}
	p.Region.West, p.Region.South, p.Region.East, p.Region.North = 9.9, 19.9, 10.1, 20.1
	p.Detail.Patches, p.Detail.Evaluations, p.Detail.Depth = 16, 64, 2
	p.Detail.Side, p.Detail.Points, p.Detail.CellPixels = 5, 4096, 8
	p.Detail.MinimumHeight, p.Detail.MaximumHeight = -100, 100
	p.Camera.At = location{Longitude: 10, Latitude: 20, Height: 5000}
	p.Camera.Look = location{Longitude: 10, Latitude: 20.05, Height: 0}
	p.Camera.Near = 0.5
	p.Ocean.Height = 0
	p.Film.Width, p.Film.Height, p.Film.Rays, p.Film.Wide = 64, 48, 1, 60
	return p
}

// TestPlanTerrain checks the terrain made from a plan, and the refusal of
// an invalid detail budget and of a missing source.
func TestPlanTerrain(t *testing.T) {
	p := smallPlan()
	sphere := testSphere(t)
	ground, err := p.terrain(sphere, &flatHeights{height: 7})
	if err != nil || ground == nil {
		t.Fatalf("terrain: %v", err)
	}
	p.Detail.Normals = &struct {
		Longitude float64 `yaml:"longitude-deg" json:"longitude-deg"`
		Latitude  float64 `yaml:"latitude-deg" json:"latitude-deg"`
	}{Longitude: 0.001, Latitude: 0.001}
	if _, err := p.terrain(sphere, &flatHeights{height: 7}); err != nil {
		t.Errorf("with normal steps: %v", err)
	}
	p.Detail.Normals.Latitude = 0
	if _, err := p.terrain(sphere, &flatHeights{height: 7}); err == nil {
		t.Error("a normal step with one zero axis was accepted")
	}
	p.Detail.Normals = nil
	p.Detail.Side = 4
	if _, err := p.terrain(sphere, &flatHeights{height: 7}); err == nil {
		t.Error("a lattice of side 4 was accepted")
	}
	p = smallPlan()
	p.Region.West = 11
	if _, err := p.terrain(sphere, &flatHeights{height: 7}); err == nil {
		t.Error("a reversed region was accepted")
	}
}

// layersOf wraps one height source as the session's layers.
func layersOf(t *testing.T, source island.Heights) *island.Layers {
	t.Helper()
	layers, err := island.NewLayers(source)
	if err != nil {
		t.Fatal(err)
	}
	return layers
}

// TestCheckCamera checks a camera over a level source: it must clear the
// source, the ocean and the near plane, and its sight line must be defined.
func TestCheckCamera(t *testing.T) {
	sphere := testSphere(t)
	cases := []struct {
		name   string
		change func(*plan)
		source *flatHeights
		want   string
	}{
		{"clear", func(*plan) {}, &flatHeights{height: 100}, ""},
		{"below the source", func(p *plan) { p.Camera.At.Height = 50 }, &flatHeights{height: 100}, "does not clear source height"},
		{"within the near distance of the source", func(p *plan) { p.Camera.At.Height = 100.2 }, &flatHeights{height: 100}, "does not clear source height"},
		{"below the ocean with no source", func(p *plan) { p.Camera.At.Height = -5 }, &flatHeights{absent: true}, "does not clear"},
		{"no source, above the ocean", func(p *plan) { p.Camera.At.Height = 5 }, &flatHeights{absent: true}, ""},
		{"source error", func(*plan) {}, &flatHeights{fail: errors.New("source failed")}, "source failed"},
		{"looking at itself", func(p *plan) { p.Camera.Look = p.Camera.At }, &flatHeights{height: 100}, "view:"},
		{"position beyond the sphere", func(p *plan) { p.Camera.At.Height = -2e6 }, &flatHeights{height: 100}, "frame:"},
		{"target beyond the sphere", func(p *plan) { p.Camera.Look.Height = -2e6 }, &flatHeights{height: 100}, "frame:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := smallPlan()
			tc.change(p)
			session := &sceneSession{sphere: sphere, layers: layersOf(t, tc.source)}
			err := session.checkCamera(p)
			switch {
			case tc.want == "" && err != nil:
				t.Errorf("unexpected refusal: %v", err)
			case tc.want != "" && (err == nil || !containsText(err.Error(), tc.want)):
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

// TestFingerprint checks the digest of a file against the published digest
// of "abc", and the refusals for a missing file and a cancelled context.
func TestFingerprint(t *testing.T) {
	path := writeFile(t, t.TempDir(), "data.bin", "abc")
	got, err := fingerprint(t.Context(), path)
	if err != nil || got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("digest = %q, %v", got, err)
	}
	emptyPath := writeFile(t, t.TempDir(), "empty", "")
	if empty, err := fingerprint(t.Context(), emptyPath); err != nil || empty != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("empty digest = %q, %v", empty, err)
	}
	nonePath := filepath.Join(t.TempDir(), "none")
	if _, err := fingerprint(t.Context(), nonePath); err == nil {
		t.Error("a missing file was accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := fingerprint(ctx, path); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	big := make([]byte, 100<<10)
	large := writeFile(t, t.TempDir(), "big", string(big))
	sum := sha256.Sum256(big)
	if got, err := fingerprint(t.Context(), large); err != nil || got != hex.EncodeToString(sum[:]) {
		t.Errorf("a file larger than one read buffer: %q, %v", got, err)
	}
}

// TestOpenSourcesRefusals checks the refusals before any raster is decoded.
func TestOpenSourcesRefusals(t *testing.T) {
	dir := t.TempDir()
	content := "not a raster"
	sum := sha256.Sum256([]byte(content))
	path := writeFile(t, dir, "heights.tif", content)
	p := smallPlan()
	p.RasterBudget.CacheBytes, p.RasterBudget.CacheTiles = 1<<20, 4
	p.RasterBudget.TileBytes, p.RasterBudget.MetadataBytes = 1<<20, 1<<20
	cases := []struct {
		name   string
		source sourcePlan
		want   string
	}{
		{"missing file", sourcePlan{File: filepath.Join(dir, "none.tif")}, "no such file"},
		{"hash differs", sourcePlan{File: path, SHA256: "00"}, "source hash changed"},
		{"not a raster", sourcePlan{File: path, SHA256: hex.EncodeToString(sum[:])}, ""},
	}
	for _, tc := range cases {
		p.Sources = []sourcePlan{tc.source}
		layers, opened, err := p.openSources(t.Context())
		if err == nil || layers != nil || len(opened) != 0 || !containsText(err.Error(), tc.want) {
			t.Errorf("%s: layers %v opened %d error %v", tc.name, layers, len(opened), err)
		}
	}
	p.Sources = nil
	if _, _, err := p.openSources(t.Context()); err == nil || !containsText(err.Error(), "at least one height layer") {
		t.Errorf("no sources: %v", err)
	}
}

// TestOpenSceneRefusesBadSource checks that a scene whose source cannot be
// opened is refused and leaves nothing open.
func TestOpenSceneRefusesBadSource(t *testing.T) {
	p := smallPlan()
	p.Sources = []sourcePlan{{File: filepath.Join(t.TempDir(), "none.tif")}}
	session, err := openScene(t.Context(), p)
	if err == nil || session != nil {
		t.Errorf("session %v error %v", session, err)
	}
}

// TestSessionCloseIsRepeatable checks that closing releases the mist floors
// and a second close does no harm.
func TestSessionCloseIsRepeatable(t *testing.T) {
	session := &sceneSession{floors: floorStore{values: map[floorKey]floorEntry{{}: {}}, bytes: 72}}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if session.floors.values != nil || session.floors.bytes != 0 || session.opened != nil {
		t.Errorf("floors %v bytes %d", session.floors.values, session.floors.bytes)
	}
	if err := session.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

// TestWorldAssembly checks the world made from a plan and empty ground: the
// sky, ocean, sun lamp and the optional parts.
func TestWorldAssembly(t *testing.T) {
	p := smallPlan()
	p.Sky = [3]float64{0.1, 0.2, 0.3}
	p.Ground = [3]float64{0.4, 0.5, 0.6}
	p.Sun.Azimuth, p.Sun.Altitude, p.Sun.Strength = 90, 30, [3]float64{2, 2, 2}
	p.Ocean.Colour = [3]float64{0, 0.1, 0.2}
	p.Ocean.Gloss, p.Ocean.Index = 1, 1.34
	sphere := testSphere(t)
	origin, _ := frame.NewRelative(maths.V(0, 100, 0))
	world, err := p.world(sphere, origin, island.Prepared{Origin: maths.V(0, 100, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 1 || world.Things[0].Material.Name != "authored smooth ocean proxy" || len(world.Lamps) != 1 {
		t.Fatalf("things %d lamps %d", len(world.Things), len(world.Lamps))
	}
	if got := world.Things[0].At(0).Origin; got != maths.V(0, -1e6-100, 0) {
		t.Errorf("ocean centre = %v", got)
	}
	if world.Sky == nil || world.VolumeIllumination.Samples() != 0 || world.RayOriginReference != (thing.RayOriginReference{}) {
		t.Error("optional parts were set without being asked")
	}
	if got := world.Sky.Light(maths.V(0, 1, 0)); got != rgb(p.Sky) {
		t.Errorf("sky = %v, want the authored sky colour", got)
	}
}

// TestWorldAssemblyOptions checks the landcover texture, the shifted ground
// pose, the lighting, the ray reference and the cloud layers.
func TestWorldAssemblyOptions(t *testing.T) {
	p := smallPlan()
	p.Sun.Strength = [3]float64{1, 1, 1}
	p.Ground = [3]float64{0.4, 0.5, 0.6}
	p.Landcover = []heightBand{{Below: 100, Colour: [3]float64{1, 0, 0}}}
	p.LandcoverTransition = 0
	p.VolumeLighting = &volumeLightingPlan{Samples: 8}
	reference := 250.0
	p.RayReference = &reference
	p.Clouds = []cloudPlan{{Low: 1000, High: 2000, PatchScale: 100, Extinction: 0.001, Coverage: 0.5}}
	sphere := testSphere(t)
	origin, _ := frame.NewRelative(maths.V(0, 100, 0))
	ground := island.Prepared{Origin: maths.V(5, 100, 0), Surfaces: []*mesh.Mesh{quad(t)}}
	world, err := p.world(sphere, origin, ground)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 2 || len(world.Volumes) != 1 || world.VolumeIllumination.Samples() != 8 {
		t.Fatalf("things %d volumes %d", len(world.Things), len(world.Volumes))
	}
	paint, ok := world.Things[0].Material.Texture.(*altitudePaint)
	if !ok || len(paint.bands) != 1 || paint.base != rgb(p.Ground) || paint.origin != ground.Origin {
		t.Errorf("texture = %#v", world.Things[0].Material.Texture)
	}
	if got := world.Things[0].At(0).Origin; got != maths.V(5, 0, 0) {
		t.Errorf("ground offset = %v, want its origin less the render origin", got)
	}
	if world.RayOriginReference == (thing.RayOriginReference{}) {
		t.Error("the ray reference was dropped")
	}
	p.RayReference = ptr(-1.0)
	if _, err := p.world(sphere, origin, island.Prepared{}); err == nil {
		t.Error("a negative ray reference was accepted")
	}
	p.RayReference = nil
	p.VolumeLighting = &volumeLightingPlan{Samples: 3}
	if _, err := p.world(sphere, origin, island.Prepared{}); err == nil {
		t.Error("odd sky samples were accepted")
	}
	p.VolumeLighting = nil
	p.Sun.Strength = [3]float64{-1, -1, -1}
	if _, err := p.world(sphere, origin, island.Prepared{}); err == nil {
		t.Error("a negative sun was accepted")
	}
	p.Sun.Strength = [3]float64{1, 1, 1}
	p.Clouds[0].Low = -1
	if _, err := p.world(sphere, origin, island.Prepared{}); err == nil {
		t.Error("a cloud shell below the surface was accepted")
	}
	p.Clouds = nil
	p.Radius = -1
	if _, err := p.world(sphere, origin, island.Prepared{}); err == nil {
		t.Error("a negative ocean radius was accepted")
	}
}

// ptr returns the address of a copy of value.
func ptr(value float64) *float64 { return &value }

// TestPrepareGround checks the stateless path prepares a mesh for a level
// source and that a transition owner takes over when one is set.
func TestPrepareGround(t *testing.T) {
	p := smallPlan()
	sphere := testSphere(t)
	ground, err := p.terrain(sphere, &flatHeights{height: 7})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	view := lookDown(t, maths.V(0, 3000, 0), maths.V(0, 7, 0))
	origin, _ := frame.NewRelative(maths.V(0, 3000, 0))
	session := &sceneSession{ground: ground, sphere: sphere}
	prepared, err := session.prepareGround(t.Context(), p, view, origin)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Surfaces) == 0 || len(prepared.Patches) == 0 || prepared.Samples == 0 {
		t.Errorf("surfaces %d patches %d samples %d", len(prepared.Surfaces), len(prepared.Patches), prepared.Samples)
	}
	if time.Since(start) > time.Second {
		t.Errorf("preparation took %v", time.Since(start))
	}
	motion, err := island.NewMotion(ground, time.Second, 8, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	session.terrainMotion = motion
	p.Time = 2
	moved, err := session.prepareGround(t.Context(), p, view, origin)
	if err != nil || len(moved.Surfaces) == 0 {
		t.Errorf("transition path: %v, surfaces %d", err, len(moved.Surfaces))
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	session.terrainMotion = nil
	if _, err := session.prepareGround(ctx, p, view, origin); err == nil {
		t.Error("a cancelled context was ignored")
	}
}

// quad is a flat two-triangle surface, standing in for prepared ground.
func quad(t *testing.T) *mesh.Mesh {
	t.Helper()
	made, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(1, 0, 1), maths.V(0, 0, 1)},
		[][3]int{{0, 2, 1}, {0, 3, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}
