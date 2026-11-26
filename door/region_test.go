package door

import (
	"context"
	"image"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
)

func TestDrawRegionOverrideAppearsInDryRun(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", jobLine+
		"scene: {smoke: 1}\nfilm: {width: 20, height: 20, region: {left: 0, top: 0, width: 1, height: 1}}\n")
	env := envIn(t, context.Background(), directory)
	said := drew(t, env, path, "--dry-run", "--region", "2,3,4,5")
	for _, line := range []string{"region:", "left: 2", "top: 3", "width: 4", "height: 5"} {
		if !strings.Contains(said, line) {
			t.Errorf("dry run lacks %q: %s", line, said)
		}
	}
}

func TestDrawAcceptsSixtyFourRaysForWholeAndRegion(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	for _, extra := range [][]string{nil, {"--region", "0,0,1,1"}} {
		args := append([]string{"draw", path, "--dry-run", "--rays", "64"}, extra...)
		got := running(Tool(env), args...)
		if got.code != nothing || !strings.Contains(got.said, "rays: 64") {
			t.Errorf("%v: %d %s%s", extra, got.code, got.said, got.wrongSaid)
		}
	}
}

// Anti: neither whole pictures nor regions accept 65 rays per side.
func TestDrawRejectsSixtyFiveRaysForWholeAndRegion(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	for _, extra := range [][]string{nil, {"--region", "0,0,1,1"}} {
		args := append([]string{"draw", path, "--dry-run", "--rays", "65"}, extra...)
		got := running(Tool(env), args...)
		if got.code == nothing {
			t.Errorf("accepted 65 rays with %v", extra)
		}
	}
}

func TestRegionDevelopmentUsesTheEffectiveTone(t *testing.T) {
	directory := t.TempDir()
	for _, each := range []struct {
		after    string
		fallback float64
		refuse   bool
	}{
		{"", 0, false},
		{"", 2, true},
		{"print: {knee: 0.8, top: 4, bloom: 0}\n", 2, false},
		{"print: {knee: 0.8, top: 4, bloom: 2}\n", 0, true},
		{"stock: {name: lantern}\n", 0, true},
	} {
		path := written(t, directory, "still.yaml", stillJob(t, each.after))
		wanted, err := job.Read(path)
		if err != nil {
			t.Fatal(err)
		}
		wanted, err = wanted.With(job.Region(image.Rect(0, 0, 1, 1)))
		if err != nil {
			t.Fatal(err)
		}
		err = regionDevelopment(wanted, colour.Tone{Bloom: each.fallback})
		if (err != nil) != each.refuse {
			t.Errorf("%q fallback %v: %v", each.after, each.fallback, err)
		}
	}
}

// Anti: malformed and overflowing region values are refused before a draw.
func TestDrawRegionOverrideRefusesMalformedValues(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	for _, value := range []string{"", "0,,1,1", "0,0,1", "0,0,1,1,1", "a,0,1,1", "-1,0,1,1", "0,0,0,1", "1,0,9223372036854775807,1", "0,0,999999999999999999999999999,1"} {
		got := running(Tool(env), "draw", path, "--dry-run", "--region="+value)
		if got.code == nothing || !strings.Contains(got.wrongSaid, "region") {
			t.Errorf("%q gave %d: %s", value, got.code, got.wrongSaid)
		}
	}
}

// Anti: another verb can use its own flags without querying draw's region.
func TestWatchWithoutRegionStillReadsItsFlags(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "watch", path, "--dry-run", "--width", "4")
	if got.code != nothing {
		t.Errorf("watch refused its width: %s", got.wrongSaid)
	}
}

// A region is not published until its development and overlays preserve
// the full picture's coordinates and neighbouring light.
func TestDrawRefusesRegionWithBloomBeforeWritingAnOutput(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, "print: {knee: 0.8, top: 4, bloom: 2}\n"))
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "draw", path, "--region", "0,0,1,1")
	if got.code == nothing || !strings.Contains(got.wrongSaid, "region") {
		t.Fatalf("region draw gave %d: %s", got.code, got.wrongSaid)
	}
	if found := files(t, directory); len(found) != 1 || found[0] != "still.yaml" {
		t.Fatalf("region draw wrote %v", found)
	}
}

// Anti: region refusal does not reject a whole-picture dry run.
func TestWholePictureDryRunSurvivesRegionGuard(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "draw", path, "--dry-run")
	if got.code != nothing {
		t.Fatalf("whole-picture dry run failed: %s", got.wrongSaid)
	}
}

func TestWatchRefusesAnAuthoredRegion(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", jobLine+
		"scene: {smoke: 1}\nfilm: {width: 10, height: 10, region: {left: 1, top: 1, width: 2, height: 2}}\n")
	_, err := reading(nil, false)(path)
	if err == nil || !strings.Contains(err.Error(), "region") {
		t.Fatalf("watch read a region: %v", err)
	}
}

func TestDevelopRefusesAnAuthoredRegion(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", jobLine+
		"scene: {smoke: 1}\nfilm: {width: 10, height: 10, region: {left: 1, top: 1, width: 2, height: 2}}\n")
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "develop", path, "missing.pfm")
	if got.code == nothing || !strings.Contains(got.wrongSaid, "region") {
		t.Fatalf("develop gave %d: %s", got.code, got.wrongSaid)
	}
}
