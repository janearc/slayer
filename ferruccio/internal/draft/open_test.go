package draft

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

func exampleSource(t *testing.T, name string) []byte {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func mapping(t *testing.T, name string, source []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := yamlfile.Read(name, source, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestOpenRecognizesJobAndV2SetWithoutChangingSource(t *testing.T) {
	for _, fixture := range []struct {
		name string
		kind string
	}{
		{"first-picture.yaml", "job"},
		{"webb-at-l2.yaml", "set"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			source := exampleSource(t, fixture.name)
			document, err := Open(fixture.name, source)
			if err != nil || document == nil {
				t.Fatalf("Open: document=%v, err=%v", document, err)
			}
			if document.Kind() != fixture.kind {
				t.Errorf("Kind = %q; want %q", document.Kind(), fixture.kind)
			}
			if document.Text() != string(source) || !bytes.Equal(source, exampleSource(t, fixture.name)) {
				t.Error("opening changed the fixture source")
			}
			if err := document.Check(); err != nil {
				t.Errorf("fixture did not validate: %v", err)
			}
		})
	}
}

func TestOpenRecognizesAndChecksCatalogueSettings(t *testing.T) {
	source := exampleSource(t, "subaru-settings.yaml")
	document, err := Open("subaru-settings.yaml", source)
	if err != nil || document == nil {
		t.Fatalf("Open catalogue: document=%v, err=%v", document, err)
	}
	if document.Kind() != "catalogue" {
		t.Fatalf("Kind = %q; want catalogue", document.Kind())
	}
	if document.Text() != string(source) {
		t.Fatal("opening changed catalogue settings")
	}
	if err := document.Check(); err != nil {
		t.Fatalf("pinned catalogue schema rejected fixture, including local-lights: %v", err)
	}
}

func TestCatalogueEditKeepsItsKindAndUnrelatedSource(t *testing.T) {
	source := exampleSource(t, "subaru-settings.yaml")
	document, err := Open("subaru-settings.yaml", source)
	if err != nil || document == nil {
		t.Fatalf("Open catalogue: document=%v, err=%v", document, err)
	}
	if err := document.Set(Change{Path: []string{"width"}, Value: "640", Tag: "!!int"}); err != nil {
		t.Fatalf("edit catalogue width: %v", err)
	}
	if document.Kind() != "catalogue" {
		t.Fatalf("edited Kind = %q; want catalogue", document.Kind())
	}
	if document.Text() == string(source) {
		t.Fatal("successful edit did not update the editable source")
	}
	if !strings.Contains(document.Text(), "# Authored catalogue display controls") ||
		!strings.Contains(document.Text(), "local-lights:") ||
		!strings.Contains(document.Text(), "height: 480") ||
		!strings.Contains(document.Text(), "width: 640") {
		t.Fatal("catalogue edit overwrote comments or unrelated settings")
	}
	if string(source) != string(exampleSource(t, "subaru-settings.yaml")) {
		t.Fatal("editing the draft overwrote the original fixture source")
	}
	if err := document.Check(); err != nil {
		t.Fatalf("edited catalogue no longer validates: %v", err)
	}
}

func TestNavigateCatalogueAddsOrbitAndPreservesOtherSettings(t *testing.T) {
	source := exampleSource(t, "subaru-settings.yaml")
	document, err := Open("subaru-settings.yaml", source)
	if err != nil || document == nil {
		t.Fatalf("Open catalogue: document=%v, err=%v", document, err)
	}
	if !document.CanNavigate() {
		t.Fatal("catalogue document cannot navigate")
	}
	if err := document.Navigate(12.5, -4.5, 1.25); err != nil {
		t.Fatalf("Navigate catalogue: %v", err)
	}
	if err := document.Check(); err != nil {
		t.Fatalf("navigated catalogue is invalid: %v", err)
	}
	if got, _ := document.Value("orbit", "yaw"); got != "12.5" {
		t.Errorf("orbit.yaw = %q; want 12.5", got)
	}
	if got, _ := document.Value("orbit", "pitch"); got != "-4.5" {
		t.Errorf("orbit.pitch = %q; want -4.5", got)
	}
	if got, _ := document.Value("wide"); got != "62.5" {
		t.Errorf("wide = %q; want 62.5", got)
	}
	if !strings.Contains(document.Text(), "# Authored catalogue display controls") || !strings.Contains(document.Text(), "# A uniform backdrop") {
		t.Error("navigation edit removed authored comments")
	}
	got := mapping(t, document.Name(), []byte(document.Text()))
	want := mapping(t, "source.yaml", source)
	want["orbit"] = map[string]any{"yaw": 12.5, "pitch": -4.5}
	want["wide"] = 62.5
	if !reflect.DeepEqual(got, want) {
		t.Errorf("navigation edit changed settings beyond orbit and wide:\n got %#v\nwant %#v", got, want)
	}
	if !bytes.Equal(source, exampleSource(t, "subaru-settings.yaml")) {
		t.Error("navigation edit changed the original source bytes")
	}
}

func TestNavigateCatalogueWrapsYawClampsPitchAndZoom(t *testing.T) {
	source := strings.Replace(string(exampleSource(t, "subaru-settings.yaml")), "wide: 50\n", "wide: 100\n", 1) + "orbit: {yaw: 170, pitch: 70}\n"
	document, err := Open("subaru-settings.yaml", []byte(source))
	if err != nil || document == nil {
		t.Fatalf("Open catalogue with orbit: document=%v, err=%v", document, err)
	}
	if err := document.Navigate(20, 30, 2); err != nil {
		t.Fatalf("Navigate toward upper bounds: %v", err)
	}
	for path, want := range map[string]string{"orbit.yaw": "-170", "orbit.pitch": "80", "wide": "120"} {
		var got string
		if path == "wide" {
			got, err = document.Value("wide")
		} else {
			got, err = document.Value("orbit", strings.TrimPrefix(path, "orbit."))
		}
		if err != nil || got != want {
			t.Errorf("%s = %q, %v; want %s", path, got, err, want)
		}
	}
	if err := document.Navigate(-25, -200, 0.01); err != nil {
		t.Fatalf("Navigate toward lower bounds: %v", err)
	}
	for path, want := range map[string]string{"orbit.yaw": "165", "orbit.pitch": "-80", "wide": "5"} {
		var got string
		if path == "wide" {
			got, err = document.Value("wide")
		} else {
			got, err = document.Value("orbit", strings.TrimPrefix(path, "orbit."))
		}
		if err != nil || got != want {
			t.Errorf("%s = %q, %v; want %s", path, got, err, want)
		}
	}
}

func TestNavigateZeroDeltasLeaveValidSourceExactlyUnchanged(t *testing.T) {
	source := exampleSource(t, "subaru-settings.yaml")
	document, err := Open("subaru-settings.yaml", source)
	if err != nil || document == nil {
		t.Fatalf("Open catalogue: document=%v, err=%v", document, err)
	}
	if err := document.Navigate(0, 0, 0); err != nil {
		t.Fatalf("zero deltas: %v", err)
	}
	if document.Text() != string(source) {
		t.Error("zero deltas rewrote source")
	}
	if got, _ := document.Value("wide"); got != "50" {
		t.Errorf("zero zoom changed wide to %q; want 50", got)
	}
}

func TestNavigateRejectsInvalidInputWithoutChangingSource(t *testing.T) {
	valid := exampleSource(t, "subaru-settings.yaml")
	cases := []struct {
		name                   string
		yaw, pitch, zoomFactor float64
	}{
		{"nan yaw", math.NaN(), 0, 1},
		{"negative infinite yaw", math.Inf(-1), 0, 1},
		{"nan pitch", 0, math.NaN(), 1},
		{"infinite pitch", 0, math.Inf(1), 1},
		{"nan zoom", 0, 0, math.NaN()},
		{"infinite zoom", 0, 0, math.Inf(1)},
		{"negative zoom", 0, 0, -1},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := Open("subaru-settings.yaml", valid)
			if err != nil || document == nil {
				t.Fatalf("Open catalogue: document=%v, err=%v", document, err)
			}
			before := document.Text()
			if err := document.Navigate(testCase.yaw, testCase.pitch, testCase.zoomFactor); err == nil {
				t.Fatal("Navigate accepted non-finite input or a negative zoom factor")
			}
			if document.Text() != before {
				t.Error("rejected navigation changed source")
			}
		})
	}
}

