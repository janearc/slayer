package preview

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

func TestFullNativeGPUUsesNativeRendererAndGPUQueue(t *testing.T) {
	plan := Plan{Input: "/scene.yaml", Output: "/image.png", Kind: "set",
		Profile: Profile{Full: true, Width: 320, Height: 180, Rays: 1, NativeGPU: true}}
	args, err := plan.arguments(Tools{Slayer: "/slayer"})
	if err != nil || !strings.Contains(strings.Join(args, " "), "--renderer gpu-native") || plan.lane() != "gpu" {
		t.Fatalf("native GPU request = %v, lane %q, %v", args, plan.lane(), err)
	}
	job := "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\n" +
		"scene: {smoke: 1}\nfilm: {width: 320, height: 180, samples: 7}\n"
	for _, selected := range []int{0, 7} {
		profile, err := InspectFull("job.yaml", job, Profile{NativeGPU: true, Samples: selected})
		if err != nil || profile.Samples != 7 || !profile.NativeGPU || profile.CPURenderer {
			t.Fatalf("native GPU exact profile = %+v, %v", profile, err)
		}
		plan.Profile = profile
		exact, err := plan.arguments(Tools{Slayer: "/slayer"})
		if err != nil || !strings.Contains(strings.Join(exact, " "), "--spp 7 --quality exact --renderer gpu-native") {
			t.Fatalf("native GPU exact request = %v, %v", exact, err)
		}
	}
}

