package manufacture

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
)

const assemblyProbeMode = "SLAYER_BOUNDED_ASSEMBLY_PROBE"

// Small fixed recipes must terminate. Run the probes in child processes so
// an inverted loop cannot keep allocating after the parent records a failure.
// This is a hang guard, not an animation or rendering performance budget.
func TestSmallManufacturedAssembliesFinish(t *testing.T) {
	if mode := os.Getenv(assemblyProbeMode); mode != "" {
		probeSmallAssembly(t, mode)
		return
	}
	for _, mode := range []string{"mirror", "foliage"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestSmallManufacturedAssembliesFinish$"}
			if dir := flag.Lookup("test.gocoverdir"); testing.CoverMode() != "" &&
				dir != nil && dir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+dir.Value.String())
			}
			child := exec.CommandContext(ctx, os.Args[0], args...)
			child.Env = append(os.Environ(), assemblyProbeMode+"="+mode)
			out, err := child.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("small %s assembly did not terminate: %v", mode, ctx.Err())
			}
			if err != nil {
				t.Fatalf("small %s assembly: %v\n%s", mode, err, out)
			}
		})
	}
}

// probeSmallAssembly uses real valid geometry, with no replacement factory.
func probeSmallAssembly(t *testing.T, mode string) {
	t.Helper()
	switch mode {
	case "mirror":
		// Hex centres inside the four-span usable radius occupy five
		// rows of 3,4,5,4,3 tiles. Each closed hex has 24 triangles.
		skin := (mirrorDish{radius: 1, depth: .02, segmented: true}).panels()
		if len(skin) != 19*24 {
			t.Fatalf("small segmented mirror: %d faces, want 456", len(skin))
		}
	case "foliage":
		trunk, err := newTube(measure.Metres(1),
			measure.Metres(.1), measure.Metres(.08))
		if err != nil {
			t.Fatal(err)
		}
		axis, err := newTube(measure.Metres(.4),
			measure.Metres(.01), measure.Metres(.001))
		if err != nil {
			t.Fatal(err)
		}
		scaffold := good{
			parts: []part{
				{of: body, shape: trunk, fine: cut{around: 6}},
				{of: leaf, shape: axis, terminal: true},
			},
			dressed: looks{
				body: plain("wood", .2, .1, .05),
				leaf: plain("leaves", .1, .3, .1),
			},
		}
		made, err := recursiveFoliage(scaffold, Sow(11), 1, "quick")
		if err != nil {
			t.Fatal(err)
		}
		if group := made.groups[leaf]; group == nil ||
			len(group.Members()) != 1 {
			t.Fatal("one terminal cell did not produce one foliage member")
		}
	default:
		t.Fatalf("unknown assembly probe %q", mode)
	}
}
