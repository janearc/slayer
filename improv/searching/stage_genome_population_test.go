package searching

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	jobpkg "github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func populationSail(colour, glow [3]float64, rate float64) SailPlan {
	return SailPlan{Count: 3, Segments: 12, SpanSegments: 3,
		Length: 2, Width: .7, Sweep: .4, Camber: .2, TwistDegrees: 15,
		SpinRate: rate, Colour: colour, Glow: glow}
}

func examplePopulation() []AuthoredBeingPlan {
	secondSail := populationSail([3]float64{.2, .1, .8}, [3]float64{.02, .01, .1}, -.5)
	secondSail.Rotation = [3]float64{0, 25, 0}
	return []AuthoredBeingPlan{
		{Identity: 4, At: [3]float64{-3, 0, 0}, Scale: 1,
			Anatomy: &AnatomyPlan{
				Cores: []CorePlan{{Form: "sphere", Radius: .7,
					Colour: [3]float64{.2, .4, .8}, Glow: [3]float64{.01, .02, .1}}},
				Sails: []SailPlan{populationSail([3]float64{.8, .2, .1}, [3]float64{.1, .02, .01}, .3)},
			}},
		{Identity: 9, At: [3]float64{3, 0, 0}, Scale: 1.4,
			Anatomy: &AnatomyPlan{
				Cores: []CorePlan{{Form: "octahedron", Radius: .9,
					Colour: [3]float64{.1, .8, .2}, Glow: [3]float64{.02, .1, .01}}},
				Sails: []SailPlan{secondSail},
			}},
	}
}

func populationStagePlan(population []AuthoredBeingPlan) *StagePlan {
	return &StagePlan{Kind: "authored-multitude", Population: population,
		Eye: [3]float64{0, 3, 24}, EndEye: [3]float64{1, 4, 22},
		Look: [3]float64{0, 0, 0}, Wide: 60, Exposure: 1.6}
}

func TestAuthoredStagePlanJSONRoundTripKeepsYAMLFieldNames(t *testing.T) {
	disabled := false
	plan := &StagePlan{Kind: "individual", Anatomy: exampleAnatomy(), ShareAnatomyPrototypes: &disabled}
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var decoded StagePlan
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Anatomy == nil || decoded.Anatomy.Cores[0].Form != plan.Anatomy.Cores[0].Form ||
		decoded.ShareAnatomyPrototypes == nil || *decoded.ShareAnatomyPrototypes {
		t.Fatalf("anatomy JSON round-trip changed accepted field names: %s", encoded)
	}
	plan = populationStagePlan(examplePopulation())
	plan.ShareAnatomyPrototypes = &disabled
	encoded, err = json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	decoded = StagePlan{}
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Population) != len(plan.Population) || decoded.Population[0].Anatomy == nil ||
		decoded.ShareAnatomyPrototypes == nil || *decoded.ShareAnatomyPrototypes {
		t.Fatalf("population JSON round-trip changed accepted field names: %s", encoded)
	}
}

