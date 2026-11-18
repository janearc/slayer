// Command alfa-surface-clip prepares the recorded native ten-second Alfa recipe.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// main runs the tool and exits with its status.
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run registers the prepare, preview and render verbs and runs them on args.
func run(args []string, out, errout io.Writer) int {
	tool := pzm.New("alfa-surface-clip", "draw the recorded moonlit Alfa trajectory", pzm.Build(""), pzm.Built(""), pzm.Who(""), pzm.Sessions(""))
	for _, mode := range []string{"prepare", "preview", "render"} {
		load := verb.FullGPU
		if mode == "prepare" {
			load = verb.Light
		}
		tool.Verb(verb.Name(mode), verb.NoShort, "prepare or draw the native Alfa clip", "Takes OUTPUT TEMPLATE. Uses OUTPUT/inputs/slayer. Existing inputs must match exactly; existing pictures require the matching input pin. Render writes120nativeframes at12fps and encodes10seconds.", verb.Example("Example_"+mode), verb.NewFacts(verb.WritesTo("the named output directory"), verb.Changes, verb.ThisMachine, verb.NoOne, load), func(call pzm.Call) error { return invoke(call, mode) }, pzm.Value("camera-east", verb.NoShort, "camera easting in metres; default -1650, range +/-1000000"), pzm.Value("aim-east", verb.NoShort, "target easting in metres; default -1535, range +/-1000000"))
	}
	return tool.Run(args, out, errout)
}

type config struct {
	output, template, mode string
	camera, aim            float64
	dry                    bool
}

// invoke reads the output and template arguments and the camera and aim
// coordinates, then executes the plan for mode.
func invoke(call pzm.Call, mode string) error {
	if len(call.Args()) != 2 {
		return fmt.Errorf("%s needs exactly OUTPUT TEMPLATE", mode)
	}
	camera, err := coordinate(call, "camera-east", -1650)
	if err != nil {
		return err
	}
	aim, err := coordinate(call, "aim-east", -1535)
	if err != nil {
		return err
	}
	return execute(config{output: call.Args()[0], template: call.Args()[1], mode: mode, camera: camera, aim: aim, dry: call.DryRun()}, call.Out())
}

