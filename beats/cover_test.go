package beats_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/beats"
)

// TestSettingsCheck feeds Settings.Check one value that breaks each rule and a few that keep it.
func TestSettingsCheck(t *testing.T) {
	good := beats.Settings{Tempo: 120, Meter: 4, First: -1}
	bad := map[string]func(*beats.Settings){
		"meter zero":      func(s *beats.Settings) { s.Meter = 0 },
		"meter too large": func(s *beats.Settings) { s.Meter = 33 },
		"tempo nan":       func(s *beats.Settings) { s.Tempo = math.NaN() },
		"tempo negative":  func(s *beats.Settings) { s.Tempo = -1 },
		"tempo too slow":  func(s *beats.Settings) { s.Tempo = 59 },
		"tempo too fast":  func(s *beats.Settings) { s.Tempo = 201 },
		"first infinite":  func(s *beats.Settings) { s.First = math.Inf(1) },
	}
	for name, mutate := range bad {
		s := good
		mutate(&s)
		if err := s.Check(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	for _, tempo := range []float64{0, 60, 200} {
		s := good
		s.Tempo = tempo
		if err := s.Check(); err != nil {
			t.Errorf("tempo %v: %v", tempo, err)
		}
	}
}

// TestDetectEstimatesTheTempoWhenAskedTo checks that with tempo zero the click track at 120 beats a minute is found at half-second spacing.
func TestDetectEstimatesTheTempoWhenAskedTo(t *testing.T) {
	grid, err := beats.Detect(clicks(), 1000, beats.Settings{Tempo: 0, Meter: 4, First: -1})
	if err != nil {
		t.Fatal(err)
	}
	times := grid.Score.Beats.Times
	if len(times) < 20 {
		t.Fatalf("only %d beats found", len(times))
	}
	for i := 1; i < len(times); i++ {
		if gap := times[i] - times[i-1]; math.Abs(gap-0.5) > 0.011 {
			t.Fatalf("gap %d is %v seconds, want 0.5", i, gap)
		}
	}
}

// TestDetectWithNoOnsetsAndNoTempoFindsNoRhythm checks silence with an estimated tempo is refused for lack of evidence.
func TestDetectWithNoOnsetsAndNoTempoFindsNoRhythm(t *testing.T) {
	_, err := beats.Detect(make([]float32, 4000), 1000, beats.Settings{Tempo: 0, Meter: 4, First: -1})
	if err == nil || !strings.Contains(err.Error(), "no rhythmic onset evidence") {
		t.Fatalf("err=%v", err)
	}
}

// TestDetectTrimsToTheFirstBeat checks a First setting drops every beat before it and a First past the music leaves too few beats.
func TestDetectTrimsToTheFirstBeat(t *testing.T) {
	grid, err := beats.Detect(clicks(), 1000, beats.Settings{Tempo: 120, Meter: 4, First: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range grid.Score.Beats.Times {
		if at < 3-0.005 {
			t.Fatalf("beat at %v is before the chosen first beat 3", at)
		}
	}
	if grid.Score.Beats.Times[0] > 3.011 {
		t.Fatalf("first kept beat %v is not near 3", grid.Score.Beats.Times[0])
	}
	if _, err := beats.Detect(clicks(), 1000, beats.Settings{Tempo: 120, Meter: 4, First: 11.9}); err == nil || !strings.Contains(err.Error(), "two or more beats") {
		t.Fatalf("first past the music: %v", err)
	}
}

// TestDetectRefusesBadSound feeds Detect each kind of bad recording and bad setting.
func TestDetectRefusesBadSound(t *testing.T) {
	settings := beats.Settings{Tempo: 120, Meter: 4, First: -1}
	loud := clicks()
	loud[10] = 1.5
	notANumber := clicks()
	notANumber[10] = float32(math.NaN())
	cases := []struct {
		name    string
		samples []float32
		rate    int
		with    beats.Settings
	}{
		{"rate under 100", clicks(), 50, settings},
		{"rate not a multiple of 100", clicks(), 1050, settings},
		{"odd sample count", clicks()[:len(clicks())-1], 1000, settings},
		{"under a second", make([]float32, 1000), 1000, settings},
		{"sample over one", loud, 1000, settings},
		{"sample not a number", notANumber, 1000, settings},
		{"bad settings", clicks(), 1000, beats.Settings{Tempo: 120, Meter: 0}},
	}
	for _, tc := range cases {
		if _, err := beats.Detect(tc.samples, tc.rate, tc.with); err == nil {
			t.Errorf("%s: accepted", tc.name)
		}
	}
}

// TestGridCheckRefusals feeds Grid.Check a grid that breaks each rule in turn.
func TestGridCheckRefusals(t *testing.T) {
	good := func() *beats.Grid {
		return &beats.Grid{Score: beats.Score{Duration: 4, Meter: 4, Status: "verified",
			Beats: beats.Pulse{Times: []float64{1, 2, 3}, Measured: []int{1, 0, 1}}}}
	}
	if err := good().Check(); err != nil {
		t.Fatalf("a good grid: %v", err)
	}
	var none *beats.Grid
	if err := none.Check(); err == nil || !strings.Contains(err.Error(), "no grid") {
		t.Errorf("nil grid: %v", err)
	}
	cases := map[string]func(*beats.Grid){
		"duration zero":    func(g *beats.Grid) { g.Score.Duration = 0 },
		"duration nan":     func(g *beats.Grid) { g.Score.Duration = math.NaN() },
		"meter zero":       func(g *beats.Grid) { g.Score.Meter = 0 },
		"status unknown":   func(g *beats.Grid) { g.Score.Status = "certain" },
		"one beat":         func(g *beats.Grid) { g.Score.Beats = beats.Pulse{Times: []float64{1}, Measured: []int{1}} },
		"marks too few":    func(g *beats.Grid) { g.Score.Beats.Measured = []int{1, 1} },
		"negative time":    func(g *beats.Grid) { g.Score.Beats.Times[0] = -1 },
		"time not number":  func(g *beats.Grid) { g.Score.Beats.Times[1] = math.NaN() },
		"reversed times":   func(g *beats.Grid) { g.Score.Beats.Times[2] = 1.5; g.Score.Beats.Times[1] = 2 },
		"time at duration": func(g *beats.Grid) { g.Score.Beats.Times[2] = 4 },
		"mark of two":      func(g *beats.Grid) { g.Score.Beats.Measured[1] = 2 },
	}
	for name, mutate := range cases {
		g := good()
		mutate(g)
		if err := g.Check(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// TestReadYAMLAndJSON checks a grid file of either kind is read and checked, and a missing, malformed or invalid one is refused.
func TestReadYAMLAndJSON(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	yaml := "score:\n  audio: a.ogg\n  sha256: \"\"\n  duration: 4\n  meter: 4\n  status: estimated\n  method: test\n  beats:\n    times: [1, 1.5]\n    measured: [1, 0]\n"
	grid, err := beats.Read(write("good.yaml", yaml))
	if err != nil {
		t.Fatal(err)
	}
	if grid.Score.Meter != 4 || grid.Score.Duration != 4 || len(grid.Score.Beats.Times) != 2 || grid.Score.Beats.Measured[1] != 0 {
		t.Fatalf("grid=%+v", grid.Score)
	}
	js := `{"score":{"audio":"a.ogg","duration":4,"meter":3,"status":"verified","beats":{"times":[1,2],"measured":[1,1]}}}`
	grid, err = beats.Read(write("good.json", js))
	if err != nil || grid.Score.Meter != 3 || grid.Score.Status != "verified" {
		t.Fatalf("json grid=%+v err=%v", grid, err)
	}
	if _, err := beats.Read(filepath.Join(dir, "absent.yaml")); err == nil {
		t.Error("missing file accepted")
	}
	if _, err := beats.Read(write("broken.yaml", "score: [unclosed")); err == nil {
		t.Error("malformed file accepted")
	}
	if _, err := beats.Read(write("one.yaml", strings.Replace(yaml, "[1, 1.5]", "[1]", 1))); err == nil {
		t.Error("a grid that fails its check was accepted")
	}
}

// TestCheckAudio checks a grid is bound to audio whose SHA-256 it names, and refused for any other bytes or missing digest.
func TestCheckAudio(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audio.bin")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The SHA-256 of "abc" is a published test vector.
	const abc = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	grid := &beats.Grid{Score: beats.Score{SHA256: abc}}
	if err := grid.CheckAudio(path); err != nil {
		t.Fatalf("matching audio: %v", err)
	}
	grid.Score.SHA256 = strings.Repeat("0", 64)
	if err := grid.CheckAudio(path); err == nil || !strings.Contains(err.Error(), "different audio") {
		t.Errorf("other audio: %v", err)
	}
	grid.Score.SHA256 = ""
	if err := grid.CheckAudio(path); err == nil || !strings.Contains(err.Error(), "SHA256") {
		t.Errorf("no digest: %v", err)
	}
	var none *beats.Grid
	if err := none.CheckAudio(path); err == nil {
		t.Error("nil grid accepted")
	}
	grid.Score.SHA256 = abc
	if err := grid.CheckAudio(filepath.Join(dir, "absent.bin")); err == nil {
		t.Error("missing audio accepted")
	}
	if err := grid.CheckAudio(dir); err == nil {
		t.Error("a directory read as audio was accepted")
	}
}
