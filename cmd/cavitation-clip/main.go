// Command cavitation-clip authors checked renderer jobs for a rigid moving
// subject and its attached analytic density field. It renders no images.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/set"
	yaml "gopkg.in/yaml.v3"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

//go:embed plan.schema.json
var planSchema []byte

// writeSameOrNew writes raw to a new file at path. An existing file with
// the same bytes is left alone; one with other bytes is an error.
func writeSameOrNew(path string, raw []byte) error {
	if old, err := os.ReadFile(path); err == nil {
		if bytes.Equal(old, raw) {
			return nil
		}
		return fmt.Errorf("refusing to replace different existing output %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

type plan struct {
	Template       string       `yaml:"template"`
	Output         string       `yaml:"output"`
	Subject        string       `yaml:"subject"`
	Field          string       `yaml:"field"`
	Camera         string       `yaml:"camera"`
	Renderer       string       `yaml:"renderer"`
	AttachedFizzes []string     `yaml:"attached-fizzes"`
	Width          int          `yaml:"width"`
	Height         int          `yaml:"height"`
	FPS            int          `yaml:"fps"`
	Seconds        int          `yaml:"seconds"`
	Velocity       [3]float64   `yaml:"velocity-metres-per-second"`
	Track          float64      `yaml:"camera-tracking-fraction"`
	StartSeconds   float64      `yaml:"start-seconds"`
	Orbit          *cameraOrbit `yaml:"camera-orbit"`
	Rays           *int         `yaml:"rays"`
	VerticalView   *float64     `yaml:"vertical-view-deg"`
}

var build, built, who, sessions string

// main registers the plan verb and runs the tool with the process arguments.
func main() {
	tool := pzm.New("cavitation-clip", "author real moving-subject and propeller-field render jobs", pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	tool.Verb("plan", verb.NoShort, "write deterministic scene snapshots and renderer jobs", "A single SI linear trajectory moves both subject and density field. Native Slayer draw jobs evaluate each real scene time; no pixels are synthesized.", "Example_plan", verb.NewFacts(verb.WritesTo("the configured new jobs/scenes directories"), verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light), func(call pzm.Call) error {
		if len(call.Args()) != 0 || !call.Given("recipe") {
			return fmt.Errorf("plan requires --recipe PATH and no positional arguments")
		}
		return generate(call.Value("recipe"), call.Out())
	}, pzm.Value("recipe", verb.NoShort, "strict YAML clip plan; all paths resolve relative to this file"))
	os.Exit(tool.RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// nodeAt follows keys through nested YAML mappings from node and returns
// the node found at the end of the path.
func nodeAt(node *yaml.Node, keys ...string) (*yaml.Node, error) {
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) != 1 {
			return nil, fmt.Errorf("expected one document")
		}
		node = node.Content[0]
	}
	for _, key := range keys {
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("expected mapping before %q", key)
		}
		var next *yaml.Node
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				next = node.Content[i+1]
				break
			}
		}
		if next == nil {
			return nil, fmt.Errorf("template missing %q", key)
		}
		node = next
	}
	return node, nil
}

// shiftScalar adds delta to a numeric YAML scalar in place and rewrites
// it as a float.
func shiftScalar(node *yaml.Node, delta float64) error {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!int" && node.Tag != "!!float" {
		return fmt.Errorf("coordinate must be a numeric scalar")
	}
	value, err := strconv.ParseFloat(node.Value, 64)
	if err != nil {
		return err
	}
	if math.IsNaN(value+delta) || math.IsInf(value+delta, 0) {
		return fmt.Errorf("coordinate must stay finite")
	}
	node.Tag = "!!float"
	node.Value = strconv.FormatFloat(value+delta, 'g', 17, 64)
	return nil
}

// shiftTriple adds delta to each of the three numbers in a YAML sequence
// in place.
func shiftTriple(node *yaml.Node, delta [3]float64) error {
	if node.Kind != yaml.SequenceNode || len(node.Content) != 3 {
		return fmt.Errorf("expected map-coordinate triple")
	}
	for i := range 3 {
		if err := shiftScalar(node.Content[i], delta[i]); err != nil {
			return err
		}
	}
	return nil
}

// shiftMetreQuantity shifts a position value that must be given in metres.
// It fails if the unit under key is anything other than m.
func shiftMetreQuantity(position *yaml.Node, key string, delta float64) error {
	units, err := nodeAt(position, key, "unit")
	if err != nil || units.Kind != yaml.ScalarNode || units.Value != "m" {
		return fmt.Errorf("subject coordinates must use explicit metre units")
	}
	value, err := nodeAt(position, key, "value")
	if err != nil {
		return err
	}
	return shiftScalar(value, delta)
}

// requireUnitWorld fails unless the clip template sets world.unit to 1,
// so that coordinates are map-space metres.
func requireUnitWorld(document *yaml.Node) error {
	unit, err := nodeAt(document, "scene", "world", "unit")
	if err != nil || unit.Kind != yaml.ScalarNode ||
		(unit.Tag != "!!int" && unit.Tag != "!!float") {
		return fmt.Errorf("clip template requires numeric world.unit: 1")
	}
	value, err := strconv.ParseFloat(unit.Value, 64)
	if err != nil || value != 1 {
		return fmt.Errorf("clip template requires world.unit: 1 for map-space metres")
	}
	return nil
}

// attachedFeet returns the foot node of each named attached fizz in the
// template.
func attachedFeet(document *yaml.Node, names []string) ([]*yaml.Node, error) {
	var feet []*yaml.Node
	for _, name := range names {
		foot, err := nodeAt(document, "scene", "world", "fizzes", name, "foot")
		if err != nil {
			return nil, fmt.Errorf("attached fizz %q: %w", name, err)
		}
		if err = shiftTriple(foot, [3]float64{}); err != nil {
			return nil, fmt.Errorf("attached fizz %q: %w", name, err)
		}
		feet = append(feet, foot)
	}
	return feet, nil
}