// coordinate reads a finite metre coordinate flag, or returns fallback when the
// flag is absent.
func coordinate(call pzm.Call, name string, fallback float64) (float64, error) {
	if !call.Given(name) {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(call.Value(name), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 1e6 {
		return 0, fmt.Errorf("--%s needs a finite metre coordinate in [-1000000,1000000]", name)
	}
	return value, nil
}

type frame struct {
	index           int
	seconds         float64
	name, directory string
	set, job        []byte
}

// plan expands the template into one set and one job for each frame of the mode.
func plan(c config, base []byte) ([]frame, error) {
	for _, key := range []string{"{{HULL_EAST}}", "{{CAMERA_EAST}}", "{{AIM_EAST}}", "{{SECONDS}}"} {
		if !bytes.Contains(base, []byte(key)) {
			return nil, fmt.Errorf("template is missing %s", key)
		}
	}
	indices := []int{0, 60, 119}
	if c.mode != "preview" {
		indices = make([]int, 120)
		for i := range indices {
			indices[i] = i
		}
	}
	frames := make([]frame, 0, len(indices))
	for _, i := range indices {
		seconds := float64(i) / 12
		name := fmt.Sprintf("%04d", i)
		width, height, dir, preset := 1080, 960, "frames", "preset: half-portrait, "
		if c.mode == "preview" {
			width, height, dir, preset = 360, 320, "previews", ""
			name = "preview-" + name
		}
		scene := strings.NewReplacer("{{HULL_EAST}}", fmt.Sprintf("%.9f", -1500+15*seconds), "{{CAMERA_EAST}}", fmt.Sprintf("%.9f", c.camera+15*seconds), "{{AIM_EAST}}", fmt.Sprintf("%.9f", c.aim+15*seconds), "{{SECONDS}}", fmt.Sprintf("%.9f", seconds)).Replace(string(base))
		if strings.Contains(scene, "{{") {
			return nil, fmt.Errorf("template has an unknown placeholder")
		}
		job := fmt.Sprintf("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: ../sets/%s.yaml}\nrenderer: gpu\nquality: exact\nfilm: {%swidth: %d, height: %d, rays: 1}\nprint: {knee: 0.7, top: 8, stops: 20}\ncamera: {name: deck}\nat: {value: %.9f, unit: s}\noutputs:\n  picture: ../%s/%s.png\n", name, preset, width, height, seconds, dir, name)
		frames = append(frames, frame{i, seconds, name, dir, []byte(scene), []byte(job)})
	}
	return frames, nil
}

type pinned struct {
	Template, Renderer string
	Camera, Aim        float64
}

// execute writes the inputs and draws each missing frame. A render also encodes
// the movie.
func execute(c config, output io.Writer) error {
	base, err := os.ReadFile(c.template)
	if err != nil {
		return err
	}
	frames, err := plan(c, base)
	if err != nil {
		return err
	}
	renderer, err := os.ReadFile(filepath.Join(c.output, "inputs", "slayer"))
	if err != nil {
		return err
	}
	pin, err := json.Marshal(pinned{fmt.Sprintf("%x", sha256.Sum256(base)), fmt.Sprintf("%x", sha256.Sum256(renderer)), c.camera, c.aim})
	if err != nil {
		return err
	}
	family := "film"
	if c.mode == "preview" {
		family = "preview"
	}
	pinPath := filepath.Join(c.output, "inputs", family+"-pin.json")
	if err = preflight(c.output, frames, pinPath, pin); err != nil {
		return err
	}
	if c.dry {
		_, err = fmt.Fprintf(output, "would %s %d native frames in %s\n", c.mode, len(frames), c.output)
		return err
	}
	for _, dir := range []string{"sets", "jobs", "frames", "previews", "logs", "inputs"} {
		if err = os.MkdirAll(filepath.Join(c.output, dir), 0755); err != nil {
			return err
		}
	}
	if err = writeNewOrSame(pinPath, pin); err != nil {
		return err
	}
	for _, one := range frames {
		if err = writeNewOrSame(filepath.Join(c.output, "sets", one.name+".yaml"), one.set); err != nil {
			return err
		}
		if err = writeNewOrSame(filepath.Join(c.output, "jobs", one.name+".yaml"), one.job); err != nil {
			return err
		}
		if c.mode == "prepare" {
			continue
		}
		if _, err = os.Stat(filepath.Join(c.output, one.directory, one.name+".png")); err == nil {
			continue
		}
		if err = draw(c, one, output); err != nil {
			return err
		}
	}
	if c.mode != "render" {
		return nil
	}
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "warning", "-y", "-framerate", "12", "-i", filepath.Join(c.output, "frames", "%04d.png"), "-frames:v", "120", "-c:v", "libx264", "-crf", "18", "-pix_fmt", "yuv420p", "-movflags", "+faststart", filepath.Join(c.output, "alfa-moonlit-surface-10s.mp4"))
	cmd.Stdout, cmd.Stderr = output, output
	return cmd.Run()
}

// preflight refuses an output directory whose inputs or pictures do not match
// the pin.
func preflight(out string, frames []frame, pinPath string, pin []byte) error {
	existing, err := os.ReadFile(pinPath)
	hasPin := err == nil
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if hasPin && !bytes.Equal(existing, pin) {
		return fmt.Errorf("existing input pin differs; use a fresh output directory")
	}
	for _, one := range frames {
		for path, want := range map[string][]byte{filepath.Join(out, "sets", one.name+".yaml"): one.set, filepath.Join(out, "jobs", one.name+".yaml"): one.job} {
			if err = checkSame(path, want); err != nil {
				return err
			}
		}
		if _, err = os.Stat(filepath.Join(out, one.directory, one.name+".png")); err == nil && !hasPin {
			return fmt.Errorf("existing picture has no matching input pin; use a fresh output directory")
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// checkSame reports an error when the file at path holds different contents
// from want.
func checkSame(path string, want []byte) error {
	old, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(old, want) {
		return fmt.Errorf("existing input %s differs; use a fresh output directory", path)
	}
	return nil
}

// writeNewOrSame creates the file, or checks that it matches data when it
// already exists.
func writeNewOrSame(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return checkSame(path, data)
	}
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// draw runs the renderer on one job, logs its output and appends a receipt for
// the frame.
func draw(c config, one frame, output io.Writer) error {
	log, err := os.Create(filepath.Join(c.output, "logs", one.name+".log"))
	if err != nil {
		return err
	}
	start := time.Now()
	cmd := exec.Command(filepath.Join(c.output, "inputs", "slayer"), "draw", filepath.Join(c.output, "jobs", one.name+".yaml"))
	cmd.Stdout, cmd.Stderr = log, log
	drawErr := cmd.Run()
	closeErr := log.Close()
	receipt := fmt.Sprintf("frame=%d t=%.9f hull_east=%.9f camera_east=%.9f wake_east=%.9f elapsed_seconds=%.4f error=%v set_sha256=%x\n", one.index, one.seconds, -1500+15*one.seconds, c.camera+15*one.seconds, -1545.902645+15*one.seconds, time.Since(start).Seconds(), drawErr, sha256.Sum256(one.set))
	f, err := os.OpenFile(filepath.Join(c.output, "frame-receipts.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, writeErr := f.WriteString(receipt)
	receiptCloseErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if receiptCloseErr != nil {
		return receiptCloseErr
	}
	if _, err = fmt.Fprint(output, receipt); err != nil {
		return err
	}
	if drawErr != nil {
		return drawErr
	}
	return closeErr
}
