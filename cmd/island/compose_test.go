package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/thing"
)

// composePlan is a plan whose site sits at the test sphere's anchor with a
// can, a saguaro forest, a mist bank and a laser, over a level source.
func composePlan(t *testing.T) *plan {
	t.Helper()
	site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	p := smallPlan()
	p.Site = &sitePlan{At: geographicPoint{Longitude: 10, Latitude: 20}, LightExposure: 1,
		Objects: []objectPlan{{Name: "tin", Kind: "can", Detail: "quick", Seed: 3, East: 0, South: 0}},
		Forest:  site.plan.Forest,
		Atmosphere: []mistPlan{{PatchScale: 50, Extinction: 0.01, Depth: 20, Coverage: 0.5, Base: 10, Seed: 1,
			Bounds:    localBounds{MinEast: -20, MinSouth: -20, MaxEast: 20, MaxSouth: 20},
			FloorSide: 3, FloorTolerance: 1e-6, FloorIterations: 20}},
		Lasers: &laserPlan{Origins: [][3]float64{{0, 0, 0}}, Azimuth: 90, Altitude: 45, Radius: 0.1,
			Reach: 1000, Divergence: 0.01, Wavelength: 589, Strength: 1}}
	p.Site.Budget.MistFloors, p.Site.Budget.Cells, p.Site.Budget.Candidates = 1024, 100000, 100000
	p.Site.Budget.Instances, p.Site.Budget.Faces = 100000, 1<<30
	p.Site.Forest.Thresholds = [4]float64{2, 4, 12, 40}
	return p
}

// composeInputs gathers the arguments compose needs around a level source.
func composeInputs(t *testing.T, p *plan) (*thing.World, *flatHeights, *prototypeStore, *floorStore, frame.Relative) {
	t.Helper()
	origin, err := frame.NewRelative(maths.V(0, 40, 0))
	if err != nil {
		t.Fatal(err)
	}
	var limit int64
	if p.Site != nil {
		limit = p.Site.Budget.MistFloors
	}
	return thing.NewWorld(), &flatHeights{height: 7}, &prototypeStore{values: map[prototypeKey]prototype{}}, &floorStore{limit: limit}, origin
}

