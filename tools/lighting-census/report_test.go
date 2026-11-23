package census

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

const reportTestDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestReportKeepsUnrunUnsupportedAndMeasuredSeparate(t *testing.T) {
	root := t.TempDir()
	cases := []Case{reportCase("cpu"), reportCase("gpu-cached-lighting"), reportCase("gpu-fast-lighting"), reportCase("gpu-faster-lighting"), reportCase("gpu-native-shadows")}
	reportWrite(t, filepath.Join(root, "manifest.json"), map[string]any{"version": 1, "identity": map[string]string{"executable_sha256": reportTestDigest}, "options": map[string]any{"scene_selection": "all", "mode_selection": "all", "times_ns": []int64{0}, "width": 1, "height": 1}, "cases": cases})
	reportSaveFrame(t, root, cases[0], "ok", 0, 4)
	reportSaveFrame(t, root, cases[1], "ok", 0, 3)
	reportWrite(t, filepath.Join(root, "scene-001", "gpu-fast-lighting", "result.json"), map[string]any{"executable_sha256": reportTestDigest, "case": cases[2], "status": "unsupported", "reason": "test refusal"})
	reportWrite(t, filepath.Join(root, "scene-001", "gpu-faster-lighting", "result.json"), map[string]any{"executable_sha256": reportTestDigest, "case": cases[3], "status": "failed", "reason": "test failure"})
	report, err := Summarize(root, "cpu")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cases) != len(cases) {
		t.Fatalf("got %d cases", len(report.Cases))
	}
	if report.Cases[0].Status != "reference" || report.Cases[1].Status != "measured" || report.Cases[2].Status != "unsupported" || report.Cases[3].Status != "failed" || report.Cases[4].Status != "missing" {
		t.Fatalf("incorrect statuses: %+v", report.Cases)
	}
	measured := report.Cases[1].Frames
	if len(measured) != 2 || measured[0].Difference == nil || measured[0].Difference.Relative == nil || *measured[0].Difference.Relative != .25 {
		t.Fatalf("incorrect comparison: %+v", measured)
	}
	if measured[0].ReferenceDurationNS != 10 || measured[0].CandidateDurationNS != 10 || !measured[0].TimingComparable || measured[0].ReferencePNG != "scene-001/cpu/frame-000.png" || measured[0].CandidatePNG != "scene-001/gpu-cached-lighting/frame-000.png" {
		t.Fatalf("missing timing or previews: %+v", measured[0])
	}
	if _, err := Summarize(root, "gpu-fast-lighting"); err == nil {
		t.Fatal("approximate lighting was allowed as a reference")
	}
}

func TestGPUComparisonWithUnknownDeviceDoesNotClaimComparableTimings(t *testing.T) {
	root := t.TempDir()
	direct, cached := reportCase("gpu-native-meshes"), reportCase("gpu-cached-lighting")
	reportWrite(t, filepath.Join(root, "manifest.json"), map[string]any{"version": 1, "identity": map[string]string{"executable_sha256": reportTestDigest}, "options": map[string]any{"scene_selection": "all", "mode_selection": "all", "times_ns": []int64{0}, "width": 1, "height": 1}, "cases": []Case{direct, cached}})
	reportSaveFrame(t, root, direct, "ok", 0, 4)
	reportSaveFrame(t, root, cached, "ok", 0, 3)
	dropDevice(t, filepath.Join(root, "scene-001", "gpu-cached-lighting", "result.json"))
	report, err := Summarize(root, "gpu-native-meshes")
	if err != nil || report.Cases[1].Status != "measured" || len(report.Cases[1].Frames) == 0 || report.Cases[1].Frames[0].TimingComparable {
		t.Fatalf("unknown GPU device was treated as matching: %+v, %v", report, err)
	}
}

func TestCPUGPUComparisonWithUnknownDeviceDoesNotClaimComparableTimings(t *testing.T) {
	root := t.TempDir()
	cpu, gpu := reportCase("cpu"), reportCase("gpu-native-meshes")
	reportWrite(t, filepath.Join(root, "manifest.json"), map[string]any{"version": 1, "identity": map[string]string{"executable_sha256": reportTestDigest}, "options": map[string]any{"scene_selection": "all", "mode_selection": "all", "times_ns": []int64{0}, "width": 1, "height": 1}, "cases": []Case{cpu, gpu}})
	reportSaveFrame(t, root, cpu, "ok", 0, 4)
	reportSaveFrame(t, root, gpu, "ok", 0, 3)
	dropDevice(t, filepath.Join(root, "scene-001", "gpu-native-meshes", "result.json"))
	report, err := Summarize(root, "cpu")
	if err != nil || report.Cases[1].Status != "measured" || len(report.Cases[1].Frames) == 0 || report.Cases[1].Frames[0].TimingComparable {
		t.Fatalf("unknown GPU device was treated as known: %+v, %v", report, err)
	}
}

func dropDevice(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	for _, f := range result["frames"].([]any) {
		delete(f.(map[string]any), "profile")
	}
	reportWrite(t, path, result)
}

func TestReportRejectsMismatchAndDoesNotCallMissingReferencePass(t *testing.T) {
	root := t.TempDir()
	cpu, gpu := reportCase("cpu"), reportCase("gpu-native-meshes")
	reportWrite(t, filepath.Join(root, "manifest.json"), map[string]any{"version": 1, "identity": map[string]string{"executable_sha256": reportTestDigest}, "options": map[string]any{"scene_selection": "all", "mode_selection": "all", "times_ns": []int64{0}, "width": 1, "height": 1}, "cases": []Case{cpu, gpu}})
	reportSaveFrame(t, root, gpu, "ok", 0, 2)
	report, err := Summarize(root, "cpu")
	if err != nil || report.Cases[1].Status != "missing-reference" {
		t.Fatalf("missing reference: %+v, %v", report, err)
	}
	reportSaveFrame(t, root, cpu, "ok", 1, 2)
	report, err = Summarize(root, "cpu")
	if err != nil || report.Cases[1].Status != "missing-reference" || len(report.Cases[1].Frames) != 0 {
		t.Fatalf("different scene times were compared: %+v, %v", report, err)
	}
}

