//go:build darwin

package gpu

import (
	stdflag "flag"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// Observe actual command-buffer times independently of the profile splitter.
// The subprocess's only GPU jobs build one hierarchy, prepare one smoke
// grid and trace. No timestamps or native results are replaced.
func TestTraceGPUTimingBalancesObservedCommandDurations(t *testing.T) {
	if path := os.Getenv("SLAYER_METAL_TIMING_LOG"); path != "" {
		world, eye := fire(t)
		world.Volumes = world.Volumes[1:]
		picture, profile, err := SampledCached(world, eye,
			film.Pinhole{Wide: 50 * maths.Degree}, 0, 4, 3, 1)
		if err != nil || picture == nil {
			t.Fatalf("observed cached render failed: %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		total, count := 0.0, 0
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				t.Fatalf("incomplete command observation: %q", line)
			}
			start, startErr := strconv.ParseFloat(fields[0], 64)
			end, endErr := strconv.ParseFloat(fields[1], 64)
			if startErr != nil || endErr != nil || !(end >= start) ||
				math.IsInf(start, 0) || math.IsInf(end, 0) {
				t.Fatalf("invalid real command times: %q", line)
			}
			total += end - start
			count++
		}
		observed := time.Duration(total * float64(time.Second))
		reported := profile.AccelerationGPU + profile.LightingGPU + profile.TraceGPU
		t.Logf("%d actual commands: observed %v, reported %v (AS %v, lighting %v, trace %v)",
			count, observed, reported, profile.AccelerationGPU,
			profile.LightingGPU, profile.TraceGPU)
		if count < 4 || profile.AccelerationGPU <= 0 ||
			profile.LightingGPU <= 0 || profile.TraceGPU <= 0 {
			t.Fatal("observer fixture lost an actual GPU stage")
		}
		if observed != reported {
			t.Fatalf("profile does not partition the actual GPU work: observed %v, reported %v",
				observed, reported)
		}
		return
	}
	dir := t.TempDir()
	library := filepath.Join(dir, "observer.dylib")
	compile := exec.Command("cc", "-dynamiclib", "-fobjc-arc",
		"-framework", "Foundation", "-framework", "Metal",
		"testdata/command_observer.m", "-o", library)
	if out, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile command observer: %v\n%s", err, out)
	}
	args := []string{"-test.run=^TestTraceGPUTimingBalancesObservedCommandDurations$", "-test.v"}
	if flag := stdflag.Lookup("test.gocoverdir"); testing.CoverMode() != "" &&
		flag != nil && flag.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+flag.Value.String())
	}
	child := exec.Command(os.Args[0], args...)
	child.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+library,
		"SLAYER_METAL_TIMING_LOG="+filepath.Join(dir, "commands.log"))
	out, err := child.CombinedOutput()
	t.Logf("isolated actual command observations:\n%s", out)
	if err != nil {
		t.Fatalf("command observer subprocess: %v", err)
	}
}
