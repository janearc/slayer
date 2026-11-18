package searching

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// enrichRubin makes the native stepped facility read as a working site.
// HVAC, facade openings, pipework and yard details are authored scenery,
// not claims about the surveyed Rubin building or its operating equipment.
func (s *heardStage) enrichRubin() error {
	base := s.ground(0, 0)
	var slab, metal, vents, glass, markings siteBlocks
	box := func(batch *siteBlocks, x, y, z, w, h, d float64) { batch.box(maths.V(x, base+y, z), maths.V(w, h, d)) }
	// Match the native surveyTier's two-metre roof inset. Windows and
	// mullions lie on that slope instead of floating in front of the wall.
	facade := func(batch *siteBlocks, x, y, z, w, h, d, floor, slope float64) {
		first := len(batch.vertices)
		box(batch, x, y, z, w, h, d)
		for i := first; i < len(batch.vertices); i++ {
			batch.vertices[i].Z += slope * (batch.vertices[i].Y - base - floor)
		}
	}
	box(&slab, -23, -1.5, -19, 97, 1.5, 40)
	// Service-building facades: occupied rooms above the loading level.
	for i := range 16 {
		x := 18 + float64(i)*2.7
		facade(&glass, x, 1.8, -11.06, 1.5, 1.35, .08, 0, .4)
		if i < 12 {
			facade(&glass, x, 6.5, -10.06, 1.45, 1.1, .08, 5.16, .5)
		}
		facade(&metal, x-.12, .4, -11.12, .12, 4.45, .14, 0, .4)
	}
	// Louvred air-handling equipment on each of the three roof levels.
	for _, row := range [][3]float64{{49, 5.25, 0}, {58, 5.25, 0}, {41, 9.55, 0}, {22, 12.6, 0}, {30, 12.6, 0}} {
		x, y, z := row[0], row[1], row[2]
		box(&metal, x, y, z, 5.2, 2.3, 4)
		for i := range 11 {
			box(&vents, x+.12, y+.15+float64(i)*.18, z-.12, 4.95, .07, .13)
		}
		box(&metal, x+1.8, y+2.3, z+1.1, 1.6, .55, 1.8)
	}
	// Roof parapets and a narrow service gallery with railings and posts.
	for _, row := range [][4]float64{{15.6, 5.12, 48, -9.7}, {15.9, 9.45, 36, -8.7}, {16.2, 12.4, 20, -7.2}} {
		box(&metal, row[0], row[1], row[3], row[2], .1, .75)
		box(&metal, row[0], row[1]+1, row[3], row[2], .09, .09)
		for i := 0; i < int(row[2]/2); i++ {
			box(&metal, row[0]+float64(i)*2, row[1], row[3], .07, 1, .07)
		}
	}
	// Cooling supply/return and cable trays descend to the service yard.
	for i := range 3 {
		z := 4 + float64(i)*.8
		box(&metal, 63, 1, z, 9, .22, .25)
		box(&metal, 71.5, 1, z, .22, 2.8, .25)
	}
	for i := range 5 {
		x := 67 + float64(i)*1.3
		box(&metal, x, 0, 7, 1, 2.4, 1.8)
		box(&vents, x+.1, 1.6, 6.95, .8, .45, .06)
	}
	// Door panel ribs, human-scale bollards and a marked loading apron.
	for i := range 13 {
		facade(&vents, 48, 1+float64(i)*.24, -11.08, 5, .06, .08, 0, .4)
	}
	for i := range 7 {
		x := 46 + float64(i)*2.2
		box(&markings, x, 0, -16, .15, .85, .15)
		box(&markings, x, -.01, -24, .1, .04, 6)
	}
	for _, row := range []struct {
		batch    *siteBlocks
		material *thing.Material
	}{
		{&slab, &thing.Material{Name: "Rubin authored concrete terrace", Colour: maths.RGB(.2, .22, .24)}},
		{&metal, &thing.Material{Name: "Rubin authored service metal", Colour: maths.RGB(.34, .39, .44), Gloss: .22}},
		{&vents, &thing.Material{Name: "Rubin authored vent recesses", Colour: maths.RGB(.035, .045, .055)}},
		{&glass, &thing.Material{Name: "Rubin authored occupied rooms", Colour: maths.RGB(.02, .03, .04), Glow: maths.RGB(.18, .26, .32)}},
		{&markings, &thing.Material{Name: "Rubin authored loading markers", Colour: maths.RGB(.65, .42, .08), Glow: maths.RGB(.025, .012, .002)}},
	} {
		if err := row.batch.place(s, row.material); err != nil {
			return err
		}
	}
	s.world.Lamp(emit.Lit{At: maths.V(51, base+7, -17), Strength: maths.RGB(45, 55, 70)})
	return nil
}
