package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/thing"
)

// roomySite is a test site whose face and instance budgets are large.
func roomySite(t *testing.T) *siteBuilder {
	t.Helper()
	site, _ := newTestSite(t, 7, 0)
	site.plan.Budget.Faces, site.plan.Budget.Instances = 1<<30, 1<<20
	return site
}

// TestLoadStoresAndReuses checks a manufactured prototype is stored once and
// returned from the store the second time.
func TestLoadStoresAndReuses(t *testing.T) {
	site := roomySite(t)
	key := prototypeKey{kind: "can", detail: "quick", seed: 3}
	first, err := site.load(key)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.parts) != 1 || site.store.faces != 64 || len(site.store.values) != 1 {
		t.Fatalf("parts %d faces %d stored %d", len(first.parts), site.store.faces, len(site.store.values))
	}
	if first.bounds.Min.Y != 0 || first.bounds.Max.Y <= 0 {
		t.Errorf("a can should stand on y = 0: %+v", first.bounds)
	}
	second, err := site.load(key)
	if err != nil {
		t.Fatal(err)
	}
	if site.store.faces != 64 || second.bounds != first.bounds {
		t.Errorf("a repeat load changed the store: faces %d", site.store.faces)
	}
	lit, err := site.load(prototypeKey{kind: "campfire", detail: "quick", seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(lit.lights) != 1 {
		t.Errorf("a campfire carries %d lights, want 1", len(lit.lights))
	}
}

// TestLoadRefusals checks unknown kinds, unknown detail names and the face
// budget.
func TestLoadRefusals(t *testing.T) {
	site := roomySite(t)
	if _, err := site.load(prototypeKey{kind: "dragon", detail: "quick"}); err == nil || !containsText(err.Error(), "dragon") {
		t.Errorf("unknown kind: %v", err)
	}
	if _, err := site.load(prototypeKey{kind: "can", detail: "lavish"}); err == nil || !containsText(err.Error(), "lavish") {
		t.Errorf("unknown detail: %v", err)
	}
	site.plan.Budget.Faces = 63
	_, err := site.load(prototypeKey{kind: "can", detail: "quick", seed: 3})
	if err == nil || !containsText(err.Error(), "needs 64 stored faces; 0 of 63 already used") {
		t.Errorf("face budget: %v", err)
	}
	if len(site.store.values) != 0 || site.store.faces != 0 {
		t.Error("a refused prototype was stored")
	}
}

// TestLoadCrown checks a crown prototype: it is built from the clump
// prototype, keyed by the area side when matching is on, and its probe rays
// are counted.
func TestLoadCrown(t *testing.T) {
	site := roomySite(t)
	site.plan.Forest = &forestPlan{CrownAreaMatch: true, CrownAreaSide: 8}
	value, err := site.load(prototypeKey{kind: "spruce", detail: "crown", seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	if value.fit == nil || value.fit.Side != 8 || site.record.CrownAreaRays != 3*8*8 {
		t.Errorf("fit %+v rays %d", value.fit, site.record.CrownAreaRays)
	}
	if _, found := site.store.values[prototypeKey{kind: "spruce", detail: "clumps", seed: 3}]; !found {
		t.Error("the clump prototype was not retained")
	}
	if _, found := site.store.values[prototypeKey{kind: "spruce", detail: "crown", seed: 3, areaSide: 8}]; !found {
		t.Error("the crown was not stored under its area side")
	}
	if len(value.parts) == 0 || value.bounds.Max.Y <= 0 {
		t.Errorf("crown has %d parts and bounds %+v", len(value.parts), value.bounds)
	}
}

// TestLoadCrownWithoutMatching checks the unfitted crown and its refusals.
func TestLoadCrownWithoutMatching(t *testing.T) {
	site := roomySite(t)
	value, err := site.load(prototypeKey{kind: "spruce", detail: "crown", seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	if value.fit != nil || site.record.CrownAreaRays != 0 {
		t.Errorf("an unmatched crown has a fit or rays: %d", site.record.CrownAreaRays)
	}
	probe := roomySite(t)
	if _, err := probe.load(prototypeKey{kind: "spruce", detail: "clumps", seed: 3}); err != nil {
		t.Fatal(err)
	}
	clumpFaces := probe.store.faces
	tight := roomySite(t)
	tight.plan.Budget.Faces = clumpFaces - 1
	if _, err := tight.load(prototypeKey{kind: "spruce", detail: "crown", seed: 3}); err == nil {
		t.Error("clumps over the face budget were accepted")
	}
	exact := roomySite(t)
	exact.plan.Budget.Faces = clumpFaces
	if _, err := exact.load(prototypeKey{kind: "spruce", detail: "crown", seed: 3}); err == nil || !containsText(err.Error(), "crown exceeds retained prototype face budget") {
		t.Errorf("crown over budget: %v", err)
	}
	bad := roomySite(t)
	bad.plan.Forest = &forestPlan{CrownAreaMatch: true, CrownAreaSide: 4}
	if _, err := bad.load(prototypeKey{kind: "spruce", detail: "crown", seed: 3}); err == nil || !containsText(err.Error(), "probe grid") {
		t.Errorf("bad area side: %v", err)
	}
	if _, err := bad.load(prototypeKey{kind: "dragon", detail: "crown", seed: 3}); err == nil {
		t.Error("a crown of an unknown kind was accepted")
	}
}

// TestAddPlacesPartsAndLights checks the placed things, the instance count
// and that a light source is lit in the world.
func TestAddPlacesPartsAndLights(t *testing.T) {
	site := roomySite(t)
	value, err := site.load(prototypeKey{kind: "campfire", detail: "quick", seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	site.relative, _ = frame.NewRelative(maths.V(1, 2, 3))
	site.plan.LightExposure = 1
	pose := maths.NewSubspace(maths.V(11, 12, 13), maths.Identity, 1)
	if err := site.add(value, pose); err != nil {
		t.Fatal(err)
	}
	if len(site.world.Things) != 2 || site.record.Instances != 2 || len(site.world.Lamps) != 1 {
		t.Fatalf("things %d instances %d lamps %d", len(site.world.Things), site.record.Instances, len(site.world.Lamps))
	}
	if got := site.world.Things[0].At(0).Origin; got != maths.V(10, 10, 10) {
		t.Errorf("part origin = %v, want the pose less the render origin", got)
	}
}

// TestAddRefusesOverBudget checks the instance allowance names the counts.
func TestAddRefusesOverBudget(t *testing.T) {
	site := roomySite(t)
	value, _ := site.load(prototypeKey{kind: "campfire", detail: "quick", seed: 3})
	site.plan.Budget.Instances = 1
	err := site.add(value, maths.Local())
	if err == nil || !containsText(err.Error(), "next object needs 2; allowance 1") {
		t.Errorf("error = %v", err)
	}
	if len(site.world.Things) != 0 {
		t.Error("parts were placed despite the refusal")
	}
}

// TestObjectPlacesOnGround checks the position, recorded height and the
// clearing reserved for an authored object.
func TestObjectPlacesOnGround(t *testing.T) {
	site := roomySite(t)
	err := site.object(objectPlan{Name: "tin", Kind: "can", Detail: "quick", Seed: 3, East: 10, South: -20, Above: 1, Heading: 90})
	if err != nil {
		t.Fatal(err)
	}
	if len(site.record.Objects) != 1 || len(site.world.Things) != 1 || len(site.clearings) != 1 {
		t.Fatalf("objects %d things %d clearings %d", len(site.record.Objects), len(site.world.Things), len(site.clearings))
	}
	got := site.record.Objects[0]
	if got.Name != "tin" || got.Kind != "can" || got.Detail != "quick" || got.Ground != 7 {
		t.Errorf("record = %+v", got)
	}
	// One metre above ground at height 7 puts the object 8 m above the sphere's radius.
	if radial := got.Position.Sub(site.frame.Centre()).Len(); !near(radial, 1e6+8, 1e-6) {
		t.Errorf("radial distance = %v, want %v", radial, 1e6+8)
	}
	clearing := site.clearings[0]
	if clearing.east != 10 || clearing.south != -20 || clearing.radius <= 0.03 || clearing.radius > 0.05 {
		t.Errorf("clearing = %+v, want a radius near 0.048 for a can 0.068 wide", clearing)
	}
}

// TestObjectRefusals checks each stage of placement stops the object.
func TestObjectRefusals(t *testing.T) {
	good := objectPlan{Name: "tin", Kind: "can", Detail: "quick", Seed: 3}
	site := roomySite(t)
	if err := site.object(objectPlan{Kind: "dragon", Detail: "quick"}); err == nil {
		t.Error("an unknown kind was accepted")
	}
	site.heights.(*flatHeights).absent = true
	if err := site.object(good); err == nil || !containsText(err.Error(), "no source height") {
		t.Errorf("no ground: %v", err)
	}
	site = roomySite(t)
	site.plan.Budget.Instances = 0
	if err := site.object(good); err == nil || !containsText(err.Error(), "instance budget exhausted") {
		t.Errorf("no instances: %v", err)
	}
	if len(site.clearings) != 0 || len(site.record.Objects) != 0 {
		t.Error("a refused object left a record")
	}
}

// TestLasersNil checks that a site without lasers adds nothing.
func TestLasersNil(t *testing.T) {
	site := roomySite(t)
	if err := site.lasers(); err != nil || len(site.world.Lamps) != 0 {
		t.Errorf("err %v lamps %d", err, len(site.world.Lamps))
	}
}

// TestLasersPlaceBeams checks one lamp per origin, and launcher housings when
// they are asked for.
func TestLasersPlaceBeams(t *testing.T) {
	plan := &laserPlan{Origins: [][3]float64{{0, 0, 0}, {5, 0, 5}}, Azimuth: 90, Altitude: 45,
		Radius: 0.1, Reach: 1000, Divergence: 0.01, Wavelength: 589, Strength: 1}
	site := roomySite(t)
	site.plan.Lasers = plan
	if err := site.lasers(); err != nil {
		t.Fatal(err)
	}
	if len(site.world.Lamps) != 2 || len(site.world.Things) != 0 {
		t.Errorf("lamps %d things %d, want 2 and 0", len(site.world.Lamps), len(site.world.Things))
	}
	launched := roomySite(t)
	withHousing := *plan
	withHousing.Launchers = true
	launched.plan.Lasers = &withHousing
	if err := launched.lasers(); err != nil {
		t.Fatal(err)
	}
	if len(launched.world.Lamps) != 2 || len(launched.world.Things) != 4 || launched.record.Instances != 4 {
		t.Errorf("lamps %d things %d instances %d, want 2, 4 and 4", len(launched.world.Lamps), len(launched.world.Things), launched.record.Instances)
	}
}

// TestLasersRefusals checks the beam, direction and housing refusals.
func TestLasersRefusals(t *testing.T) {
	base := laserPlan{Origins: [][3]float64{{0, 0, 0}}, Azimuth: 90, Altitude: 45,
		Radius: 0.1, Reach: 1000, Divergence: 0.01, Wavelength: 589, Strength: 1}
	cases := []struct {
		name   string
		change func(*laserPlan)
		budget int
	}{
		{"zero radius", func(p *laserPlan) { p.Radius = 0 }, 100},
		{"zero reach", func(p *laserPlan) { p.Reach = 0 }, 100},
		{"altitude beyond the zenith", func(p *laserPlan) { p.Altitude = 91 }, 100},
		{"no room for the housing", func(p *laserPlan) { p.Launchers = true }, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			site := roomySite(t)
			site.plan.Budget.Instances = tc.budget
			plan := base
			tc.change(&plan)
			site.plan.Lasers = &plan
			if err := site.lasers(); err == nil {
				t.Error("no error")
			}
		})
	}
}

// TestCloudLayers checks one shell volume per layer, placed about the
// sphere's centre, and the refusal of a layer below the surface.
func TestCloudLayers(t *testing.T) {
	sphere := testSphere(t)
	relative, _ := frame.NewRelative(maths.V(0, 100, 0))
	world := thing.NewWorld()
	p := plan{Radius: 1e6, Clouds: []cloudPlan{
		{Low: 1000, High: 2000, PatchScale: 100, Extinction: 0.001, Coverage: 0.5, Seed: 1, Colour: [3]float64{1, 1, 1}},
		{Low: 3000, High: 4000, PatchScale: 100, Extinction: 0.001, Coverage: 0.5, Seed: 2}}}
	if err := p.cloudLayers(world, sphere, relative); err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != 2 || world.Volumes[0].Material.Name != "authored planetary cloud" {
		t.Fatalf("%d volumes", len(world.Volumes))
	}
	if got := world.Volumes[0].Motion.At(0).Origin; got != maths.V(0, -1e6-100, 0) {
		t.Errorf("shell centre = %v, want the sphere centre less the render origin", got)
	}
	p.Clouds[0].Low = -5
	if err := p.cloudLayers(thing.NewWorld(), sphere, relative); err == nil {
		t.Error("a shell below the surface was accepted")
	}
	p.Clouds = nil
	if err := p.cloudLayers(thing.NewWorld(), sphere, relative); err != nil {
		t.Errorf("no layers: %v", err)
	}
}

// TestVolumeIllumination checks the legacy, plain and gridded forms and the
// refusals.
func TestVolumeIllumination(t *testing.T) {
	light, err := (plan{}).volumeIllumination()
	if err != nil || light.Samples() != 0 || light.Grid() != nil {
		t.Errorf("legacy: %v samples %d", err, light.Samples())
	}
	plain := plan{VolumeLighting: &volumeLightingPlan{Samples: 8}}
	if light, err = plain.volumeIllumination(); err != nil || light.Samples() != 8 || light.Grid() != nil {
		t.Errorf("plain: %v samples %d", err, light.Samples())
	}
	grid := &incidentGridPlan{Low: [3]float64{-10, 0, -10}, High: [3]float64{10, 20, 10}, Nodes: [3]int{3, 4, 5}}
	gridded := plan{VolumeLighting: &volumeLightingPlan{Samples: 8, Grid: grid}}
	if light, err = gridded.volumeIllumination(); err != nil || light.Grid() == nil {
		t.Fatalf("gridded: %v", err)
	}
	box, nodes := light.Grid().Parameters()
	if box.Min != maths.V(-10, 0, -10) || box.Max != maths.V(10, 20, 10) || nodes != [3]int{3, 4, 5} {
		t.Errorf("grid = %+v %v", box, nodes)
	}
	odd := plan{VolumeLighting: &volumeLightingPlan{Samples: 7, Grid: grid}}
	if _, err := odd.volumeIllumination(); err == nil {
		t.Error("odd sample count accepted")
	}
	flat := *grid
	flat.High = flat.Low
	if _, err := (plan{VolumeLighting: &volumeLightingPlan{Samples: 8, Grid: &flat}}).volumeIllumination(); err == nil {
		t.Error("an empty grid was accepted")
	}
}

// TestRelativeIllumination checks the grid is carried by the render origin
// and that missing lighting or a missing grid passes through.
func TestRelativeIllumination(t *testing.T) {
	origin, _ := frame.NewRelative(maths.V(1, 2, 3))
	if light, err := (plan{}).relativeIllumination(origin); err != nil || light.Samples() != 0 {
		t.Errorf("legacy: %v", err)
	}
	if light, err := (plan{VolumeLighting: &volumeLightingPlan{Samples: 4}}).relativeIllumination(origin); err != nil || light.Samples() != 4 || light.Grid() != nil {
		t.Errorf("plain: %v", err)
	}
	grid := &incidentGridPlan{Low: [3]float64{-10, 0, -10}, High: [3]float64{10, 20, 10}, Nodes: [3]int{2, 2, 2}}
	light, err := (plan{VolumeLighting: &volumeLightingPlan{Samples: 4, Grid: grid}}).relativeIllumination(origin)
	if err != nil {
		t.Fatal(err)
	}
	box, nodes := light.Grid().Parameters()
	if box.Min != maths.V(-11, -2, -13) || box.Max != maths.V(9, 18, 7) || nodes != [3]int{2, 2, 2} || light.Samples() != 4 {
		t.Errorf("grid = %+v %v", box, nodes)
	}
	if _, err := (plan{VolumeLighting: &volumeLightingPlan{Samples: 5}}).relativeIllumination(origin); err == nil {
		t.Error("odd sample count accepted")
	}
	huge, _ := frame.NewRelative(maths.V(math.MaxFloat64, 0, 0))
	if _, err := (plan{VolumeLighting: &volumeLightingPlan{Samples: 4, Grid: grid}}).relativeIllumination(huge); err == nil {
		t.Error("an origin that overflows the grid was accepted")
	}
}

// TestAddRefusesUnlitExposure checks that a light source with no exposure
// is refused.
func TestAddRefusesUnlitExposure(t *testing.T) {
	site := roomySite(t)
	value, _ := site.load(prototypeKey{kind: "campfire", detail: "quick", seed: 3})
	if err := site.add(value, maths.Local()); err == nil || !containsText(err.Error(), "exposure") {
		t.Errorf("error = %v", err)
	}
}
