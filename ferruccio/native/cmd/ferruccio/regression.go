package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"gopkg.in/yaml.v3"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type regressionCase struct {
	Name, Source, Gesture  string
	Full, Bounded, Starter bool
}
type regressionResult struct {
	Name, Source, SourceSHA256, OriginalSHA256 string
	Mode, SourceManifestSHA256                 string
	Arguments                                  []string
	Status, Boundary, Diagnostic               string
	Elapsed                                    time.Duration
}
type regressionReport struct {
	Started, Finished                time.Time
	Executable, Build, PriorWebbFix  string
	ExecutableSHA256, ProducerSHA256 string
	Cases                            []regressionResult
}

// registerRegression declares the serial native-window runner through Puzzlemouth.
func registerRegression(tool *pzm.Tool) {
	facts := verb.NewFacts(verb.WritesTo("one new regression evidence directory"), verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("regress", verb.NoShort, "exercise scenes through native Ferruccio", "Run serial proof child windows from this executable; preserve sources, logs, captures and failure boundaries. Run unwrapped; leave other Ferruccio windows alone.", "Example_proof", facts, runRegression,
		append([]pzm.Flag{pzm.Switch("inventory", verb.NoShort, "probe uncovered sets in wire and bounded full modes"), pzm.Value("out", verb.NoShort, "existing evidence parent directory"), pzm.Value("sets", verb.NoShort, "existing Slayer sets directory"), pzm.Value("venus", verb.NoShort, "explicit Venus YAML source"), pzm.Value("kilauea", verb.NoShort, "explicit Kilauea YAML source"), pzm.Value("full-case", verb.NoShort, "small authored full-render YAML")}, runtimeFlags()...)...)
}

// runRegression runs the regression cases and writes the summary under the
// --out directory. It takes no positional arguments.
func runRegression(call pzm.Call) error {
	if len(call.Args()) != 0 || call.Value("out") == "" || call.Value("examples") == "" {
		return fmt.Errorf("regress needs --out EXISTING_PARENT --examples DIR and no positional arguments")
	}
	for _, path := range []string{call.Value("out"), call.Value("examples")} {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
	}
	examples, err := filepath.Abs(call.Value("examples"))
	if err != nil {
		return err
	}
	sets := call.Value("sets")
	if sets == "" {
		sets = filepath.Join(examples, "..", "..", "sets")
	}
	choose := func(flag, fallback string) string {
		value := call.Value(flag)

		if value != "" {
			return value
		}
		return fallback
	}
	cases := []regressionCase{
		{Name: "larson-wire", Source: filepath.Join(examples, "ferruccio-playground.yaml")},
		{Name: "larson-trackball", Source: filepath.Join(examples, "ferruccio-playground.yaml"), Gesture: "trackball", Starter: true},
		{Name: "webb-wire", Source: filepath.Join(examples, "webb-inspection.yaml")},
		{Name: "webb-trackball", Source: filepath.Join(examples, "webb-inspection.yaml"), Gesture: "trackball"},
		{Name: "service-court-wire", Source: filepath.Join(examples, "spacecraft-service-court.yaml")},
		{Name: "service-court-trackball", Source: filepath.Join(examples, "spacecraft-service-court.yaml"), Gesture: "trackball"},
		{Name: "larson-full", Source: choose("full-case", filepath.Join(examples, "full-render-check.yaml")), Full: true},
		{Name: "venus-full-probe", Source: choose("venus", filepath.Join(sets, "venera-13-plain.yaml")), Full: true, Bounded: true},
		{Name: "kilauea-full-probe", Source: choose("kilauea", filepath.Join(sets, "kilauea-iki.yaml")), Full: true, Bounded: true},
	}
	if call.On("inventory") {
		cases, err = appendRegressionInventory(cases, sets)
		if err != nil {
			return err
		}
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "regress: %d serial native proof children; no windows or renders started\n", len(cases))
		return err
	}
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	parent, err := filepath.Abs(call.Value("out"))
	if err != nil {
		return err
	}
	root, err := os.MkdirTemp(parent, "ferruccio-regression-")
	if err != nil {
		return err
	}
	pinnedSlayer := call.Value("slayer")
	if pinnedSlayer != "" {
		pinnedSlayer, err = filepath.EvalSymlinks(pinnedSlayer)
		if err != nil {
			return err
		}
		pinnedSlayer, err = filepath.Abs(pinnedSlayer)
		if err != nil {
			return err
		}
	}
	report := regressionReport{Started: time.Now().UTC(), Executable: binary, Build: build, PriorWebbFix: "Webb remains an explicit regression after its earlier corrected viewport defect; historical success is not a current pass."}
	report.ExecutableSHA256 = regressionFileHash(binary)
	report.ProducerSHA256 = regressionFileHash(pinnedSlayer)
	fmt.Fprintln(call.Out(), root)
	failed := 0
	blocked := false
	for _, c := range cases {
		result := regressionResult{Name: c.Name, Source: c.Source, Status: "blocked", Boundary: "unresolved", Diagnostic: "earlier proof timed out; renderer quiescence unconfirmed, no new child started"}
		if !blocked {
			result = runRegressionCase(call, binary, root, pinnedSlayer, c)
		}
		if result.Status == "cleanup-unconfirmed" {
			blocked = true
		}
		if result.Status != "passed" {
			failed++
		}
		report.Cases = append(report.Cases, result)
		if err := writeRegressionSummary(root, report); err != nil {
			return err
		}
		fmt.Fprintf(call.Out(), "%s: %s (%s)\n", result.Name, result.Status, result.Boundary)
	}
	report.Finished = time.Now().UTC()
	if err := writeRegressionSummary(root, report); err != nil {
		return err
	}
	if failed != 0 {
		return fmt.Errorf("%d cases failed or unavailable; evidence: %s", failed, root)
	}
	return nil
}