func TestAuthoredMultitudeComposesDistinctBeingsAndSharesOnlyGeometry(t *testing.T) {
	plan := populationStagePlan(examplePopulation())
	stage, err := newHeardStage(plan, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stage.crowd.beings) != 2 || len(stage.world.Things) != 8 {
		t.Fatalf("beings/surfaces = %d/%d, want 2/8", len(stage.crowd.beings), len(stage.world.Things))
	}
	first, second := stage.crowd.beings[0], stage.crowd.beings[1]
	firstSail := first.parts[1:]
	secondSail := second.parts[1:]
	shared := firstSail[0].part.Shape
	for _, part := range append(firstSail, secondSail...) {
		if part.part.Shape != shared {
			t.Fatal("identical authored sails did not share their immutable mesh")
		}
	}
	if firstSail[0].part.Material == secondSail[0].part.Material ||
		firstSail[0].local.Orientation == secondSail[0].local.Orientation ||
		first.at == second.at || first.scale == second.scale {
		t.Fatal("shared geometry collapsed independent materials or placements")
	}
	if _, ok := first.parts[0].part.Shape.(shape.Sphere); !ok {
		t.Fatal("first being lost its sphere body")
	}
	if _, ok := second.parts[0].part.Shape.(*mesh.Mesh); !ok {
		t.Fatal("second being lost its octahedral body")
	}
	things := make([]thing.Thing, len(stage.world.Things))
	for index, part := range stage.world.Things {
		things[index] = *part
	}
	stored := thing.Storage(things)
	const sailFaces = 12 * 3 * 2
	if stored.UniqueMeshes != 2 || stored.StoredFaces != 8+sailFaces ||
		stored.ReferencedFaces != 8+6*sailFaces {
		t.Fatalf("geometry storage did not count shared instance meshes: %+v", stored)
	}

	eye, err := stage.update(plan, State{At: 2}, .5)
	if err != nil || eye.Origin != maths.V(.5, 3.5, 23) {
		t.Fatalf("authored population camera = %v, err %v", eye.Origin, err)
	}
	if firstSail[0].part.At(0).Orientation == secondSail[0].part.At(0).Orientation ||
		firstSail[0].part.Material.Colour != maths.RGB(.8, .2, .1) ||
		secondSail[0].part.Material.Colour != maths.RGB(.2, .1, .8) ||
		firstSail[0].part.Material.Glow.Luminance() == 0 ||
		secondSail[0].part.Material.Glow.Luminance() == 0 {
		t.Fatal("population tick lost independent motion, colour, or emission")
	}
}

