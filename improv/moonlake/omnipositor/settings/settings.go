package settings

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	Frame          Frame         `yaml:"frame"`
	Score          string        `yaml:"score"`
	Cues           string        `yaml:"cues"`
	Passes         Passes        `yaml:"passes"`
	Fire           Fire          `yaml:"fire"`
	Song           *OptionalPath `yaml:"song"`
	Camera         Camera        `yaml:"camera"`
	Night          Night         `yaml:"night"`
	Beam           Beam          `yaml:"beam"`
	Gates          Gates         `yaml:"gates"`
	MoonRef        *OptionalPath `yaml:"moon_ref"`
	DebugMask      *OptionalPath `yaml:"debug_mask"`
	FPS            float64       `yaml:"fps"`
	DiskFloorBytes int64         `yaml:"disk_floor_bytes"`
	Output         Output        `yaml:"output"`
}

type Frame struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

type Passes struct {
	Nodisc     string        `yaml:"nodisc"`
	Disc       string        `yaml:"disc"`
	Mask       string        `yaml:"mask"`
	Water      string        `yaml:"water"`
	Trees      string        `yaml:"trees"`
	BackLeft   string        `yaml:"back_left"`
	BackRight  string        `yaml:"back_right"`
	Overhead   string        `yaml:"overhead"`
	SpotLeft0  *OptionalPath `yaml:"spot_left_0"`
	SpotLeft1  *OptionalPath `yaml:"spot_left_1"`
	SpotLeft2  *OptionalPath `yaml:"spot_left_2"`
	SpotRight0 *OptionalPath `yaml:"spot_right_0"`
	SpotRight1 *OptionalPath `yaml:"spot_right_1"`
	SpotRight2 *OptionalPath `yaml:"spot_right_2"`
}

type OptionalPath struct {
	Path string `yaml:"path"`
	Off  bool   `yaml:"off"`
}

type Fire struct {
	Frames     *OptionalPath `yaml:"frames"`
	Swell      *OptionalPath `yaml:"swell"`
	Burst      *OptionalPath `yaml:"burst"`
	Gain       float64       `yaml:"gain"`
	Decay      float64       `yaml:"decay"`
	FPS        float64       `yaml:"fps"`
	SteadyFrom int           `yaml:"steady_from"`
	Pulse      float64       `yaml:"pulse"`
	Flash      float64       `yaml:"flash"`
	Snap       float64       `yaml:"snap"`
	SwellFall  float64       `yaml:"swell_fall"`
	BurstGain  float64       `yaml:"burst_gain"`
	Top        float64       `yaml:"top"`
}

type Camera struct {
	Eye           float64 `yaml:"eye"`
	Look          float64 `yaml:"look"`
	Focal         float64 `yaml:"focal"`
	FrameHeight   float64 `yaml:"frame_height"`
	MoonAlt       float64 `yaml:"moon_alt"`
	MoonScale     float64 `yaml:"moon_scale"`
	ShoreDistance float64 `yaml:"shore_distance"`
	MoonRadius    float64 `yaml:"moon_radius"`
	MoonDiscAlt   float64 `yaml:"moon_disc_alt"`
}

type Night struct {
	HazeGain     float64    `yaml:"haze_gain"`
	FloodGain    float64    `yaml:"flood_gain"`
	SpotGain     float64    `yaml:"spot_gain"`
	SearchGain   float64    `yaml:"search_gain"`
	SkyGlow      float64    `yaml:"sky_glow"`
	OverheadGain float64    `yaml:"overhead_gain"`
	OverheadMist float64    `yaml:"overhead_mist"`
	MistThick    float64    `yaml:"mist_thick"`
	RigDeck      float64    `yaml:"rig_deck"`
	Bloom        float64    `yaml:"bloom"`
	Exposure     float64    `yaml:"exposure"`
	Wind         float64    `yaml:"wind"`
	MistSeed     uint64     `yaml:"mist_seed"`
	StrobeMist   float64    `yaml:"strobe_mist"`
	SearchCount  int        `yaml:"search_count"`
	FloodPasses  [2]string  `yaml:"flood_passes"`
	MistColour   [3]float64 `yaml:"mist_colour"`
}

