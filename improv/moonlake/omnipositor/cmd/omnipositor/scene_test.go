package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestCompositeSceneDryRun uses the real Pzm verb and checked YAML without
// opening a GPU or creating its proposed output directory.
func TestCompositeSceneDryRun(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"job.yaml":  "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: summit.yaml}\nfilm: {width: 320, height: 180, rays: 1}\nrenderer: gpu\noutputs: {picture: unused.png}\n",
		"plan.yaml": "version: 1\njob: job.yaml\nfrom: 0\nto: 1\ntransport-scale: 2\ntransport-rays: 1\nplate-every: 1\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "new-output")
	var stdout, stderr bytes.Buffer
	status := tool().Run([]string{"scene", filepath.Join(dir, "plan.yaml"), "--out", out, "--dry-run"}, &stdout, &stderr)
	if status != 0 {
		t.Fatalf("status %d: %s", status, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("[0,1)")) {
		t.Fatal(stdout.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("dry-run made output: %v", err)
	}
}
