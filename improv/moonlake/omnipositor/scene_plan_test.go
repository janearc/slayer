package omnipositor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompositePlanRefusesStaleCameraAndUnknownFields exercises the actual
// YAML reader: a moving eye may never silently borrow the previous plate.
func TestCompositePlanRefusesStaleCameraAndUnknownFields(t *testing.T) {
	dir := t.TempDir()
	jobFile := filepath.Join(dir, "job.yaml")
	base := `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: summit.yaml}
film: {width: 320, height: 180, rays: 1}
renderer: gpu
fps: 6
duration: {value: 2, unit: s}
outputs: {frames: unused}
`
	plan := `version: 1
job: job.yaml
from: 0
to: 12
transport-scale: 2
transport-rays: 1
plate-every: 12
`
	path := filepath.Join(dir, "plan.yaml")
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(jobFile, base)
	write(path, plan)
	p, j, err := ReadScenePlan(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.JobFile != jobFile || j.Clock().Frames() != 12 {
		t.Fatal("relative job or film clock lost")
	}
	write(path, plan+"typo: true\n")
	if _, _, err := ReadScenePlan(path); err == nil {
		t.Fatal("accepted an unknown configuration key")
	}
	write(path, strings.Replace(plan, "to: 12", "to: 13", 1))
	if _, _, err := ReadScenePlan(path); err == nil {
		t.Fatal("accepted a frame outside the job")
	}
	write(path, plan)
	write(jobFile, base+`camera:
  name: summit
  zoom:
    from: {value: 60, unit: deg}
    to: {value: 62, unit: deg}
    duration: {value: 2, unit: s}
`)
	if _, _, err := ReadScenePlan(path); err == nil || !strings.Contains(err.Error(), "plate-every") {
		t.Fatalf("zoom reuse: %v", err)
	}
	write(path, strings.Replace(plan, "plate-every: 12", "plate-every: 1", 1))
	if _, _, err := ReadScenePlan(path); err != nil {
		t.Fatalf("fresh plates with a zoom: %v", err)
	}
}