func TestNavigateIsUnavailableForJobs(t *testing.T) {
	source := exampleSource(t, "first-picture.yaml")
	document, err := Open("first-picture.yaml", source)
	if err != nil || document == nil {
		t.Fatalf("Open job: document=%v, err=%v", document, err)
	}
	if document.CanNavigate() {
		t.Fatal("job document reports camera navigation support")
	}
	before := document.Text()
	if err := document.Navigate(1, 2, 1); err == nil {
		t.Fatal("Navigate accepted a non-catalogue document")
	}
	if document.Text() != before {
		t.Error("rejected job navigation changed source")
	}
}

func TestNavigateSimpleSetCameraInMapCoordinates(t *testing.T) {
	source := exampleSource(t, "fizz.yaml")
	document, err := Open("fizz.yaml", source)
	if err != nil || document == nil {
		t.Fatalf("Open set: document=%v, err=%v", document, err)
	}
	if !document.CanNavigate() {
		t.Fatal("simple set camera does not report navigation support")
	}
	if err := document.Navigate(90, 30, 2); err != nil {
		t.Fatalf("Navigate set camera: %v", err)
	}
	if err := document.Check(); err != nil {
		t.Fatalf("navigated set is invalid YAML: %v", err)
	}
	if !strings.Contains(document.Text(), "# level with the column's middle") {
		t.Error("navigation removed the camera's authored comment")
	}
	got := mapping(t, "fizz.yaml", []byte(document.Text()))
	want := mapping(t, "fizz.yaml", source)
	gotScene := got["scene"].(map[string]any)
	wantScene := want["scene"].(map[string]any)
	gotFilm := gotScene["film"].(map[string]any)
	gotFilm["wide"] = 100.0
	wantFilm := wantScene["film"].(map[string]any)
	wantFilm["wide"] = 100.0
	gotCamera := gotScene["cameras"].(map[string]any)["front"].(map[string]any)
	wantCamera := wantScene["cameras"].(map[string]any)["front"].(map[string]any)
	gotLocation := gotCamera["location"].(map[string]any)
	wantLocation := wantCamera["location"].(map[string]any)
	gotAt, ok := gotLocation["at"].([]any)
	if !ok || len(gotAt) != 3 {
		t.Fatalf("edited camera location = %#v; want three coordinates", gotLocation["at"])
	}
	for index, expected := range []float64{3.897114317029974, 0, 3.25} {
		actual, numeric := gotAt[index].(float64)
		if !numeric || math.Abs(actual-expected) > 1e-8 {
			t.Errorf("edited camera coordinate %d = %#v; want %.12f", index, gotAt[index], expected)
		}
	}
	wantLocation["at"] = gotAt
	if !reflect.DeepEqual(got, want) {
		t.Errorf("navigation changed set data beyond camera and wide:\n got %#v\nwant %#v", got, want)
	}
	if !bytes.Equal(source, exampleSource(t, "fizz.yaml")) {
		t.Error("navigation changed the original set source")
	}
}

