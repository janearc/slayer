package omnipositor

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

// TestLoadAssetsRefusals checks each setting that the loader refuses, and
// that the error names the setting.
func TestLoadAssetsRefusals(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(s *settings.Settings)
		want   string
	}{
		{"zero width", func(s *settings.Settings) { s.Frame.Width = 0 }, "invalid or excessive dimensions"},
		{"zero height", func(s *settings.Settings) { s.Frame.Height = 0 }, "invalid or excessive dimensions"},
		{"width past the limit", func(s *settings.Settings) { s.Frame.Width = math.MaxInt / 2 }, "invalid or excessive dimensions"},
		{"no cache", func(s *settings.Settings) { s.Output.MemoryCacheBytes = 0 }, "memory_cache_bytes"},
		{"blank plate path", func(s *settings.Settings) { s.Passes.Nodisc = "  " }, "invalid path"},
		{"empty first flood name", func(s *settings.Settings) { s.Night.FloodPasses[0] = "" }, "empty renderer pass name"},
		{"flood name is the overhead name", func(s *settings.Settings) { s.Night.FloodPasses[0] = "overhead" }, "duplicate renderer pass name"},
		{"both flood names the same", func(s *settings.Settings) { s.Night.FloodPasses = [2]string{"same", "same"} }, "duplicate renderer pass name"},
		{"missing back pass", func(s *settings.Settings) {
			s.Passes.BackLeft = filepath.Join(filepath.Dir(s.Passes.BackLeft), "gone.pfm")
		}, "passes.back_left"},
		{"spot name taken", func(s *settings.Settings) {
			s.Night.FloodPasses[0] = "spot-left-0"
			s.Passes.SpotLeft0 = &settings.OptionalPath{Path: s.Passes.Nodisc}
		}, "passes.spot_left_0: duplicate"},
		{"missing spot pass", func(s *settings.Settings) {
			s.Passes.SpotRight2 = &settings.OptionalPath{Path: filepath.Join(filepath.Dir(s.Passes.Nodisc), "gone.pfm")}
		}, "passes.spot_right_2"},
		{"missing moon reference", func(s *settings.Settings) {
			s.MoonRef = &settings.OptionalPath{Path: filepath.Join(filepath.Dir(s.Passes.Nodisc), "gone.pfm")}
		}, "moon_ref"},
		{"blank fire directory", func(s *settings.Settings) { s.Fire.Frames = &settings.OptionalPath{Path: " "} }, "fire.steady: empty directory"},
		{"missing fire directory", func(s *settings.Settings) {
			s.Fire.Frames = &settings.OptionalPath{Path: filepath.Join(filepath.Dir(s.Passes.Nodisc), "nowhere")}
		}, "fire.steady"},
		{"swell without steady", func(s *settings.Settings) { s.Fire.Swell = &settings.OptionalPath{Path: assetFireDir(t, 1)} }, "steady directory required"},
	} {
		for _, loader := range []struct {
			name string
			load func(settings.Settings) (*LoadedAssets, error)
		}{{"LoadAssets", LoadAssets}, {"LoadStaticAssets", LoadStaticAssets}} {
			s := assetSettings(t)
			c.change(&s)
			if got, err := loader.load(s); err == nil || got != nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("%s with %s: %v", c.name, loader.name, err)
			}
		}
	}
}

// TestLoadAssetsSharesAFileUsedTwice checks that two settings that name one
// file read it once and carry the same digest.
func TestLoadAssetsSharesAFileUsedTwice(t *testing.T) {
	s := assetSettings(t)
	s.Passes.Disc = s.Passes.Nodisc
	a, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	ids := a.Identities()
	if ids["passes.disc"].SHA256 == "" || ids["passes.disc"] != ids["passes.nodisc"] {
		t.Errorf("shared file identities differ: %+v %+v", ids["passes.disc"], ids["passes.nodisc"])
	}
}

// TestReadAssetRefusals checks a missing file, expected dimensions that are
// not usable, and a file of the wrong size.
func TestReadAssetRefusals(t *testing.T) {
	s := assetSettings(t)
	if _, _, err := readAsset("named", filepath.Join(t.TempDir(), "gone.pfm"), 2, 2); err == nil || !strings.Contains(err.Error(), "named") {
		t.Errorf("missing file: %v", err)
	}
	for _, size := range [][2]int{{0, 2}, {2, 0}, {-1, 2}, {math.MaxInt / 2, 2}} {
		if _, _, err := readAsset("named", s.Passes.Nodisc, size[0], size[1]); err == nil || !strings.Contains(err.Error(), "invalid expected dimensions") {
			t.Errorf("size %v: %v", size, err)
		}
	}
	image, id, err := readAsset("named", s.Passes.Nodisc, 2, 2)
	if err != nil || image.Width != 2 || id.Size < 48 || id.Path != s.Passes.Nodisc {
		t.Errorf("good file: %+v %+v %v", image, id, err)
	}
}

// fakeInfo is file information whose system part is whatever a test sets.
type fakeInfo struct {
	os.FileInfo
	sys any
}

// Sys returns the system part the test chose.
func (f fakeInfo) Sys() any { return f.sys }

