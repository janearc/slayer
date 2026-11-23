// Package hanauma holds the Hānauma Bay terrain and its scene materials.
package hanauma

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
)

//go:embed noaa-cudem-window.bin.gz
var elevations []byte

//go:embed noaa-structure.bin.gz
var structure []byte

const west, south = -850., -620.

// groundForWater returns the surveyed relief, with sea level at Y=0.
// The wet finish includes a low-order diffuse light return from the seabed.
func groundForWater(wet bool) (thing.Thing, error) {
	bed, err := ground()
	if err != nil || !wet {
		return bed, err
	}
	look := *bed.Material
	look.Shine = seabedFill{paint: look.Texture.(groundPaint)}
	bed.Material = &look
	return bed, nil
}

var ground = sync.OnceValues(func() (thing.Thing, error) {
	raw, err := unpack(elevations)
	if err != nil {
		return thing.Thing{}, fmt.Errorf("hanauma relief: %w", err)
	}
	window, err := terrain.WindowFromBytes(raw)
	if err != nil {
		return thing.Thing{}, err
	}
	hardbottom, err := unpack(structure)
	if err != nil {
		return thing.Thing{}, fmt.Errorf("hanauma seabed structure: %w", err)
	}
	across, down, _ := window.Size()
	if len(hardbottom) != across*down {
		return thing.Thing{}, fmt.Errorf("hanauma seabed structure: need %d samples, got %d", across*down, len(hardbottom))
	}
	solid, lowest, err := window.Ground()
	if err != nil {
		return thing.Thing{}, err
	}
	offset := maths.V(west, lowest, south)
	return thing.Thing{
		Shape:  solid,
		Motion: thing.Still(maths.NewSubspace(offset, maths.Identity, 1)),
		Material: &thing.Material{
			Name:    "Hanauma sand, reef pavement and tuff",
			Texture: groundPaint{offset: offset, relief: window, hardbottom: hardbottom},
		},
	}, nil
})

// unpack is the bytes of a gzip stream, uncompressed.
func unpack(data []byte) ([]byte, error) {
	input, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer input.Close()
	return io.ReadAll(input)
}

type groundPaint struct {
	offset     maths.Vec
	relief     terrain.Window
	hardbottom []byte
}

type seabedFill struct{ paint groundPaint }

// At is the colour the fill adds at a point under the water: none at the
// surface, rising over the first third of a metre, then fading with depth.
func (fill seabedFill) At(local maths.Vec) maths.Light {
	depth := -(local.Y + fill.paint.offset.Y)
	if depth <= 0 {
		return maths.Light{}
	}
	share := maths.Smooth(0, .35, depth) * math.Exp(-.026*depth)
	// Approximate diffuse return below the surface without a separate fill
	// lamp crossing the boundary. Camera rays still undergo absorption.
	return fill.paint.At(local).Times(maths.RGB(.16, .52, .73)).Scale(share)
}

// At is the ground's colour at a point: sand varied by noise, changed by
// the slope and by the hard bottom.
func (paint groundPaint) At(local maths.Vec) maths.Light {
	point := local.Add(paint.offset)
	broad := noise.Gradient(239, point.X/30, 4.9, point.Z/30)
	fine := noise.Gradient(811, point.X/6.5, point.Y/7-.11, point.Z/6.5)
	sand := maths.RGB(.67, .55, .36).Scale(1 + broad*.12 + fine*.09)
	slope := paint.grade(local)
	if point.Y >= 0 {
		// Reserve pale sand for low, gently sloping shoreline in the cove.
		cove := 1 - maths.Smooth(.84, 1.08, math.Hypot((point.X+92)/352, (point.Z-43)/355))
		beach := cove * (1 - maths.Smooth(1.8, 5.3, point.Y)) * (1 - maths.Smooth(.21, .67, slope))
		stone := maths.RGB(.24, .158, .083).Scale(1 + broad*.47 + fine*.17)
		scrub := maths.RGB(.115, .173, .046).Scale(1 + broad*.57 + fine*.45)
		cover := maths.Smooth(.26, .73, .61+broad*.81+fine*.18-slope*.30)
		land := stone.Scale(1 - cover).Add(scrub.Scale(cover))
		return land.Scale(1 - beach).Add(sand.Scale(beach))
	}
	depth := -point.Y
	rough := maths.Smooth(.05, .32, slope)
	pavement := maths.Smooth(.14, .66, .33+broad*.77+fine*.25+rough*.63)
	mapped, coverage := paint.mappedHardbottom(local)
	detail := math.Max(0, math.Min(1, mapped+.20*broad+.34*fine))
	pavement = pavement*(1-coverage) + detail*coverage
	pavement *= maths.Smooth(.45, 1.30, depth) * (1 - maths.Smooth(8, 21, depth))
	reef := maths.RGB(.105, .128, .084).Scale(1 + broad*.42 + fine*.52)
	return sand.Scale(1 - pavement).Add(reef.Scale(pavement))
}

// mappedHardbottom is the share of hard bottom at a point and how much of it
// is covered, read from the mapped cells and blended between them.
func (paint groundPaint) mappedHardbottom(point maths.Vec) (share, coverage float64) {
	across, down, step := paint.relief.Size()
	x := math.Max(0, math.Min(float64(across-1), point.X/step))
	z := math.Max(0, math.Min(float64(down-1), point.Z/step))
	column, row := min(int(x), across-2), min(int(z), down-2)
	u, v := x-float64(column), z-float64(row)
	for index, weight := range [4]float64{(1 - u) * (1 - v), u * (1 - v), (1 - u) * v, u * v} {
		value := paint.hardbottom[(row+index/2)*across+column+index%2]
		if value != 255 {
			coverage += weight
			share += weight * float64(value) / 200
		}
	}
	if coverage > 0 {
		share /= coverage
	}
	return share, coverage
}

// grade is the slope of the ground at a point, from the heights of the
// relief cells either side of it.
func (paint groundPaint) grade(point maths.Vec) float64 {
	across, down, step := paint.relief.Size()
	column := min(across-2, max(1, int(math.Round(point.X/step))))
	row := min(down-2, max(1, int(math.Round(point.Z/step))))
	dx := paint.relief.At(column+1, row) - paint.relief.At(column-1, row)
	dz := paint.relief.At(column, row+1) - paint.relief.At(column, row-1)
	return math.Hypot(dx, dz) / (2 * step)
}
