package searching

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

func TestFlowGeometryReferenceLoadsExactValues(t *testing.T) {
	geometry := []byte(`paths:
- points: [[10, 0, 0], [9, 0, 0], [6, 0, 0]]
  width: 0.02
  delay-beats: 0.25
landmarks:
- at: [8, 2, -4]
  radius: 0.1
  rgb: [1, 0.5, 2]
clouds:
- at: [4, 5, 6]
  rgb: [0.1, 0.2, 0.3]
  radius: 2
  density: 0.04
  seed: 71
`)
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "geometry.yaml"), geometry, 0o600); err != nil {
		t.Fatal(err)
	}
	plan := &FlowPlan{Provenance: "authored-previs", Start: "1.1", Fold: "2.1", Answer: "3.1",
		Eye: [3]float64{0, 0, 10}, Wide: 60, ReturnPath: 0,
		Geometry: &FlowGeometryReference{File: "geometry.yaml", SHA: fmt.Sprintf("%x", sha256.Sum256(geometry))}}
	if err := resolveFlowGeometry(filepath.Join(directory, "shoot.yaml"), plan); err != nil {
		t.Fatal(err)
	}
	want := FlowGeometry{
		Paths:     []FlowPath{{Points: [][3]float64{{10, 0, 0}, {9, 0, 0}, {6, 0, 0}}, Width: 0.02, Delay: 0.25}},
		Landmarks: []FlowPoint{{At: [3]float64{8, 2, -4}, Radius: 0.1, RGB: [3]float64{1, 0.5, 2}}},
		Clouds:    []FlowCloud{{At: [3]float64{4, 5, 6}, RGB: [3]float64{0.1, 0.2, 0.3}, Radius: 2, Density: 0.04, Seed: 71}},
	}
	got := FlowGeometry{Paths: plan.Paths, Landmarks: plan.Landmarks, Clouds: plan.Clouds}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded geometry differs:\n got: %#v\nwant: %#v", got, want)
	}
	if plan.Eye != ([3]float64{0, 0, 10}) || plan.Start != "1.1" || plan.Geometry.File != "geometry.yaml" {
		t.Fatal("resolving geometry changed camera, timing or the pinned reference")
	}
}

func TestFlowGeometryReferenceRefusesMalformedInputs(t *testing.T) {
	geometry := []byte("paths: []\nlandmarks: []\nclouds: []\n")
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "geometry.yaml"), geometry, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "geometry.txt"), geometry, 0o600); err != nil {
		t.Fatal(err)
	}
	validSHA := fmt.Sprintf("%x", sha256.Sum256(geometry))
	for _, test := range []struct {
		name   string
		file   string
		sha    string
		bytes  []byte
		inline bool
	}{
		{name: "missing file", file: "missing.yaml", sha: validSHA},
		{name: "changed bytes", file: "geometry.yaml", sha: fmt.Sprintf("%x", sha256.Sum256([]byte("different"))), bytes: []byte("different")},
		{name: "malformed YAML", file: "geometry.yaml", sha: fmt.Sprintf("%x", sha256.Sum256([]byte("paths: ["))), bytes: []byte("paths: [")},
		{name: "oversize", file: "geometry.yaml", sha: validSHA, bytes: make([]byte, FlowGeometryMaxBytes+1)},
		{name: "bad digest", file: "geometry.yaml", sha: "ABC"},
		{name: "absolute path", file: filepath.Join(directory, "geometry.yaml"), sha: validSHA},
		{name: "parent traversal", file: "../geometry.yaml", sha: validSHA},
		{name: "wrong extension", file: "geometry.txt", sha: validSHA},
		{name: "inline conflict", file: "geometry.yaml", sha: validSHA, inline: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(directory, test.file)
			if test.bytes != nil {
				if err := os.WriteFile(path, test.bytes, 0o600); err != nil {
					t.Fatal(err)
				}
				defer os.WriteFile(path, geometry, 0o600)
			}
			plan := &FlowPlan{Geometry: &FlowGeometryReference{File: test.file, SHA: test.sha}}
			if test.inline {
				plan.Paths = []FlowPath{{Points: [][3]float64{{1, 0, 0}, {0, 0, 0}}}}
			}
			if err := resolveFlowGeometry(filepath.Join(directory, "shoot.yaml"), plan); err == nil {
				t.Fatal("malformed or changed geometry reference was accepted")
			}
			if len(plan.Landmarks) != 0 || len(plan.Clouds) != 0 {
				t.Fatal("failed reference partially populated the plan")
			}
		})
	}
}

func TestInlineFlowGeometryRemainsCompatible(t *testing.T) {
	want := []FlowPath{{Points: [][3]float64{{2, 0, 0}, {1, 0, 0}}}}
	plan := &FlowPlan{Paths: want}
	if err := resolveFlowGeometry(filepath.Join(t.TempDir(), "shoot.yaml"), plan); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Paths, want) {
		t.Fatal("legacy inline geometry changed")
	}
}

