package settings

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const validYAML = `frame:
  width: 4096
  height: 3002
score: score.yaml
cues: cues.yaml
passes:
  nodisc: passes/nodisc.pfm
  disc: passes/disc.pfm
  mask: passes/mask.pfm
  water: passes/water.pfm
  trees: passes/trees.pfm
  back_left: passes/back-left.pfm
  back_right: passes/back-right.pfm
  overhead: passes/overhead.pfm
fire:
  frames: {path: fire/frames}
  swell: {off: true}
  burst: {path: fire-burst/frames}
  gain: 0.1
  decay: 1
  fps: 24
  steady_from: 42
  pulse: 3
  flash: 8
  snap: 0.12
  swell_fall: 0.3
  burst_gain: 1
  top: 0.6
song: {off: true}
camera:
  eye: 71.7
  look: -6
  focal: 85
  frame_height: 39.61
  moon_alt: 2.3
  moon_scale: 6
  shore_distance: 1660
  moon_radius: 1.554
  moon_disc_alt: 5.45
night:
  haze_gain: 2.5
  flood_gain: 400
  spot_gain: 150
  search_gain: 0.9
  sky_glow: 0.02
  overhead_gain: 0.002
  overhead_mist: 3
  mist_thick: 12
  rig_deck: 20.4
  bloom: 1.6
  exposure: 23
  wind: 8
  mist_seed: 3
  strobe_mist: 0.25
  search_count: 8
  flood_passes: [back-left, back-right]
  mist_colour: [0.78, 0.88, 1.08]
beam:
  out: 900
  radius: 90
  apex: 450
  gain: 40
  halo: 0.1
gates:
  load_limit: 16
  swap_limit_bytes: 8589934592
fps: 24
disk_floor_bytes: 37580963840
output:
  memory_cache_bytes: 536870912
`

func TestSettingsOptionalSpotPasses(t *testing.T) {
	for _, side := range []string{"left", "right"} {
		for _, index := range []string{"0", "1", "2"} {
			name := "spot_" + side + "_" + index
			for _, value := range []string{"{path: spot.pfm}", "{off: true}"} {
				text := strings.Replace(validYAML, "passes:\n", "passes:\n  "+name+": "+value+"\n", 1)
				if _, err := parseText(text); err != nil {
					t.Fatal(err)
				}
			}
			text := strings.Replace(validYAML, "passes:\n", "passes:\n  "+name+": {path: ''}\n", 1)
			if _, err := parseText(text); err == nil || !strings.Contains(err.Error(), "passes."+name) {
				t.Fatalf("%s: %v", name, err)
			}
		}
	}
}

func parseText(text string) (Settings, error) {
	return Parse(strings.NewReader(text))
}

