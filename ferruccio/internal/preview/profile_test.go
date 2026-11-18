package preview

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestCPUProfileUsesTestsLaneAndExplicitRenderer keeps CPU work off GPU waits.
func TestCPUProfileUsesTestsLaneAndExplicitRenderer(t *testing.T) {
	plan := Plan{Kind: "set", Input: "scene.yaml", Output: "preview.png", Profile: Profile{Width: 320, Height: 200, CPURenderer: true}}
	tools := Tools{Game: "game5", Slayer: "slayer"}
	got, err := plan.arguments(tools)
	want := []string{"slayer", "draw", "scene.yaml", "--out", "preview.png", "--width", "320", "--height", "200", "--rays", "1", "--quality", "preview", "--renderer", "cpu"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("CPU arguments %v, %v; want %v", got, err, want)
	}
}

// TestCatalogueProfileKeepsProducerAndGPULane preserves its renderer authority.
func TestCatalogueProfileKeepsProducerAndGPULane(t *testing.T) {
	plan := Plan{Kind: "catalogue", Input: "settings.yaml", Directory: "out", Specimen: "subaru", Profile: Profile{Width: 320, Height: 200, CPURenderer: true}}
	got, err := plan.arguments(Tools{Game: "game5", Catalogue: "catalogue"})
	want := []string{"catalogue", "render", "subaru", "--plan", "settings.yaml", "--out", filepath.Join("out", "book")}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("catalogue arguments %v, %v", got, err)
	}
}

// TestProfileEvidenceUsesUniqueChildrenAndPreservesSource binds real output destinations.
func TestProfileEvidenceUsesUniqueChildrenAndPreservesSource(t *testing.T) {
	source := string(firstPicture(t))
	root := t.TempDir()
	original := filepath.Join(t.TempDir(), "source.yaml")
	if err := os.WriteFile(original, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	var previous string
	for index := 0; index < 2; index++ {
		plan, err := Prepare(Request{Path: original, Text: source, OutputRoot: root, Profile: Profile{Width: 640, Height: 400, CPURenderer: true}})
		if err != nil {
			t.Fatal(err)
		}
		defer plan.Close()
		if filepath.Dir(plan.Directory) != root || !filepath.IsAbs(plan.Output) || plan.Directory == previous {
			t.Fatalf("evidence path not unique child: %+v", plan)
		}
		if plan.Profile.Width != 640 || plan.Profile.Height != 400 || !plan.Profile.CPURenderer {
			t.Fatalf("lost frozen profile: %+v", plan.Profile)
		}
		previous = plan.Directory
	}
	got, err := os.ReadFile(original)
	if err != nil || string(got) != source {
		t.Fatalf("source changed: %v", err)
	}
}

// TestProfileRefusesPartialOrNegativeDimensionsBeforeArtifacts rejects malformed work.
func TestProfileRefusesPartialOrNegativeDimensionsBeforeArtifacts(t *testing.T) {
	for _, profile := range []Profile{{Width: 320}, {Height: 200}, {Width: -1, Height: 200}, {Width: 320, Height: -1}} {
		root := t.TempDir()
		plan, err := Prepare(Request{Path: filepath.Join(root, "source.yaml"), Text: string(firstPicture(t)), OutputRoot: root, Profile: profile})
		if plan != nil {
			plan.Close()
		}
		if err == nil {
			t.Fatalf("accepted invalid profile %+v", profile)
		}
		entries, readErr := os.ReadDir(root)
		if readErr != nil || len(entries) != 0 {
			t.Fatalf("invalid profile wrote artifacts: %v %v", entries, readErr)
		}
	}
}

// TestMissingEvidenceParentDoesNotFallBackToTemporaryDirectory makes destination failure visible.
func TestMissingEvidenceParentDoesNotFallBackToTemporaryDirectory(t *testing.T) {
	root := t.TempDir()
	plan, err := Prepare(Request{Path: filepath.Join(root, "source.yaml"), Text: string(firstPicture(t)), OutputRoot: filepath.Join(root, "missing")})
	if plan != nil {
		plan.Close()
	}
	if err == nil {
		t.Fatal("missing evidence parent was ignored")
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("destination refusal wrote artifacts: %v %v", entries, readErr)
	}
}

// TestCatalogueSnapshotUsesFrozenDimensions leaves the original settings unchanged.
func TestCatalogueSnapshotUsesFrozenDimensions(t *testing.T) {
	source := catalogueSettings(t)
	original := settingsFile(t, source, "")
	plan, err := Prepare(Request{Path: original, Text: string(source), Specimen: "subaru", Profile: Profile{Width: 320, Height: 200}})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	snapshot, err := os.ReadFile(plan.Input)
	if err != nil {
		t.Fatal(err)
	}
	value := readMapping(t, plan.Input, snapshot)
	if value["width"] != 320 || value["height"] != 200 {
		t.Fatalf("snapshot dimensions %v x %v", value["width"], value["height"])
	}
	got, err := os.ReadFile(original)
	if err != nil || !reflect.DeepEqual(got, source) {
		t.Fatalf("catalogue source changed: %v", err)
	}
}

// TestReceiptUsesActualProfileDimensions rejects mismatched renderer output.
func TestReceiptUsesActualProfileDimensions(t *testing.T) {
	for _, height := range []int{400, 360} {
		root := t.TempDir()
		plan := Plan{Directory: root, Output: filepath.Join(root, "preview.png"), Kind: "set", Profile: Profile{Width: 640, Height: 400, CPURenderer: true}}
		file, err := os.Create(plan.Output)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(file, image.NewNRGBA(image.Rect(0, 0, 640, height))); err != nil {
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
		err = plan.record(Tools{Game: "game5", Slayer: "slayer"})
		receiptPath := filepath.Join(root, "receipt.json")
		if height != 400 {
			if err == nil {
				t.Fatal("accepted wrong output dimensions")
			}
			if _, statErr := os.Stat(receiptPath); !os.IsNotExist(statErr) {
				t.Fatalf("failed preview got receipt: %v", statErr)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(receiptPath)
		if err != nil {
			t.Fatal(err)
		}
		var receipt map[string]any
		if err = json.Unmarshal(raw, &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt["width"] != float64(640) || receipt["height"] != float64(400) || receipt["lane"] != "tests" || receipt["renderer_override"] != "cpu" {
			t.Fatalf("incorrect receipt %v", receipt)
		}
	}
}