func TestSetCameraNavigationRefusesUnsupportedCameraPaths(t *testing.T) {
	base := string(exampleSource(t, "fizz.yaml"))
	cases := map[string]string{
		"moving camera": strings.Replace(base,
			"      title: \"the column\"\n", "      title: \"the column\"\n      moves: []\n", 1),
		"relative camera": strings.Replace(base,
			"        at: [0, -4.5, 1]\n", "        at: [0, -4.5, 1]\n        relative: {object: floor}\n", 1),
		"missing look target":  strings.Replace(base, "        look_at: [0, 0, 1]\n", "", 1),
		"non-numeric location": strings.Replace(base, "at: [0, -4.5, 1]", "at: [0, north, 1]", 1),
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			document, err := Open("fizz.yaml", []byte(source))
			if err != nil || document == nil {
				t.Fatalf("Open editable set: document=%v, err=%v", document, err)
			}
			if document.CanNavigate() {
				t.Fatal("unsupported camera reported navigation support")
			}
			before := document.Text()
			if err := document.Navigate(10, 5, 1.5); err == nil {
				t.Fatal("Navigate accepted unsupported camera structure")
			}
			if document.Text() != before {
				t.Error("refused navigation changed source")
			}
		})
	}
}

func TestSetCameraPitchDeltaClampsInsteadOfWrapping(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		delta    float64
		expected float64
	}{
		{"positive full turn reaches upper pole", 360, 80},
		{"negative full turn reaches lower pole", -360, -80},
		{"maximum finite delta remains at upper pole", math.MaxFloat64, 80},
		{"minimum finite delta remains at lower pole", -math.MaxFloat64, -80},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := Open("fizz.yaml", exampleSource(t, "fizz.yaml"))
			if err != nil || document == nil {
				t.Fatalf("Open set: document=%v, err=%v", document, err)
			}
			if err := document.Navigate(0, testCase.delta, 0); err != nil {
				t.Fatalf("Navigate pitch: %v", err)
			}
			form := mapping(t, "fizz.yaml", []byte(document.Text()))
			scene := form["scene"].(map[string]any)
			camera := scene["cameras"].(map[string]any)["front"].(map[string]any)
			at := camera["location"].(map[string]any)["at"].([]any)
			dx := at[0].(float64)
			dy := at[1].(float64)
			dz := at[2].(float64) - 1
			pitch := math.Atan2(dz, math.Hypot(dx, dy)) * 180 / math.Pi
			if math.IsNaN(pitch) || math.IsInf(pitch, 0) || math.Abs(pitch-testCase.expected) > 1e-6 {
				t.Errorf("camera pitch = %v degrees; want %v", pitch, testCase.expected)
			}
		})
	}
}