func TestParseConcreteSettings(t *testing.T) {
	got, err := parseText(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	if got.Frame != (Frame{Width: 4096, Height: 3002}) || got.FPS != 24 || got.DiskFloorBytes != 37580963840 || got.Output.MemoryCacheBytes != 536870912 {
		t.Fatalf("wrong resource settings: %+v", got)
	}
	if got.Score != "score.yaml" || got.Cues != "cues.yaml" || got.Passes.BackLeft != "passes/back-left.pfm" || got.Passes.BackRight != "passes/back-right.pfm" || got.Passes.Overhead != "passes/overhead.pfm" {
		t.Fatalf("wrong inputs: %+v", got)
	}
	if got.Camera.Look != -6 || got.Camera.FrameHeight != 39.61 || got.Night.HazeGain != 2.5 || got.Night.MistThick != 12 || got.Night.Exposure != 23 {
		t.Fatalf("wrong scene values: %+v", got)
	}
	if got.Fire.Frames == nil || got.Fire.Frames.Path != "fire/frames" || got.Fire.Swell == nil || !got.Fire.Swell.Off || got.Song == nil || !got.Song.Off {
		t.Fatalf("wrong optional inputs: %+v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalInputs(t *testing.T) {
	text := strings.Replace(validYAML, "  frames: {path: fire/frames}\n", "", 1)
	got, err := parseText(text)
	if err != nil || got.Fire.Frames != nil || got.Fire.Swell == nil || !got.Fire.Swell.Off {
		t.Fatalf("omitted and off must remain distinct: %+v, %v", got.Fire, err)
	}
	for _, value := range []string{"{}", "{path: ''}", "{path: '   '}", "{off: false}", "{path: fire, off: true}"} {
		t.Run(value, func(t *testing.T) {
			_, err := parseText(strings.Replace(validYAML, "{path: fire/frames}", value, 1))
			if err == nil || !strings.Contains(err.Error(), "fire.frames") {
				t.Fatalf("expected fire.frames error, got %v", err)
			}
		})
	}
}

func TestParseRejectsInvalidFields(t *testing.T) {
	cases := []struct{ old, replacement, field string }{
		{"width: 4096", "width: 0", "frame.width"},
		{"height: 3002", "height: -1", "frame.height"},
		{"width: 4096", "width: 9223372036854775807", "frame"},
		{"width: 4096", "width: 1.5", "frame.width"},
		{"width: 4096", "width: 18446744073709551616", "frame.width"},
		{"\nfps: 24", "\nfps: 0", "fps"},
		{"\nfps: 24", "\nfps: .nan", "fps"},
		{"\nfps: 24", "\nfps: .inf", "fps"},
		{"\nfps: 24", "\nfps: -2", "fps"},
		{"disk_floor_bytes: 37580963840", "disk_floor_bytes: 0", "disk_floor_bytes"},
		{"disk_floor_bytes: 37580963840", "disk_floor_bytes: -1", "disk_floor_bytes"},
		{"memory_cache_bytes: 536870912", "memory_cache_bytes: 0", "output.memory_cache_bytes"},
		{"memory_cache_bytes: 536870912", "memory_cache_bytes: 18446744073709551616", "output.memory_cache_bytes"},
		{"score: score.yaml", "score: ''", "score"},
		{"cues: cues.yaml", "cues: '  '", "cues"},
		{"focal: 85", "focal: 0", "camera.focal"},
		{"frame_height: 39.61", "frame_height: -2", "camera.frame_height"},
		{"eye: 71.7", "eye: .nan", "camera.eye"},
		{"look: -6", "look: -.inf", "camera.look"},
		{"moon_alt: 2.3", "moon_alt: .inf", "camera.moon_alt"},
		{"shore_distance: 1660", "shore_distance: 0", "camera.shore_distance"},
		{"moon_radius: 1.554", "moon_radius: 0", "camera.moon_radius"},
		{"moon_scale: 6", "moon_scale: 0", "camera.moon_scale"},
		{"flood_gain: 400", "flood_gain: -1", "night.flood_gain"},
		{"spot_gain: 150", "spot_gain: .inf", "night.spot_gain"},
		{"haze_gain: 2.5", "haze_gain: -1", "night.haze_gain"},
		{"search_gain: 0.9", "search_gain: .nan", "night.search_gain"},
		{"sky_glow: 0.02", "sky_glow: .inf", "night.sky_glow"},
		{"overhead_gain: 0.002", "overhead_gain: -1", "night.overhead_gain"},
		{"overhead_mist: 3", "overhead_mist: -1", "night.overhead_mist"},
		{"mist_thick: 12", "mist_thick: 0", "night.mist_thick"},
		{"bloom: 1.6", "bloom: -1", "night.bloom"},
		{"exposure: 23", "exposure: .nan", "night.exposure"},
		{"rig_deck: 20.4", "rig_deck: .inf", "night.rig_deck"},
		{"wind: 8", "wind: -1", "night.wind"},
		{"strobe_mist: 0.25", "strobe_mist: -1", "night.strobe_mist"},
		{"\nfps: 24", "\nfps: 24\nscene: moonlake", "scene"},
		{"focal: 85", "focal: 85\n  typo: 3", "typo"},
		{"memory_cache_bytes: 536870912", "memory_cache_bytes: 536870912\n  typo: 3", "typo"},
		{"{path: fire/frames}", "{path: fire/frames, typo: true}", "typo"},
		{"\nfps: 24", "\nfps: 24\nfps: 25", "fps"},
	}
	for _, tc := range cases {
		t.Run(tc.field+"/"+tc.replacement, func(t *testing.T) {
			_, err := parseText(strings.Replace(validYAML, tc.old, tc.replacement, 1))
			if err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("expected %s error, got %v", tc.field, err)
			}
		})
	}
	for _, name := range []string{"nodisc", "disc", "mask", "water", "trees", "back_left", "back_right", "overhead"} {
		t.Run("missing/"+name, func(t *testing.T) {
			lines := strings.Split(validYAML, "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, "  "+name+": passes/") {
					lines[i] = ""
				}
			}
			_, err := parseText(strings.Join(lines, "\n"))
			if err == nil || !strings.Contains(err.Error(), "passes."+name) {
				t.Fatalf("expected passes.%s error, got %v", name, err)
			}
		})
	}
}

func TestParseDocumentBoundary(t *testing.T) {
	for _, text := range []string{"", "null", "[]", "{", validYAML + "---\n", validYAML + "---\nfps: 25\n"} {
		if _, err := parseText(text); err == nil {
			t.Fatalf("accepted invalid document %q", text)
		}
	}
	if _, err := parseText(validYAML + "...\n"); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("reader broke")
	if _, err := Parse(brokenReader{failure}); !errors.Is(err, failure) {
		t.Fatalf("lost reader error: %v", err)
	}
}

type brokenReader struct{ err error }

func (r brokenReader) Read([]byte) (int, error) { return 0, r.err }

var _ io.Reader = brokenReader{}

func TestRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.yaml")
	if _, err := Read(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("expected named read error, got %v", err)
	}
	if err := os.WriteFile(path, []byte(validYAML), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	want, parseErr := parseText(validYAML)
	if err != nil || parseErr != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("read differs: %+v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("fps: .nan"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("expected named validation error, got %v", err)
	}
}

func TestValidateRejectsProgrammaticNonFinite(t *testing.T) {
	got, err := parseText(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	got.Night.Bloom = math.Inf(1)
	if err := got.Validate(); err == nil || !strings.Contains(err.Error(), "night.bloom") {
		t.Fatalf("expected bloom error, got %v", err)
	}
}

func TestRequiredKeys(t *testing.T) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(validYAML), &document); err != nil {
		t.Fatal(err)
	}
	var visit func(*yaml.Node, reflect.Type, string)
	visit = func(node *yaml.Node, shape reflect.Type, prefix string) {
		for i := range shape.NumField() {
			member := shape.Field(i)
			if member.Type.Kind() == reflect.Pointer {
				continue
			}
			key := member.Tag.Get("yaml")
			field := prefix + key
			for j := 0; j < len(node.Content); j += 2 {
				if node.Content[j].Value != key {
					continue
				}
				t.Run(field, func(t *testing.T) {
					original := node.Content
					node.Content = append(append([]*yaml.Node{}, original[:j]...), original[j+2:]...)
					text, err := yaml.Marshal(&document)
					node.Content = original
					if err != nil {
						t.Fatal(err)
					}
					if _, err := parseText(string(text)); err == nil || !strings.Contains(err.Error(), field) {
						t.Fatalf("expected missing %s error, got %v", field, err)
					}
				})
				if member.Type.Kind() == reflect.Struct {
					visit(node.Content[j+1], member.Type, field+".")
				}
				break
			}
		}
	}
	visit(document.Content[0], reflect.TypeOf(Settings{}), "")
}

