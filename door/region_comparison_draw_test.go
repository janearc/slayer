//go:build draw

package door

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComparisonDrawsAndReusesReference(t *testing.T) {
	root := t.TempDir()
	shelf := filepath.Join(root, "shelf")
	if err := os.Mkdir(shelf, 0700); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	t.Setenv("SLAYER_SHELF", shelf)
	t.Setenv("SLAYER_CACHE", cache)
	t.Setenv("SLAYER_SCROOGE", "")
	t.Setenv("SLAYER_TRIAL_PLAN", "")
	files := map[string]string{
		"job.yaml": `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: set.yaml}
camera: {name: view}
film: {width: 4, height: 4, rays: 1}
renderer: cpu
outputs: {picture: picture.png}
`,
		"set.yaml": `scene:
  name: small
  light: ambient
  film: {width: 4, height: 4, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 1]}
      orientation: {look_at: [0, 0, 1]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      object:
        material: matte
        middle: [0, 0, 1]
        size: [1, 1, 1]
        source: eye
`,
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	set := filepath.Join(t.TempDir(), "regions.yaml")
	if err := os.WriteFile(set, []byte(`job: job.yaml
threshold: 0.1
regions:
  - {name: box, observation: box, left: 1, top: 1, width: 2, height: 2}
`), 0600); err != nil {
		t.Fatal(err)
	}
	refs := t.TempDir()
	env := envAt(t, Given{Context: context.Background(), Now: noon, Directory: root, Build: "test"}, "SLAYER_SHELF", shelf, "SLAYER_CACHE", cache)
	out := filepath.Join(t.TempDir(), "result")
	args := []string{"compare-regions", set, "--snapshot", root, "--inputs", "job.yaml,set.yaml", "--complete-inputs", "--references", refs, "--out", out, "--width", "4", "--height", "4", "--rays", "1", "--reference-rays", "2", "--repeat", "2"}
	for _, count := range []string{"0", "9"} {
		bad := append([]string(nil), args...)
		bad[len(bad)-1] = count
		if got := running(Tool(env), bad...); got.code == 0 {
			t.Fatalf("accepted --repeat %s", count)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("invalid repeat wrote output: %v", err)
		}
	}
	dry := running(Tool(env), append(args, "--dry-run")...)
	if dry.code != 0 || !strings.Contains(dry.said, "would compare") {
		t.Fatalf("dry: %+v", dry)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote output: %v", err)
	}
	got := running(Tool(env), args...)
	if got.code != 0 {
		t.Fatalf("draw: %+v", got)
	}
	for _, name := range []string{"report.json", "sheet.png", "references.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadFile(filepath.Join(out, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Regions []struct {
			Repeats []json.RawMessage `json:"repeats"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(before, &report); err != nil || len(report.Regions) != 1 || len(report.Regions[0].Repeats) != 1 {
		t.Fatalf("repeat report: %+v, %v", report, err)
	}
	got = running(Tool(env), args...)
	if got.code == 0 {
		t.Fatal("existing output was accepted")
	}
	after, err := os.ReadFile(filepath.Join(out, "report.json"))
	if err != nil || string(after) != string(before) {
		t.Fatal("existing result changed")
	}
	kept, err := os.ReadDir(refs)
	if err != nil || len(kept) != 1 {
		t.Fatalf("references: %v %v", kept, err)
	}
	second := filepath.Join(t.TempDir(), "result")
	for i := range args {
		if args[i] == out {
			args[i] = second
		}
	}
	args = append(args, "--reference-plan", filepath.Join(out, "references.json"))
	got = running(Tool(env), args...)
	if got.code != 0 {
		t.Fatalf("reuse: %+v", got)
	}
	keptAgain, err := os.ReadDir(refs)
	if err != nil || len(keptAgain) != 1 || kept[0].Name() != keptAgain[0].Name() {
		t.Fatalf("references changed: %v %v", keptAgain, err)
	}
	if err := os.WriteFile(filepath.Join(refs, kept[0].Name()), []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	third := filepath.Join(t.TempDir(), "result")
	for i := range args {
		if args[i] == second {
			args[i] = third
		}
	}
	got = running(Tool(env), args...)
	if got.code == 0 {
		t.Fatal("damaged reference was accepted")
	}
	if _, err := os.Stat(third); !os.IsNotExist(err) {
		t.Fatalf("failure wrote completed output: %v", err)
	}
}