type Beam struct {
	Out    float64 `yaml:"out"`
	Radius float64 `yaml:"radius"`
	Apex   float64 `yaml:"apex"`
	Gain   float64 `yaml:"gain"`
	Halo   float64 `yaml:"halo"`
}

type Gates struct {
	LoadLimit      float64 `yaml:"load_limit"`
	SwapLimitBytes int64   `yaml:"swap_limit_bytes"`
}

type Output struct {
	MemoryCacheBytes int64 `yaml:"memory_cache_bytes"`
}

// Read opens the file at path and parses it with Parse. Its errors name the
// path.
func Read(path string) (Settings, error) {
	file, err := os.Open(path)
	if err != nil {
		return Settings{}, fmt.Errorf("settings %s: %w", path, err)
	}
	got, parseErr := Parse(file)
	closeErr := file.Close()
	if parseErr != nil {
		return Settings{}, fmt.Errorf("settings %s: %w", path, parseErr)
	}
	if closeErr != nil {
		return Settings{}, fmt.Errorf("settings %s: %w", path, closeErr)
	}
	return got, nil
}

// Parse reads settings as YAML from input, which may be at most 1 MiB. It
// needs exactly one document, refuses unknown or repeated fields, fields of
// the wrong type, and missing required fields, then runs Validate on the
// result.
func Parse(input io.Reader) (Settings, error) {
	const maxSettingsBytes = 1 << 20
	text, err := io.ReadAll(io.LimitReader(input, maxSettingsBytes+1))
	if err != nil {
		return Settings{}, fmt.Errorf("settings: read: %w", err)
	}
	if len(text) > maxSettingsBytes {
		return Settings{}, fmt.Errorf("settings: input size exceeds %d bytes", maxSettingsBytes)
	}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(text))
	if err := decoder.Decode(&document); err != nil {
		return Settings{}, fmt.Errorf("settings: document: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return Settings{}, fmt.Errorf("settings: document: %w", err)
		}
		return Settings{}, fmt.Errorf("settings: document: expected exactly one YAML document")
	}
	if len(document.Content) != 1 {
		return Settings{}, fmt.Errorf("settings: document: expected a mapping")
	}
	if err := checkNode(document.Content[0], reflect.TypeOf(Settings{}), "settings"); err != nil {
		return Settings{}, err
	}
	var got Settings
	decoder = yaml.NewDecoder(bytes.NewReader(text))
	decoder.KnownFields(true)
	if err := decoder.Decode(&got); err != nil {
		return Settings{}, fmt.Errorf("settings: %w", err)
	}
	if err := got.Validate(); err != nil {
		return Settings{}, err
	}
	return got, nil
}

// checkNode checks a YAML node against the Go type shape. A struct needs a
// mapping of known, unrepeated keys and every field that is not a pointer; a
// struct reached through a pointer may leave fields out. An array needs a
// sequence of exactly its length. A scalar needs the right YAML tag, a float
// taking an integer too, and must decode. field names the place in errors.
func checkNode(node *yaml.Node, shape reflect.Type, field string) error {
	optional := shape.Kind() == reflect.Pointer
	if optional {
		shape = shape.Elem()
	}
	if shape.Kind() == reflect.Struct {
		if node.Kind != yaml.MappingNode || node.Tag != "!!map" {
			return fmt.Errorf("%s: expected a mapping", field)
		}
		fields := make(map[string]reflect.Type, shape.NumField())
		for i := range shape.NumField() {
			member := shape.Field(i)
			fields[member.Tag.Get("yaml")] = member.Type
		}
		seen := make(map[string]bool)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			name := field + "." + key.Value
			member, found := fields[key.Value]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || !found {
				return fmt.Errorf("%s: unknown field", name)
			}
			if seen[key.Value] {
				return fmt.Errorf("%s: duplicate field", name)
			}
			seen[key.Value] = true
			if err := checkNode(node.Content[i+1], member, name); err != nil {
				return err
			}
		}
		if !optional {
			for i := range shape.NumField() {
				member := shape.Field(i)
				key := member.Tag.Get("yaml")
				if member.Type.Kind() != reflect.Pointer && !seen[key] {
					return fmt.Errorf("%s.%s: required field is missing", field, key)
				}
			}
		}
		return nil
	}
	if shape.Kind() == reflect.Array {
		if node.Kind != yaml.SequenceNode || node.Tag != "!!seq" || len(node.Content) != shape.Len() {
			return fmt.Errorf("%s: expected exactly %d entries", field, shape.Len())
		}
		for i, child := range node.Content {
			if err := checkNode(child, shape.Elem(), fmt.Sprintf("%s[%d]", field, i)); err != nil {
				return err
			}
		}
		return nil
	}
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("%s: expected a scalar", field)
	}
	var expected string
	switch shape.Kind() {
	case reflect.String:
		expected = "!!str"
	case reflect.Bool:
		expected = "!!bool"
	case reflect.Int, reflect.Int64, reflect.Uint64:
		expected = "!!int"
	case reflect.Float64:
		if node.Tag != "!!int" && node.Tag != "!!float" {
			return fmt.Errorf("%s: expected a number", field)
		}
	}
	if expected != "" && node.Tag != expected {
		return fmt.Errorf("%s: expected %s", field, strings.TrimPrefix(expected, "!!"))
	}
	value := reflect.New(shape)
	if err := node.Decode(value.Interface()); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

