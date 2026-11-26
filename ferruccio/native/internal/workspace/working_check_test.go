package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkingSource is a space set the working display takes: one plain kind,
// a seed, a film size and a fixed camera.
const checkingSource = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Checking
  setting: space
  seed: 19
  light: none
  set:
    things:
      visitor: {kind: human-scale-2m, seed: 7, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -60, 20]}
      orientation: {look_at: [0, 0, 1]}
`

// checkingFile writes text to a new file and returns its path.
func checkingFile(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "set.yaml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckWorkingTakesAGoodSetAtBothDepths(t *testing.T) {
	path := checkingFile(t, checkingSource)
	for _, meshes := range []bool{false, true} {
		if err := CheckWorking(path, meshes); err != nil {
			t.Fatalf("meshes %v: a good set was refused: %v", meshes, err)
		}
	}
}

func TestCheckWorkingRefusesAMissingFile(t *testing.T) {
	if err := CheckWorking(filepath.Join(t.TempDir(), "nowhere.yaml"), false); err == nil {
		t.Fatal("a file that is not there was taken")
	}
}

func TestCheckWorkingRefusesAFileOverTheEditorLimit(t *testing.T) {
	big := checkingFile(t, strings.Repeat("# padding\n", 5*1024*1024/len("# padding\n")))
	err := CheckWorking(big, false)
	if err == nil || !strings.Contains(err.Error(), "4 MiB editor limit") {
		t.Fatalf("a file over the limit was not refused for it: %v", err)
	}
}

func TestCheckWorkingNamesTheSchemaWhenItRefuses(t *testing.T) {
	err := CheckWorking(checkingFile(t, "scene: [unclosed\n"), false)
	if err == nil || !strings.HasPrefix(err.Error(), "schema: ") {
		t.Fatalf("a broken file was not refused by the schema: %v", err)
	}
}

func TestCheckWorkingRefusesNoSeedInTheDisplaysWords(t *testing.T) {
	bare := strings.Replace(checkingSource, "  seed: 19\n", "", 1)
	err := CheckWorking(checkingFile(t, bare), false)
	if err == nil || !strings.Contains(err.Error(), "an explicit seed") {
		t.Fatalf("a set with no seed was not refused for it: %v", err)
	}
}

func TestCheckWorkingGivesTheSameRefusalAtBothDepths(t *testing.T) {
	burning := strings.Replace(checkingSource, "visitor: {kind:", "visitor: {alight: true, kind:", 1)
	path := checkingFile(t, burning)
	cheap, deep := CheckWorking(path, false), CheckWorking(path, true)
	if cheap == nil || deep == nil || cheap.Error() != deep.Error() {
		t.Fatalf("the depths disagree: %v and %v", cheap, deep)
	}
	if !strings.Contains(cheap.Error(), "visitor: ") {
		t.Fatalf("the refusal does not name the placement: %v", cheap)
	}
}

func TestWorkingRefusalDoesNotClaimRenderSuccess(t *testing.T) {
	work := playWorkspace(t)
	work.editor.ApplyText("scene: [\n")
	work.updateWorking()
	if work.working.failure == "" || work.diagnostic == "" {
		t.Fatal("invalid source did not report its working-view refusal")
	}
	if strings.Contains(work.caption, "can still draw") || !strings.Contains(work.caption, "unavailable") {
		t.Fatalf("working-view refusal made an unsupported render claim: %q", work.caption)
	}
}

func TestWorkingViewNamesJobLimitation(t *testing.T) {
	work := playWorkspace(t)
	path := filepath.Join("..", "..", "..", "..", "jobs", "voyager-1-hero.yaml")
	if err := work.state.load(path); err != nil {
		t.Fatal(err)
	}
	if work.state.document.Kind() != "job" {
		t.Fatalf("document kind = %q", work.state.document.Kind())
	}
	work.editor.SetText(work.state.document.Text())
	work.updateWorking()
	if !strings.Contains(work.diagnostic, "job documents") {
		t.Fatalf("job limitation not explained: %q", work.diagnostic)
	}
	if strings.Contains(work.diagnostic, "jsonschema validation failed") {
		t.Fatalf("a job was described as an invalid set: %q", work.diagnostic)
	}
}

func TestInvalidJobKeepsItsValidationError(t *testing.T) {
	work := playWorkspace(t)
	path := filepath.Join("..", "..", "..", "..", "jobs", "voyager-1-hero.yaml")
	if err := work.state.load(path); err != nil {
		t.Fatal(err)
	}
	work.editor.SetText(work.state.document.Text())
	work.editor.ApplyText("scene: [\n")
	work.updateWorking()
	if work.diagnostic == "" || strings.Contains(work.diagnostic, "does not yet display job documents") {
		t.Fatalf("invalid job hid its validation error: %q", work.diagnostic)
	}
}
