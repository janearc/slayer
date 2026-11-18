package searching

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// songBytes stands in for the encoded audio the score is bound to.
var songBytes = []byte("not really music, only bytes to hash")

// shootFiles writes a song, a score of eighty half-second beats and a plan
// in a new directory, and returns the plan's path. The change function may
// edit the plan text, which has one line per setting, before it is written.
func shootFiles(t *testing.T, change func(plan string) string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "song.bin"), songBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	var times, marks []string
	for index := range 40 {
		times = append(times, fmt.Sprint(float64(index)/2))
		marks = append(marks, "1")
	}
	score := fmt.Sprintf(`score:
  audio: song.bin
  sha256: %x
  duration: 20
  meter: 4
  status: estimated
  method: test
  beats:
    times: [%s]
    measured: [%s]
`, sha256.Sum256(songBytes), strings.Join(times, ", "), strings.Join(marks, ", "))
	if err := os.WriteFile(filepath.Join(dir, "score.yaml"), []byte(score), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := `name: test shoot
music: song.bin
score: score.yaml
width: 160
height: 90
fps: 4
rays: 1
lock: "2.1"
radio-one: "3.1"
radio-two: "4.1"
reveal: "5.1"
coda-bars: 2
search-cut-bars: 2
beat-motion: [1, 0.5, 0.25, 0.125]
beat-two-accents: ["1.2", "2.2"]
search-days: 2
shared-azimuth-deg: 25
shared-altitude-deg: 55
optical-diameter-metres: 38
optical-aperture-metres: 8
array-count: 6
exposure: 1.5
bloom: 0.65
`
	if change != nil {
		plan = change(plan)
	}
	path := filepath.Join(dir, "shoot.yaml")
	if err := os.WriteFile(path, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// replaceLine swaps one setting's line in the plan text.
func replaceLine(plan, key, line string) string {
	lines := strings.Split(plan, "\n")
	for index, each := range lines {
		if strings.HasPrefix(each, key+":") {
			lines[index] = line
			return strings.Join(lines, "\n")
		}
	}
	panic("no setting " + key)
}

// TestReadBindsAPlanToItsScore checks every cue the plan reads, worked from
// half-second beats: bar b beat k falls at half a second a beat from the
// start, so bar two beat one is two seconds in.
func TestReadBindsAPlanToItsScore(t *testing.T) {
	path := shootFiles(t, nil)
	edit, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if edit.Lock != 2 || edit.RadioOne != 4 || edit.RadioTwo != 6 || edit.Reveal != 8 {
		t.Errorf("cues %g %g %g %g", edit.Lock, edit.RadioOne, edit.RadioTwo, edit.Reveal)
	}
	// Forty beats at the end make ten bars; two coda bars start at bar nine.
	if edit.Coda != 16 {
		t.Errorf("coda at %g", edit.Coda)
	}
	if len(edit.Accents) != 2 || edit.Accents[0] != 0.5 || edit.Accents[1] != 2.5 {
		t.Errorf("accents %v", edit.Accents)
	}
	if edit.Plan.Name != "test shoot" || edit.Plan.FPS != 4 || edit.Plan.ArrayCount != 6 {
		t.Errorf("plan %+v", edit.Plan)
	}
	if want := filepath.Join(filepath.Dir(path), "song.bin"); edit.Plan.Music != want {
		t.Errorf("music %q, wanted %q", edit.Plan.Music, want)
	}
	if edit.Frames() != 80 {
		t.Errorf("%d frames for twenty seconds at four a second", edit.Frames())
	}
}

// TestReadKeepsAbsolutePaths checks that a plan naming its files by full
// path is not rewritten beside itself.
func TestReadKeepsAbsolutePaths(t *testing.T) {
	elsewhere := filepath.Join(filepath.Dir(shootFiles(t, nil)), "song.bin")
	path := shootFiles(t, func(plan string) string {
		return replaceLine(plan, "music", "music: "+elsewhere)
	})
	edit, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if edit.Plan.Music != elsewhere {
		t.Errorf("music %q, wanted %q", edit.Plan.Music, elsewhere)
	}
}

// TestReadRefusals gives Read each kind of bad plan, score or song.
func TestReadRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(string) string
		want   string
	}{
		{"width too small", func(p string) string { return replaceLine(p, "width", "width: 100") }, "invalid or excessive"},
		{"cue past the end", func(p string) string { return replaceLine(p, "lock", `lock: "40.1"`) }, "invalid or out of order"},
		{"cue before the last", func(p string) string { return replaceLine(p, "radio-one", `radio-one: "1.1"`) }, "invalid or out of order"},
		{"cue not a position", func(p string) string { return replaceLine(p, "reveal", `reveal: "five"`) }, "invalid or out of order"},
		{"coda over the reveal", func(p string) string { return replaceLine(p, "coda-bars", "coda-bars: 9") }, "coda overlaps reveal"},
		{"accent not a position", func(p string) string { return replaceLine(p, "beat-two-accents", `beat-two-accents: ["1.9"]`) }, "invalid cue"},
		{"score missing", func(p string) string { return replaceLine(p, "score", "score: none.yaml") }, "no such file"},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			edit, err := Read(shootFiles(t, each.change))
			if err == nil || edit != nil || !strings.Contains(err.Error(), each.want) {
				t.Fatalf("got %v, %v; wanted an error containing %q", edit, err, each.want)
			}
		})
	}
	if _, err := Read(filepath.Join(t.TempDir(), "none.yaml")); err == nil {
		t.Error("a missing plan was read")
	}
	broken := filepath.Join(t.TempDir(), "broken.yaml")
	if err := os.WriteFile(broken, []byte("name: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(broken); err == nil {
		t.Error("a broken plan was read")
	}
}