func TestExplicitZeroValues(t *testing.T) {
	text := validYAML
	for _, pair := range [][2]string{
		{"look: -6", "look: 0"},
		{"moon_alt: 2.3", "moon_alt: 0"}, {"haze_gain: 2.5", "haze_gain: 0"},
		{"flood_gain: 400", "flood_gain: 0"}, {"spot_gain: 150", "spot_gain: 0"},
		{"search_gain: 0.9", "search_gain: 0"}, {"sky_glow: 0.02", "sky_glow: 0"},
		{"overhead_gain: 0.002", "overhead_gain: 0"}, {"overhead_mist: 3", "overhead_mist: 0"},
		{"rig_deck: 20.4", "rig_deck: 0"}, {"bloom: 1.6", "bloom: 0"},
		{"exposure: 23", "exposure: 0"}, {"wind: 8", "wind: 0"},
		{"mist_seed: 3", "mist_seed: 0"}, {"strobe_mist: 0.25", "strobe_mist: 0"},
	} {
		text = strings.Replace(text, pair[0], pair[1], 1)
	}
	if _, err := parseText(text); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"0.5", "0.999"} {
		if _, err := parseText(strings.Replace(validYAML, "mist_thick: 12", "mist_thick: "+value, 1)); err == nil || !strings.Contains(err.Error(), "night.mist_thick") {
			t.Fatalf("expected mist_thick minimum error, got %v", err)
		}
	}
	if _, err := parseText(strings.Replace(validYAML, "mist_thick: 12", "mist_thick: 1", 1)); err != nil {
		t.Fatal(err)
	}
}