// runRegressionCase runs one case and records its outcome and timing in the
// result.
func runRegressionCase(call pzm.Call, binary, root, pinnedSlayer string, c regressionCase) (result regressionResult) {
	started := time.Now()
	result = regressionResult{Name: c.Name, Source: c.Source, Status: "failed", Boundary: "unresolved", Mode: "wire"}
	if c.Full {
		result.Mode = "authored full"
	}
	if c.Bounded {
		result.Mode = "bounded full160x100 rays1"
	}
	dir := filepath.Join(root, c.Name)
	if err := os.Mkdir(dir, 0755); err != nil {
		result.Diagnostic = err.Error()
		return
	}
	defer func() {
		result.Elapsed = time.Since(started)
		raw, _ := json.MarshalIndent(result, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, "result.json"), raw, 0644)
	}()
	raw, err := os.ReadFile(c.Source)
	if err != nil {
		result.Status = "unavailable"
		result.Boundary = "input"
		result.Diagnostic = err.Error()
		return
	}
	if err = preview.CopySourceManifest(c.Source, dir); err != nil {
		result.Boundary = "Ferruccio input context"
		result.Diagnostic = err.Error()
		return
	}
	result.SourceManifestSHA256 = regressionFileHash(filepath.Join(dir, "sources.yaml"))
	hash := sha256.Sum256(raw)
	result.OriginalSHA256 = hex.EncodeToString(hash[:])
	if err = os.WriteFile(filepath.Join(dir, "original.yaml"), raw, 0644); err != nil {
		result.Diagnostic = err.Error()
		return
	}
	if c.Bounded {
		raw, err = boundedRegressionFilm(c.Source, raw)
		if err != nil {
			result.Boundary = "input"
			if errors.Is(err, errBoundedPhysicalFilm) {
				result.Status = "unavailable"
				result.Boundary = "Ferruccio regression harness"
			}
			result.Diagnostic = err.Error()
			return
		}
	}
	hash = sha256.Sum256(raw)
	result.SourceSHA256 = hex.EncodeToString(hash[:])
	source := filepath.Join(dir, "source.yaml")
	if err = os.WriteFile(source, raw, 0644); err != nil {
		result.Diagnostic = err.Error()
		return
	}
	out := filepath.Join(dir, "view.png")
	args := []string{"proof", "--background", "--file", source, "--out", out, "--composition", "play"}
	if c.Full {
		args = append(args, "--full", "--display", "rendered")
	} else {
		args = append(args, "--display", "wire")
	}
	if c.Gesture != "" {
		args = append(args, "--gesture", c.Gesture)
		if c.Starter && call.Value("starter") != "" {
			args = append(args, "--starter", call.Value("starter"))
		} else {
			args = append(args, "--render")
		}
	}
	for _, flag := range []string{"preview-transport", "azazel", "azazel-queue", "agent", "evidence", "slayer", "catalogue", "game", "game-home", "shelf", "cache", "examples", "theme"} {
		value := call.Value(flag)
		if flag == "slayer" {
			value = pinnedSlayer
		}
		if value != "" {
			args = append(args, "--"+flag, value)
		}
	}
	result.Arguments = args
	log, err := os.OpenFile(filepath.Join(dir, "child.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		result.Diagnostic = err.Error()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute+30*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, binary, args...)
	child.Env = regressionChildEnvironment()
	child.Stdout = log
	child.Stderr = log
	// This process group belongs only to the new proof child.
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	child.Cancel = func() error {
		if child.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-child.Process.Pid, syscall.SIGTERM)
	}
	child.WaitDelay = 5 * time.Second
	err = child.Run()
	if ctx.Err() != nil && child.Process != nil {
		_ = syscall.Kill(-child.Process.Pid, syscall.SIGKILL)
	}
	_ = log.Close()
	if err != nil {
		data, _ := os.ReadFile(filepath.Join(dir, "child.log"))
		result.Diagnostic = fmt.Sprintf("%v\n%s", err, data)
		result.Boundary = regressionBoundary(string(data))
		if ctx.Err() != nil {
			result.Boundary = "unresolved"
			result.Status = "cleanup-unconfirmed"
			result.Diagnostic = "parent timeout; bounded child cleanup did not complete: " + result.Diagnostic
		}
		return
	}
	if err = validateRegressionCapture(out, c); err != nil {
		result.Boundary = "Ferruccio"
		result.Diagnostic = err.Error()
		return
	}
	result.Status = "passed"
	result.Boundary = "none"
	return
}

