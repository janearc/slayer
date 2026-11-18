package census

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/janearc/slayer/render/frame"
)

// Report is an analysis of the saved cases, not an acceptance verdict.
type Report struct {
	Reference      string       `json:"reference"`
	ReferenceScope string       `json:"reference_scope"`
	Cases          []CaseReport `json:"cases"`
}

// CaseReport retains missing, unsupported and partial cases in the matrix.
// A measured result is not called a pass: no error threshold is implied.
type CaseReport struct {
	Case   Case          `json:"case"`
	Status string        `json:"status"`
	Reason string        `json:"reason,omitempty"`
	Frames []FrameReport `json:"frames,omitempty"`
}

// FrameReport is one matched scene time, before any display development.
type FrameReport struct {
	Index               int         `json:"index"`
	TimeNS              int64       `json:"time_ns"`
	Phase               string      `json:"phase"`
	ReferenceDurationNS int64       `json:"reference_duration_ns"`
	CandidateDurationNS int64       `json:"candidate_duration_ns"`
	ReferencePNG        string      `json:"reference_png"`
	CandidatePNG        string      `json:"candidate_png"`
	ReferenceDevice     string      `json:"reference_device,omitempty"`
	CandidateDevice     string      `json:"candidate_device,omitempty"`
	TimingComparable    bool        `json:"timing_comparable"`
	TimingNote          string      `json:"timing_note,omitempty"`
	Difference          *Difference `json:"difference"`
}

type savedOptions struct {
	Scene  string  `json:"scene_selection"`
	Mode   string  `json:"mode_selection"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Times  []int64 `json:"times_ns"`
}

type savedManifest struct {
	Version  int          `json:"version"`
	Options  savedOptions `json:"options"`
	Cases    []Case       `json:"cases"`
	Identity struct {
		ExecutableSHA256 string `json:"executable_sha256"`
	} `json:"identity"`
}

type savedFrame struct {
	Index         int    `json:"index"`
	TimeNS        int64  `json:"time_ns"`
	Phase         string `json:"phase"`
	Raw           string `json:"raw"`
	CoverageDepth string `json:"coverage_depth"`
	PNG           string `json:"png"`
	DurationNS    int64  `json:"duration_ns"`
	Profile       *struct {
		Device string `json:"Device"`
	} `json:"profile"`
}

type savedResult struct {
	ExecutableSHA256 string            `json:"executable_sha256"`
	Case             Case              `json:"case"`
	Status           string            `json:"status"`
	Reason           string            `json:"reason"`
	Bands            string            `json:"spectral_band_status"`
	Inputs           map[string]string `json:"input_sha256"`
	Camera           string            `json:"camera"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	Frames           []savedFrame      `json:"frames"`
}