func TestParseSizeLimit(t *testing.T) {
	const limit = 1 << 20
	padding := strings.Repeat(" ", limit-len(validYAML))
	if _, err := parseText(validYAML + padding); err != nil {
		t.Fatalf("at limit: %v", err)
	}
	if _, err := parseText(validYAML + padding + " "); err == nil || !strings.Contains(err.Error(), "size") {
		t.Fatalf("expected size error, got %v", err)
	}
	reader := &countedSpaces{}
	if _, err := Parse(reader); err == nil || !strings.Contains(err.Error(), "size") || reader.count != limit+1 {
		t.Fatalf("unbounded read: %d bytes, %v", reader.count, err)
	}
}

type countedSpaces struct{ count int }

func (r *countedSpaces) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	r.count += len(p)
	return len(p), nil
}

func TestAppearanceControls(t *testing.T) {
	got, err := parseText(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	if got.Beam != (Beam{Out: 900, Radius: 90, Apex: 450, Gain: 40, Halo: 0.1}) || got.Fire.Gain != 0.1 || got.Fire.Decay != 1 || got.Fire.FPS != 24 || got.Fire.SteadyFrom != 42 || got.Fire.Pulse != 3 || got.Fire.Flash != 8 || got.Fire.Snap != 0.12 || got.Fire.SwellFall != 0.3 || got.Fire.BurstGain != 1 || got.Fire.Top != 0.6 || got.Camera.MoonDiscAlt != 5.45 || got.Night.SearchCount != 8 || got.Night.FloodPasses != [2]string{"back-left", "back-right"} || got.Night.MistColour != [3]float64{0.78, 0.88, 1.08} || got.Gates.LoadLimit != 16 || got.Gates.SwapLimitBytes != 8589934592 {
		t.Fatalf("appearance controls changed: %+v", got)
	}
	cases := []struct{ old, replacement, field string }{
		{"out: 900", "out: .nan", "beam.out"},
		{"radius: 90", "radius: -1", "beam.radius"},
		{"apex: 450", "apex: 0", "beam.apex"},
		{"  gain: 40\n", "  gain: .inf\n", "beam.gain"},
		{"halo: 0.1", "halo: -1", "beam.halo"},
		{"gain: 0.1", "gain: -1", "fire.gain"},
		{"decay: 1", "decay: 0", "fire.decay"},
		{"  fps: 24", "  fps: 0", "fire.fps"},
		{"steady_from: 42", "steady_from: -1", "fire.steady_from"},
		{"steady_from: 42", "steady_from: 1.5", "fire.steady_from"},
		{"pulse: 3", "pulse: .nan", "fire.pulse"},
		{"flash: 8", "flash: -1", "fire.flash"},
		{"snap: 0.12", "snap: 0", "fire.snap"},
		{"swell_fall: 0.3", "swell_fall: 0", "fire.swell_fall"},
		{"burst_gain: 1", "burst_gain: .inf", "fire.burst_gain"},
		{"top: 0.6", "top: -1", "fire.top"},
		{"moon_disc_alt: 5.45", "moon_disc_alt: .inf", "camera.moon_disc_alt"},
		{"search_count: 8", "search_count: 0", "night.search_count"},
		{"search_count: 8", "search_count: 1.5", "night.search_count"},
		{"search_count: 8", "search_count: 18446744073709551616", "night.search_count"},
		{"[back-left, back-right]", "[one]", "night.flood_passes"},
		{"[back-left, back-right]", "[one, two, three]", "night.flood_passes"},
		{"[back-left, back-right]", "['', right]", "night.flood_passes[0]"},
		{"[back-left, back-right]", "['../escape', right]", "night.flood_passes[0]"},
		{"[0.78, 0.88, 1.08]", "[1, 2]", "night.mist_colour"},
		{"[0.78, 0.88, 1.08]", "[1, 2, 3, 4]", "night.mist_colour"},
		{"[0.78, 0.88, 1.08]", "[1, .nan, 3]", "night.mist_colour[1]"},
		{"load_limit: 16", "load_limit: 0", "gates.load_limit"},
		{"load_limit: 16", "load_limit: .inf", "gates.load_limit"},
		{"swap_limit_bytes: 8589934592", "swap_limit_bytes: -1", "gates.swap_limit_bytes"},
	}
	for _, tc := range cases {
		t.Run(tc.field+tc.replacement, func(t *testing.T) {
			if _, err := parseText(strings.Replace(validYAML, tc.old, tc.replacement, 1)); err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("expected %s error, got %v", tc.field, err)
			}
		})
	}
	text := strings.NewReplacer("out: 900", "out: -20", "radius: 90", "radius: 0", "moon_disc_alt: 5.45", "moon_disc_alt: -4", "search_count: 8", "search_count: 1", "[back-left, back-right]", "[custom, custom]", "[0.78, 0.88, 1.08]", "[-0.1, 0, 2]", "steady_from: 42", "steady_from: 0", "pulse: 3", "pulse: 0", "top: 0.6", "top: 0", "swap_limit_bytes: 8589934592", "swap_limit_bytes: 0").Replace(validYAML)
	if _, err := parseText(text); err != nil {
		t.Fatalf("unnecessarily constrained scene: %v", err)
	}
}

