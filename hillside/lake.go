package hillside

import (
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// the lake is improv/moonlake's, a sketch never merged: still water as
// the earth's own curve, its top at an elevation in metres, dark and
// glossy as island's ocean is (gloss 1, index 1.34)
const lakeRadius = 6371e3

// flooded fills the terrain with still water to an elevation, metres
// above the sea.
func (stage *staging) flooded(elevation float64) error {
	depth := elevation - stage.lowest
	water, err := globe.New(lakeRadius)
	if err != nil {
		return err
	}
	material := &thing.Material{Name: "still water",
		Colour: maths.RGB(0.004, 0.012, 0.02), Gloss: 1, Index: 1.34}
	// -lakeglow makes the water shine on its own and nothing else, so a
	// render of it is a mask of where the lake is seen
	if stage.given.lakeGlow {
		material = &thing.Material{Name: "water mask",
			Glow: maths.Grey(1)}
	}
	stage.world.Add(thing.Thing{Shape: water, Material: material,
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(0, depth-lakeRadius, 0)})})
	return nil
}
