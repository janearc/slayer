package main

import (
	"context"
	stdflag "flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/janearc/slayer/scene/thing"
)

// The actual checked 8-by-8 CPU fixture completes without unbounded assembly.
// Keep loop-fault probes in an owned child so failures cannot grow the suite.
func TestFuelBedPreparationTerminatesForCheckedInput(t *testing.T) {
	if os.Getenv("SLAYER_FUEL_PREPARATION_PROBE") == "1" {
		// This is test containment, not a renderer memory budget. Stop a
		// growing faulty loop before the outer runner's resource watchdog.
		var initial runtime.MemStats
		runtime.ReadMemStats(&initial)
		go func() {
			for range time.Tick(10 * time.Millisecond) {
				var current runtime.MemStats
				runtime.ReadMemStats(&current)
				if current.HeapAlloc > initial.HeapAlloc+(32<<20) {
					fmt.Fprintln(os.Stderr, "checked fuel-bed preparation grew its heap by more than 32 MiB")
					os.Exit(2)
				}
			}
		}()
		given := fuelOnFixture(t)
		given.onGPU = false
		read := readBoundarySet(t, given.set)
		stage := staging{given: given, recipes: read, world: thing.NewWorld()}
		if err := stage.terrain(read); err != nil {
			t.Fatal(err)
		}
		if err := stage.prepareBeds(read); err != nil {
			t.Fatal(err)
		}
		defer (&liveScene{beds: stage.beds}).closeBeds()
		if len(stage.beds) != 1 || len(stage.beds[0].groups) != 1 {
			t.Fatal("checked fixture did not prepare its one group")
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	args := []string{"-test.run=^TestFuelBedPreparationTerminatesForCheckedInput$", "-test.v"}
	if flag := stdflag.Lookup("test.gocoverdir"); testing.CoverMode() != "" && flag != nil && flag.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+flag.Value.String())
	}
	child := exec.CommandContext(ctx, os.Args[0], args...)
	child.Env = append(os.Environ(), "SLAYER_FUEL_PREPARATION_PROBE=1", "GOMAXPROCS=1", "GOMEMLIMIT=128MiB", "TMPDIR="+t.TempDir())
	out, err := child.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("checked fuel-bed preparation did not terminate within 3s:\n%s", out)
	}
	if err != nil {
		t.Fatalf("checked fuel-bed preparation failed: %v\n%s", err, out)
	}
}