func TestAuthoredMultitudeCanDisablePrototypeSharingWithoutChangingScene(t *testing.T) {
	sharedPlan := populationStagePlan(examplePopulation())
	separatePlan := populationStagePlan(examplePopulation())
	disabled := false
	separatePlan.ShareAnatomyPrototypes = &disabled
	shared, err := newHeardStage(sharedPlan, "")
	if err != nil {
		t.Fatal(err)
	}
	separate, err := newHeardStage(separatePlan, "")
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
		if shared.crowd.beings[index/4].parts[index%4].local != separate.crowd.beings[index/4].parts[index%4].local {
			t.Fatalf("thing %d local pose changed", index)
		}
	}
	sharedThings := make([]thing.Thing, len(shared.world.Things))
	separateThings := make([]thing.Thing, len(separate.world.Things))
	for index := range shared.world.Things {
		sharedThings[index], separateThings[index] = *shared.world.Things[index], *separate.world.Things[index]
	}
	sharedStorage, separateStorage := thing.Storage(sharedThings), thing.Storage(separateThings)
	if sharedStorage.UniqueMeshes != 2 || separateStorage.UniqueMeshes != 7 ||
		sharedStorage.ReferencedFaces != separateStorage.ReferencedFaces ||
		sharedStorage.StoredFaces >= separateStorage.StoredFaces {
		t.Fatalf("sharing storage = %+v, separate storage = %+v", sharedStorage, separateStorage)
	}
	for _, at := range []float64{.75, 1.5, 4.25} {
		sharedEye, sharedErr := shared.update(sharedPlan, State{At: at}, .37)
		separateEye, separateErr := separate.update(separatePlan, State{At: at}, .37)
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

func assertSameAnatomyGeometry(t *testing.T, shared, separate interface{}) {
	t.Helper()
	sharedMesh, sharedOK := shared.(*mesh.Mesh)
	separateMesh, separateOK := separate.(*mesh.Mesh)
	if sharedOK != separateOK {
		t.Fatalf("shape types differ: %T and %T", shared, separate)
	}
	if sharedOK {
		if sharedMesh == separateMesh {
			t.Fatal("explicitly unshared anatomy retained a shared mesh")
		}
		if !reflect.DeepEqual(sharedMesh.Points(), separateMesh.Points()) ||
			!reflect.DeepEqual(sharedMesh.Faces(), separateMesh.Faces()) {
			t.Fatal("prototype sharing changed mesh geometry")
		}
	}
	if !sharedOK && !reflect.DeepEqual(shared, separate) {
		t.Fatalf("nonmesh anatomy geometry changed: %#v vs %#v", shared, separate)
	}
}

func TestAuthoredMultitudeHasStrictSceneInputAndTotalResourceBound(t *testing.T) {
	with := map[string]any{
		"stage": map[string]any{
			"kind": "authored-multitude",
			"eye":  []float64{0, 3, 24}, "end-eye": []float64{1, 4, 22},
			"look": []float64{0, 0, 0}, "wide": 60.0, "exposure": 1.6,
			"share-anatomy-prototypes": false,
			"population":               examplePopulation(),
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
	if _, err = readStageJob(decoded); err != nil {
		t.Fatal(err)
	}
	stagePlan, err := readStageJob(decoded)
	if err != nil || stagePlan.Stage.ShareAnatomyPrototypes == nil || *stagePlan.Stage.ShareAnatomyPrototypes {
		t.Fatalf("explicit prototype sharing opt-out was not parsed: %#v, %v", stagePlan.Stage.ShareAnatomyPrototypes, err)
	}
	badKind := stagePlan.Stage
	badKind.Kind = "lantern"
	if err := badKind.check(); err == nil {
		t.Fatal("prototype sharing control was accepted on an unsupported stage")
	}
	individualWithoutAnatomy := stagePlan.Stage
	individualWithoutAnatomy.Kind = "individual"
	individualWithoutAnatomy.Anatomy = nil
	individualWithoutAnatomy.Population = nil
	if err := individualWithoutAnatomy.check(); err == nil {
		t.Fatal("prototype sharing control was accepted without authored individual anatomy")
	}
	badStage := decoded["stage"].(map[string]any)
	population := badStage["population"].([]any)
	member := population[1].(map[string]any)
	anatomy := member["anatomy"].(map[string]any)
	anatomy["surprise"] = true
	if _, err = readStageJob(decoded); err == nil {
		t.Fatal("unknown population anatomy field was accepted")
	}
	delete(anatomy, "surprise")
	badStage["share-anatomy-prototypess"] = false
	if _, err = readStageJob(decoded); err == nil {
		t.Fatal("unknown prototype sharing field was accepted")
	}
	tooMuch := make([]AuthoredBeingPlan, 2)
	for index := range tooMuch {
		cores := make([]CorePlan, maxAnatomyParts)
		for i := range cores {
			cores[i] = CorePlan{Form: "sphere", Radius: 1}
		}
		tooMuch[index] = AuthoredBeingPlan{Scale: 1,
			Anatomy: &AnatomyPlan{Cores: cores}}
	}
	if err = populationStagePlan(tooMuch).check(); err == nil {
		t.Fatal("total population work exceeded its bound")
	}
	tooManyFaces := make([]AuthoredBeingPlan, 6)
	for index := range tooManyFaces {
		tooManyFaces[index] = AuthoredBeingPlan{Scale: 1,
			Anatomy: &AnatomyPlan{Sails: []SailPlan{{
				Count: 10, Segments: 128, SpanSegments: 16,
				Length: 1, Width: 1,
			}}}}
	}
	if err = populationStagePlan(tooManyFaces).check(); err == nil {
		t.Fatal("total population face work exceeded its bound")
	}
}

func TestAnatomyPrototypeSharingSettingChangesJobPictureKey(t *testing.T) {
	defaultKey := prototypeSharingJobKey(t, nil)
	disabled := false
	separateKey := prototypeSharingJobKey(t, &disabled)
	enabled := true
	explicitKey := prototypeSharingJobKey(t, &enabled)
	if defaultKey == separateKey || defaultKey == explicitKey || separateKey == explicitKey {
		t.Fatalf("omitted/false/true settings did not remain distinct in picture keys: %s / %s / %s",
			defaultKey, separateKey, explicitKey)
	}
}

func prototypeSharingJobKey(t *testing.T, sharing *bool) string {
	t.Helper()
	stage := map[string]any{"kind": "authored-multitude"}
	if sharing != nil {
		stage["share-anatomy-prototypes"] = *sharing
	}
	jobWire := map[string]any{
		"$schema":  "https://github.com/janearc/slayer/schemas/job-v1.json",
		"scene":    map[string]any{"named": "heard", "with": map[string]any{"stage": stage}},
		"renderer": "cpu", "film": map[string]any{"width": 32, "height": 24, "rays": 1},
	}
	raw, err := json.Marshal(jobWire)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "job.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	read, err := jobpkg.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	return read.PictureKey()
}
