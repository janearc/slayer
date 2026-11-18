package preview

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestFullRenderPreservesAuthoredFilmAndAvoidsPreviewOverrides locks the producer contract.
func TestFullRenderPreservesAuthoredFilmAndAvoidsPreviewOverrides(t *testing.T) {
	for _, name := range []string{"first-picture.yaml", "ferruccio-playground.yaml"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "examples", name))
			if err != nil {
				t.Fatal(err)
			}
			source := strings.ReplaceAll(string(raw), "rays: 1", "rays: 3")
			root := t.TempDir()
			plan, err := Prepare(Request{Path: filepath.Join(root, name), Text: source, Profile: Profile{Full: true, CPURenderer: true}})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()
			got, err := plan.arguments(Tools{Slayer: "slayer"})
			want := []string{"slayer", "draw", plan.Input, "--out", plan.Output, "--quality", "exact", "--renderer", "cpu"}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("full arguments %v %v", got, err)
			}
			if plan.Profile.Rays != 3 || plan.lane() != "tests" {
				t.Fatalf("lost full sampling: %+v", plan.Profile)
			}
			frozen, err := os.ReadFile(plan.Input)
			if err != nil || !strings.Contains(string(frozen), "rays: 3") {
				t.Fatalf("sampling mutated: %s %v", frozen, err)
			}
			if name == "first-picture.yaml" && (plan.Profile.Width != 160 || plan.Profile.Height != 90) {
				t.Fatalf("authored job dimensions changed %+v", plan.Profile)
			}
			if name == "ferruccio-playground.yaml" && (plan.Profile.Width != 640 || plan.Profile.Height != 400) {
				t.Fatalf("authored set dimensions changed %+v", plan.Profile)
			}
		})
	}
}

// TestFullRenderRefusesUnconnectedFilmBeforeArtifacts prevents a silent preview fallback.
func TestFullRenderRefusesUnconnectedFilmBeforeArtifacts(t *testing.T) {
	source := strings.Replace(string(firstPicture(t)), "film: {width: 160, height: 90, rays: 1}", "film: {preset: 720}", 1)
	root := t.TempDir()
	plan, err := Prepare(Request{Path: filepath.Join(root, "scene.yaml"), Text: source, OutputRoot: root, Profile: Profile{Full: true}})
	if plan != nil {
		plan.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "explicit authored film") {
		t.Fatalf("unsupported film accepted: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("refused render created artifacts: %v %v", entries, err)
	}
}

// TestFullCatalogueRefusesBeforeSnapshot prevents reduced catalogue settings masquerading as full.
func TestFullCatalogueRefusesBeforeSnapshot(t *testing.T) {
	root := t.TempDir()
	plan, err := Prepare(Request{Path: filepath.Join(root, "settings.yaml"), Text: string(catalogueSettings(t)), Specimen: "subaru", Profile: Profile{Full: true}})
	if plan != nil {
		plan.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "not supported for catalogue") {
		t.Fatalf("catalogue full accepted: %v", err)
	}
}

// TestFullReceiptRecordsDimensionsAndSampling binds evidence to the authored request.
func TestFullReceiptRecordsDimensionsAndSampling(t *testing.T) {
	root := t.TempDir()
	plan := Plan{Directory: root, Output: filepath.Join(root, "preview.png"), Kind: "set", Profile: Profile{Full: true, Width: 12, Height: 8, Rays: 3, CPURenderer: true}}
	file, err := os.Create(plan.Output)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewNRGBA(image.Rect(0, 0, 12, 8))); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := plan.record(Tools{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt["preview"] != false || receipt["quality"] != "exact" || receipt["rays_per_axis"] != float64(3) || receipt["width"] != float64(12) || receipt["height"] != float64(8) {
		t.Fatalf("false full evidence: %v", receipt)
	}
}