// TestComposeWithoutSite checks that a plan with no site returns no record.
func TestComposeWithoutSite(t *testing.T) {
	p := smallPlan()
	world, heights, store, floors, origin := composeInputs(t, p)
	globe := testSphere(t)
	looking := lookDown(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	record, err := p.compose(t.Context(), world, globe, heights, looking, origin, store, floors)
	if record != nil || err != nil {
		t.Errorf("record %v error %v", record, err)
	}
}

// TestComposeBuildsSite checks the whole site: its parts are placed, the
// forest paint is linked to the ground texture, and the record totals match
// the stores.
func TestComposeBuildsSite(t *testing.T) {
	p := composePlan(t)
	world, heights, store, floors, origin := composeInputs(t, p)
	paint := paintedWorld(&siteBuilder{world: world})
	paint.forest = nil
	globe := testSphere(t)
	looking := lookDown(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	record, err := p.compose(t.Context(), world, globe, heights, looking, origin, store, floors)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Objects) != 1 || record.Objects[0].Name != "tin" || record.Objects[0].Ground != 7 {
		t.Errorf("objects = %+v", record.Objects)
	}
	if record.Trees == 0 || len(world.Volumes) != 1 || len(world.Lamps) != 1 {
		t.Errorf("trees %d volumes %d lamps %d", record.Trees, len(world.Volumes), len(world.Lamps))
	}
	if record.Prototypes != len(store.values) || record.StoredFaces != store.faces || record.MistFloorBytes != 72 || floors.bytes != 72 {
		t.Errorf("prototypes %d/%d faces %d/%d mist bytes %d", record.Prototypes, len(store.values), record.StoredFaces, store.faces, record.MistFloorBytes)
	}
	if paint.forest == nil {
		t.Fatal("the ground texture was not given the forest")
	}
	diameter := math.Sqrt(8*9 + 18*18)
	if !near(paint.forest.fadeFrom, diameter/4, 1e-12) || !near(paint.forest.fadeTo, diameter/2, 1e-12) {
		t.Errorf("fade runs %v to %v, want %v to %v", paint.forest.fadeFrom, paint.forest.fadeTo, diameter/4, diameter/2)
	}
	if len(paint.forest.clearings) != 1 || paint.forest.clearings[0].east != 0 || paint.forest.bounds != p.Site.Forest.Bounds {
		t.Errorf("forest paint = %+v", paint.forest)
	}
}

// TestComposeSortsCrownFits checks that retained crown fits are listed by
// kind, then seed, then sampling side.
func TestComposeSortsCrownFits(t *testing.T) {
	p := smallPlan()
	p.Site = &sitePlan{At: geographicPoint{Longitude: 10, Latitude: 20}}
	world, heights, store, floors, origin := composeInputs(t, p)
	fit := func(side int) *crownAreaFit { return &crownAreaFit{Side: side} }
	for key, side := range map[prototypeKey]int{
		{kind: "spruce", seed: 9, detail: "crown"}:               10,
		{kind: "fern", seed: 4, detail: "crown"}:                 11,
		{kind: "spruce", seed: 2, detail: "crown"}:               12,
		{kind: "spruce", seed: 2, detail: "crown", areaSide: 16}: 16,
		{kind: "spruce", seed: 2, detail: "crown", areaSide: 8}:  8,
	} {
		store.values[key] = prototype{fit: fit(side)}
	}
	store.values[prototypeKey{kind: "oak", seed: 1, detail: "clumps"}] = prototype{}
	globe := testSphere(t)
	looking := lookDown(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	record, err := p.compose(t.Context(), world, globe, heights, looking, origin, store, floors)
	if err != nil {
		t.Fatal(err)
	}
	var got [][3]any
	for _, fit := range record.CrownFits {
		got = append(got, [3]any{fit.Kind, fit.Seed, fit.Fit.Side})
	}
	want := [][3]any{{"fern", uint64(4), 11}, {"spruce", uint64(2), 8}, {"spruce", uint64(2), 12}, {"spruce", uint64(2), 16}, {"spruce", uint64(9), 10}}
	if len(got) != len(want) {
		t.Fatalf("fits = %v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("fit %d = %v, want %v", index, got[index], want[index])
		}
	}
	if record.Prototypes != 6 {
		t.Errorf("prototypes = %d, want 6", record.Prototypes)
	}
}

// TestComposeRefusals checks each refusal before and during composition.
func TestComposeRefusals(t *testing.T) {
	view := lookDown(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	cases := []struct {
		name   string
		change func(*plan, *flatHeights, **floorStore)
		want   string
	}{
		{"no floor store", func(_ *plan, _ *flatHeights, f **floorStore) { *f = nil }, "cache budget changed"},
		{"floor budget changed", func(_ *plan, _ *flatHeights, f **floorStore) { (*f).limit = 5 }, "cache budget changed"},
		{"site without source height", func(_ *plan, h *flatHeights, _ **floorStore) { h.absent = true }, "site has no source height"},
		{"source failure", func(_ *plan, h *flatHeights, _ **floorStore) { h.fail = errBoom }, "boom"},
		{"unknown object", func(p *plan, _ *flatHeights, _ **floorStore) { p.Site.Objects[0].Kind = "dragon" }, "dragon"},
		{"forest refusal", func(p *plan, _ *flatHeights, _ **floorStore) { p.Site.Forest.Radius = 0.1 }, "visibility envelope"},
		{"mist refusal", func(p *plan, _ *flatHeights, _ **floorStore) { p.Site.Atmosphere[0].FloorSide = 1 }, "mist floor sampling side"},
		{"laser refusal", func(p *plan, _ *flatHeights, _ **floorStore) { p.Site.Lasers.Radius = 0 }, "invalid beam"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := composePlan(t)
			world, heights, store, floors, origin := composeInputs(t, p)
			tc.change(p, heights, &floors)
			record, err := p.compose(t.Context(), world, testSphere(t), heights, view, origin, store, floors)
			if err == nil || record != nil || !containsText(err.Error(), tc.want) {
				t.Errorf("record %v error %v, want one containing %q", record, err, tc.want)
			}
		})
	}
}
