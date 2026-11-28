package workspace

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestBundledCampVisitorSavesAndReopensAtTwoMetres(t *testing.T) {
	path, study, err := copyBundledStudy("../../../examples", campExample)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path))
	document, _, err := ReadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(t.TempDir(), "camp.yaml")
	if err := saveBundledStudyDocument(document, saved, filepath.Dir(path), study); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(saved)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := set.Read(saved, raw)
	if err != nil {
		t.Fatal(err)
	}
	world, ok := reopened.World()
	if !ok || len(world.Imports()) != 1 || world.Imports()[0].Name() != "visitor" {
		t.Fatal("saved camp lost the visitor")
	}
	parts := world.Imports()[0].Asset().Instances()
	if len(parts) != 1 {
		t.Fatalf("visitor has %d imported parts", len(parts))
	}
	box := parts[0].Geometry().Bounds()
	placed := parts[0].Placement()
	low, high := placed.Out(box.Min).Y, placed.Out(box.Max).Y
	if math.Abs(low) > 1e-9 || math.Abs(high-2) > 1e-9 {
		t.Fatalf("imported visitor bounds = %.12g to %.12g metres", low, high)
	}
	notice, err := os.ReadFile(filepath.Join(filepath.Dir(saved), "larson-camp-visitor-LICENSE.txt"))
	if err != nil || !strings.Contains(string(notice), "Max Planck") || !strings.Contains(string(notice), "Changes:") {
		t.Fatalf("saved camp did not carry the credit and change notice: %v", err)
	}
}
