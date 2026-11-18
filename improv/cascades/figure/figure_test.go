package main

import (
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/cascades/flatland"
)

// captureStdout runs work and returns what it printed to standard output.
func captureStdout(t *testing.T, work func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = writer
	done := make(chan string)
	go func() {
		data, _ := io.ReadAll(reader)
		done <- string(data)
	}()
	func() {
		defer func() { os.Stdout = saved }()
		work()
	}()
	writer.Close()
	return <-done
}

// TestEncode checks the tone curve at points worked out by hand.
func TestEncode(t *testing.T) {
	cases := []struct {
		name            string
		value, exposure float64
		want            uint8
	}{
		{"zero", 0, 4, 0},
		{"negative clamps to zero", -3, 4, 0},
		{"linear toe", 0.001, 1, 3},
		{"quarter after the curve", 1.0 / 3, 1, 137},
		{"very bright saturates", 1e9, 1, 255},
	}
	for _, c := range cases {
		if got := encode(c.value, c.exposure); got != c.want {
			t.Errorf("%s: encode(%g, %g) = %d, want %d", c.name, c.value, c.exposure, got, c.want)
		}
	}
}

// TestAbsoluteDifference checks the gap and the gain on two small fields.
func TestAbsoluteDifference(t *testing.T) {
	first := flatland.Field{Side: 1, Values: []float64{1, 2, 3}}
	second := flatland.Field{Side: 1, Values: []float64{0.5, 4, 3}}
	got := absoluteDifference(first, second, 2)
	want := []float64{1, 4, 0}
	if got.Side != 1 || len(got.Values) != 3 {
		t.Fatalf("shape %d/%d", got.Side, len(got.Values))
	}
	for i := range want {
		if got.Values[i] != want[i] {
			t.Errorf("value %d = %g, want %g", i, got.Values[i], want[i])
		}
	}
}

// TestWritePicture writes two panels and reads the PNG back.
func TestWritePicture(t *testing.T) {
	bright := flatland.Field{Side: 2, Values: make([]float64, 2*2*flatland.Bands)}
	for i := range bright.Values {
		bright.Values[i] = 1e9
	}
	dark := flatland.Field{Side: 2, Values: make([]float64, 2*2*flatland.Bands)}
	path := filepath.Join(t.TempDir(), "out.png")
	if err := writePicture(path, []flatland.Field{bright, dark}, 1, 2); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 2*4+gapPixels || b.Dy() != 4 {
		t.Fatalf("size %v, want %dx4", b, 2*4+gapPixels)
	}
	check := func(x, y int, want uint32) {
		r, g, b, a := img.At(x, y).RGBA()
		if r>>8 != want || g>>8 != want || b>>8 != want || a>>8 != 255 {
			t.Errorf("pixel (%d,%d) = %d,%d,%d,%d, want grey %d", x, y, r>>8, g>>8, b>>8, a>>8, want)
		}
	}
	check(0, 0, 255)
	check(3, 3, 255)
	check(4, 0, gapLevel)
	check(4+gapPixels-1, 3, gapLevel)
	check(4+gapPixels, 0, 0)
}

// TestWritePictureBadPath checks that an unwritable path is an error.
func TestWritePictureBadPath(t *testing.T) {
	field := flatland.Field{Side: 1, Values: make([]float64, flatland.Bands)}
	path := filepath.Join(t.TempDir(), "missing", "out.png")
	if err := writePicture(path, []flatland.Field{field}, 1, 1); err == nil {
		t.Fatal("want an error for a missing directory")
	}
}

// TestBuildScene checks the three scene names.
func TestBuildScene(t *testing.T) {
	if _, _, err := buildScene(32, "nope"); err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("unknown scene error = %v", err)
	}
	grid, opaque, err := buildScene(32, "single")
	if err != nil {
		t.Fatal(err)
	}
	// The lone disc is centred at cell 16: the cell at (16,16) has its centre
	// 0.5*sqrt(2) from the middle, inside radius 1.28.
	if !opaque[16*32+16] || opaque[0] {
		t.Error("single scene mask wrong")
	}
	if got := grid.Opacity(16, 16, 1); got != opaqueOpacity {
		t.Errorf("opacity %g, want %g", got, opaqueOpacity)
	}
	if grid.Opacity(0, 0, 1) != 0 {
		t.Error("empty corner should not absorb")
	}

	grid, opaque, err = buildScene(64, "figure")
	if err != nil {
		t.Fatal(err)
	}
	// Square runs from int(0.28*64)=17 to int(0.72*64)=46. Red depth is
	// 1.5 per frame, so 1.5/64 per cell; green is 150/64.
	if got := grid.Opacity(30, 30, 0); got != 1.5/64 {
		t.Errorf("square red opacity %g, want %g", got, 1.5/64)
	}
	if got := grid.Opacity(30, 30, 1); got != 150.0/64 {
		t.Errorf("square green opacity %g, want %g", got, 150.0/64)
	}
	if opaque[30*64+30] {
		t.Error("square must not be in the opaque mask")
	}
	if !opaque[10*64+32] {
		t.Error("blocker near the top is missing from the mask")
	}
}

// TestPrintFalloff compares the printed exact value with a hand-worked one.
func TestPrintFalloff(t *testing.T) {
	const side = 16
	fields := make([]flatland.Field, 3)
	for i := range fields {
		fields[i] = flatland.Field{Side: side, Values: make([]float64, side*side*flatland.Bands)}
		fields[i].Values[(8*side+9)*flatland.Bands+1] = float64(i+1) / 4
	}
	out := captureStdout(t, func() { printFalloff(side, fields) })
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != len(falloffDistances) {
		t.Fatalf("got %d lines: %q", len(lines), out)
	}
	// Radius is clamped to 1.2 cells; at 1.5 cells, J = 4*asin(0.8)/pi.
	want := 4 * math.Asin(0.8) / math.Pi
	if !strings.HasPrefix(lines[0], "falloff at 1.5 cells: exact 1.18067, reference 0.25000, cascades 0.50000, third 0.75000") {
		t.Errorf("first line %q (exact should be %.5f)", lines[0], want)
	}
}