// The regression child is another Ferruccio; it must not duplicate the
// parent's summaries. Slayer's explicit summary settings remain available.
func regressionChildEnvironment() []string {
	var env []string
	for _, entry := range preview.ChildEnvironment() {
		if !strings.HasPrefix(entry, "FERRUCCIO_STATS_") {
			env = append(env, entry)
		}
	}
	return env
}

var errBoundedPhysicalFilm = errors.New("bounded full probe unavailable for physical film rig; authored optics preserved")

// boundedRegressionFilm changes parsed raster scalars only; physical rigs are refused.
func boundedRegressionFilm(name string, raw []byte) ([]byte, error) {
	var node yaml.Node
	if err := yamlfile.Read(name, raw, &node); err != nil {
		return nil, err
	}
	var refusal error
	var visit func(*yaml.Node) bool
	visit = func(n *yaml.Node) bool {
		if n.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(n.Content); i += 2 {
				if n.Content[i].Value == "film" && n.Content[i+1].Kind == yaml.MappingNode {
					film := n.Content[i+1]
					for j := 0; j+1 < len(film.Content); j += 2 {
						switch film.Content[j].Value {
						case "frame", "lens", "shutter", "glimpses":
							refusal = errBoundedPhysicalFilm
							return true
						}
					}
					for _, field := range []struct{ k, v string }{{"width", "160"}, {"height", "100"}, {"rays", "1"}} {
						found := false
						for j := 0; j+1 < len(film.Content); j += 2 {
							if film.Content[j].Value == field.k {
								film.Content[j+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: field.v}
								found = true
							}
						}
						if !found {
							film.Content = append(film.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: field.k}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: field.v})
						}
					}
					return true
				}
			}
		}
		for _, child := range n.Content {
			if visit(child) {
				return true
			}
		}
		return false
	}
	if !visit(&node) {
		return nil, fmt.Errorf("source has no authored film to bound; refusing unbounded full probe")
	}
	if refusal != nil {
		return nil, refusal
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// validateRegressionCapture checks that the capture file at path exists and
// matches the case c.
func validateRegressionCapture(path string, c regressionCase) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	_, err = png.DecodeConfig(file)
	_ = file.Close()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		return err
	}
	var receipt map[string]any
	if err = json.Unmarshal(raw, &receipt); err != nil {
		return err
	}
	if receipt["native_window_created"] != true && receipt["native_window_opened"] != true {
		return fmt.Errorf("receipt does not confirm native window")
	}
	previewPath, _ := receipt["preview_file"].(string)
	if c.Full && (receipt["full_render"] != true || previewPath == "") {
		return fmt.Errorf("full proof has no accepted full render")
	}
	if c.Full {
		if err := validateFullReceipt(previewPath, filepath.Join(filepath.Dir(path), "source.yaml")); err != nil {
			return err
		}
	}

	if !c.Full {
		key := "working"
		if c.Gesture != "" {
			key = "working_held"
		}
		stats, ok := receipt[key].(map[string]any)
		if !ok {
			return fmt.Errorf("working receipt has no GPU statistics")
		}
		draws, _ := stats["Draws"].(float64)
		triangles, _ := stats["Triangles"].(float64)
		if draws < 1 || triangles < 1 {
			return fmt.Errorf("working receipt has no completed geometry draw")
		}
	}
	if c.Gesture != "" {
		if receipt["held_source_unchanged"] != true || receipt["source_text_before"] != receipt["source_text_held"] || receipt["source_text_before"] == receipt["source_text_after_release"] {
			return fmt.Errorf("held gesture source invariants failed")
		}
		before, ok := receipt["before_capture"].(string)
		if !ok {
			return fmt.Errorf("held gesture has no before capture")
		}
		if _, err := os.Stat(before); err != nil {
			return err
		}
	}
	return nil
}

