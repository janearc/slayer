package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func attachedFixture(t *testing.T, root, selection string) string {
	t.Helper()
	raw, err := os.ReadFile("../../doc/examples/jobs/alfa/template.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document, volumes yaml.Node
	if err = yaml.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	extra := `residual:
  foot: [10, 20, -55]
  radius: 0.6
  height: 20
  thick: 0.02
  rise: 10
  churn: 0.8
  seed: 7
  colour: {srgb_linear: [0.84, 0.84, 0.84]}
  source: eye
environment:
  foot: [150, 250, -5]
  radius: 0.6
  height: 2
  thick: 0.02
  rise: 0.1
  churn: 0.5
  seed: 9
  colour: {srgb_linear: [0.84, 0.84, 0.84]}
  source: eye
`
	yaml.Unmarshal([]byte(extra), &volumes)
	world, _ := nodeAt(&document, "scene", "world")
	world.Content = append(world.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "fizzes"}, volumes.Content[0])
	raw, err = yaml.Marshal(&document)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "template.yaml"), raw, 0644)
	plan := "template: template.yaml\noutput: output\nsubject: alfa\nfield: main-screw\ncamera: underwater-60m\nwidth: 1080\nheight: 960\nfps: 2\nseconds: 2\nvelocity-metres-per-second: [2, 4, 0.5]\ncamera-tracking-fraction: 1\n" + selection
	file := filepath.Join(root, "plan.yaml")
	os.WriteFile(file, []byte(plan), 0644)
	return file
}

func TestAttachedFizzTracksSubjectWhileUnselectedPopulationStaysFixed(t *testing.T) {
	root := t.TempDir()
	plan := attachedFixture(t, root, "attached-fizzes: [residual]\n")
	if err := generate(plan, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 2, 3} {
		sceneName := fmt.Sprintf("frame-%06d.yaml", index)
		raw, err := os.ReadFile(filepath.Join(root, "output/scenes", sceneName))
		if err != nil {
			t.Fatal(err)
		}
		var doc yaml.Node
		yaml.Unmarshal(raw, &doc)
		foot, _ := nodeAt(&doc, "scene", "world", "fizzes", "residual", "foot")
		fixed, _ := nodeAt(&doc, "scene", "world", "fizzes", "environment", "foot")
		subject, _ := nodeAt(&doc, "scene", "set", "things", "alfa", "position", "east", "value")
		east, _ := strconv.ParseFloat(subject.Value, 64)
		tseconds := float64(index) / 2
		for axis, base := range []float64{10, 20, -55} {
			v, _ := strconv.ParseFloat(foot.Content[axis].Value, 64)
			velocity := []float64{2, 4, .5}[axis]
			if math.Abs(v-base-velocity*tseconds) > 1e-9 {
				t.Fatalf("frame%d attachment axis%d=%g", index, axis, v)
			}
		}
		x, _ := strconv.ParseFloat(foot.Content[0].Value, 64)
		if math.Abs((x-east)-1510) > 1e-9 {
			t.Fatal("residual lost alignment with vessel")
		}
		if fixed.Content[0].Value != "150" || fixed.Content[1].Value != "250" || fixed.Content[2].Value != "-5" {
			t.Fatal("unselected environmental Fizz moved")
		}
	}
}

func TestAttachedFizzRefusesMissingDuplicateAndBadFootBeforeWriting(t *testing.T) {
	for _, selection := range []string{"attached-fizzes: [missing]\n", "attached-fizzes: [residual, residual]\n"} {
		root := t.TempDir()
		plan := attachedFixture(t, root, selection)
		if err := generate(plan, &bytes.Buffer{}); err == nil {
			t.Fatal("accepted missing/duplicate attachment")
		}
		files, _ := filepath.Glob(filepath.Join(root, "output/scenes/*.yaml"))
		if len(files) != 0 {
			t.Fatal("invalid plan wrote snapshots")
		}
	}
	for _, bad := range []string{"[10, nope, -55]", "{value: [10, 20, -55], unit: km}"} {
		root := t.TempDir()
		plan := attachedFixture(t, root, "attached-fizzes: [residual]\n")
		raw, _ := os.ReadFile(filepath.Join(root, "template.yaml"))
		raw = bytes.Replace(raw, []byte("foot: [10, 20, -55]"), []byte("foot: "+bad), 1)
		os.WriteFile(filepath.Join(root, "template.yaml"), raw, 0644)
		if err := generate(plan, &bytes.Buffer{}); err == nil {
			t.Fatal("accepted bad attachment coordinate/type")
		}
		files, _ := filepath.Glob(filepath.Join(root, "output/scenes/*.yaml"))
		if len(files) != 0 {
			t.Fatal("bad foot wrote snapshots")
		}
	}
}

func TestAttachedFizzOmissionAndEmptyListKeepLegacyOutputs(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	a := attachedFixture(t, first, "")
	b := attachedFixture(t, second, "attached-fizzes: []\n")
	for _, plan := range []string{a, b} {
		if err := generate(plan, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"scenes/frame-000000.yaml", "scenes/frame-000003.yaml", "jobs/frame-000003.yaml"} {
		old, _ := os.ReadFile(filepath.Join(first, "output", name))
		same, _ := os.ReadFile(filepath.Join(second, "output", name))
		if !bytes.Equal(old, same) {
			t.Fatalf("empty attachment list changed %s", name)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(first, "output/scenes/frame-000003.yaml"))
	var doc yaml.Node
	yaml.Unmarshal(raw, &doc)
	foot, _ := nodeAt(&doc, "scene", "world", "fizzes", "residual", "foot")
	if foot.Content[0].Value != "10" || foot.Content[1].Value != "20" {
		t.Fatal("omitted attachment moved a population")
	}
}

func TestAttachedFizzRefusesWrongWorldUnitBeforeWriting(t *testing.T) {
	root := t.TempDir()
	plan := attachedFixture(t, root, "attached-fizzes: [residual]\n")
	raw, _ := os.ReadFile(filepath.Join(root, "template.yaml"))
	raw = bytes.Replace(raw, []byte("unit: 1"), []byte("unit: 2"), 1)
	os.WriteFile(filepath.Join(root, "template.yaml"), raw, 0644)
	if err := generate(plan, &bytes.Buffer{}); err == nil {
		t.Fatal("translated non-metre world as metres")
	}
	files, _ := filepath.Glob(filepath.Join(root, "output/scenes/*.yaml"))
	if len(files) != 0 {
		t.Fatal("wrong world unit wrote snapshots")
	}
}
