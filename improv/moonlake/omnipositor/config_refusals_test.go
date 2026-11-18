package omnipositor

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/oceantex"
)

// TestConfigValidateRefusals checks that each bad value in a compositor
// config is refused with its own message, starting from one that is valid.
func TestConfigValidateRefusals(t *testing.T) {
	if err := testConfig(t).validate(); err != nil {
		// The base needs a light stage to be valid; give it a recording one.
		cfg := testConfig(t)
		cfg.Light = &recordingLight{}
		if err := cfg.validate(); err != nil {
			t.Fatalf("base config: %v", err)
		}
	}
	nan := math.NaN()
	for _, c := range []struct {
		name string
		edit func(c *Config)
		want string
	}{
		{"nan fps", func(c *Config) { c.FPS = nan }, "finite positive fps"},
		{"negative from", func(c *Config) { c.From = -1 }, "finite positive fps"},
		{"empty range", func(c *Config) { c.To = c.From }, "finite positive fps"},
		{"zero fps", func(c *Config) { c.FPS = 0 }, "finite positive fps"},
		{"timestamps merge", func(c *Config) { c.From, c.To, c.FPS = 0, 1<<53+2, 1 }, "unrepresentable"},
		{"zero width", func(c *Config) { c.Width = 0 }, "invalid or excessive dimensions"},
		{"zero height", func(c *Config) { c.Height = 0 }, "invalid or excessive dimensions"},
		{"too many pixels", func(c *Config) { c.Width, c.Height = 1<<30, 1<<30 }, "invalid or excessive dimensions"},
		{"no show", func(c *Config) { c.Show = nil }, "cues and lighting are required"},
		{"no light", func(c *Config) { c.Light = nil }, "cues and lighting are required"},
		{"nan top", func(c *Config) { c.Top = nan }, "nonfinite configuration"},
		{"infinite mist colour", func(c *Config) { c.MistColour[2] = math.Inf(1) }, "nonfinite configuration"},
		{"nan white", func(c *Config) { c.Develop.White = nan }, "nonfinite configuration"},
		{"bloom too large", func(c *Config) { c.Develop.Bloom = 100001 }, "bloom exceeds"},
		{"top below bottom", func(c *Config) { c.Top, c.Bottom = -3, 3 }, "invalid geometry"},
		{"shore above top", func(c *Config) { c.Shore = 4 }, "invalid geometry"},
		{"shore on the bottom row", func(c *Config) { c.Shore = -3 }, "invalid geometry"},
		{"shore in the first row", func(c *Config) { c.Shore = 2.9 }, "invalid geometry"},
		{"zero moon radius", func(c *Config) { c.MoonRadius = 0 }, "invalid geometry"},
		{"thin mist", func(c *Config) { c.MistLayers = 0.5 }, "invalid geometry"},
		{"negative overhead mist", func(c *Config) { c.OverheadMist = -1 }, "invalid geometry"},
		{"negative strobe mist", func(c *Config) { c.StrobeMist = -1 }, "invalid geometry"},
		{"moon reference of the wrong size", func(c *Config) { c.MoonReference.Width = 3 }, "moon reference"},
		{"water mask of the wrong size", func(c *Config) { c.WaterMask.Light = c.WaterMask.Light[:3] }, "water mask"},
		{"negative water", func(c *Config) {
			c.WaterMask.Light = append([]float64(nil), c.WaterMask.Light...)
			c.WaterMask.Light[0] = -0.5
		}, "negative water mask"},
		{"no water light", func(c *Config) { c.WaterMask.Light = make([]float64, 12) }, "positive light"},
	} {
		cfg := testConfig(t)
		cfg.Light = &recordingLight{}
		c.edit(&cfg)
		if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// recordingLight is a light stage that is never asked for light.
type recordingLight struct{}

// Light returns an empty result; the validation tests never call it.
func (*recordingLight) Light(_ context.Context, _ LightRequest) (LightResult, error) {
	return LightResult{}, nil
}

// TestBottomUpReversesRows checks the row order and values for a hand-worked
// two-row image, and the refusals for a wrong size and values that do not fit.
func TestBottomUpReversesRows(t *testing.T) {
	input := lights.Image{Width: 1, Height: 2, Light: []float64{1, 2, 3, 4, 5, 6}}
	got, err := bottomUp(input, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{4, 5, 6, 1, 2, 3}
	for i := range want {
		if got.Pixels[i] != want[i] {
			t.Fatalf("pixels %v, want %v", got.Pixels, want)
		}
	}
	if got.Scale != -1 || got.Width != 1 || got.Height != 2 {
		t.Errorf("shape %+v", got)
	}
	for name, light := range map[string][]float64{
		"nan":      {1, 2, 3, math.NaN(), 5, 6},
		"infinite": {1, 2, 3, math.Inf(-1), 5, 6},
		"too big":  {1, 2, 3, 4, 5, 1e39},
		"too few":  {1, 2, 3},
	} {
		if _, err := bottomUp(lights.Image{Width: 1, Height: 2, Light: light}, 1, 2); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := bottomUp(input, 2, 1); err == nil || !strings.Contains(err.Error(), "dimensions") {
		t.Errorf("wrong size: %v", err)
	}
}

// TestSeaFloatKeepsRowOrder checks that an ocean texture keeps its row order
// and refuses a missing texture, a wrong size and values that do not fit.
func TestSeaFloatKeepsRowOrder(t *testing.T) {
	texture := &oceantex.Texture{Width: 1, Height: 2, Light: [][3]float64{{1, 2, 3}, {4, 5, 6}}}
	got, err := seaFloat(texture, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{1, 2, 3, 4, 5, 6}
	for i := range want {
		if got.Pixels[i] != want[i] {
			t.Fatalf("pixels %v, want %v", got.Pixels, want)
		}
	}
	for name, input := range map[string]*oceantex.Texture{
		"nil":        nil,
		"wrong size": {Width: 2, Height: 1, Light: texture.Light},
		"nan":        {Width: 1, Height: 2, Light: [][3]float64{{1, math.NaN(), 3}, {4, 5, 6}}},
		"too big":    {Width: 1, Height: 2, Light: [][3]float64{{1, 2, 3}, {4, 5, -1e39}}},
	} {
		if _, err := seaFloat(input, 1, 2); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// TestNewProfileLimits checks the span limits NewProfile takes and refuses.
func TestNewProfileLimits(t *testing.T) {
	for _, limit := range []int{0, -1, 1<<20 + 1} {
		if p, err := NewProfile(limit); err == nil || p != nil {
			t.Errorf("limit %d: %v %v", limit, p, err)
		}
	}
	for _, limit := range []int{1, 1 << 20} {
		if p, err := NewProfile(limit); err != nil || p == nil {
			t.Errorf("limit %d: %v %v", limit, p, err)
		}
	}
}

// TestNilProfile checks that a profile that is not there has no spans and
// refuses to record or write a trace.
func TestNilProfile(t *testing.T) {
	var p *Profile
	if spans := p.Spans(); spans != nil {
		t.Errorf("spans of nil: %v", spans)
	}
	if err := p.WriteTrace(&strings.Builder{}); err == nil {
		t.Error("wrote a trace for a nil profile")
	}
	real, err := NewProfile(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := real.WriteTrace(nil); err == nil {
		t.Error("wrote a trace to nowhere")
	}
}

// brokenWriter is a writer that refuses every write.
type brokenWriter struct{}

// Write refuses p.
func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("disk gone") }

// TestWriteTraceReportsWriterFailure checks that a failing writer is named in
// the error and that an empty profile writes an empty event list.
func TestWriteTraceReportsWriterFailure(t *testing.T) {
	p, err := NewProfile(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.WriteTrace(brokenWriter{}); err == nil || !strings.Contains(err.Error(), "disk gone") {
		t.Errorf("failing writer: %v", err)
	}
	var out strings.Builder
	if err := p.WriteTrace(&out); err != nil || strings.TrimSpace(out.String()) != `{"traceEvents":[]}` {
		t.Errorf("empty trace: %q %v", out.String(), err)
	}
}