func TestYAMLFlowGeometryDistinguishesAbsentAndEmptySequences(t *testing.T) {
	var absent, empty FlowPlan
	if err := yamlfile.Read("absent.yaml", []byte("provenance: authored-previs\n"), &absent); err != nil {
		t.Fatal(err)
	}
	if err := yamlfile.Read("empty.yaml", []byte("provenance: authored-previs\npaths: []\nlandmarks: []\nclouds: []\n"), &empty); err != nil {
		t.Fatal(err)
	}
	if absent.Paths != nil || absent.Landmarks != nil || absent.Clouds != nil {
		t.Fatal("absent geometry keys were not nil")
	}
	if empty.Paths == nil || empty.Landmarks == nil || empty.Clouds == nil {
		t.Fatal("explicit empty geometry sequences lost their presence")
	}
	directory := t.TempDir()
	geometry := []byte("paths:\n- points: [[1, 0, 0], [0, 0, 0]]\n  width: 0.02\n  delay-beats: 0\nlandmarks: []\nclouds: []\n")
	if err := os.WriteFile(filepath.Join(directory, "geometry.yaml"), geometry, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, inline := range []string{"paths: []", "landmarks: []", "clouds: []",
		"paths:\n- points: [[1, 0, 0], [0, 0, 0]]\n  width: 0.02\n  delay-beats: 0"} {
		raw := []byte(fmt.Sprintf("geometry:\n  file: geometry.yaml\n  sha256: %x\n%s\n", sha256.Sum256(geometry), inline))
		var ambiguous FlowPlan
		if err := yamlfile.Read("ambiguous.yaml", raw, &ambiguous); err != nil {
			t.Fatal(err)
		}
		if err := resolveFlowGeometry(filepath.Join(directory, "shoot.yaml"), &ambiguous); err == nil {
			t.Fatalf("inline geometry %q was silently replaced by the reference", inline)
		}
	}
}

// Hashes below were computed independently from typed FlowGeometry values
// parsed from the original study files at the pre-migration HEAD, then checked
// against the referenced assets and the untouched embedded reveal source.
func TestPinnedStudyGeometryMatchesPreMigrationSnapshots(t *testing.T) {
	read := func(path string, embedded bool) FlowPlan {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if embedded {
			var plan FlowPlan
			if err := yamlfile.Read(path, raw, &plan); err != nil {
				t.Fatal(err)
			}
			return plan
		}
		var plan Plan
		if err := yamlfile.Read(path, raw, &plan); err != nil {
			t.Fatal(err)
		}
		if err := resolveFlowGeometry(path, plan.FlowStudy); err != nil {
			t.Fatal(err)
		}
		return *plan.FlowStudy
	}
	paths := []struct {
		name     string
		geometry FlowGeometry
		plan     FlowPlan
	}{
		{"flow", FlowGeometry{}, read("studies/heard-flow-720.yaml", false)},
		{"full", FlowGeometry{}, read("studies/heard-full-720.yaml", false)},
		{"depth", FlowGeometry{}, read("studies/heard-depth-refined-720.yaml", false)},
		{"glass-hour", FlowGeometry{}, read("studies/heard-glass-hour-full-320.yaml", false)},
		{"embedded-reveal", FlowGeometry{}, read("reveal-reference.yaml", true)},
	}
	for index := range paths {
		paths[index].geometry = FlowGeometry{Paths: paths[index].plan.Paths,
			Landmarks: paths[index].plan.Landmarks, Clouds: paths[index].plan.Clouds}
	}
	a, b := paths[0].geometry, paths[2].geometry
	if !reflect.DeepEqual(a, paths[1].geometry) || !reflect.DeepEqual(a, paths[4].geometry) {
		t.Fatal("flow, full-cut and embedded reveal geometry diverged")
	}
	if !reflect.DeepEqual(b, paths[3].geometry) {
		t.Fatal("depth-refined and glass-hour geometry diverged")
	}
	if len(a.Paths) != 120 || len(a.Landmarks) != 1600 || len(a.Clouds) != 28 ||
		len(b.Paths) != 120 || len(b.Landmarks) != 2600 || len(b.Clouds) != 18 {
		t.Fatalf("geometry counts changed: A=(%d,%d,%d) B=(%d,%d,%d)",
			len(a.Paths), len(a.Landmarks), len(a.Clouds), len(b.Paths), len(b.Landmarks), len(b.Clouds))
	}
	for _, check := range []struct {
		name string
		data FlowGeometry
		want string
	}{
		{"set A", a, "5414754239aca642be7e00245c48c0679149e13eae9b28f170fb86eb35d92683"},
		{"set B", b, "52374cba60c3555dea23f3fa174ab5ac4179c34e5523abd36d4da30efa47af34"},
	} {
		raw, err := json.Marshal(check.data)
		if err != nil {
			t.Fatal(err)
		}
		got := fmt.Sprintf("%x", sha256.Sum256(raw))
		if got != check.want {
			t.Errorf("%s differs from the pre-migration typed snapshot: %s", check.name, got)
		}
	}
	revealWant := FlowPlan{Provenance: "authored-previs", Start: "117.1",
		Fold: "121.1", Answer: "123.1", Eye: [3]float64{0, 0, 50},
		Wide: 72, ReturnPath: 31, Paths: a.Paths, Landmarks: a.Landmarks, Clouds: a.Clouds}
	if !reflect.DeepEqual(paths[4].plan, revealWant) {
		t.Fatal("the embedded reveal FlowPlan lost timing, camera, return style or geometry")
	}
}
