//go:build darwin

package fuelbed

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"testing"
	"time"
)

const fuelProbeMode = "SLAYER_BOUNDED_FUEL_PROBE"

// A one-cell tick must finish. Children contain a broken solver loop so the
// ordinary suite can report a failure without leaving it allocating forever.
func TestOneCellSolversFinish(t *testing.T) {
	if mode := os.Getenv(fuelProbeMode); mode != "" {
		var solver Evolution
		var err error
		if mode == "cpu" {
			solver, err = New(quietBoundarySpec())
		} else {
			device, opened := NewMetal(quietBoundarySpec())
			if opened == nil {
				defer device.Close()
			}
			solver, err = device, opened
		}
		if err != nil {
			t.Fatal(err)
		}
		cells, err := solver.At(.5)
		if err != nil || len(cells) != 1 || cells[0] != (Cell{Excess: 100, Fuel: 0}) {
			t.Fatalf("one completed tick: %v %v", cells, err)
		}
		return
	}
	for _, mode := range []string{"cpu", "metal"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestOneCellSolversFinish$"}
			if dir := flag.Lookup("test.gocoverdir"); testing.CoverMode() != "" && dir != nil && dir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+dir.Value.String())
			}
			child := exec.CommandContext(ctx, os.Args[0], args...)
			child.Env = append(os.Environ(), fuelProbeMode+"="+mode)
			out, err := child.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("one-cell %s solver did not terminate: %v", mode, ctx.Err())
			}
			if err != nil {
				t.Fatalf("one-cell %s solver: %v\n%s", mode, err, out)
			}
		})
	}
}
