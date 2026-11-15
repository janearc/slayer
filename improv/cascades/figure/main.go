// Command figure builds the scene of figure 1 of Osborne and Sannikov
// (2024), solves it by brute force and by radiance cascades, writes the
// three results side by side as one PNG and prints a report.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/janearc/slayer/improv/cascades/flatland"
)

// options are the command line flags.
type options struct {
	side        int
	directions  int
	workers     int
	firstLength float64
	exposure    float64
	fix         bool
	gain        float64
	zoom        int
	out         string
	scene       string
}

// main reads the flags and runs. A missing -out exits with 2, and a failed
// run prints the error and exits with 1.
func main() {
	var opt options
	flag.IntVar(&opt.side, "side", 256, "grid side in cells, a power of two")
	flag.IntVar(&opt.directions, "directions", 256, "reference directions per cell")
	flag.IntVar(&opt.workers, "workers", 2, "parallel workers")
	flag.Float64Var(&opt.firstLength, "firstlength", 1.5, "length of cascade 0's shell, in cells")
	flag.Float64Var(&opt.exposure, "exposure", 4, "exposure before tone mapping")
	flag.BoolVar(&opt.fix, "bilinear", true, "third panel is the bilinear fix; if false, the difference")
	flag.Float64Var(&opt.gain, "diffgain", 8, "scale applied to the difference panel")
	flag.IntVar(&opt.zoom, "zoom", 1, "pixels per cell in the picture")
	flag.StringVar(&opt.out, "out", "", "path of the PNG to write")
	flag.StringVar(&opt.scene, "scene", "figure", "figure, or single (one disc, for checking falloff)")
	flag.Parse()
	if opt.out == "" {
		fmt.Fprintln(os.Stderr, "figure: -out is required")
		os.Exit(2)
	}
	if err := run(opt); err != nil {
		fmt.Fprintln(os.Stderr, "figure:", err)
		os.Exit(1)
	}
}

// run makes the plan and the scene, solves it, prints the falloff for the
// single scene, and writes the picture.
func run(opt options) error {
	plan, err := flatland.MakePlan(opt.side, opt.firstLength)
	if err != nil {
		return err
	}
	grid, opaque, err := buildScene(opt.side, opt.scene)
	if err != nil {
		return err
	}
	fmt.Printf("grid: %dx%d cells, %d reference directions, %d workers\n",
		opt.side, opt.side, opt.directions, opt.workers)
	printPlan(plan)
	panels := solve(opt, plan, grid, opaque)
	if opt.scene == "single" {
		printFalloff(opt.side, panels)
	}
	return writePicture(opt.out, panels, opt.exposure, opt.zoom)
}

// solve runs the methods, prints their cost and error, and returns the
// panels of the picture: reference, cascades, and the bilinear fix or the
// difference.
func solve(opt options, plan flatland.Plan, grid *flatland.Grid, opaque []bool) []flatland.Field {
	reference, referenceStats := flatland.Reference(grid, opt.directions, opt.workers)
	printStats("reference", referenceStats)
	basic, basicStats := flatland.Cascades(grid, plan, opt.workers, false)
	printStats("cascades", basicStats)
	printErrors("cascades", basic, reference, opaque)
	if !opt.fix {
		fmt.Printf("third panel: absolute difference, cascades minus reference, times %g\n", opt.gain)
		return []flatland.Field{reference, basic, absoluteDifference(basic, reference, opt.gain)}
	}
	fixed, fixedStats := flatland.Cascades(grid, plan, opt.workers, true)
	printStats("cascades with bilinear fix", fixedStats)
	printErrors("bilinear fix", fixed, reference, opaque)
	return []flatland.Field{reference, basic, fixed}
}