// regressionBoundary names the boundary that a failed case log points at.
// It returns "unresolved" when the log names none.
func regressionBoundary(log string) string {
	if strings.Contains(log, "no sources.yaml") {
		return "Ferruccio input context"
	}
	if strings.Contains(log, "proof interrupted") || strings.Contains(log, "Preview cancellation confirmed") {
		return "input interruption; renderer verdict unresolved"
	}
	for _, s := range []string{"working display needs supported", "working display does not support", "working display needs a fixed camera", "has no finite working display mesh", "working display contains no surfaces"} {
		if strings.Contains(log, s) {
			return "Ferruccio live-geometry adapter unsupported"
		}
	}
	for _, s := range []string{"working view shader:", "working triangles:", "working edges:", "working depth:", "working GPU readback:", "wgpu validation", "Metal shader compilation"} {
		if strings.Contains(log, s) {
			return "GoGPU/Metal boundary (root cause unproven)"
		}
	}
	if strings.Contains(log, "shared Slayer preview failed:") {
		return "Slayer renderer/input boundary"
	}
	if strings.Contains(log, "held gesture") || strings.Contains(log, "working viewport:") || strings.Contains(log, "widget export") {
		return "Ferruccio"
	}
	return "unresolved"
}

// writeRegressionSummary writes the report as indented JSON, then moves it
// to summary.json under root.
func writeRegressionSummary(root string, report regressionReport) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(root, ".summary-")
	if err != nil {
		return err
	}
	path := temp.Name()
	defer os.Remove(path)
	if _, err = temp.Write(raw); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(path, filepath.Join(root, "summary.json"))
}

// regressionFileHash returns the SHA256 of the file at path, or the empty
// string when it cannot be read.
func regressionFileHash(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

// validateFullReceipt follows the image to its measured producer record.
func validateFullReceipt(picture, source string) error {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(picture), "receipt.json"))
	if err != nil {
		return err
	}
	var receipt map[string]any
	if err = json.Unmarshal(raw, &receipt); err != nil {
		return err
	}
	if receipt["preview"] != false || receipt["quality"] != "exact" || receipt["source_sha256"] != regressionFileHash(source) {
		return fmt.Errorf("full image receipt does not match exact source request")
	}
	rays, ok := receipt["rays_per_axis"].(float64)
	if !ok || rays < 1 {
		return fmt.Errorf("full image has no recorded sampling")
	}
	file, err := os.Open(picture)
	if err != nil {
		return err
	}
	config, err := png.DecodeConfig(file)
	_ = file.Close()
	if err != nil {
		return err
	}
	if receipt["width"] != float64(config.Width) || receipt["height"] != float64(config.Height) {
		return fmt.Errorf("full image differs from measured producer dimensions")
	}
	if args, ok := receipt["producer_arguments"].([]any); ok {
		for _, arg := range args {
			if arg == "--width" || arg == "--height" || arg == "--rays" {
				return fmt.Errorf("full producer received a preview sampling override")
			}
		}
	}
	return nil
}

// appendRegressionInventory adds each uncovered direct sets YAML source once.
// filepath.Glob sorts inputs, keeping evidence names and execution order stable.
func appendRegressionInventory(cases []regressionCase, sets string) ([]regressionCase, error) {
	info, err := os.Stat(sets)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("inventory sets path is not a directory: %s", sets)
	}
	covered := make(map[string]bool)
	for _, c := range cases {
		source, err := filepath.Abs(c.Source)
		if err != nil {
			return nil, err
		}
		covered[source] = true
	}
	sources, err := filepath.Glob(filepath.Join(sets, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(sources) > 20 {
		return nil, fmt.Errorf("inventory has %d sources; bounded maximum is 20, refusing partial coverage", len(sources))
	}
	for _, source := range sources {
		source, err = filepath.Abs(source)
		if err != nil {
			return nil, err
		}
		if covered[source] {
			continue
		}
		info, err := os.Stat(source)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		covered[source] = true
		name := "inventory-" + strings.TrimSuffix(filepath.Base(source), ".yaml")
		cases = append(cases,
			regressionCase{Name: name + "-wire", Source: source},
			regressionCase{Name: name + "-full", Source: source, Full: true, Bounded: true},
		)
	}
	return cases, nil
}