// TestPrintPlanAndStats checks the plan and the cost lines.
func TestPrintPlanAndStats(t *testing.T) {
	plan, err := flatland.MakePlan(4, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		printPlan(plan)
		printStats("reference", flatland.Stats{Rays: 7, Steps: 9})
	})
	if !strings.Contains(out, "cascades used: 2\n") {
		t.Errorf("missing count in %q", out)
	}
	if !strings.Contains(out, "cascade 0: probes 4x4 (spacing 1 cells), directions 4, shell [0.00, 1.50], rays 64") {
		t.Errorf("missing cascade 0 in %q", out)
	}
	if !strings.Contains(out, "shell [1.50, edge of grid]") {
		t.Errorf("missing open end in %q", out)
	}
	if !strings.Contains(out, "reference: rays traced 7, march steps 9, wall time 0.00s") {
		t.Errorf("missing stats in %q", out)
	}
}

// TestEndText checks both forms of a shell end.
func TestEndText(t *testing.T) {
	if got := endText(2.345); got != "2.35" && got != "2.34" {
		t.Errorf("endText(2.345) = %q", got)
	}
	if got := endText(math.Inf(1)); got != "edge of grid" {
		t.Errorf("endText(inf) = %q", got)
	}
}

// TestMean checks an average and the empty case.
func TestMean(t *testing.T) {
	if got := mean([]float64{1, 2, 6}); got != 3 {
		t.Errorf("mean = %g, want 3", got)
	}
	if got := mean(nil); !math.IsNaN(got) {
		t.Errorf("mean of nothing = %g, want NaN", got)
	}
}

// TestPrintErrors checks the error lines for a two by two field with one
// opaque cell.
func TestPrintErrors(t *testing.T) {
	reference := flatland.Field{Side: 2, Values: make([]float64, 4*flatland.Bands)}
	field := flatland.Field{Side: 2, Values: make([]float64, 4*flatland.Bands)}
	// Red band: reference 1 in the three open cells; the field is 1.5, 1, 1.
	// Cell 3 is opaque and must be ignored whatever it holds.
	for cell := 0; cell < 4; cell++ {
		reference.Values[cell*flatland.Bands] = 1
		field.Values[cell*flatland.Bands] = 1
	}
	field.Values[0] = 1.5
	field.Values[3*flatland.Bands] = 99
	opaque := []bool{false, false, false, true}
	out := captureStdout(t, func() { printErrors("test", field, reference, opaque) })
	// Gaps are 0.5, 0, 0: mean 0.1667, largest 0.5, band mean is 1.
	want := "test error, red relative: mean 0.1667, 99th percentile 0.5000, largest 0.5000"
	if !strings.Contains(out, want) {
		t.Errorf("missing %q in %q", want, out)
	}
	if !strings.Contains(out, "test error, red against band mean: mean 0.1667") {
		t.Errorf("missing band-mean line in %q", out)
	}
	if strings.Count(out, "\n") != 2*flatland.Bands {
		t.Errorf("want %d lines, got %q", 2*flatland.Bands, out)
	}
}

// TestRun runs the whole program on a tiny grid in each mode.
func TestRun(t *testing.T) {
	base := options{side: 16, directions: 8, workers: 1, firstLength: 1.5,
		exposure: 4, fix: true, gain: 8, zoom: 1, scene: "figure"}
	cases := []struct {
		name    string
		change  func(*options)
		wantOut []string
		width   int
	}{
		{"figure with fix", func(o *options) {}, []string{"cascades with bilinear fix", "bilinear fix error"}, 3*16 + 2*gapPixels},
		{"difference panel", func(o *options) { o.fix = false }, []string{"third panel: absolute difference, cascades minus reference, times 8"}, 3*16 + 2*gapPixels},
		{"single scene", func(o *options) { o.scene = "single"; o.zoom = 2 }, []string{"falloff at "}, 3*32 + 2*gapPixels},
	}
	for _, c := range cases {
		opt := base
		c.change(&opt)
		opt.out = filepath.Join(t.TempDir(), "out.png")
		var err error
		out := captureStdout(t, func() { err = run(opt) })
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !strings.Contains(out, "grid: 16x16 cells, 8 reference directions, 1 workers") {
			t.Errorf("%s: no header in %q", c.name, out)
		}
		for _, want := range c.wantOut {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q in %q", c.name, want, out)
			}
		}
		file, err := os.Open(opt.out)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(file)
		file.Close()
		if err != nil || cfg.Width != c.width {
			t.Errorf("%s: width %d (%v), want %d", c.name, cfg.Width, err, c.width)
		}
	}
}

// TestRunErrors checks that a bad side and a bad scene are reported.
func TestRunErrors(t *testing.T) {
	opt := options{side: 12, directions: 8, workers: 1, firstLength: 1.5, scene: "figure"}
	if err := run(opt); err == nil || !strings.Contains(err.Error(), "power of two") {
		t.Errorf("bad side error = %v", err)
	}
	opt.side, opt.scene = 16, "nope"
	if err := run(opt); err == nil || !strings.Contains(err.Error(), "unknown scene") {
		t.Errorf("bad scene error = %v", err)
	}
}
