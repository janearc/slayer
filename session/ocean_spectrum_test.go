package session

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/set"
)

func activeOceanText(t *testing.T) string {
	t.Helper()
	text := strings.Replace(authoredOceanText("1"),
		"wind: {value: 0, unit: m/s}", "wind: {value: 7, unit: m/s}", 1)
	text = strings.Replace(text, "fetch: {value: 0, unit: m}",
		"fetch: {value: 12000, unit: m}", 1)
	return text
}

func TestOceanSpectrumEstimateUsesOceantexSeaAndCarriesReproductionInputs(t *testing.T) {
	source := activeOceanText(t)
	read, err := set.Read("ocean.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	surface, _ := world.OceanSurface()
	got, err := oceanSpectrumEstimate(surface)
	if err != nil {
		t.Fatal(err)
	}
	wind, fetch, depth, toward := surface.Sea()
	sea := oceantex.Sea{Wind: wind, Fetch: fetch, Depth: depth,
		Toward: toward}
	wantPeriod := 2 * math.Pi / sea.Peak()
	if got.Model != "oceantex.Sea" || got.Source != surface.Source() ||
		got.SignificantHeightMetres != sea.Height() ||
		got.PeakPeriodSeconds != wantPeriod || !got.HasPeakPeriod ||
		got.WindMetresPerSecond != wind || got.FetchMetres != fetch ||
		got.DepthMetres != depth || got.TowardRadians != toward ||
		got.ExtentMetres != surface.Extent() || got.Samples != surface.Samples() ||
		got.Seed != surface.Seed() || got.Seconds != surface.Seconds() {
		t.Fatalf("spectrum estimate = %+v", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil || !bytes.Contains(encoded, []byte(`"has_peak_period":true`)) {
		t.Fatalf("structured report %s, error %v", encoded, err)
	}
	if line := got.line(); !strings.Contains(line, "not observation") ||
		!strings.Contains(line, "finite sampled wave field may differ") ||
		!strings.Contains(line, "peak period=") {
		t.Fatalf("human report does not label its model limits: %s", line)
	}
}

func TestOceanSpectrumCalmReportsZeroHeightAndNoPeak(t *testing.T) {
	source := authoredOceanText("1")
	read, err := set.Read("calm-ocean.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	surface, _ := world.OceanSurface()
	got, err := oceanSpectrumEstimate(surface)
	if err != nil {
		t.Fatal(err)
	}
	if got.SignificantHeightMetres != 0 || got.HasPeakPeriod ||
		got.PeakPeriodSeconds != 0 || !strings.Contains(got.line(),
		"peak period=none (calm/no spectral peak)") {
		t.Fatalf("calm estimate = %+v, line %q", got, got.line())
	}
}

func TestOceanSpectrumRefusesNonfiniteAndDegenerateInputs(t *testing.T) {
	for _, each := range []struct {
		name string
		sea  oceantex.Sea
	}{
		{"nan wind", oceantex.Sea{Wind: math.NaN(), Fetch: 1}},
		{"infinite fetch", oceantex.Sea{Wind: 1, Fetch: math.Inf(1)}},
		{"negative depth", oceantex.Sea{Wind: 1, Fetch: 1, Depth: -1}},
		{"invalid direction", oceantex.Sea{Wind: 1, Fetch: 1,
			Toward: 2*math.Pi + 0.01}},
		{"degenerate positive inputs", oceantex.Sea{Wind: math.SmallestNonzeroFloat64,
			Fetch: math.SmallestNonzeroFloat64}},
	} {
		t.Run(each.name, func(t *testing.T) {
			got, err := estimateSeaInputs(each.sea, "test", 8, 9, 1, 0)
			if err == nil {
				t.Fatalf("estimate=%+v, accepted invalid input", got)
			}
		})
	}
	if _, err := estimateSeaInputs(oceantex.Sea{Wind: 1, Fetch: 1},
		"test", 8, arid.MaxHeightFieldSamples+1, 1, 0); err == nil {
		t.Fatal("estimate accepted a sample count outside the supported field limit")
	}
}

func TestOceanSpectrumIsAvailableOnWorldOnlyAndTerrainBackedSessions(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	for _, each := range []struct {
		name      string
		terrain   bool
		wantOcean bool
	}{
		{"world-only", false, true},
		{"terrain-backed", true, true},
		{"world-only-no-ocean", false, false},
	} {
		t.Run(each.name, func(t *testing.T) {
			text := activeOceanText(t)
			if each.terrain {
				text = withSpectrumTerrain(t, text)
			}
			if !each.wantOcean {
				text = withoutSpectrumOcean(t, text)
			}
			path := writeSpectrumSet(t, text, each.terrain)
			wanted := jobFrom(t, "scene: {set: "+path+"}\nrenderer: cpu\n")
			session, err := Open(context.Background(), wanted, Stamp{})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			report, has := session.OceanSpectrum()
			if has != each.wantOcean {
				t.Fatalf("session ocean report presence = %t, want %t", has, each.wantOcean)
			}
			if has && (report.Model != oceanSpectrumModel ||
				!report.HasPeakPeriod || report.SignificantHeightMetres <= 0) {
				t.Fatalf("session report = %+v", report)
			}
			if has {
				report.Source = "changed by caller"
				again, _ := session.OceanSpectrum()
				if again.Source == report.Source {
					t.Fatal("session exposed mutable report storage")
				}
			}
		})
	}
}

func TestHillsideCLIReportsOceanOnceAndAddsNothingWithoutOcean(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	for _, each := range []struct {
		name string
		text string
		want int
	}{
		{"ocean", activeOceanText(t), 1},
		{"no-ocean", withoutSpectrumOcean(t, authoredOceanText("1")), 0},
	} {
		t.Run(each.name, func(t *testing.T) {
			path := writeSpectrumSet(t, each.text, false)
			outFile := filepath.Join(t.TempDir(), "image.png")
			var said, wrong bytes.Buffer
			code := Run([]string{"-set", path, "-w", "1", "-h", "1",
				"-n", "1", "-out", outFile}, &said, &wrong,
				Stamped("", "", "", ""))
			if code != 0 {
				t.Fatalf("Run=%d, stderr=%s, stdout=%s", code, wrong.String(), said.String())
			}
			got := strings.Count(said.String(), "ocean spectrum model estimate")
			if got != each.want {
				t.Fatalf("spectrum report count = %d, want %d; output %s", got, each.want, said.String())
			}
			if each.want == 1 && !strings.Contains(said.String(), "peak period=") {
				t.Fatalf("CLI report missing peak period: %s", said.String())
			}
		})
	}
}

func writeSpectrumSet(t *testing.T, text string, terrainBacked bool) string {
	t.Helper()
	root := t.TempDir()
	if terrainBacked {
		text = withSpectrumTerrain(t, text)
		// Keep a single terrain insertion when callers already supplied one.
	}
	path := filepath.Join(root, "scene.yaml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if terrainBacked {
		raw, err := os.ReadFile("../terrain/testdata/fixture-plain.tif")
		if err != nil {
			t.Fatal(err)
		}
		shelf := filepath.Join(root, "shelf")
		if err := os.MkdirAll(shelf, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(shelf, "fixture.tif"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		registry := fmt.Sprintf("sources:\n  - key: fixture\n    from: https://example.test/fixture.tif\n    licence: test fixture\n    sha256: %s\n    bytes: %d\n", hex.EncodeToString(sum[:]), len(raw))
		if err := os.WriteFile(filepath.Join(root, "sources.yaml"), []byte(registry), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SLAYER_SHELF", shelf)
	}
	return path
}

func withSpectrumTerrain(t *testing.T, text string) string {
	t.Helper()
	if strings.Contains(text, "  terrain:\n") {
		return text
	}
	terrain := "  terrain:\n" +
		"    from: fixture\n" +
		"    file: fixture.tif\n" +
		"    west: 1001\n" +
		"    north: 8999\n" +
		"    across: 8\n" +
		"    down: 8\n" +
		"    step: 2\n"
	if !strings.Contains(text, "  film:") {
		t.Fatal("fixture has no film insertion point")
	}
	return strings.Replace(text, "  film:", terrain+"  film:", 1)
}

func withoutSpectrumOcean(t *testing.T, text string) string {
	t.Helper()
	start := strings.Index(text, "    ocean:\n")
	end := strings.Index(text, "    materials:\n")
	if start < 0 || end <= start {
		t.Fatal("fixture has no authored ocean block to remove")
	}
	return text[:start] + text[end:]
}

func TestOceanReceiptReportsConfiguredDirectSunWithoutClaimingVisibleEffect(t *testing.T) {
	read := directSunSet(t, "")
	world, has := read.World()
	if !has {
		t.Fatal("fixture has no world")
	}
	surface, has := world.OceanSurface()
	if !has {
		t.Fatal("fixture has no ocean")
	}
	report, err := oceanSpectrumEstimate(surface)
	if err != nil {
		t.Fatal(err)
	}
	if report.DirectSunMode != "mean-plane" ||
		!strings.Contains(report.Summary(), "direct-sun=mean-plane") ||
		!strings.Contains(report.Summary(), "actual effect depends on the rays") {
		t.Fatalf("direct-sun receipt is not explicit about its scope: %+v, %q", report, report.Summary())
	}
	encoded, err := json.Marshal(report)
	if err != nil || !bytes.Contains(encoded, []byte(`"direct_sun_mode":"mean-plane"`)) {
		t.Fatalf("structured receipt = %s, error %v", encoded, err)
	}
	plainSource := activeOceanText(t)
	plain, err := set.Read("plain-ocean.yaml", []byte(plainSource))
	if err != nil {
		t.Fatal(err)
	}
	plainWorld, _ := plain.World()
	plainSurface, _ := plainWorld.OceanSurface()
	plainReport, err := oceanSpectrumEstimate(plainSurface)
	if err != nil {
		t.Fatal(err)
	}
	if plainReport.DirectSunMode != "" || strings.Contains(plainReport.Summary(), "direct-sun=") {
		t.Fatalf("unconfigured ocean gained a direct-sun receipt: %+v", plainReport)
	}
}
