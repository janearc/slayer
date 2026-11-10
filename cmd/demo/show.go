package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/scene/smoke"
)

type preview struct {
	width, height, rays int
	output              string
	cells               terminal.Size
	pixelAspect         float64
	grace               time.Duration
	sampleBudget        int64
}

// readPreview checks optional overrides before creating any output.
func readPreview(call pzm.Call, menu *catalogue) (*preview, error) {
	grace := menu.Preview.ShutdownGrace * float64(time.Second)
	given := preview{width: menu.Preview.Width, height: menu.Preview.Height,
		rays: menu.Preview.Rays, output: filepath.Join("out", "demos"),
		pixelAspect:  menu.Preview.PixelAspect,
		grace:        time.Duration(grace),
		sampleBudget: menu.Preview.SampleBudget}
	for _, option := range []struct {
		name   string
		target *int
		limit  int
	}{
		{"width", &given.width, menu.Preview.Limits.Width},
		{"height", &given.height, menu.Preview.Limits.Height},
		{"rays", &given.rays, menu.Preview.Limits.Rays},
	} {
		if !call.Given(option.name) {
			continue
		}
		value, err := strconv.Atoi(call.Value(option.name))
		if err != nil || value < one || value > option.limit {
			return nil, fmt.Errorf("%s must be an integer in 1..%d",
				option.name, option.limit)
		}
		*option.target = value
	}
	if err := given.checkSamples(given.rays); err != nil {
		return nil, err
	}
	if call.Given("out") {
		given.output = call.Value("out")
		if given.output == "" {
			return nil, fmt.Errorf("out needs a directory")
		}
	}
	cells := menu.Preview.Cells
	if call.Given("cells") {
		cells = call.Value("cells")
	}
	var err error
	given.cells, err = terminal.ParseSize(cells)
	if err != nil {
		return nil, err
	}
	return &given, nil
}

// show resolves one selector before constructing or building a scene.
func (app application) show(call pzm.Call) error {
	if len(call.Args()) != one {
		return fmt.Errorf("show wants one smoke number or demo name; " +
			"use demo list")
	}
	menu, catalogueErr := readCatalogue()
	if catalogueErr != nil {
		return catalogueErr
	}
	given, err := readPreview(call, menu)
	if err != nil {
		return err
	}
	selector := call.Args()[nothing]
	if number, err := strconv.Atoi(selector); err == nil {
		scene, exists := smoke.Get(number)
		if !exists {
			return fmt.Errorf("no smoke %d; use demo list", number)
		}
		return showSmoke(call, *given, scene)
	}
	for _, preset := range menu.Demos {
		if preset.Name == selector {
			return showNamed(app.ctx, call, *given, preset)
		}
	}
	return fmt.Errorf("no demo %q; use demo list", selector)
}

// showSmoke draws a real milestone without claiming a smokecheck verdict.
func showSmoke(call pzm.Call, given preview,
	scene smoke.Smoke) error {
	name := fmt.Sprintf("smoke-%d.png", scene.Number)
	target := filepath.Join(given.output, name)
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(),
			"would draw smoke %d, %s -> %s\n",
			scene.Number, scene.Name, target)
		return err
	}
	if _, err := scene.RequireInputs(); err != nil {
		return err
	}
	rays := max(given.rays, scene.Samples)
	if err := given.checkSamples(rays); err != nil {
		return err
	}
	picture, err := drawSmoke(scene, given)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(given.output, 0755); err != nil {
		return err
	}
	if err := file.PNG(target, picture, one); err != nil {
		return err
	}
	if call.On("terminal") {
		err := terminalPreview(call.Out(), picture, given)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(call.Out(), target)
	return err
}

// checkSamples bounds the supersampled CPU scratch image before allocation.
func (given preview) checkSamples(rays int) error {
	samples := int64(given.width) * int64(given.height)
	samples *= int64(rays) * int64(rays)
	if samples > given.sampleBudget {
		return fmt.Errorf("preview exceeds its sample budget; " +
			"reduce size or rays, or use the scene command")
	}
	return nil
}
