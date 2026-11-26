package draft

import (
	"math"
	"strings"
	"testing"
)

const poseSource = `# Keep the scene note.
$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Pose study
  film: {width: 640, height: 480, rays: 1, exposure: 2000, wide: 45}
  cameras:
    viewer:
      location: {at: [80, 60, 70]}
      orientation: {look_at: [30, 40, 50]}
  setting: space
  set:
    things:
      craft:
        kind: mcrn-larson
        seed: 7
        east: 30 # Keep the east note.
        north: 40
        height: {value: 50, unit: m}
        heading: {value: 60, unit: deg}
      companion:
        kind: webb
        seed: 9
        east: 10
        north: 20
        height: {value: 2, unit: m}
  world:
    imports:
      craft:
        file: model.gltf
        at: [3, 4, 5]
        heading: {value: 10, unit: deg}
        materials: {hull: aluminium}
        source: eye
        license: CC0
`

func poseDraft(t *testing.T, source string) *Document {
	t.Helper()
	document, err := Open("pose.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Check(); err != nil {
		t.Fatal(err)
	}
	return document
}

func TestObjectsExposeStableDistinctImmutablePlacements(t *testing.T) {
	document := poseDraft(t, poseSource)
	objects, err := document.Objects()
	if err != nil || len(objects) != 3 {
		t.Fatalf("Objects = %v, %v", objects, err)
	}
	craft := objects[0]
	heading, explicit := craft.Heading()
	if craft.Name() != "craft" || craft.Kind() != "mcrn-larson" || craft.Position() != [3]float64{30, 40, 50} || craft.Unit() != "m" || heading != 60 || !explicit {
		t.Fatalf("craft = %+v", craft)
	}
	if objects[2].ID() == craft.ID() || objects[2].Kind() != "import" {
		t.Fatal("collection identities collided")
	}
	objects[0] = Object{}
	again, _ := document.Objects()
	if again[0].ID() != craft.ID() || document.Text() != poseSource {
		t.Fatal("snapshot changed document")
	}
}

func TestChangePoseMovesOnlySelectedObjectAndRestoresFromText(t *testing.T) {
	document := poseDraft(t, poseSource)
	objects, _ := document.Objects()
	if len(objects) == 0 {
		t.Fatal("missing objects")
	}
	id := objects[0].ID()
	err := document.ChangePose(PoseChange{ID: id, East: 2, North: -3, Up: 4, Heading: 5, LengthUnit: "m", AngleUnit: "deg"})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := document.Objects()
	heading, _ := after[0].Heading()
	if after[0].Position() != [3]float64{32, 37, 54} || heading != 65 || after[0].ID() != id {
		t.Fatalf("changed = %+v", after[0])
	}
	if after[1] != objects[1] || after[2] != objects[2] {
		t.Fatal("edit moved another object")
	}
	for _, text := range []string{"Keep the scene note.", "Keep the east note.", "seed: 7", "license: CC0"} {
		if !strings.Contains(document.Text(), text) {
			t.Fatalf("lost %s", text)
		}
	}
	document.Replace(poseSource)
	if document.Dirty() {
		t.Fatal("text undo did not restore saved source")
	}
	restored, _ := document.Objects()
	if restored[0] != objects[0] {
		t.Fatal("text undo did not restore pose")
	}
}