// TestSameChangeTime checks that the change time of one file matches itself,
// and that information with no change time field is never the same.
func TestSameChangeTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !sameChangeTime(info, info) {
		t.Error("a file's change time differs from itself")
	}
	if sameChangeTime(fakeInfo{info, nil}, info) || sameChangeTime(info, fakeInfo{info, 5}) {
		t.Error("information with no system part matched")
	}
	if sameChangeTime(fakeInfo{info, &struct{ Other int }{1}}, fakeInfo{info, &struct{ Other int }{1}}) {
		t.Error("information with no change time field matched")
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.Chmod(path, 0640); err != nil {
		t.Fatal(err)
	}
	changed, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if sameChangeTime(info, changed) {
		t.Error("a change was not seen in the change time")
	}
}

// TestFireFrameCountRefusals checks a directory that cannot be read, a path
// that is a file, a frame that is not a regular file, and a gap in the
// numbering, and the count for a good directory.
func TestFireFrameCountRefusals(t *testing.T) {
	dir := t.TempDir()
	if _, err := fireFrameCount(filepath.Join(dir, "gone")); err == nil {
		t.Error("counted a missing directory")
	}
	file := filepath.Join(dir, "plain")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := fireFrameCount(file); err == nil {
		t.Error("counted a file")
	}
	linked := t.TempDir()
	if err := os.Symlink(file, filepath.Join(linked, "000.pfm")); err != nil {
		t.Fatal(err)
	}
	if _, err := fireFrameCount(linked); err == nil || !strings.Contains(err.Error(), "not a regular PFM file") {
		t.Errorf("link as frame: %v", err)
	}
	gap := assetFireDir(t, 3)
	if err := os.Remove(filepath.Join(gap, "001.pfm")); err != nil {
		t.Fatal(err)
	}
	if _, err := fireFrameCount(gap); err == nil || !strings.Contains(err.Error(), "missing 001.pfm") {
		t.Errorf("gap: %v", err)
	}
	good := assetFireDir(t, 3)
	if err := os.WriteFile(filepath.Join(good, "notes.txt"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if n, err := fireFrameCount(good); err != nil || n != 3 {
		t.Errorf("good directory: %d %v", n, err)
	}
}

// TestLoadFireRefusals checks the fire requests that are refused before any
// frame is read, and a request for no fire.
func TestLoadFireRefusals(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	a, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, Cues: cues.Frame{Fire: 1, FireSince: []float64{0}}}
	var noContext context.Context
	if _, err := a.LoadFire(noContext, request); err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Errorf("nil context: %v", err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.LoadFire(ended, request); err == nil {
		t.Error("loaded fire for an ended context")
	}
	nan := request
	nan.Cues.Fire = math.NaN()
	if _, err := a.LoadFire(context.Background(), nan); err == nil || !strings.Contains(err.Error(), "nonfinite") {
		t.Errorf("NaN level: %v", err)
	}
	dark := request
	dark.Cues.Fire = 0
	frames, err := a.LoadFire(context.Background(), dark)
	if err != nil || len(frames[0].Images) != 0 || frames[0].Count != 3 {
		t.Errorf("no fire: %+v %v", frames[0], err)
	}
	lit, err := a.LoadFire(context.Background(), request)
	if err != nil || len(lit[0].Images) == 0 {
		t.Errorf("fire: %+v %v", lit[0], err)
	}
	tight := lights.FireByteBudget(1, 1)
	if _, err := a.loadFireBudget(context.Background(), request, false, 100, tight); err == nil {
		t.Error("loaded fire past a one byte budget")
	}
	if _, err := a.loadFireBudget(context.Background(), request, false, 0, 1<<20); err == nil {
		t.Error("loaded fire past a limit of no frames")
	}
}

// TestLoadFireStopsWhenFileChanges checks that a fire frame that is replaced
// after it was first loaded is refused, whether the cache holds it or not.
func TestLoadFireStopsWhenFileChanges(t *testing.T) {
	s := assetSettings(t)
	dir := assetFireDir(t, 2)
	s.Fire.Frames = &settings.OptionalPath{Path: dir}
	s.Output.MemoryCacheBytes = 1
	a, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, Cues: cues.Frame{Fire: 1, FireSince: []float64{0}}}
	if _, err := a.LoadFire(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if a.CacheBytes() != 0 {
		t.Fatalf("a frame larger than the cache was cached: %d", a.CacheBytes())
	}
	other := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
	other.Light[5] = 9
	if err := lights.Write(filepath.Join(dir, "000.pfm"), other, -1); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoadFire(context.Background(), request); err == nil || !strings.Contains(err.Error(), "identity changed since first load") {
		t.Errorf("replaced frame: %v", err)
	}
}

// TestNewLightStageFromLoadedAssetsRefusesBadConfig checks that a lighting
// config that the renderer refuses stops the stage from being made, for the
// private and the shared loaders.
func TestNewLightStageFromLoadedAssetsRefusesBadConfig(t *testing.T) {
	for name, load := range map[string]func(settings.Settings) (*LoadedAssets, error){"shared": LoadAssets, "private": LoadStaticAssets} {
		a, err := load(assetSettings(t))
		if err != nil {
			t.Fatal(err)
		}
		bad := lights.DefaultConfig()
		bad.Focal = 0
		if stage, err := a.NewLightStage(bad); err == nil || stage != nil {
			t.Errorf("%s: %v %v", name, stage, err)
		}
	}
}