// Validate checks the values in s and returns the first problem, named by its
// field. Paths must be nonempty and without NUL, numbers finite and in the
// sign each allows, the camera angles in range, and the flood pass names
// plain names rather than paths. A switched-off optional path must have no
// path. Frame size, disk floor and cache size must be positive and fit.
func (s Settings) Validate() error {
	if s.Frame.Width <= 0 {
		return fmt.Errorf("frame.width: must be positive")
	}
	if s.Frame.Height <= 0 {
		return fmt.Errorf("frame.height: must be positive")
	}
	maxInt := int64(int(^uint(0) >> 1))
	if int64(s.Frame.Width) > maxInt/12/int64(s.Frame.Height) {
		return fmt.Errorf("frame: width * height * 12 overflows addressable RGB float32 bytes")
	}
	paths := []struct{ field, value string }{
		{"score", s.Score}, {"cues", s.Cues},
		{"passes.nodisc", s.Passes.Nodisc}, {"passes.disc", s.Passes.Disc},
		{"passes.mask", s.Passes.Mask}, {"passes.water", s.Passes.Water},
		{"passes.trees", s.Passes.Trees}, {"passes.back_left", s.Passes.BackLeft},
		{"passes.back_right", s.Passes.BackRight}, {"passes.overhead", s.Passes.Overhead},
	}
	for _, path := range paths {
		if err := validPath(path.field, path.value); err != nil {
			return err
		}
	}
	for _, input := range []struct {
		field string
		value *OptionalPath
	}{
		{"fire.frames", s.Fire.Frames}, {"fire.swell", s.Fire.Swell},
		{"fire.burst", s.Fire.Burst}, {"song", s.Song},
		{"moon_ref", s.MoonRef}, {"debug_mask", s.DebugMask},
		{"passes.spot_left_0", s.Passes.SpotLeft0}, {"passes.spot_left_1", s.Passes.SpotLeft1},
		{"passes.spot_left_2", s.Passes.SpotLeft2}, {"passes.spot_right_0", s.Passes.SpotRight0},
		{"passes.spot_right_1", s.Passes.SpotRight1}, {"passes.spot_right_2", s.Passes.SpotRight2},
	} {
		if input.value == nil {
			continue
		}
		if input.value.Off {
			if input.value.Path != "" {
				return fmt.Errorf("%s: off and path are mutually exclusive", input.field)
			}
		} else if err := validPath(input.field+".path", input.value.Path); err != nil {
			return err
		}
	}
	for _, number := range []struct {
		field    string
		value    float64
		positive bool
		negative bool
	}{
		{"fps", s.FPS, true, false},
		{"beam.out", s.Beam.Out, false, true},
		{"beam.radius", s.Beam.Radius, false, false},
		{"beam.apex", s.Beam.Apex, true, false},
		{"beam.gain", s.Beam.Gain, false, false},
		{"beam.halo", s.Beam.Halo, false, false},
		{"fire.gain", s.Fire.Gain, false, false},
		{"fire.decay", s.Fire.Decay, true, false},
		{"fire.fps", s.Fire.FPS, true, false},
		{"fire.pulse", s.Fire.Pulse, false, false},
		{"fire.flash", s.Fire.Flash, false, false},
		{"fire.snap", s.Fire.Snap, true, false},
		{"fire.swell_fall", s.Fire.SwellFall, true, false},
		{"fire.burst_gain", s.Fire.BurstGain, false, false},
		{"fire.top", s.Fire.Top, false, false},
		{"gates.load_limit", s.Gates.LoadLimit, true, false},
		{"camera.moon_disc_alt", s.Camera.MoonDiscAlt, false, true},
		{"night.mist_colour[0]", s.Night.MistColour[0], false, true},
		{"night.mist_colour[1]", s.Night.MistColour[1], false, true},
		{"night.mist_colour[2]", s.Night.MistColour[2], false, true},
		{"camera.eye", s.Camera.Eye, true, false},
		{"camera.look", s.Camera.Look, false, true},
		{"camera.focal", s.Camera.Focal, true, false},
		{"camera.frame_height", s.Camera.FrameHeight, true, false},
		{"camera.moon_alt", s.Camera.MoonAlt, false, true},
		{"camera.moon_scale", s.Camera.MoonScale, true, false},
		{"camera.shore_distance", s.Camera.ShoreDistance, true, false},
		{"camera.moon_radius", s.Camera.MoonRadius, true, false},
		{"night.haze_gain", s.Night.HazeGain, false, false},
		{"night.flood_gain", s.Night.FloodGain, false, false},
		{"night.spot_gain", s.Night.SpotGain, false, false},
		{"night.mist_thick", s.Night.MistThick, true, false},
		{"night.search_gain", s.Night.SearchGain, false, false},
		{"night.sky_glow", s.Night.SkyGlow, false, false},
		{"night.overhead_gain", s.Night.OverheadGain, false, false},
		{"night.overhead_mist", s.Night.OverheadMist, false, false},
		{"night.rig_deck", s.Night.RigDeck, false, true},
		{"night.bloom", s.Night.Bloom, false, false},
		{"night.exposure", s.Night.Exposure, false, true},
		{"night.wind", s.Night.Wind, false, false},
		{"night.strobe_mist", s.Night.StrobeMist, false, false},
	} {
		if math.IsNaN(number.value) || math.IsInf(number.value, 0) {
			return fmt.Errorf("%s: must be finite", number.field)
		}
		if number.positive && number.value <= 0 {
			return fmt.Errorf("%s: must be positive", number.field)
		}
		if !number.negative && number.value < 0 {
			return fmt.Errorf("%s: must not be negative", number.field)
		}
	}
	if s.Camera.Look <= -90 || s.Camera.Look >= 90 {
		return fmt.Errorf("camera.look: must be strictly between -90 and 90 degrees")
	}
	if s.Camera.MoonAlt < -90 || s.Camera.MoonAlt > 90 {
		return fmt.Errorf("camera.moon_alt: must be between -90 and 90 degrees inclusive")
	}
	if s.Fire.SteadyFrom < 0 {
		return fmt.Errorf("fire.steady_from: must not be negative")
	}
	if s.Night.SearchCount <= 0 {
		return fmt.Errorf("night.search_count: must be positive")
	}
	if s.Gates.SwapLimitBytes < 0 {
		return fmt.Errorf("gates.swap_limit_bytes: must not be negative")
	}
	for i, name := range s.Night.FloodPasses {
		field := fmt.Sprintf("night.flood_passes[%d]", i)
		if err := validPath(field, name); err != nil {
			return err
		}
		if name == "." || name == ".." || strings.ContainsAny(name, "/\\\\") {
			return fmt.Errorf("%s: expected a pass name, not a path", field)
		}
	}
	if s.Night.MistThick < 1 {
		return fmt.Errorf("night.mist_thick: must be at least 1")
	}
	if s.DiskFloorBytes <= 0 {
		return fmt.Errorf("disk_floor_bytes: must be positive")
	}
	if s.Output.MemoryCacheBytes <= 0 || s.Output.MemoryCacheBytes > maxInt {
		return fmt.Errorf("output.memory_cache_bytes: must be positive and addressable")
	}
	return nil
}

// validPath refuses a value that is blank or holds a NUL byte, naming field.
func validPath(field, value string) error {
	if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s: must be a nonempty path without NUL", field)
	}
	return nil
}
