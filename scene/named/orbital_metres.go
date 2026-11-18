package named

import (
	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"math"
)

// orbitalMetreFloor retains a finite macro biome world and an implicit fine
// lattice. Rivers, lakes and coastal cities keep their established geography;
// physical cell size changes fine seams and lamps, not continental structure.
type orbitalMetreFloor struct {
	tiles     hexes.Band
	fields    biome.Fields
	glowing   bool
	geography *biome.World
	macro     hexes.Band
}

// newOrbitalMetreFloor returns the floor of a band with the given radius. Its
// tiles are metres wide, counted around the band. Its biome fields use seed.
func newOrbitalMetreFloor(radius float64, metres int, seed uint64) orbitalMetreFloor {
	tiles := hexes.Band{Radius: radius, Width: bandWidth, Around: int(math.Round(2 * math.Pi * radius * 1e6 / float64(metres)))}
	return orbitalMetreFloor{tiles: tiles, fields: biome.Fields{Tiling: tiles, Seed: seed, Scale: floorScale, Sea: floorSea}}
}

// geographyPaint returns the paint of the tile under p. Without a geography it
// uses the floor's own fields, and a glowing floor paints no light. With a
// geography, a glowing floor uses its shine paint.
func (f orbitalMetreFloor) geographyPaint(p maths.Vec) maths.Light {
	if f.geography == nil {
		if f.glowing {
			return maths.Light{}
		}
		return f.fields.Paint(f.tiles.Nearest(p))
	}
	tile := f.macro.Nearest(p)
	if f.glowing {
		return f.geography.Shine().Paint(tile)
	}
	return f.geography.Paint(tile)
}

// mean returns the paint at p averaged over the seams. A glowing floor returns
// its paint as it is. Otherwise the paint is darkened by the seam coverage.
func (f orbitalMetreFloor) mean(p maths.Vec) maths.Light {
	if f.glowing {
		return f.geographyPaint(p)
	}
	const seamCoverage = 2*.06 - 1.2*.06*.06
	return f.geographyPaint(p).Scale(1 - floorLines*seamCoverage)
}

// At returns the colour at p. A glowing floor scales its geography paint by a
// per-tile lamp grain. Otherwise the paint takes a per-tile grain and is
// darkened along the tile edges.
func (f orbitalMetreFloor) At(p maths.Vec) maths.Light {
	tile := f.tiles.Nearest(p)
	id := f.tiles.ID(tile)
	if f.glowing {
		// Resolved lamp grain has unit mean. Distant cities keep their emitted
		// brightness and branching shape when the fine cells become unresolved.
		return f.geographyPaint(p).Scale(.55 + .9*noise.Hash(f.fields.Seed+11, id))
	}
	paint := f.geographyPaint(p)
	grain := .92 + .16*noise.Hash(f.fields.Seed+12, id)
	line := 1 - maths.Smooth(0, f.tiles.Spacing()*.06, f.tiles.Edge(p, tile))
	return paint.Scale(grain * (1 - floorLines*line))
}

// Within uses the actual local-space pixel footprint. Unresolved ten-metre
// boundaries converge to their mean; macro geography is sampled across the
// footprint to retain a stable image as the bands rotate.
func (f orbitalMetreFloor) Within(p maths.Vec, footprint float64) maths.Light {
	step := f.tiles.Spacing()
	if footprint <= step/2 {
		return f.At(p)
	}
	axis := maths.V(p.Z, 0, -p.X).Unit()
	var mean maths.Light
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			q := p.Add(axis.Scale(float64(x) * footprint / 3))
			q.Y += float64(y) * footprint / 3
			radius := math.Hypot(q.X, q.Z)
			if radius > 0 {
				q.X *= f.tiles.Radius / radius
				q.Z *= f.tiles.Radius / radius
			}
			mean = mean.Add(f.mean(q))
		}
	}
	mean = mean.Scale(1.0 / 9)
	if footprint >= 2*step {
		return mean
	}
	blend := maths.Smooth(step/2, 2*step, footprint)
	return f.At(p).Scale(1 - blend).Add(mean.Scale(blend))
}
