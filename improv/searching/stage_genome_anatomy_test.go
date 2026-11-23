package searching

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func exampleAnatomy() *anatomyPlan {
	return &anatomyPlan{
		Cores: []corePlan{{
			Form: "octahedron", Radius: 1.5,
			SpinRate: .2,
			Colour:   [3]float64{.1, .3, .6}, Glow: [3]float64{.02, .01, .08},
		}},
		Bands: []bandPlan{{
			Form: "undulating", Count: 2, Segments: 24, Radius: 2, Width: .4, Thickness: .05,
			Spacing: .6, Eccentricity: .2, WarpShare: .3, Waves: 2, SpinRate: .4,
		}},
		Sails: []sailPlan{{
			Count: 3, Segments: 12, SpanSegments: 4, Length: 2, Width: .9,
			Sweep: .45, Camber: .35, TwistDegrees: 25, SpinRate: .8,
			Colour: [3]float64{.2, .15, .1}, Glow: [3]float64{.03, .01, .02},
		}},
	}
}

func TestAuthoredIndividualCanDisablePrototypeSharingWithoutChangingScene(t *testing.T) {
	sharedPlan := &stagePlan{Kind: "individual", Anatomy: exampleAnatomy(),
		Eye: [3]float64{2, 3, 12}, EndEye: [3]float64{2, 3, 12},
		Look: [3]float64{}, Wide: 65, Exposure: 1}
	separatePlan := *sharedPlan
	separatePlan.Anatomy = exampleAnatomy()
	disabled := false
	separatePlan.ShareAnatomyPrototypes = &disabled
	shared, err := newHeardStage(sharedPlan, "")
	if err != nil {
		t.Fatal(err)
	}
	separate, err := newHeardStage(&separatePlan, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(shared.world.Things) != len(separate.world.Things) {
		t.Fatalf("shared/separate thing counts = %d/%d", len(shared.world.Things), len(separate.world.Things))
	}
	for index := range shared.world.Things {
		sharedThing, separateThing := shared.world.Things[index], separate.world.Things[index]
		assertSameAnatomyGeometry(t, sharedThing.Shape, separateThing.Shape)
		if sharedThing.Material == separateThing.Material || !reflect.DeepEqual(sharedThing.Material, separateThing.Material) {
			t.Fatalf("thing %d material ownership/value changed", index)
		}
		if shared.crowd.beings[0].parts[index].local != separate.crowd.beings[0].parts[index].local {
			t.Fatalf("thing %d local pose changed", index)
		}
	}
	for _, at := range []float64{.5, 2.25, 7} {
		sharedEye, sharedErr := shared.update(sharedPlan, State{At: at}, .4)
		separateEye, separateErr := separate.update(&separatePlan, State{At: at}, .4)
		if sharedErr != nil || separateErr != nil || sharedEye != separateEye {
			t.Fatalf("time %g eyes/errors = %+v/%+v, %v/%v", at, sharedEye, separateEye, sharedErr, separateErr)
		}
		for index := range shared.world.Things {
			sharedThing, separateThing := shared.world.Things[index], separate.world.Things[index]
			if sharedThing.At(0) != separateThing.At(0) || !reflect.DeepEqual(sharedThing.Material, separateThing.Material) {
				t.Fatalf("time %g thing %d motion/material changed", at, index)
			}
		}
	}
}

func TestAuthoredAnatomyComposesIndependentTopologyAndDimensions(t *testing.T) {
	plan := exampleAnatomy()
	stage, err := newHeardStage(&stagePlan{
		Kind: "individual", Anatomy: plan,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stage.world.Things) != 6 {
		t.Fatalf("parts = %d, want core + two bands + three sails", len(stage.world.Things))
	}
	being := stage.crowd.beings[0]
	if one, two := being.parts[3].part.Shape, being.parts[4].part.Shape; one != two || one != being.parts[5].part.Shape {
		t.Fatal("identical sails did not share their immutable mesh")
	}
	if being.parts[3].part.Material == being.parts[4].part.Material ||
		being.parts[3].local.Orientation == being.parts[4].local.Orientation {
		t.Fatal("sail instances lost independent material or pose")
	}
	faces := make(map[int]bool)
	closed, open := false, false
	for _, part := range stage.world.Things {
		solid, ok := part.Shape.(*mesh.Mesh)
		if !ok {
			continue
		}
		faces[solid.FaceCount()] = true
		closed = closed || solid.Closed()
		open = open || !solid.Closed()
	}
	if !faces[8] || !closed || !open {
		t.Fatalf("mesh topology faces=%v closed=%t open=%t", faces, closed, open)
	}

	otherCore, err := newHeardStage(&stagePlan{Kind: "individual", Anatomy: &anatomyPlan{
		Cores: []corePlan{
			{Form: "octahedron", Radius: 1, Colour: [3]float64{.2, .1, .1}},
			{Form: "octahedron", Radius: 2, At: [3]float64{3, 0, 0}, Colour: [3]float64{.1, .2, .3}},
		},
	}}, "")
	if err != nil {
		t.Fatal(err)
	}
	cores := otherCore.crowd.beings[0].parts
	if cores[0].part.Shape != cores[1].part.Shape || cores[0].local == cores[1].local {
		t.Fatal("unit octahedra did not share geometry with distinct placements")
	}

	eye, err := stage.update(&stagePlan{Kind: "individual", Identity: 9,
		Eye: [3]float64{2, 3, 12}, EndEye: [3]float64{2, 3, 12},
		Look: [3]float64{0, 0, 0}, Wide: 65, Exposure: 1, Anatomy: plan},
		State{At: 1}, 0)
	if err != nil || eye.Origin != maths.V(2, 3, 12) {
		t.Fatalf("authored camera origin = %v, err %v", eye.Origin, err)
	}
	if being.parts[3].part.At(0).Orientation == being.parts[4].part.At(0).Orientation {
		t.Fatal("authored sails did not move independently on stage tick")
	}
	if being.parts[0].part.Material.Colour != maths.RGB(.1, .3, .6) ||
		being.parts[0].part.Material.Glow.Luminance() == 0 ||
		being.parts[3].part.Material.Glow.Luminance() == 0 {
		t.Fatalf("stage tick lost authored appearance: core colour=%v glow=%v; sail glow=%v",
			being.parts[0].part.Material.Colour, being.parts[0].part.Material.Glow,
			being.parts[3].part.Material.Glow)
	}

	other := exampleAnatomy()
	other.Cores = append(other.Cores, corePlan{
		Form: "sphere", Radius: .45, At: [3]float64{3, 0, 0},
		Colour: [3]float64{.3, .1, .05},
	})
	other.Bands = nil
	other.Sails[0].Count = 1
	other.Sails[0].Sweep = -.8
	second, err := newHeardStage(&stagePlan{Kind: "individual", Anatomy: other}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(second.world.Things) != 3 {
		t.Fatalf("alternate parts = %d, want two cores and one sail", len(second.world.Things))
	}
}

func TestAuthoredAnatomyIsStrictlyValidatedInHeardSceneInput(t *testing.T) {
	with := map[string]any{
		"stage": map[string]any{
			"kind": "individual",
			"eye":  []float64{0, 0, 12}, "end-eye": []float64{0, 0, 12},
			"look": []float64{0, 0, 0}, "wide": 65.0, "exposure": 1.0,
			"anatomy": exampleAnatomy(),
		},
		"from": 0.0, "until": 4.0, "bpm": 66.0,
	}
	raw, err := json.Marshal(with)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	job, err := readStageJob(decoded)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := newHeardStage(&job.Stage, "")
	if err != nil || len(stage.world.Things) != 6 {
		t.Fatalf("authored stage has %d surfaces: %v", len(stage.world.Things), err)
	}

	bad := map[string]any{
		"stage": map[string]any{
			"kind": "individual",
			"eye":  []float64{0, 0, 12}, "end-eye": []float64{0, 0, 12},
			"look": []float64{0, 0, 0}, "wide": 65.0, "exposure": 1.0,
			"anatomy": map[string]any{"cores": []any{}, "surprise": true},
		},
		"from": 0.0, "until": 4.0, "bpm": 66.0,
	}
	if _, err = readStageJob(bad); err == nil {
		t.Fatal("unknown anatomy field was accepted")
	}
}

func TestAuthoredAnatomyRefusesInvalidOrUnboundedComponents(t *testing.T) {
	for _, plan := range []*anatomyPlan{
		{},
		{Cores: []corePlan{{Form: "sphere", Radius: 0}}},
		{Bands: []bandPlan{{Form: "annular", Count: 1, Segments: 7, Radius: 1, Width: 1, Thickness: .1, Spacing: .1}}},
		{Sails: []sailPlan{{Count: 0, Segments: 12, Length: 1, Width: 1}}},
		{Cores: []corePlan{{Form: "sphere", Radius: 1}}, Bands: []bandPlan{{Form: "annular", Count: 1, Segments: 12, Radius: 1, Width: 1, Thickness: .1, Spacing: .1, Eccentricity: 1}}},
	} {
		if err := plan.check(); err == nil {
			t.Fatalf("invalid anatomy plan was accepted: %#v", plan)
		}
	}
	if err := (*anatomyPlan)(nil).check(); err == nil {
		t.Fatal("nil anatomy plan was accepted")
	}
	if err := (&anatomyPlan{Cores: []corePlan{{Form: "sphere", Radius: 1}}}).check(); err != nil {
		t.Fatal(err)
	}
}
