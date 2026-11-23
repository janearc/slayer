package thing

import (
	"fmt"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
)

// Made is the bridge from a material and a finish to one look. A
// material is what a thing is made of, a finish is what was done to its
// surface, and a look is made from the two here and nowhere else: the
// material's readings per band become Metal, in the tracer's red, green
// and blue order, and the finish's numbers sit on top, in fields a
// Material already has.
//
// Paint covers whatever it meets, so it gives a
// plain coloured look over any material; every other finish reads the
// metal, and a material with no optics is refused by name.
//
// The look's Name is the material's, so named looks can be made again.
func Made(kind materials.Material, finish Finish) (*Material, error) {
	row, known := finishes[finish.name]
	if !known {
		return nil, fmt.Errorf("thing: no finish %q", finish.name)
	}
	if kind.Name() == "" {
		return nil, fmt.Errorf("thing: a material never made")
	}
	look := &Material{Name: kind.Name(), Gloss: row.Gloss, Satin: row.Satin}
	if len(row.Colour) == metalBands {
		look.Colour = maths.RGB(row.Colour[metalRed], row.Colour[metalGreen],
			row.Colour[metalBlue])
	}
	if row.Paint {
		look.Colour = finish.colour
		return look, nil
	}
	sight := kind.Optics()
	look.Metal = &Metal{}
	for at, band := range [...]materials.Band{materials.Red, materials.Green,
		materials.Blue} {
		if !sight.Has(band) {
			return nil, fmt.Errorf("thing: %s has no optics for finish %q",
				kind.Name(), finish.name)
		}
		read := sight.At(band)
		look.Metal.N[at], look.Metal.K[at] = read.N(), read.K()
	}
	if row.FilmIndex > metalNone {
		look.Film = &Film{Index: row.FilmIndex, Thickness: row.FilmThickness}
	}
	return look, nil
}
