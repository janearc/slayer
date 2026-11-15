package job

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	yaml "gopkg.in/yaml.v3"
)

// TestABrokenSchemaOrTableIsSaid holds that a schema that did not
// compile, or a presets table that did not read, refuses every job.
// anti-test of TestEveryKeyReads
func TestABrokenSchemaOrTableIsSaid(t *testing.T) {
	path := written(t, still)
	before, beforeError := schema, schemaError
	t.Cleanup(func() { schema, schemaError = before, beforeError })
	_, schemaError = yamlfile.NewSchema("https://example.test/broken.json",
		[]byte("{"))
	if _, err := Read(path); !errors.Is(err, schemaError) {
		t.Errorf("a broken schema read as %v", err)
	}
	schemaError = nil
	presetsBefore := presetsError
	t.Cleanup(func() { presetsError = presetsBefore })
	presetsError = errors.New("presets.yaml: unreadable")
	if _, err := Read(path); !errors.Is(err, presetsError) {
		t.Errorf("a broken presets table read as %v", err)
	}
}

// TestTheDecoderCatchesASchemaThatDrifted holds the typed decoding to
// its own refusals where a looser schema lets a wrong value through.
// anti-test of TestEveryKeyReads
func TestTheDecoderCatchesASchemaThatDrifted(t *testing.T) {
	before := schema
	t.Cleanup(func() { schema = before })
	loose, err := yamlfile.NewSchema("https://example.test/loose.json",
		[]byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	schema = loose
	for _, text := range []string{
		"fps: {}\n",
		"titles: {elements: {a: {kind: [x]}}}\n",
		"titles: {elements: {a: {count: x}}}\n",
		"titles: {elements: {a: {count: {from: x}}}}\n",
	} {
		_, err := reading(t, text)
		if err == nil || !strings.Contains(err.Error(), "unmarshal") {
			t.Errorf("%q drifted through as %v", text, err)
		}
	}
}

// TestAPlaceIsTheNearestTheDocumentHas holds a refusal's place to the
// key, or the deepest node on the way where the key is not there.
func TestAPlaceIsTheNearestTheDocumentHas(t *testing.T) {
	var document yaml.Node
	text := "a:\n  b: [x, y]\nc: d\n"
	if err := yaml.Unmarshal([]byte(text), &document); err != nil {
		t.Fatal(err)
	}
	top := document.Content[nothing]
	for _, each := range []struct {
		path []string
		line int
	}{
		{[]string{"c"}, len("abc")},
		{[]string{"a", "b", "1"}, len("ab")},
	} {
		if at := keyAt(top, each.path); at.Line != each.line {
			t.Errorf("%v is placed at line %d, want %d", each.path,
				at.Line, each.line)
		}
	}
	// anti-test of TestAPlaceIsTheNearestTheDocumentHas
	for _, each := range []struct {
		path []string
		line int
	}{
		{[]string{"a", "b", "x"}, len("ab")},
		{[]string{"a", "b", "9"}, len("ab")},
		{[]string{"absent"}, len("a")},
		{nil, len("a")},
	} {
		if at := keyAt(top, each.path); at.Line != each.line {
			t.Errorf("%v is placed at line %d, want %d", each.path,
				at.Line, each.line)
		}
	}
}

// TestNoWorkingDirectoryIsSaid holds Read, Lift and Output to the
// working directory's error when there is none to resolve against.
// anti-test of TestAnOverridesPathIsTheWorkingDirectorys
// anti-test of TestALiftedPlanHasTheSameFrames
func TestNoWorkingDirectoryIsSaid(t *testing.T) {
	job := read(t, film)
	values := plan(t)
	directory := t.TempDir()
	t.Chdir(directory)
	t.Setenv("PWD", "/")
	if err := os.Chmod(directory, nothing); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(directory, private|executable); err != nil {
			t.Error(err)
		}
	})
	if _, err := Read("job.yaml"); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Read with no working directory: %v", err)
	}
	if _, err := Lift(values); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Lift with no working directory: %v", err)
	}
	_, err := job.With(Output("frames"))
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("Output with no working directory: %v", err)
	}
}

// executable is defined as the owner's right to enter a directory.
const executable = 0o100

// TestAPresetTheTableDoesNotHaveIsRefused holds the check behind the
// schema's enum: were the two to part, the table's word stands.
// anti-test of TestThePresetsAreOneTable
func TestAPresetTheTableDoesNotHaveIsRefused(t *testing.T) {
	before := presets
	t.Cleanup(func() { presets = before })
	presets = presets[:one]
	refused(t, still+"film: {preset: 4k}\n", "job.yaml:3:",
		`film preset "4k"`)
}