func TestCatalogueSchemaRejectsInvalidEditableSettings(t *testing.T) {
	valid := string(exampleSource(t, "subaru-settings.yaml"))
	cases := []struct {
		name string
		text string
	}{
		{"missing required field", strings.Replace(valid, "rays: 2\n", "", 1)},
		{"unknown field", valid + "unknown-display-option: true\n"},
		{"duplicate field", strings.Replace(valid, "width: 720\n", "width: 720\nwidth: 721\n", 1)},
		{"wrongly typed camera", strings.Replace(valid, "camera: {x: 1.4, y: 0.8, z: -2}", "camera: [1.4, 0.8, -2]", 1)},
		{"wrongly typed local lamp", strings.Replace(valid, "strength: {r: 260, g: 130, b: 36}", "strength: [260, 130, 36]", 1)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.text == valid {
				t.Fatal("test mutation did not change the fixture")
			}
			document, err := Open("invalid-catalogue.yaml", []byte(testCase.text))
			if err != nil || document == nil {
				t.Fatalf("Open must retain invalid editable settings: document=%v, err=%v", document, err)
			}
			if document.Kind() != "catalogue" || document.Text() != testCase.text {
				t.Fatalf("invalid draft kind/source changed: kind=%q", document.Kind())
			}
			if err := document.Check(); err == nil {
				t.Fatal("catalogue schema accepted invalid settings")
			}
			document.Replace(valid)
			if err := document.Check(); err != nil {
				t.Fatalf("corrected draft did not remain editable: %v", err)
			}
		})
	}
}

