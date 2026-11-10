//go:build darwin

package gpu

import (
	_ "embed"
	"errors"
	"math"
	"time"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/research"
)

//go:embed flame.metal
var flameSource string

var fireGridStruct = structure{"FireGrid", []field{
	floatField("shape", "radius, height, absorption, puff period"),
	floatField("heat", "edge, hottest, time, unused"),
	floatField("low", ""), floatField("high", ""),
	intField("cells", ""), intField("seed", "low and high halves"),
	floatField("speeds", "continuous, intermittent, plume, lower border"),
	floatField("borders", "upper border"),
	floatField("offset", "source position"),
}}

// bakeFlame prepares a paired density/emission grid on Metal.
func bakeFlame(grid boxed) ([]float32, []float32, time.Duration, error) {
	kernel, inputs, err := prepareFlameBake(grid)
	if err != nil {
		return nil, nil, 0, err
	}
	defer kernel.Close()
	samples, err := kernel.Grid(grid.cells[0]*grid.cells[1],
		grid.cells[2], rows, 4, inputs)
	if err != nil {
		return nil, nil, kernel.GPUTime(), err
	}
	density := make([]float32, grid.count())
	shine := make([]float32, grid.count()*3)
	for i := range density {
		density[i] = samples[4*i]
		copy(shine[3*i:3*i+3], samples[4*i+1:4*i+4])
	}
	return density, shine, kernel.GPUTime(), nil
}

// prepareFlameBake shares validation, parameters and source between full-grid
// reference callers and production's streamed writes into reserved storage.
func prepareFlameBake(grid boxed) (*metal.Kernel, [][]byte, error) {
	field, ok := flameField(grid)
	if !ok {
		return nil, nil, unsupported{"a custom flame grid"}
	}
	for _, cells := range grid.cells {
		if cells < 1 || cells > mostBoxCells {
			return nil, nil,
				errors.New(
					"gpu: invalid flame grid dimensions")
		}
	}
	if grid.cells[0] > (math.MaxInt32/16)/grid.cells[1]/grid.cells[2] {
		return nil, nil, errors.New("gpu: flame grid exceeds its packed buffer capacity")
	}
	source := metal.NoiseLibrary + fireGridStruct.msl() + flameSource
	kernel, err := metal.Open(source, "fire_grid")
	if err != nil {
		return nil, nil, err
	}
	params, parts, glow := flameParameters(field, grid)
	return kernel, [][]byte{params, parts, glow}, nil
}

// flameParameters carries physical inputs and the same pinned rise constants
// as flame.Body. The shader's authored shape constants are checked against it.
func flameParameters(field flameDescription, grid boxed) ([]byte, []byte,
	[]byte) {
	var header, components, table packer
	composite := 0
	if field.composite {
		composite = 1
	}
	for i, part := range field.parts {
		fire := part.fire
		given := values{
			"shape": {fire.Radius, fire.Height,
				fire.Absorption, part.period.Seconds()},
			"heat": {fire.Edge.InKelvin(),
				fire.Hottest.InKelvin(), part.at.Seconds(), float64(flag(fire.HotTongues))},
			"low": point(grid.low), "high": point(grid.high),
			"cells": wholes(grid.cells[:]...),
			"seed": wholes(int(uint32(fire.Seed)),
				int(uint32(fire.Seed>>32)),
				len(field.parts), composite),
			"speeds": {research.Value("continuous-speed"),
				research.Value("tongue-speed"),
				research.Value("plume-speed"),
				research.Value("tongues-from")},
			"borders": {research.Value("plume-from")},
			"offset":  append(point(part.offset), fire.Intermittency),
		}
		if i == 0 {
			header.row(fireGridStruct, given)
		}
		components.row(fireGridStruct, given)
		for _, light := range part.glow {
			for _, number := range []float64{light.R, light.G,
				light.B, 0} {
				table.word(math.Float32bits(float32(number)))
			}
		}
	}
	return header.bytes, components.bytes, table.bytes
}

// bakeFlames fills reserved density/emission regions without moving any grid.
func (encoded *scene) bakeFlames() (time.Duration, error) {
	var elapsed time.Duration
	for _, job := range encoded.flames {
		spent, err := encoded.bakeFlameInto(job)
		elapsed += spent
		if err != nil {
			return elapsed, err
		}
	}
	return elapsed, nil
}
