package draft

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func cameraMoveFixture(t *testing.T) *Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "ferruccio-playground.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := Open("ferruccio-playground.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Check(); err != nil {
		t.Fatal(err)
	}
	return document
}

// TestMoveCameraTranslatesEyeAndTargetPreservingScene checks authored ENU values
// while comparing the entire remaining scene, including lens and lighting.
func TestMoveCameraTranslatesEyeAndTargetPreservingScene(t *testing.T) {
	document := cameraMoveFixture(t)
	original := document.Text()
	var before, after map[string]any
	if err := yaml.Unmarshal([]byte(original), &before); err != nil {
		t.Fatal(err)
	}
	if err := document.MoveCamera(3, -5, 7); err != nil {
		t.Fatal(err)
	}
	nodes, err := document.simpleSetCamera()
	if err != nil {
		t.Fatal(err)
	}
	at, err := readVector(nodes.at, "location.at")
	if err != nil {
		t.Fatal(err)
	}
	target, err := readVector(nodes.lookAt, "orientation.look_at")
	if err != nil {
		t.Fatal(err)
	}
	if at != [3]float64{21, -29, 19} || target != [3]float64{4, -5, 9} {
		t.Fatalf("wrong ENU translation eye=%v target=%v", at, target)
	}
	if err := document.Check(); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte(document.Text()), &after); err != nil {
		t.Fatal(err)
	}
	for _, tree := range []map[string]any{before, after} {
		camera := tree["scene"].(map[string]any)["cameras"].(map[string]any)["arrival"].(map[string]any)
		delete(camera["location"].(map[string]any), "at")
		delete(camera["orientation"].(map[string]any), "look_at")
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("camera movement changed other authored scene fields")
	}
	for _, line := range strings.Split(original, "\n") {
		if strings.HasPrefix(line, "#") && !strings.Contains(document.Text(), line) {
			t.Fatalf("lost comment %q", line)
		}
	}
}

// TestMoveCameraNoOpKeepsExactSource avoids spurious draft/history edits.
func TestMoveCameraNoOpKeepsExactSource(t *testing.T) {
	document := cameraMoveFixture(t)
	original := document.Text()
	if err := document.MoveCamera(0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if document.Text() != original {
		t.Fatal("no-op rewrote source")
	}
}

// TestMoveCameraRefusesInvalidDeltaAndPrecisionCollapse keeps failed edits atomic.
func TestMoveCameraRefusesInvalidDeltaAndPrecisionCollapse(t *testing.T) {
	for _, delta := range [][3]float64{{math.NaN(), 0, 0}, {0, math.Inf(1), 0}, {0, 0, math.Inf(-1)}, {1e308, 1e308, 1e308}} {
		document := cameraMoveFixture(t)
		original := document.Text()
		if err := document.MoveCamera(delta[0], delta[1], delta[2]); err == nil {
			t.Fatalf("accepted unsafe translation %v", delta)
		}
		if document.Text() != original {
			t.Fatalf("failed translation changed source: %v", delta)
		}
	}
}

// TestMoveCameraRefusesUnsupportedAndInvalidDocuments avoids invented camera semantics.
func TestMoveCameraRefusesUnsupportedAndInvalidDocuments(t *testing.T) {
	for _, kind := range []string{"job", "catalogue", "legacy-set"} {
		document := cameraMoveFixture(t)
		document.kind = kind
		original := document.Text()
		if err := document.MoveCamera(1, 2, 3); err == nil {
			t.Fatalf("accepted unsupported %s", kind)
		}
		if document.Text() != original {
			t.Fatalf("unsupported %s changed source", kind)
		}
	}
	document := cameraMoveFixture(t)
	document.Replace(strings.Replace(document.Text(), "look_at: [1, 0, 2]", "look_at: [bad, 0, 2]", 1))
	original := document.Text()
	if err := document.MoveCamera(1, 2, 3); err == nil {
		t.Fatal("accepted invalid original camera")
	}
	if document.Text() != original {
		t.Fatal("invalid candidate changed source")
	}
}