func TestReportRejectsChangedRecordedCaseMetadata(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"field of view", func(c map[string]any) { c["field_of_view_radians"] = 2.0 }},
		{"bands", func(c map[string]any) { c["additional_spectral_bands"] = 2 }},
		{"input declarations", func(c map[string]any) { c["declared_inputs"] = []string{"changed"} }},
		{"mode definition", func(c map[string]any) { c["mode"].(map[string]any)["fidelity"] = "reference" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			cpu, gpu := reportCase("cpu"), reportCase("gpu-native-meshes")
			reportWrite(t, filepath.Join(root, "manifest.json"), map[string]any{"version": 1, "identity": map[string]string{"executable_sha256": reportTestDigest}, "options": map[string]any{"scene_selection": "all", "mode_selection": "all", "times_ns": []int64{0}, "width": 1, "height": 1}, "cases": []Case{cpu, gpu}})
			reportSaveFrame(t, root, cpu, "ok", 0, 2)
			reportSaveFrame(t, root, gpu, "ok", 0, 2)
			path := filepath.Join(root, "scene-001", "gpu-native-meshes", "result.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			test.edit(result["case"].(map[string]any))
			reportWrite(t, path, result)
			report, err := Summarize(root, "cpu")
			if err != nil || report.Cases[1].Status != "invalid" {
				t.Fatalf("changed case was accepted: %+v, %v", report, err)
			}
		})
	}
}

func TestReportRefusesAResultFromAnotherExecutable(t *testing.T) {
	root := t.TempDir()
	cpu, gpu := reportCase("cpu"), reportCase("gpu-native-meshes")
	reportWrite(t, filepath.Join(root, "manifest.json"), map[string]any{"version": 1, "identity": map[string]string{"executable_sha256": reportTestDigest}, "options": map[string]any{"scene_selection": "all", "mode_selection": "all", "times_ns": []int64{0}, "width": 1, "height": 1}, "cases": []Case{cpu, gpu}})
	reportSaveFrame(t, root, cpu, "ok", 0, 2)
	reportSaveFrame(t, root, gpu, "ok", 0, 2)
	path := filepath.Join(root, "scene-001", "gpu-native-meshes", "result.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	result["executable_sha256"] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	reportWrite(t, path, result)
	report, err := Summarize(root, "cpu")
	if err != nil || report.Cases[1].Status != "invalid" {
		t.Fatalf("accepted a different executable: %+v, %v", report, err)
	}
}

func TestReportRefusesAnApproximateModeRelabelledInManifestAndResult(t *testing.T) {
	root := t.TempDir()
	cpu, forged := reportCase("cpu"), reportCase("gpu-cached-lighting")
	forged.Mode.Fidelity = "reference"
	reportWrite(t, filepath.Join(root, "manifest.json"), map[string]any{"version": 1, "identity": map[string]string{"executable_sha256": reportTestDigest}, "options": map[string]any{"scene_selection": "all", "mode_selection": "all", "times_ns": []int64{0}, "width": 1, "height": 1}, "cases": []Case{cpu, forged}})
	reportSaveFrame(t, root, cpu, "ok", 0, 2)
	reportSaveFrame(t, root, forged, "ok", 0, 2)
	if _, err := Summarize(root, "cpu"); err == nil {
		t.Fatal("accepted a manifest and result that relabel approximate lighting")
	}
}

func reportCase(name string) Case {
	for _, mode := range Modes() {
		if mode.Name == name {
			return Case{Scene: 1, Samples: 1, Mode: mode, Status: "not-run"}
		}
	}
	panic("unknown test mode " + name)
}

func reportSaveFrame(t *testing.T, root string, c Case, status string, when int64, red float32) {
	t.Helper()
	dir := filepath.Join(root, "scene-001", c.Mode.Name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	f := frame.New(1, 1)
	f.Pix[0].R = red
	if err := raw.Write(filepath.Join(dir, "frame-000.pfm"), f); err != nil {
		t.Fatal(err)
	}
	writeDetails(t, filepath.Join(dir, "frame-000.cd32"), f)
	if err := raw.Write(filepath.Join(dir, "frame-001.pfm"), f); err != nil {
		t.Fatal(err)
	}
	writeDetails(t, filepath.Join(dir, "frame-001.cd32"), f)
	for _, name := range []string{"frame-000.png", "frame-001.png"} {
		file, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		blank := image.NewRGBA(image.Rect(0, 0, 1, 1))
		if err := png.Encode(file, blank); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	reportWrite(t, filepath.Join(dir, "result.json"), map[string]any{
		"executable_sha256": reportTestDigest, "case": c, "status": status, "camera": "test", "width": 1, "height": 1,
		"input_sha256": map[string]string{"fixture": "digest"},
		"frames":       []any{map[string]any{"index": 0, "time_ns": when, "phase": "cold", "raw": "frame-000.pfm", "coverage_depth": "frame-000.cd32", "png": "frame-000.png", "duration_ns": 10, "profile": map[string]string{"Device": "test-gpu"}}, map[string]any{"index": 1, "time_ns": when, "phase": "same-time-repeat", "raw": "frame-001.pfm", "coverage_depth": "frame-001.cd32", "png": "frame-001.png", "duration_ns": 20, "profile": map[string]string{"Device": "test-gpu"}}},
	})
}

func reportWrite(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}