func TestAllAppearanceNumbersFinite(t *testing.T) {
	base, err := parseText(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	var visit func(reflect.Value, string)
	visit = func(value reflect.Value, field string) {
		switch value.Kind() {
		case reflect.Struct:
			for i := range value.NumField() {
				visit(value.Field(i), strings.TrimPrefix(field+"."+value.Type().Field(i).Tag.Get("yaml"), "."))
			}
		case reflect.Array:
			for i := range value.Len() {
				visit(value.Index(i), field)
			}
		case reflect.Float64:
			old := value.Float()
			for _, invalid := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
				value.SetFloat(invalid)
				if err := base.Validate(); err == nil || !strings.Contains(err.Error(), field) {
					t.Errorf("expected finite %s error, got %v", field, err)
				}
			}
			value.SetFloat(old)
		}
	}
	visit(reflect.ValueOf(&base).Elem(), "")
}

func TestReferenceAndDebugInputs(t *testing.T) {
	for _, field := range []string{"moon_ref", "debug_mask"} {
		for _, state := range []string{"{off: true}", "{path: custom.pfm}"} {
			got, err := parseText(validYAML + field + ": " + state + "\n")
			if err != nil {
				t.Fatal(err)
			}
			input := got.MoonRef
			if field == "debug_mask" {
				input = got.DebugMask
			}
			if input == nil || (!input.Off && input.Path != "custom.pfm") {
				t.Fatalf("lost %s: %+v", field, input)
			}
		}
		for _, state := range []string{"{}", "null", "{path: a, off: true}", "{typo: true}"} {
			if _, err := parseText(validYAML + field + ": " + state + "\n"); err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("expected %s error, got %v", field, err)
			}
		}
	}
}

func TestOceanCameraBoundaries(t *testing.T) {
	for _, tc := range []struct {
		old, field string
		invalid    []string
		valid      []string
	}{
		{"eye: 71.7", "camera.eye", []string{"0", "-0.1"}, []string{"0.0001", "100"}},
		{"look: -6", "camera.look", []string{"-90", "90", "-91", "91"}, []string{"-89.999", "0", "89.999"}},
		{"moon_alt: 2.3", "camera.moon_alt", []string{"-90.001", "90.001"}, []string{"-90", "0", "90"}},
	} {
		key := strings.Split(tc.old, ":")[0]
		for _, value := range tc.invalid {
			t.Run(tc.field+"/invalid/"+value, func(t *testing.T) {
				if _, err := parseText(strings.Replace(validYAML, tc.old, key+": "+value, 1)); err == nil || !strings.Contains(err.Error(), tc.field) {
					t.Fatalf("expected %s error, got %v", tc.field, err)
				}
			})
		}
		for _, value := range tc.valid {
			t.Run(tc.field+"/valid/"+value, func(t *testing.T) {
				if _, err := parseText(strings.Replace(validYAML, tc.old, key+": "+value, 1)); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	got, err := parseText(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	got.Camera.Look = 90
	if err := got.Validate(); err == nil || !strings.Contains(err.Error(), "camera.look") {
		t.Fatalf("expected direct validation error, got %v", err)
	}
}

func TestSceneValuesAreData(t *testing.T) {
	text := strings.NewReplacer("eye: 71.7", "eye: 0.5", "look: -6", "look: 5", "moon_alt: 2.3", "moon_alt: -3", "haze_gain: 2.5", "haze_gain: 0", "bloom: 1.6", "bloom: 0", "exposure: 23", "exposure: -7", "fps: 24", "fps: 29.97", "width: 4096", "width: 1024", "mist_thick: 12", "mist_thick: 1.5").Replace(validYAML)
	got, err := parseText(text)
	if err != nil || got.Camera.Eye != 0.5 || got.Camera.Look != 5 || got.Camera.MoonAlt != -3 || got.Night.Exposure != -7 || got.FPS != 29.97 || got.Frame.Width != 1024 || got.Night.MistThick != 1.5 {
		t.Fatalf("scene values changed or refused: %+v, %v", got, err)
	}
}