// generate reads the clip plan at path and writes the job set for it to out.
func generate(path string, out io.Writer) error {
	planRaw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	schema, err := yamlfile.NewSchema("cavitation-clip-plan.schema.json", planSchema)
	if err != nil {
		return err
	}
	node, err := schema.Read(path, planRaw)
	if err != nil {
		return err
	}
	var p plan
	if err = node.Decode(&p); err != nil {
		return err
	}
	if p.Renderer == "" {
		p.Renderer = "cpu"
	}
	if p.Width < 2 || p.Height < 2 || p.Width > 4096 || p.Height > 4096 || p.FPS < 1 || p.FPS > 60 || p.Seconds < 1 || p.Seconds > 60 || math.IsNaN(p.Track) || math.IsInf(p.Track, 0) || p.Track < 0 || p.Track > 1 {
		return fmt.Errorf("invalid bounded clip dimensions, timing or tracking")
	}
	for _, v := range p.Velocity {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 100 {
			return fmt.Errorf("velocity must be finite and bounded100m/s")
		}
	}
	if err = validateOrbit(p); err != nil {
		return err
	}
	base := filepath.Dir(path)
	template := filepath.Join(base, p.Template)
	destination := filepath.Join(base, p.Output)
	raw, err := os.ReadFile(template)
	if err != nil {
		return err
	}
	if _, err = set.Read(template, raw); err != nil {
		return err
	}
	var preflight yaml.Node
	if err = yamlfile.Read(template, raw, &preflight); err != nil {
		return err
	}
	if err = requireUnitWorld(&preflight); err != nil {
		return err
	}
	if _, err = attachedFeet(&preflight, p.AttachedFizzes); err != nil {
		return err
	}
	for _, sub := range []string{"scenes", "jobs", "frames", "raw", "logs"} {
		if err = os.MkdirAll(filepath.Join(destination, sub), 0755); err != nil {
			return err
		}
	}
	for index := 0; index < p.FPS*p.Seconds; index++ {
		seconds := float64(index) / float64(p.FPS)
		delta := [3]float64{p.Velocity[0] * seconds, p.Velocity[1] * seconds, p.Velocity[2] * seconds}
		var document yaml.Node
		if err = yamlfile.Read(template, raw, &document); err != nil {
			return err
		}
		position, err := nodeAt(&document, "scene", "set", "things", p.Subject, "position")
		if err != nil {
			return err
		}
		for i, key := range []string{"east", "north", "up"} {
			if err = shiftMetreQuantity(position, key, delta[i]); err != nil {
				return err
			}
		}
		origin, err := nodeAt(&document, "scene", "world", "vapor-cavities", p.Field, "origin")
		if err != nil {
			return err
		}
		if err = shiftTriple(origin, delta); err != nil {
			return err
		}
		feet, err := attachedFeet(&document, p.AttachedFizzes)
		if err != nil {
			return err
		}
		for _, foot := range feet {
			if err = shiftTriple(foot, delta); err != nil {
				return err
			}
		}
		cameraDelta := [3]float64{delta[0] * p.Track, delta[1] * p.Track, delta[2] * p.Track}
		for _, part := range [][]string{{"location", "at"}, {"orientation", "look_at"}} {
			keys := append([]string{"scene", "cameras", p.Camera}, part...)
			point, err := nodeAt(&document, keys...)
			if err != nil {
				return err
			}
			if err = shiftTriple(point, cameraDelta); err != nil {
				return err
			}
			if p.Orbit != nil {
				eye, aim := p.Orbit.points(seconds, float64(p.Seconds), delta)
				value := aim
				if part[0] == "location" {
					value = eye
				}
				if err = replaceTriple(point, value); err != nil {
					return err
				}
			}
		}
		snapshot, err := yaml.Marshal(&document)
		if err != nil {
			return err
		}
		stem := fmt.Sprintf("frame-%06d", index)
		if err = writeSameOrNew(filepath.Join(destination, "scenes", stem+".yaml"), snapshot); err != nil {
			return err
		}
		rays, view := 1, 80.0
		if p.Rays != nil {
			rays = *p.Rays
		}
		if p.VerticalView != nil {
			view = *p.VerticalView
		}
		job := fmt.Sprintf("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: ../scenes/%s.yaml}\nfilm: {preset: half-portrait, width: %d, height: %d, rays: %d, tall: %g}\nrenderer: %s\nquality: exact\ncamera: {name: %q}\nat: {value: %.9f, unit: s}\noutputs: {picture: ../frames/%s.png, raw: ../raw/%s.pfm}\nprint: {knee: 0.7, top: 8, stops: 10}\n", stem, p.Width, p.Height, rays, view, p.Renderer, p.Camera, seconds+p.StartSeconds, stem, stem)
		if err = writeSameOrNew(filepath.Join(destination, "jobs", stem+".yaml"), []byte(job)); err != nil {
			return err
		}
	}
	movie := fmt.Sprintf("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {frames: ../raw}\nfilm: {preset: half-portrait, width: %d, height: %d}\nfps: %d\nduration: {value: %d, unit: s}\nprint: {knee: 0.7, top: 8, stops: 10}\noutputs: {movie: ../alfa-underway-moonlight.mp4}\n", p.Width, p.Height, p.FPS, p.Seconds)
	if err = writeSameOrNew(filepath.Join(destination, "jobs", "encode.yaml"), []byte(movie)); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "authored %d real frame jobs at %dx%d, %dfps, %ds; no images rendered\n", p.FPS*p.Seconds, p.Width, p.Height, p.FPS, p.Seconds)
	return err
}
