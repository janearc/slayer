package draft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/assets"
)

const example = `# Keep this note for the director.
$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene:
  set: scenes/venus.yaml
film:
  preset: 720
quality: preview
renderer: gpu
fps: 24
duration: {value: 12, unit: s}
`

func openDraft(t *testing.T) *Document {
	t.Helper()
	schema, err := assets.Read("job.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	document, err := New(Options{Name: "study.job.yaml", Source: []byte(example), Schema: schema})
	if err != nil || document == nil {
		t.Fatalf("New: %v", err)
	}
	return document
}

func TestControlPreservesCommentsAndFutureFilmFields(t *testing.T) {
	document := openDraft(t)
	if err := document.Set(Change{Path: []string{"quality"}, Value: "exact", Tag: "!!str"}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Keep this note", "fps: 24", "duration:", "value: 12", "quality: exact"} {
		if !strings.Contains(document.Text(), text) {
			t.Fatalf("edit lost %q: %s", text, document.Text())
		}
	}
	if !document.Dirty() {
		t.Fatal("edit did not mark the document dirty")
	}
}

func TestStillControlsPreserveCameraAndSoundWithoutInventingFields(t *testing.T) {
	document := openDraft(t)
	document.Replace(example + "camera: {name: rim}\nsound: {file: music/score.wav}\n")
	if err := document.Set(Change{Path: []string{"film", "wide"}, Value: "62.5", Tag: "!!float"}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		path []string
		want string
	}{
		{[]string{"camera", "name"}, "rim"}, {[]string{"sound", "file"}, "music/score.wav"},
		{[]string{"fps"}, "24"}, {[]string{"duration", "value"}, "12"},
	} {
		value, err := document.Value(field.path...)
		if err != nil || value != field.want {
			t.Fatalf("preserved field %v = %q, %v", field.path, value, err)
		}
	}
	before := document.Text()
	if err := document.Set(Change{Path: []string{"sound", "imaginary-mixer"}, Value: "on"}); err == nil {
		t.Fatal("invented sound setting accepted")
	}
	if document.Text() != before {
		t.Fatal("refusal changed camera or sound source")
	}
}

func TestConstructionAndScalarRefusalsKeepInputsIntact(t *testing.T) {
	schema, err := assets.Read("job.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, options := range []Options{{Name: " ", Source: []byte(example), Schema: schema}, {Name: "broken.yaml", Source: []byte(example), Schema: []byte("not a schema")}} {
		if document, err := New(options); err == nil || document != nil {
			t.Fatal("invalid construction accepted")
		}
	}
	document := openDraft(t)
	for _, edit := range []Change{{}, {Path: []string{"quality"}, Value: "exact", Tag: "!!bool"}, {Path: []string{"quality", "nested"}, Value: "exact"}} {
		if err := document.Set(edit); err == nil {
			t.Fatal("invalid scalar edit accepted")
		}
		if document.Text() != example || document.Dirty() {
			t.Fatal("invalid scalar edit changed original input")
		}
	}
	for _, path := range [][]string{{"missing"}, {"film"}, {"quality", "nested"}} {
		if _, err := document.Value(path...); err == nil {
			t.Fatalf("non-scalar or absent path %v accepted", path)
		}
	}
}

func TestFPSControlPreservesDuration(t *testing.T) {
	document := openDraft(t)
	if err := document.Set(Change{Path: []string{"fps"}, Value: "30", Tag: "!!int"}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		path []string
		want string
	}{{[]string{"fps"}, "30"}, {[]string{"duration", "value"}, "12"}} {
		value, err := document.Value(field.path...)
		if err != nil || value != field.want {
			t.Errorf("%s = %q, %v; want %q", field.path, value, err, field.want)
		}
	}
}

func TestDurationControlPreservesFPS(t *testing.T) {
	document := openDraft(t)
	if err := document.Set(Change{Path: []string{"duration", "value"}, Value: "20", Tag: "!!int"}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		path []string
		want string
	}{{[]string{"fps"}, "24"}, {[]string{"duration", "value"}, "20"}} {
		value, err := document.Value(field.path...)
		if err != nil || value != field.want {
			t.Errorf("%v = %q, %v; want %q", field.path, value, err, field.want)
		}
	}
}

func TestSaveReloadRoundTripPreservesComments(t *testing.T) {
	document := openDraft(t)
	if err := document.Set(Change{Path: []string{"quality"}, Value: "exact", Tag: "!!str"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "roundtrip.job.yaml")
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := assets.Read("job.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(Options{Name: filepath.Base(path), Source: source, Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	if err := reloaded.Check(); err != nil {
		t.Fatalf("reloaded document is invalid: %v", err)
	}
	if !strings.Contains(reloaded.Text(), "Keep this note") {
		t.Fatalf("save and reload lost source comment: %s", reloaded.Text())
	}
	if value, err := reloaded.Value("quality"); err != nil || value != "exact" {
		t.Fatalf("reloaded quality = %q, %v; want exact", value, err)
	}
}

func TestInvalidDraftSurvivesSaveReloadAndRefusesControl(t *testing.T) {
	document := openDraft(t)
	document.Replace("# unfinished\nscene: [")
	path := filepath.Join(t.TempDir(), "unfinished.yaml")
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := assets.Read("job.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(Options{Name: filepath.Base(path), Source: source, Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	if err := reloaded.Check(); err == nil {
		t.Fatal("invalid draft became valid after reload")
	}
	before := reloaded.Text()
	if err := reloaded.Set(Change{Path: []string{"quality"}, Value: "exact"}); err == nil {
		t.Fatal("control edited an invalid reloaded draft")
	}
	if reloaded.Text() != before {
		t.Fatalf("refused edit changed invalid draft: %q", reloaded.Text())
	}
}

func TestOptionalNestedFieldEditRefusalLeavesSourceIntact(t *testing.T) {
	document := openDraft(t)
	before := document.Text()
	if err := document.Set(Change{Path: []string{"scene", "terrain", "source"}, Value: "invented", Tag: "!!str"}); err == nil {
		t.Fatal("unknown optional nested field admitted")
	}
	if document.Text() != before || document.Dirty() {
		t.Fatalf("refused nested edit changed source or dirty state: %s", document.Text())
	}
}

func TestRefusedControlLeavesDraftUnchanged(t *testing.T) {
	document := openDraft(t)
	before := document.Text()
	if err := document.Set(Change{Path: []string{"renderer"}, Value: "pretend", Tag: "!!str"}); err == nil {
		t.Fatal("unknown renderer admitted")
	}
	if document.Text() != before || document.Dirty() {
		t.Fatal("refused control changed the draft")
	}
	document.Replace("scene: [broken")
	broken := document.Text()
	if err := document.Set(Change{Path: []string{"quality"}, Value: "preview"}); err == nil {
		t.Fatal("control edited an invalid draft")
	}
	if document.Text() != broken {
		t.Fatal("invalid work was discarded")
	}
}

func TestSchemaRefusesDuplicatesAndUnknownKeys(t *testing.T) {
	for _, suffix := range []string{"quality: exact\n", "invented: 1\n"} {
		document := openDraft(t)
		document.Replace(example + suffix)
		if err := document.Check(); err == nil {
			t.Fatal("invalid structure accepted")
		}
	}
}

func TestSavePreservesInvalidWorkAndTracksSavedState(t *testing.T) {
	document := openDraft(t)
	document.Replace("# unfinished\nscene: [")
	path := filepath.Join(t.TempDir(), "unfinished.yaml")
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != document.Text() || document.Dirty() {
		t.Fatalf("save lost work: %v", err)
	}
	document.Replace("changed again")
	if !document.Dirty() {
		t.Fatal("new edit not dirty")
	}
	if err := document.Save(filepath.Join(path, "cannot-exist.yaml")); err == nil {
		t.Fatal("failed save accepted")
	}
	if !document.Dirty() {
		t.Fatal("failed save marked work saved")
	}
}
