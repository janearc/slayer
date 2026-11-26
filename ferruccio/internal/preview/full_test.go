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

func TestFullRenderFreezesChosenOutputSizeWithoutChangingSampling(t *testing.T) {
	for _, name := range []string{"first-picture.yaml", "ferruccio-playground.yaml"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "examples", name))
			if err != nil {
				t.Fatal(err)
			}
			plan, err := Prepare(Request{Path: filepath.Join(t.TempDir(), name), Text: string(raw), Profile: Profile{Full: true, Width: 1080, Height: 960, SizePreset: "half-portrait"}})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()
			args, err := plan.arguments(Tools{Slayer: "slayer"})
			want := []string{"slayer", "draw", plan.Input, "--out", plan.Output, "--width", "1080", "--height", "960", "--quality", "exact"}
			if err != nil || !reflect.DeepEqual(args, want) || plan.Profile.Rays != 1 {
				t.Fatalf("preset request = %v, profile %+v, %v", args, plan.Profile, err)
			}
		})
	}
}

func TestInspectFullResolvesCostWithoutCreatingFiles(t *testing.T) {
	root := t.TempDir()
	source := strings.Replace(string(firstPicture(t)), "rays: 1", "rays: 3", 1)
	profile, err := InspectFull(filepath.Join(root, "unsaved.yaml"), source, Profile{Width: 1080, Height: 960, SizePreset: "half-portrait"})
	if err != nil || profile.Width != 1080 || profile.Height != 960 || profile.Rays != 3 {
		t.Fatalf("inspection = %+v, %v", profile, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("inspection created files: %v, %v", entries, err)
	}
}

func TestFullRenderPassesChosenCameraToProducer(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "full-render-check.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Prepare(Request{Path: filepath.Join(t.TempDir(), "scene.yaml"), Text: string(raw), Profile: Profile{Full: true, Camera: "arrival"}})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	args, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil || !strings.Contains(strings.Join(args, " "), "--camera arrival") {
		t.Fatalf("chosen camera arguments = %v, %v", args, err)
	}
}

func TestFullGPUChoiceUsesGPULaneAndRenderer(t *testing.T) {
	plan := Plan{Input: "/scene.yaml", Output: "/image.png", Kind: "set", Profile: Profile{Full: true, Width: 320, Height: 180, Rays: 1, GPU: true}}
	args, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil || plan.lane() != "gpu" || !strings.Contains(strings.Join(args, " "), "--renderer gpu") {
		t.Fatalf("GPU request = %v, lane %q, %v", args, plan.lane(), err)
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
	plan := Plan{Directory: root, Output: filepath.Join(root, "preview.png"), Kind: "set", Profile: Profile{Full: true, Width: 12, Height: 8, Rays: 3, CPURenderer: true, SizePreset: "chosen", Camera: "arrival"}}
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
	if receipt["preview"] != false || receipt["quality"] != "exact" || receipt["rays_per_axis"] != float64(3) || receipt["width"] != float64(12) || receipt["height"] != float64(8) || receipt["size_preset"] != "chosen" || receipt["camera_override"] != "arrival" {
		t.Fatalf("false full evidence: %v", receipt)
	}
}

func TestFullRenderDerivesPhysicalRigRowsAndChecksPresetShape(t *testing.T) {
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Rig
  setting: space
  film: {width: 640, rays: 2, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}
  cameras:
    view: {location: {at: [0, -10, 0]}, orientation: {look_at: [0, 0, 0]}}
  set:
    things:
      first: {kind: yurt, seed: 1, east: 0, north: 0, height: {value: 0, unit: m}}
`
	path := filepath.Join(t.TempDir(), "rig.yaml")
	profile, err := InspectFull(path, source, Profile{})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Width != 640 || profile.Height != 427 || profile.Rays != 2 {
		t.Fatalf("derived film %+v", profile)
	}
	plan, err := Prepare(Request{Path: path, Text: source, Profile: Profile{Full: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if plan.Profile.Height != 427 {
		t.Fatalf("frozen film %+v", plan.Profile)
	}
	if _, err := InspectFull(path, source, Profile{SizePreset: "wide", Width: 640, Height: 360}); err == nil {
		t.Fatal("incompatible preset accepted")
	}
	chosen, err := InspectFull(path, source, Profile{SizePreset: "matching", Width: 640, Height: 426})
	if err != nil || chosen.Height != 427 {
		t.Fatalf("compatible preset %+v, %v", chosen, err)
	}
}
