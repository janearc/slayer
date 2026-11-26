package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOnlySourceUsesPrivateSnapshotAndOriginalBase(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "read-only")
	if err := os.Mkdir(sourceDir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sourceDir, "job.yaml")
	source := strings.Replace(string(firstPicture(t)), "scene: {smoke: 1}", "scene: {set: ../scene.yaml}", 1)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sourceDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sourceDir, 0700) })
	plan, err := Prepare(Request{Path: path, Text: source, OutputRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if !plan.relocated || filepath.Dir(plan.Input) != plan.Directory {
		t.Fatalf("frozen copy not in private output: %+v", plan)
	}
	args, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil || !strings.Contains(strings.Join(args, "\x00"), "--source-path\x00"+path) {
		t.Fatalf("source base missing from producer arguments: %v, %v", args, err)
	}
	inputs, err := plan.declaredInputs()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "scene.yaml")
	found := false
	for _, each := range inputs {
		found = found || each == want
	}
	if !found {
		t.Fatalf("relative input %q missing from %v", want, inputs)
	}
}