func TestRenderArgumentsRefuseConflictingBackends(t *testing.T) {
	for _, profile := range []Profile{
		{CPURenderer: true, GPU: true},
		{Full: true, CPURenderer: true, GPU: true},
		{Full: true, CPURenderer: true, GPU: true, Samples: 7},
		{Full: true, CPURenderer: true, NativeGPU: true},
		{Full: true, GPU: true, NativeGPU: true},
	} {
		profile.Width, profile.Height = 32, 24
		plan := Plan{Input: "/scene.yaml", Output: "/image.png", Kind: "set", Profile: profile}
		if args, err := plan.arguments(Tools{Slayer: "/slayer"}); err == nil {
			t.Errorf("conflicting backend profile %+v produced arguments %v", profile, args)
		}
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
	if receipt["preview"] != false || receipt["quality"] != "exact" || receipt["rays_per_axis"] != float64(3) || receipt["samples_per_pixel"] != float64(9) || receipt["width"] != float64(12) || receipt["height"] != float64(8) || receipt["size_preset"] != "chosen" || receipt["camera_override"] != "arrival" {
		t.Fatalf("false full evidence: %v", receipt)
	}

	plan.Profile.Samples = 7
	if err := plan.record(Tools{}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	receipt = nil
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt["samples_per_pixel"] != float64(7) || receipt["rays_per_axis"] != nil {
		t.Fatalf("explicit count used the authored ray grid: %v", receipt)
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

func TestFullJobFollowsReferencedRigAndRecordsAuthoredStock(t *testing.T) {
	root := t.TempDir()
	setPath := filepath.Join(root, "rig.yaml")
	if err := os.WriteFile(setPath, []byte(rigFixture), 0600); err != nil {
		t.Fatal(err)
	}
	source := rigJobFixture
	plan, err := Prepare(Request{Path: filepath.Join(root, "job.yaml"), Text: source, Profile: Profile{Full: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if profile := plan.Profile; profile.Width != 18 || profile.Height != 12 || profile.Rays != 2 ||
		profile.Glimpses() != 2 || profile.PrimarySamples().String() != "1728" {
		t.Fatalf("resolved film = %+v, primary samples %s", profile, profile.PrimarySamples())
	}
	file, err := os.Create(plan.Output)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewNRGBA(image.Rect(0, 0, 18, 12))); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := plan.record(Tools{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(plan.Directory, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Rig           physicalRig       `json:"physical_rig"`
		Stock         developmentStock  `json:"development_stock"`
		Samples       int               `json:"samples_per_pixel"`
		Total         int               `json:"primary_samples_requested"`
		SourceSHA256  string            `json:"source_sha256"`
		ReferencedSet map[string]string `json:"referenced_set"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Rig != (physicalRig{WidthMM: 36, HeightMM: 24, FocalMM: 50, FStop: 8, Focus: "first", ShutterSeconds: 0.5, Glimpses: 2}) ||
		receipt.Stock.Name != "soot" || receipt.Stock.Seed == nil || *receipt.Stock.Seed != 0 ||
		receipt.Stock.Rated != nil || receipt.Samples != 4 || receipt.Total != 1728 {
		t.Fatalf("receipt = %s", raw)
	}
	if receipt.SourceSHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(source))) ||
		receipt.ReferencedSet["path"] != setPath ||
		receipt.ReferencedSet["sha256"] != fmt.Sprintf("%x", sha256.Sum256([]byte(rigFixture))) {
		t.Fatalf("receipt did not identify both job and set: %s", raw)
	}
}

const rigFixture = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
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

const rigJobFixture = `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: rig.yaml}
renderer: cpu
film: {width: 18, height: 11}
stock: {name: soot, seed: 0}
at: {value: 0, unit: s}
`

func TestFullJobRefusesUnsupportedRigOverrides(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "rig.yaml"), []byte(rigFixture), 0600); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		name, source string
		profile      Profile
		want         string
	}{
		{"shape", strings.Replace(rigJobFixture, "height: 11", "height: 9", 1), Profile{}, "does not match the physical frame"},
		{"angle", strings.Replace(rigJobFixture, "height: 11", "height: 11, wide: 30", 1), Profile{}, "takes its exposure and angle from the rig"},
		{"authored exact", strings.Replace(rigJobFixture, "height: 11", "height: 11, samples: 7", 1), Profile{}, "do not yet support physical camera rigs"},
		{"selected exact", rigJobFixture, Profile{Samples: 7}, "do not yet support physical camera rigs"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			_, err := InspectFull(filepath.Join(root, "job.yaml"), entry.source, entry.profile)
			if err == nil || !strings.Contains(err.Error(), entry.want) {
				t.Fatalf("error = %v, want %q", err, entry.want)
			}
		})
	}
}

func TestFullJobReadsExactSamplingFromTheAuthoredFilm(t *testing.T) {
	source := strings.Replace(string(firstPicture(t)), "rays: 1", "samples: 7", 1)
	path := filepath.Join(t.TempDir(), "still.yaml")
	profile, err := InspectFull(path, source, Profile{})
	if err != nil || profile.Samples != 7 || profile.SamplesPerPixel().Int64() != 7 {
		t.Fatalf("authored exact setting = %+v, %v", profile, err)
	}
	for _, shape := range []string{
		"samples: 7, rays: 2",
		"samples: 7, region: {left: 0, top: 0, width: 4, height: 4}",
		"samples: 65",
	} {
		invalid := strings.Replace(source, "samples: 7", shape, 1)
		if _, err := InspectFull(path, invalid, Profile{}); err == nil {
			t.Fatalf("accepted invalid exact film %s", invalid)
		}
	}
}

func TestFullReferenceMustMatchTheOwnersFingerprint(t *testing.T) {
	plan := Plan{Profile: Profile{reference: filmReference{path: "/source/set.yaml", digest: "expected"}}}
	for _, files := range [][]queueFile{
		nil,
		{{Path: "/source/set.yaml", SHA256: "changed"}},
		{{Path: "/source/another.yaml", SHA256: "expected"}},
	} {
		if err := plan.verifyFilmReference(queueJob{Files: files}); err == nil {
			t.Fatalf("accepted unmatched set fingerprint %+v", files)
		}
	}
	if err := plan.verifyFilmReference(queueJob{Files: []queueFile{{Path: "/source/set.yaml", SHA256: "expected"}}}); err != nil {
		t.Fatal(err)
	}
}

func TestFullStillResolvesAuthoredJobPreset(t *testing.T) {
	dir := t.TempDir()
	setSource, err := os.ReadFile(filepath.Join("..", "..", "examples", "ferruccio-playground.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scene.yaml"), setSource, 0600); err != nil {
		t.Fatal(err)
	}
	const job = `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: `
	for _, test := range []struct {
		name, film    string
		width, height int
	}{
		{"inherits set dimensions", "{}", 640, 400},
		{"preset before set dimensions", "{preset: half-portrait}", 1080, 960},
		{"explicit width", "{preset: imax, width: 123}", 123, 3002},
		{"explicit height", "{preset: imax, height: 456}", 4096, 456},
		{"explicit width and height", "{preset: imax, width: 123, height: 456}", 123, 456},
		{"bare numeric preset", "{preset: 720}", 1280, 720},
		{"rate-bearing still preset", "{preset: portrait-preview}", 360, 640},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := job + test.film + "\n"
			plan, err := Prepare(Request{Path: filepath.Join(dir, "job.yaml"), Text: source, Profile: Profile{Full: true}})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()
			if plan.Profile.Width != test.width || plan.Profile.Height != test.height || plan.Profile.Rays != 1 {
				t.Fatalf("film %+v, want %d × %d at one ray per axis", plan.Profile, test.width, test.height)
			}
			args, err := plan.arguments(Tools{Slayer: "slayer"})
			want := []string{"slayer", "draw", plan.Input, "--out", plan.Output, "--quality", "exact"}
			if err != nil || !reflect.DeepEqual(args, want) {
				t.Fatalf("authored settings changed producer arguments: %v, %v", args, err)
			}
		})
	}
}

func TestPhysicalFilmUsesPresetHeightAsShapeConstraint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rig.yaml"), []byte(rigFixture), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "job.yaml")
	compatible := strings.Replace(rigJobFixture, "width: 18, height: 11", "preset: half-portrait, width: 1440", 1)
	profile, err := InspectFull(path, compatible, Profile{})
	if err != nil || profile.Width != 1440 || profile.Height != 960 {
		t.Fatalf("compatible preset %+v, %v", profile, err)
	}
	incompatible := strings.Replace(compatible, ", width: 1440", "", 1)
	if _, err := InspectFull(path, incompatible, Profile{}); err == nil || !strings.Contains(err.Error(), "does not match the physical frame") {
		t.Fatalf("incompatible preset error = %v", err)
	}
}

func TestFullRenderRefusesMovieBeforeCreatingArtifacts(t *testing.T) {
	const source = `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {smoke: 1}
film: {width: 12, height: 8, rays: 1}
`
	for _, timing := range []string{
		"fps: 24\nduration: {value: 1, unit: s}",
		"fps: 24",
		"duration: {value: 1, unit: s}",
		"start: {value: 0, unit: s}",
		"scene-duration: {value: 1, unit: s}",
	} {
		t.Run(timing, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "movie.yaml")
			movie := source + timing + "\n"
			profile := Profile{Full: true}
			if _, err := InspectFull(path, movie, profile); err == nil || !strings.Contains(err.Error(), "movie jobs") {
				t.Fatalf("movie inspection error = %v", err)
			}
			plan, err := Prepare(Request{Path: path, Text: movie, OutputRoot: dir, Profile: profile})
			if plan != nil {
				defer plan.Close()
			}
			if err == nil || !strings.Contains(err.Error(), "movie jobs") {
				t.Fatalf("movie preparation error = %v", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("movie created artifacts: %v, %v", entries, err)
			}
		})
	}
}

func TestFullStillReceivesTheAuthoredRegionAtLegacySampling(t *testing.T) {
	dir := t.TempDir()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "ferruccio-playground.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scene.yaml"), source, 0600); err != nil {
		t.Fatal(err)
	}
	const job = `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 64, height: 48, rays: 2, region: {left: 8, top: 6, width: 12, height: 10}}
`
	path := filepath.Join(dir, "job.yaml")
	for _, test := range []struct{ source, reason string }{
		{strings.Replace(job, "rays: 2", "rays: 2, mode: wire", 1), "wire"},
		{strings.Replace(job, "rays: 2", "rays: 2, aperture: 0.01, focus: 2", 1), "aperture"},
		{strings.Replace(job, "set: scene.yaml", "smoke: 1", 1), "referenced set"},
		{strings.Replace(job, "set: scene.yaml", "named: fire-night", 1), "referenced set"},
		{job + "stock: {name: soot}\n", "stock"},
	} {
		if _, err := InspectFull(path, test.source, Profile{}); err == nil ||
			!strings.Contains(err.Error(), "film region") || !strings.Contains(err.Error(), test.reason) {
			t.Errorf("region with %s did not report its incompatibility: %v", test.reason, err)
		}
	}
	for _, profile := range []Profile{
		{Full: true, Samples: 7},
		{Full: true, NativeGPU: true},
		{Full: true, SizePreset: "too small", Width: 19, Height: 16},
	} {
		if _, err := InspectFull(path, job, profile); err == nil {
			t.Fatalf("accepted incompatible region settings: %+v", profile)
		}
	}

	profile := Profile{Full: true, SizePreset: "test", Width: 32, Height: 24}
	plan, err := Prepare(Request{Path: path, Text: job, Profile: profile})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	args, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil || !strings.Contains(strings.Join(args, " "), "--width 32 --height 24") {
		t.Fatalf("arguments lost full-frame dimensions: %v, %v", args, err)
	}
	if got := plan.Profile.PrimarySamples().Int64(); got != 480 {
		t.Errorf("estimated %d samples, want 480 for the crop", got)
	}

	queuePNG(t, plan, 32, 24)
	if err := plan.record(Tools{}); err == nil {
		t.Error("accepted a full-frame PNG for a region")
	}
	queuePNG(t, plan, 12, 10)
	if err := plan.record(Tools{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(plan.Directory, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Width   int            `json:"width"`
		Height  int            `json:"height"`
		Samples int            `json:"samples_per_pixel"`
		Crop    map[string]int `json:"region"`
	}
	want := map[string]int{"left": 8, "top": 6, "width": 12, "height": 10,
		"frame_width": 32, "frame_height": 24}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Width != 12 || got.Height != 10 || got.Samples != 4 || !reflect.DeepEqual(got.Crop, want) {
		t.Fatalf("incorrect region receipt: %s", raw)
	}
}
