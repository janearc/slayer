package main

import (
	"bytes"
	yaml "gopkg.in/yaml.v3"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestClipMovesSubjectAndFieldTogetherAtRealTimes(t *testing.T) {
	root := t.TempDir()
	template, err := os.ReadFile("../../doc/examples/jobs/alfa/template.yaml")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "template.yaml"), template, 0644)
	plan := `template: template.yaml
output: output
subject: alfa
field: main-screw
camera: underwater-60m
width: 1080
height: 960
fps: 2
seconds: 1
velocity-metres-per-second: [2, 4, 0]
camera-tracking-fraction: 1
`
	path := filepath.Join(root, "plan.yaml")
	os.WriteFile(path, []byte(plan), 0644)
	var out bytes.Buffer
	if err := generate(path, &out); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "output/scenes/frame-000001.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var doc yaml.Node
	yaml.Unmarshal(raw, &doc)
	field, _ := nodeAt(&doc, "scene", "world", "vapor-cavities", "main-screw", "origin")
	x, _ := strconv.ParseFloat(field.Content[0].Value, 64)
	y, _ := strconv.ParseFloat(field.Content[1].Value, 64)
	if math.Abs(x-(-1485.6954)) > 1e-9 || math.Abs(y-(-34.554)) > 1e-9 {
		t.Fatalf("field did not follow shared trajectory: %s", raw)
	}
	subject, _ := nodeAt(&doc, "scene", "set", "things", "alfa", "position", "east", "value")
	if subject.Value != "-1499" {
		t.Fatal("subject did not move")
	}
	job, _ := os.ReadFile(filepath.Join(root, "output/jobs/frame-000001.yaml"))
	if !strings.Contains(string(job), "value: 0.500000000") || !strings.Contains(string(job), "width: 1080") {
		t.Fatal("wrong actual frame time or master dimensions")
	}
	entries, _ := filepath.Glob(filepath.Join(root, "output/scenes/frame-*.yaml"))
	if len(entries) != 2 {
		t.Fatal("wrong frame count")
	}
}

func TestClipPlanRefusesUnknownAndTrailingDocuments(t *testing.T) {
	valid := "template: t\noutput: output\nsubject: vessel\nfield: rotor\ncamera: stern\nwidth: 1080\nheight: 960\nfps: 12\nseconds: 10\nvelocity-metres-per-second: [0, 1, 0]\ncamera-tracking-fraction: 1\n"
	for _, raw := range []string{valid + "unknown: true\n", valid + "---\ntemplate: x\n", "{} {}"} {
		dir := t.TempDir()
		path := filepath.Join(dir, "plan.yaml")
		os.WriteFile(path, []byte(raw), 0644)
		if err := generate(path, &bytes.Buffer{}); err == nil || strings.Contains(err.Error(), "no such file") {
			t.Fatalf("accepted invalid/trailing document: %q", raw)
		}
	}
}

func TestClipCoordinateTypesAndExistingOutputs(t *testing.T) {
	for _, text := range []string{"[1, 2, 3]", ".nan", "hello"} {
		var node yaml.Node
		yaml.Unmarshal([]byte(text), &node)
		if err := shiftScalar(node.Content[0], 1); err == nil {
			t.Fatalf("accepted nonnumeric/nonfinite coordinate %s", text)
		}
	}
	path := filepath.Join(t.TempDir(), "existing.yaml")
	if err := writeSameOrNew(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := writeSameOrNew(path, []byte("first")); err != nil {
		t.Fatal("exact rerun should be allowed", err)
	}
	if err := writeSameOrNew(path, []byte("different")); err == nil {
		t.Fatal("silently replaced old output")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "first" {
		t.Fatal("old output changed")
	}
}

func TestClipMetreQuantitiesRefuseWrongUnit(t *testing.T) {
	for _, text := range []string{"east: {value: 1, unit: km}", "east: {value: [1], unit: m}", "east: {value: 1}"} {
		var node yaml.Node
		yaml.Unmarshal([]byte(text), &node)
		if err := shiftMetreQuantity(&node, "east", 1); err == nil {
			t.Fatalf("accepted wrong unit/type: %s", text)
		}
	}
}

func TestClipRendererSelectionAndRefusal(t *testing.T) {
	for _, backend := range []string{"cpu", "gpu", "invented"} {
		root := t.TempDir()
		template, err := os.ReadFile("../../doc/examples/jobs/alfa/template.yaml")
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(root, "template.yaml"), template, 0644)
		raw := "template: template.yaml\noutput: output\nsubject: alfa\nfield: main-screw\ncamera: underwater-60m\nwidth: 1080\nheight: 960\nfps: 1\nseconds: 1\nvelocity-metres-per-second: [0, 1, 0]\ncamera-tracking-fraction: 1\nrenderer: " + backend + "\n"
		path := filepath.Join(root, "plan.yaml")
		os.WriteFile(path, []byte(raw), 0644)
		err = generate(path, &bytes.Buffer{})
		if backend == "invented" {
			if err == nil {
				t.Fatal("accepted unknown renderer")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		job, _ := os.ReadFile(filepath.Join(root, "output/jobs/frame-000000.yaml"))
		if !strings.Contains(string(job), "renderer: "+backend+"\n") {
			t.Fatalf("wrong renderer job: %s", job)
		}
	}
}