func TestGroundedPoseKeepsHeightRelativeAndOffersScale(t *testing.T) {
	source := strings.Replace(poseSource, "setting: space", "setting: ground", 1)
	document := poseDraft(t, source)
	objects, err := document.Objects()
	if err != nil || len(objects) != 3 || !objects[0].Grounded() || !objects[0].ScaleEditable() {
		t.Fatalf("ground placements = %+v, %v", objects, err)
	}
	if err := document.ChangePose(PoseChange{ID: objects[0].ID(), East: 2, North: -3, Heading: 5, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	changed, err := document.Objects()
	if err != nil || changed[0].Position() != [3]float64{32, 37, 50} {
		t.Fatalf("ground move = %+v, %v", changed, err)
	}
	if !strings.Contains(document.Text(), "Keep the east note.") {
		t.Fatal("ground move lost a source comment")
	}
}

func TestChangePoseImportsAndOmittedHeading(t *testing.T) {
	document := poseDraft(t, poseSource)
	objects, _ := document.Objects()
	if len(objects) != 3 {
		t.Fatal("missing objects")
	}
	for _, index := range []int{1, 2} {
		if err := document.ChangePose(PoseChange{ID: objects[index].ID(), East: 1, Heading: 15, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := document.Objects()
	angle, said := after[1].Heading()
	if angle != 15 || !said || after[2].Position() != [3]float64{4, 4, 5} {
		t.Fatal("wrong import or default heading")
	}
}

func TestChangePoseRefusalsAndNoopPreserveExactSource(t *testing.T) {
	document := poseDraft(t, poseSource)
	objects, _ := document.Objects()
	if len(objects) == 0 {
		t.Fatal("missing objects")
	}
	valid := PoseChange{ID: objects[0].ID(), LengthUnit: "m", AngleUnit: "deg"}
	if err := document.ChangePose(valid); err != nil || document.Text() != poseSource || document.Dirty() {
		t.Fatal("noop changed source", err)
	}
	changes := []PoseChange{valid, valid, valid, valid, valid, valid, valid, valid}
	changes[0].ID = "unknown"
	changes[1].LengthUnit = "km"
	changes[2].AngleUnit = "rad"
	changes[3].East = math.NaN()
	changes[4].Up = math.Inf(1)
	changes[5].East = -1e10
	changes[6].Heading = 400
	changes[7].Up = -100
	for _, change := range changes {
		if err := document.ChangePose(change); err == nil {
			t.Fatalf("accepted %+v", change)
		}
		if document.Text() != poseSource || document.Dirty() {
			t.Fatal("refusal changed source")
		}
	}
	document.Replace("scene: [unfinished")
	before := document.Text()
	if _, err := document.Objects(); err == nil {
		t.Fatal("accepted invalid source")
	}
	if err := document.ChangePose(valid); err == nil || document.Text() != before {
		t.Fatal("edited invalid source")
	}
}

func TestObjectsRefuseUnsupportedAndInvalidUnits(t *testing.T) {
	for _, source := range []string{
		strings.Replace(poseSource, "unit: m}", "unit: km}", 1),
		strings.Replace(poseSource, "unit: deg}", "unit: rad}", 1),
	} {
		document, err := Open("pose.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := document.Objects(); err == nil {
			t.Fatal("accepted invalid unit")
		}
	}
	document := openDraft(t)
	if _, err := document.Objects(); err == nil {
		t.Fatal("offered objects in job")
	}
	terrain := poseDraft(t, strings.Replace(poseSource, "setting: space", "setting: ground", 1))
	objects, err := terrain.Objects()
	if err != nil || len(objects) != 3 || !objects[0].Grounded() || objects[0].Position()[2] != 50 || objects[2].Kind() != "import" {
		t.Fatal("lost terrain-relative height or import identity", objects, err)
	}
	scaled := poseDraft(t, strings.Replace(poseSource, "  world:\n", "  world:\n    unit: 2\n", 1))
	objects, err = scaled.Objects()
	if err != nil || len(objects) != 2 {
		t.Fatal("invented import unit conversion", objects, err)
	}
}

func TestObjectsSkipRelativeAttitudeAndLegacySources(t *testing.T) {
	document := poseDraft(t, strings.Replace(poseSource, "        east: 30 # Keep the east note.\n        north: 40", "        place: {at: frame}", 1))
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 {
		t.Fatal("relative placement offered", objects, err)
	}
	legacy, err := Open("legacy.yaml", []byte("scene: {name: old}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Objects(); err == nil {
		t.Fatal("unchecked legacy poses offered")
	}
}

func TestPoseAdditionRefusesOverflowAndKeepsSource(t *testing.T) {
	source := strings.Replace(poseSource, "value: 50, unit: m", "value: 1e308, unit: m", 1)
	document := poseDraft(t, source)
	objects, err := document.Objects()
	if err != nil || len(objects) == 0 {
		t.Fatal(err)
	}
	err = document.ChangePose(PoseChange{ID: objects[0].ID(), Up: 1e308, LengthUnit: "m", AngleUnit: "deg"})
	if err == nil || document.Text() != source {
		t.Fatal("overflow changed source", err)
	}
}

func TestSetGeneratedHeadingAuthorsZeroAndPreservesOtherSource(t *testing.T) {
	source := strings.Replace(poseSource, "setting: space", "setting: ground", 1)
	source = strings.Replace(source, "        heading: {value: 60, unit: deg}\n", "", 1)
	document := poseDraft(t, source)
	if err := document.SetGeneratedHeading("scene.set.things/craft", 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(document.Text(), "# Keep the east note.") || !strings.Contains(document.Text(), "kind: webb") {
		t.Fatal("heading edit lost unrelated source")
	}
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range objects {
		if object.ID() == "scene.set.things/craft" {
			heading, said := object.Heading()
			if !said || heading != 0 {
				t.Fatalf("heading = %v, authored %v", heading, said)
			}
			return
		}
	}
	t.Fatal("authored object missing")
}

func TestSetGeneratedHeadingRefusesExistingOrInvalid(t *testing.T) {
	document := poseDraft(t, poseSource)
	for _, sample := range []struct {
		id      string
		degrees float64
	}{
		{"scene.set.things/craft", 10},
		{"scene.set.things/missing", 0},
		{"scene.set.things/craft", math.NaN()},
	} {
		if err := document.SetGeneratedHeading(sample.id, sample.degrees); err == nil || document.Text() != poseSource {
			t.Fatal("invalid edit changed source", err)
		}
	}
	ground := strings.Replace(poseSource, "setting: space", "setting: ground", 1)
	ground = strings.Replace(ground, "        heading: {value: 60, unit: deg}\n", "", 1)
	document = poseDraft(t, ground)
	if err := document.SetGeneratedHeading("scene.set.things/craft", math.NaN()); err == nil || document.Text() != ground {
		t.Fatal("non-finite generated heading changed source", err)
	}
}

func TestPoseCreatesMissingSpaceHeightWithUnits(t *testing.T) {
	source := strings.Replace(poseSource, "        height: {value: 2, unit: m}\n", "", 1)
	document := poseDraft(t, source)
	objects, err := document.Objects()
	if err != nil || len(objects) != 3 {
		t.Fatal("objects", err)
	}
	if objects[1].Position()[2] != 0 {
		t.Fatal("missing space height was not zero")
	}
	if _, explicit := objects[1].Heading(); explicit {
		t.Fatal("invented explicit heading")
	}
	position := objects[1].Position()
	position[0] = -99
	if objects[1].Position()[0] == position[0] {
		t.Fatal("mutable position snapshot")
	}
	if err := document.ChangePose(PoseChange{ID: objects[1].ID(), Up: 3, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	after, _ := document.Objects()
	if after[1].Position()[2] != 3 || after[0] != objects[0] || after[2] != objects[2] {
		t.Fatal("wrong height placement")
	}
	value, err := document.Value("scene", "set", "things", "companion", "height", "unit")
	if err != nil || value != "m" {
		t.Fatal("height unit", value, err)
	}
}

func TestChangePoseKeepsSupportedSignedSpaceCoordinates(t *testing.T) {
	document := poseDraft(t, poseSource)
	objects, err := document.Objects()
	if err != nil || len(objects) == 0 {
		t.Fatal(err)
	}
	if err := document.ChangePose(PoseChange{ID: objects[0].ID(), East: -40, North: -50, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	after, err := document.Objects()
	if err != nil || after[0].Position() != [3]float64{-10, -10, 50} {
		t.Fatal("signed native pose lost", after, err)
	}
}

func TestImplicitV2GroundExposesAuthoredPlacements(t *testing.T) {
	source := strings.Replace(poseSource, "  setting: space\n", "", 1)
	document := poseDraft(t, source)
	if !document.GroundSetting() {
		t.Fatal("v2 default terrain not recognized")
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 3 || !objects[0].Grounded() || !objects[0].ScaleEditable() {
		t.Fatalf("default terrain objects %+v: %v", objects, err)
	}
	if err := document.ChangePose(PoseChange{ID: objects[0].ID(), East: 1, LengthUnit: "m", AngleUnit: "deg"}); err != nil {
		t.Fatal(err)
	}
	if err := document.AddKindThing("second-ground", "yurt", 3, [3]float64{10, 20, 0}); err != nil {
		t.Fatal(err)
	}
	objects, err = document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, one := range objects {
		if one.Name() == "second-ground" {
			heading, said := one.Heading()
			if !one.Grounded() || !said || heading != 0 {
				t.Fatal("new default-terrain object has wrong placement semantics")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("default-terrain addition missing")
	}
}