func TestCatalogueFieldsDoNotReclassifyMalformedOrExplicitJobs(t *testing.T) {
	job := string(exampleSource(t, "first-picture.yaml"))
	jobWithCatalogueField := job + "details: [demo]\n"
	document, err := Open("job-with-details.yaml", []byte(jobWithCatalogueField))
	if err != nil || document == nil {
		t.Fatalf("Open job with a catalogue-looking field: document=%v, err=%v", document, err)
	}
	if document.Kind() != "job" || document.Text() != jobWithCatalogueField {
		t.Fatalf("explicit job reclassified or changed: kind=%q", document.Kind())
	}
	if err := document.Check(); err == nil {
		t.Fatal("job schema accepted a catalogue-only field")
	}

	malformed := []byte("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: [unfinished\ndetails: [demo]\n")
	broken, err := Open("malformed-job.yaml", malformed)
	if err != nil || broken == nil {
		t.Fatalf("Open must preserve malformed job work: document=%v, err=%v", broken, err)
	}
	if broken.Kind() != "job" || broken.Text() != string(malformed) {
		t.Fatalf("malformed job was misclassified or changed: kind=%q", broken.Kind())
	}
	if err := broken.Check(); err == nil {
		t.Fatal("malformed job unexpectedly passed Check")
	}
}

func TestOpenSetRendererEditPreservesCameras(t *testing.T) {
	source := exampleSource(t, "webb-at-l2.yaml")
	document, err := Open("webb-at-l2.yaml", source)
	if err != nil || document == nil {
		t.Fatalf("Open: document=%v, err=%v", document, err)
	}
	before := mapping(t, "before.yaml", source)
	if err := document.Set(Change{Path: []string{"scene", "renderer"}, Value: "cpu", Tag: "!!str"}); err != nil {
		t.Fatal(err)
	}
	after := mapping(t, document.Name(), []byte(document.Text()))
	beforeScene := before["scene"].(map[string]any)
	afterScene := after["scene"].(map[string]any)
	if got := afterScene["renderer"]; got != "cpu" {
		t.Errorf("scene.renderer = %v; want cpu", got)
	}
	if !reflect.DeepEqual(afterScene["cameras"], beforeScene["cameras"]) {
		t.Error("renderer edit changed the existing cameras")
	}
	if err := document.Check(); err != nil {
		t.Errorf("edited set no longer validates: %v", err)
	}
}

func TestOpenRefusesUnknownExplicitSchema(t *testing.T) {
	source := []byte("$schema: https://example.invalid/unrecognized.json\nscene: {smoke: 1}\n")
	document, err := Open("unknown.yaml", source)
	if err == nil || document != nil {
		t.Fatalf("unknown explicit schema opened: document=%v, err=%v", document, err)
	}
}

func TestOpenLegacySceneAndCheckRefusesDuplicateNestedKey(t *testing.T) {
	legacy := []byte("# old scene\nscene:\n  smoke: 1\n")
	document, err := Open("legacy.yaml", legacy)
	if err != nil || document == nil {
		t.Fatalf("Open legacy scene: document=%v, err=%v", document, err)
	}
	if document.Kind() != "legacy-set" || document.Text() != string(legacy) {
		t.Fatalf("legacy kind/source = %q / %q", document.Kind(), document.Text())
	}
	if err := document.Check(); err != nil {
		t.Fatalf("legacy scene did not validate: %v", err)
	}

	duplicate := []byte("scene:\n  name: an old scene\n  renderer: cpu\n  renderer: gpu\n")
	broken, err := Open("duplicate-legacy.yaml", duplicate)
	if err != nil || broken == nil {
		t.Fatalf("Open duplicate legacy scene should retain editable source: document=%v, err=%v", broken, err)
	}
	if broken.Text() != string(duplicate) {
		t.Error("opening duplicate legacy keys changed the source")
	}
	if err := broken.Check(); err == nil || !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		t.Errorf("Check error = %v; want duplicate-key refusal", err)
	}
}

func TestReferencedSetOnlyReadsJobScenePath(t *testing.T) {
	source := strings.Replace(string(exampleSource(t, "first-picture.yaml")), "scene: {smoke: 1}", "scene: {set: ../sets/harbor.yaml}", 1)
	document, err := Open("job.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if got := document.ReferencedSet(); got != "../sets/harbor.yaml" {
		t.Fatalf("reference = %q", got)
	}
	document, err = Open("set.yaml", []byte(poseSource))
	if err != nil {
		t.Fatal(err)
	}
	if got := document.ReferencedSet(); got != "" {
		t.Fatalf("set has job reference %q", got)
	}
}