// TestReadRefusesAScoreForOtherAudio changes the song after the score was
// bound to it and expects the grid to be refused.
func TestReadRefusesAScoreForOtherAudio(t *testing.T) {
	path := shootFiles(t, nil)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "song.bin"), []byte("another song"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil || !strings.Contains(err.Error(), "different audio") {
		t.Fatalf("error %v", err)
	}
}

// TestReadRefusesATripleMeter rewrites the score with three beats a bar.
func TestReadRefusesATripleMeter(t *testing.T) {
	path := shootFiles(t, nil)
	score := filepath.Join(filepath.Dir(path), "score.yaml")
	raw, err := os.ReadFile(score)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(score, []byte(strings.Replace(string(raw), "meter: 4", "meter: 3", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil || !strings.Contains(err.Error(), "four-beat bars") {
		t.Fatalf("error %v", err)
	}
}

// TestReadRefusesABadScore breaks the score's own checks.
func TestReadRefusesABadScore(t *testing.T) {
	path := shootFiles(t, nil)
	score := filepath.Join(filepath.Dir(path), "score.yaml")
	raw, err := os.ReadFile(score)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(score, []byte(strings.Replace(string(raw), "status: estimated", "status: guessed", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("error %v", err)
	}
}

// goodPlan is a plan that passes its own check.
func goodPlan() Plan {
	return Plan{Name: "n", Music: "m", Score: "s", Width: 160, Height: 90, FPS: 6,
		Rays: 1, SearchCutBars: 2, CodaBars: 2, ArrayCount: 4,
		BeatMotion: []float64{1, 0.5, 0.25, 0.125}, SearchDays: 1, Azimuth: 25,
		Altitude: 55, Diameter: 38, Aperture: 8, Exposure: 1.5, Bloom: 0.5}
}

// TestPlanCheckAcceptsTheEdges checks values on the inside of each limit.
func TestPlanCheckAcceptsTheEdges(t *testing.T) {
	edge := goodPlan()
	edge.Width, edge.Height, edge.FPS, edge.Rays, edge.ArrayCount = 480, 90, 24, 2, 40
	edge.Altitude, edge.Bloom, edge.SearchDays = 85, 0, 0
	if err := edge.check(); err != nil {
		t.Fatalf("a plan on every limit was refused: %v", err)
	}
	if err := goodPlan().check(); err != nil {
		t.Fatal(err)
	}
}

// TestPlanCheckRefusals changes one setting at a time past its limit.
func TestPlanCheckRefusals(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	cases := map[string]func(*Plan){
		"no name":             func(p *Plan) { p.Name = "" },
		"no music":            func(p *Plan) { p.Music = "" },
		"no score":            func(p *Plan) { p.Score = "" },
		"narrow":              func(p *Plan) { p.Width = 159 },
		"wide":                func(p *Plan) { p.Width = 2049 },
		"short":               func(p *Plan) { p.Height = 89 },
		"tall":                func(p *Plan) { p.Height = 1153 },
		"no frames":           func(p *Plan) { p.FPS = 0 },
		"too many frames":     func(p *Plan) { p.FPS = 25 },
		"no rays":             func(p *Plan) { p.Rays = 0 },
		"three rays":          func(p *Plan) { p.Rays = 5 },
		"no search bars":      func(p *Plan) { p.SearchCutBars = 0 },
		"no coda bars":        func(p *Plan) { p.CodaBars = 0 },
		"no array":            func(p *Plan) { p.ArrayCount = 0 },
		"large array":         func(p *Plan) { p.ArrayCount = 41 },
		"three motions":       func(p *Plan) { p.BeatMotion = []float64{1, 1, 1} },
		"nan days":            func(p *Plan) { p.SearchDays = nan },
		"infinite azimuth":    func(p *Plan) { p.Azimuth = inf },
		"negative altitude":   func(p *Plan) { p.Altitude = -1 },
		"nan diameter":        func(p *Plan) { p.Diameter = nan },
		"infinite aperture":   func(p *Plan) { p.Aperture = inf },
		"infinite exposure":   func(p *Plan) { p.Exposure = inf },
		"negative bloom":      func(p *Plan) { p.Bloom = -0.1 },
		"negative motion":     func(p *Plan) { p.BeatMotion[3] = -1 },
		"nan motion":          func(p *Plan) { p.BeatMotion[2] = nan },
		"no exposure":         func(p *Plan) { p.Exposure = 0 },
		"no diameter":         func(p *Plan) { p.Diameter = 0 },
		"no aperture":         func(p *Plan) { p.Aperture = 0 },
		"aperture too wide":   func(p *Plan) { p.Aperture = 19 },
		"sky-high altitude":   func(p *Plan) { p.Altitude = 85.5 },
		"no first-beat speed": func(p *Plan) { p.BeatMotion[0] = 0 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			plan := goodPlan()
			plan.BeatMotion = append([]float64(nil), plan.BeatMotion...)
			change(&plan)
			if err := plan.check(); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}
