package flame

import "github.com/janearc/slayer/maths"

// A Source is one point of a flame's light, standing for a cell of its
// body: where it is in the flame's own space, how big its cell is
// across, and its strength, the light of the cell at a unit of
// distance in the picture's light, by channel.
type Source struct {
	At       maths.Vec
	Across   float64
	Strength maths.Light
}

// Placed carries a source cell into a uniformly scaled scene placement.
// Its width scales as length and its integrated power as emitting volume.
// A zero scale is the identity convention used by maths.Subspace.
func (source Source) Placed(space maths.Subspace) Source {
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	source.At = space.Out(source.At)
	source.Across *= scale
	source.Strength = source.Strength.Scale(scale * scale * scale)
	return source
}

// Sources is the flame's light as points, for lighting what is near it.
//
// the flame's box is cut into cells a side; each cell with soot in it
// at its middle gives a source there, the light the cell gives off:
// its absorption times its glow, by kirchhoff's law, times its volume,
// at the eye's three wavelengths.
//
// the finer the cells, the closer their sum is to the flame's light.
//
// a source's light is not dimmed here by the flame around it: whatever
// draws the flame dims the light from each source by the soot between
// it and what it lights, as for any lamp in a medium.
func (body *Body) Sources(cells int) []Source {
	if cells < 1 {
		return nil
	}
	box := body.Bounds()
	size := box.Max.Sub(box.Min).Scale(1 / float64(cells))
	volume := size.X * size.Y * size.Z
	across := size.Len()
	tint := body.Tint()
	var made []Source
	for i := range cells * cells * cells {
		x, y, z := i%cells, i/cells%cells, i/(cells*cells)
		middle := box.Min.Add(maths.V((float64(x)+0.5)*size.X,
			(float64(y)+0.5)*size.Y, (float64(z)+0.5)*size.Z))
		soot := body.Density(middle)
		if !(soot > 0) {
			continue
		}
		glow := body.glows.at(body.Heat(middle))
		made = append(made, Source{At: middle, Across: across,
			Strength: glow.Times(tint).Scale(soot * volume)})
	}
	return made
}