// Summarize compares saved visible-light frames against the specified
// reference from the same run. CPU is the tracer reference; native meshes
// is the same-device direct-light baseline with production density baking.
// Approximate GPU lighting modes are refused as references. Results with
// unavailable references, mismatched metadata, dark reference light, or
// unreadable pixels never become measured successes.
func Summarize(path, reference string) (Report, error) {
	scope := ""
	switch reference {
	case "cpu":
		scope = "CPU visible-light tracer; above sixteen lamps both backends sample and may differ numerically"
	case "gpu-native-meshes":
		scope = "GPU direct lighting with production density baking; not an exact density reference"
	default:
		return Report{}, fmt.Errorf("reference must be cpu or gpu-native-meshes, not %q", reference)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return Report{}, err
	}
	defer root.Close()
	var manifest savedManifest
	if err := readJSON(root, "manifest.json", &manifest); err != nil {
		return Report{}, fmt.Errorf("manifest: %w", err)
	}
	if manifest.Version != 1 || !validSHA256(manifest.Identity.ExecutableSHA256) || manifest.Options.Width < 0 || manifest.Options.Height < 0 || (manifest.Options.Width == 0) != (manifest.Options.Height == 0) || len(manifest.Options.Times) == 0 {
		return Report{}, fmt.Errorf("invalid census manifest")
	}
	validModes := map[string]Mode{}
	for _, m := range Modes() {
		validModes[m.Name] = m
	}
	byKey := map[string]savedResult{}
	errorsByKey := map[string]error{}
	seen := map[string]bool{}
	for _, c := range manifest.Cases {
		canonical, known := validModes[c.Mode.Name]
		if c.Scene < 1 || c.Samples < 1 || !known || canonical != c.Mode || seen[c.Key()] {
			return Report{}, fmt.Errorf("invalid or duplicate case %q", c.Key())
		}
		seen[c.Key()] = true
		var result savedResult
		err := readJSON(root, resultPath(c.Scene, c.Mode.Name), &result)
		if err == nil && !reflect.DeepEqual(result.Case, c) {
			err = fmt.Errorf("case identity changed")
		}
		if err == nil && result.ExecutableSHA256 != manifest.Identity.ExecutableSHA256 {
			err = fmt.Errorf("case executable digest differs from manifest")
		}
		if err != nil {
			errorsByKey[c.Key()] = err
		} else {
			byKey[c.Key()] = result
		}
	}
	report := Report{Reference: reference, ReferenceScope: scope}
	for _, c := range manifest.Cases {
		entry := CaseReport{Case: c}
		result, exists := byKey[c.Key()]
		if !exists {
			err := errorsByKey[c.Key()]
			entry.Status = "invalid"
			if errors.Is(err, os.ErrNotExist) {
				entry.Status = "not-run"
				if selected(manifest.Options, c) {
					entry.Status = "missing"
				}
			}
			entry.Reason = err.Error()
			report.Cases = append(report.Cases, entry)
			continue
		}
		entry.Status, entry.Reason = result.Status, result.Reason
		if result.Status != "ok" && result.Status != "partial" {
			if result.Status != "failed" && result.Status != "unsupported" && result.Status != "running" {
				entry.Status, entry.Reason = "invalid", "unrecognized case status"
			}
			report.Cases = append(report.Cases, entry)
			continue
		}
		if err := validResult(result, manifest.Options); err != nil {
			entry.Status, entry.Reason = "invalid", err.Error()
			report.Cases = append(report.Cases, entry)
			continue
		}
		if c.Mode.Name == reference {
			for _, rec := range result.Frames {
				f, err := loadSaved(root, c.Scene, reference, rec)
				if err != nil || f.W != result.Width || f.H != result.Height {
					entry.Status, entry.Reason = "invalid", "reference artifact is missing, damaged or has different dimensions"
					break
				}
				if err := validPNG(root, c.Scene, reference, rec, f.W, f.H); err != nil {
					entry.Status, entry.Reason = "invalid", "reference preview is missing or has an invalid header or dimensions"
					break
				}
			}
			if entry.Status == "invalid" {
				report.Cases = append(report.Cases, entry)
				continue
			}
			entry.Status = "reference"
			if result.Status == "partial" {
				entry.Status = "partial"
			}
			report.Cases = append(report.Cases, entry)
			continue
		}
		refKey := fmt.Sprintf("%03d/%s", c.Scene, reference)
		base, exists := byKey[refKey]
		if !exists || (base.Status != "ok" && base.Status != "partial") || validResult(base, manifest.Options) != nil {
			entry.Status, entry.Reason = "missing-reference", "complete reference frames are unavailable"
			report.Cases = append(report.Cases, entry)
			continue
		}
		if !sameScene(base.Case, result.Case) || base.Camera != result.Camera || base.Width != result.Width || base.Height != result.Height || !reflect.DeepEqual(base.Inputs, result.Inputs) {
			entry.Status, entry.Reason = "inconclusive", "reference and candidate metadata differ"
			report.Cases = append(report.Cases, entry)
			continue
		}
		entry.Status = "measured"
		if result.Status == "partial" || base.Status == "partial" || (c.Bands > 0 && (c.Mode.Name != "cpu" || reference != "cpu")) {
			entry.Status = "partial"
		}
		for i, candidate := range result.Frames {
			referenceFrame := base.Frames[i]
			a, err := loadSaved(root, c.Scene, reference, referenceFrame)
			if err != nil {
				entry.Status, entry.Reason, entry.Frames = "missing-reference", err.Error(), nil
				break
			}
			b, err := loadSaved(root, c.Scene, c.Mode.Name, candidate)
			if err != nil {
				entry.Status, entry.Reason, entry.Frames = "invalid", err.Error(), nil
				break
			}
			if a.W != result.Width || a.H != result.Height || b.W != a.W || b.H != a.H {
				entry.Status, entry.Reason, entry.Frames = "invalid", "artifact dimensions differ from manifest", nil
				break
			}
			if err := validPNG(root, c.Scene, reference, referenceFrame, a.W, a.H); err != nil {
				entry.Status, entry.Reason, entry.Frames = "missing-reference", "reference preview is missing or has an invalid header or dimensions", nil
				break
			}
			if err := validPNG(root, c.Scene, c.Mode.Name, candidate, b.W, b.H); err != nil {
				entry.Status, entry.Reason, entry.Frames = "invalid", "candidate preview is missing or has an invalid header or dimensions", nil
				break
			}
			difference, err := Compare(a, b)
			if err != nil {
				entry.Status, entry.Reason, entry.Frames = "invalid", err.Error(), nil
				break
			}
			compared := FrameReport{Index: candidate.Index, TimeNS: candidate.TimeNS, Phase: candidate.Phase, ReferenceDurationNS: referenceFrame.DurationNS, CandidateDurationNS: candidate.DurationNS, ReferencePNG: previewPath(c.Scene, reference, referenceFrame), CandidatePNG: previewPath(c.Scene, c.Mode.Name, candidate), Difference: &difference}
			compared.ReferenceDevice, compared.CandidateDevice = device(referenceFrame), device(candidate)
			compared.TimingComparable = candidate.DurationNS > 0 && referenceFrame.DurationNS > 0
			if !compared.TimingComparable {
				compared.TimingNote = "a duration is unavailable"
			} else if (strings.HasPrefix(reference, "gpu-") && compared.ReferenceDevice == "") || (strings.HasPrefix(c.Mode.Name, "gpu-") && compared.CandidateDevice == "") || (strings.HasPrefix(reference, "gpu-") && strings.HasPrefix(c.Mode.Name, "gpu-") && compared.ReferenceDevice != compared.CandidateDevice) {
				compared.TimingComparable = false
				compared.TimingNote = "GPU devices are unknown or differ"
			}
			entry.Frames = append(entry.Frames, compared)
			if difference.Relative == nil {
				entry.Status, entry.Reason = "inconclusive", "reference contains no light"
			}
		}
		report.Cases = append(report.Cases, entry)
	}
	return report, nil
}

// validSHA256 reports whether s is a 64-character hexadecimal digest.
func validSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// selected reports whether a case matches the scene and mode options.
func selected(o savedOptions, c Case) bool {
	return (o.Scene == "all" || o.Scene == strconv.Itoa(c.Scene)) && (o.Mode == "all" || o.Mode == c.Mode.Name)
}

// Only mode differs between two renderings of a recorded scene. Comparing
// every other field also catches newly added scene metadata by default.
func sameScene(a, b Case) bool {
	a.Mode, b.Mode = Mode{}, Mode{}
	return reflect.DeepEqual(a, b)
}

// resultPath is the path of a case's result file, relative to the root.
func resultPath(scene int, mode string) string {
	return filepath.Join(fmt.Sprintf("scene-%03d", scene), mode, "result.json")
}

// readJSON decodes the JSON file name under root, up to the size limit.
func readJSON(root *os.Root, name string, out any) error {
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 8*1024*1024+1))
	if err != nil {
		return err
	}
	if len(data) > 8*1024*1024 {
		return fmt.Errorf("JSON exceeds size limit")
	}
	return json.Unmarshal(data, out)
}

// validResult checks a saved result against the options it was made with.
func validResult(result savedResult, o savedOptions) error {
	if result.Width < 1 || result.Width > 8192 || result.Height < 1 || result.Height > 8192 || (o.Width != 0 && (result.Width != o.Width || result.Height != o.Height)) {
		return fmt.Errorf("frame size is missing or differs from the requested size")
	}
	if result.Camera == "" {
		return fmt.Errorf("camera identity is missing")
	}
	if len(result.Case.Inputs) > 0 && len(result.Inputs) == 0 {
		return fmt.Errorf("declared input hashes are missing")
	}
	for _, hash := range result.Inputs {
		if hash == "" {
			return fmt.Errorf("an input hash is empty")
		}
	}
	if len(result.Frames) != len(o.Times)+1 {
		return fmt.Errorf("incomplete frame list")
	}
	for i, f := range result.Frames {
		at, phase := o.Times[0], "cold"
		if i == 1 {
			phase = "same-time-repeat"
		} else if i > 1 {
			at, phase = o.Times[i-1], "scene-time"
		}
		if f.Index != i || f.TimeNS != at || f.Phase != phase || f.Raw != fmt.Sprintf("frame-%03d.pfm", i) || f.CoverageDepth != fmt.Sprintf("frame-%03d.cd32", i) || f.PNG != fmt.Sprintf("frame-%03d.png", i) || f.DurationNS < 0 {
			return fmt.Errorf("frame %d identity, time or artifact names do not match", i)
		}
	}
	return nil
}

// previewPath is the path of a frame's preview PNG, relative to the root.
func previewPath(scene int, mode string, rec savedFrame) string {
	return filepath.Join(fmt.Sprintf("scene-%03d", scene), mode, rec.PNG)
}

// device returns the device a frame was profiled on, or nothing when it
// has no profile.
func device(rec savedFrame) string {
	if rec.Profile == nil {
		return ""
	}
	return rec.Profile.Device
}

// validPNG checks that a frame's preview PNG has the given size.
func validPNG(root *os.Root, scene int, mode string, rec savedFrame, width, height int) error {
	f, err := root.Open(previewPath(scene, mode, rec))
	if err != nil {
		return err
	}
	defer f.Close()
	config, err := png.DecodeConfig(f)
	if err != nil || config.Width != width || config.Height != height {
		return fmt.Errorf("invalid preview PNG")
	}
	return nil
}

// loadSaved reads one saved frame from root, with its raw light file.
func loadSaved(root *os.Root, scene int, mode string, rec savedFrame) (*frame.Frame, error) {
	dir := filepath.Join(fmt.Sprintf("scene-%03d", scene), mode)
	light, err := root.Open(filepath.Join(dir, rec.Raw))
	if err != nil {
		return nil, err
	}
	defer light.Close()
	detail, err := root.Open(filepath.Join(dir, rec.CoverageDepth))
	if err != nil {
		return nil, err
	}
	defer detail.Close()
	return readFrame(light, detail)
}
