package searching

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// keckCampus adds the connecting service building, access and scale cues
// visible in the supplied references. These are authored architectural
// approximations; only site coordinates and regional terrain are sourced.
func (s *heardStage) keckCampus(plan *stagePlan) error {
	var walls, roofs, trim, vents, glass, lights siteBlocks
	a, b := maths.V(-28.669, 0, 18.299), maths.V(22.005, 0, -49.972)
	centre := a.Add(b).Scale(.5)
	delta := b.Sub(a)
	turn := maths.About(maths.YAxis, math.Atan2(delta.X, delta.Z))
	put := func(batch *siteBlocks, x, y, z, w, h, d float64) { batch.box(maths.V(x, y+2.5, z), maths.V(w, h, d)) }
	put(&walls, -6, 0, -28, 12, 4.8, 56)
	put(&roofs, -6.4, 4.8, -28.3, 12.8, .35, 56.6)
	put(&walls, 6, 0, -7, 20, 5.4, 25)
	put(&roofs, 5.7, 5.4, -7.3, 20.6, .3, 25.6)
	for i := range 18 {
		z := -26 + float64(i)*3
		put(&trim, -6.2, 0, z, .25, 4.9, .18)
		put(&glass, -6.24, 1.6, z+.45, .12, 1.4, 1.4)
		put(&trim, 6, 0, z, .25, 4.9, .18)
	}
	for i := range 7 {
		z := -5 + float64(i)*3
		put(&glass, 26.02, 1.6, z, .1, 1.8, 1.6)
		put(&trim, 26, 0, z-.25, .2, 5.3, .15)
	}
	put(&vents, 25.95, .2, 9, .18, 3.2, 3.5)
	for i := range 14 {
		put(&trim, 26.08, .35+float64(i)*.21, 9, .1, .06, 3.5)
	}
	for i := range 4 {
		z := -20 + float64(i)*12
		put(&trim, -2, 5.15, z, 4, 1.2, 3)
		for j := range 8 {
			put(&vents, -2.03, 5.25+float64(j)*.11, z, .06, .04, 3)
		}
	}
	// Rotate the service wing into the actual line between the two sites.
	for _, batch := range []*siteBlocks{&walls, &roofs, &trim, &vents, &glass} {
		for i, p := range batch.vertices {
			batch.vertices[i] = turn.Out(p).Add(centre)
		}
	}
	// Warm work lights are inside each enclosure; exterior light stays cool.
	for _, at := range []maths.Vec{a, b} {
		floor := s.ground(at.X, at.Z)
		if !plan.Burst {
			s.world.Lamp(emit.Lit{At: at.Add(maths.V(0, floor+17, -2)), Strength: maths.RGB(180, 105, 35)})
		}
		for i := range 6 {
			lights.box(at.Add(maths.V(-4+float64(i)*1.6, floor+24, -3)), maths.V(.8, .12, .2))
		}
		for i := range 10 {
			angle := float64(i) * 2 * math.Pi / 10
			p := at.Add(maths.V(16*math.Cos(angle), floor+.2, 16*math.Sin(angle)))
			trim.box(p, maths.V(.18, 1.2, .18))
		}
	}
	if !plan.Burst {
		s.world.Lamp(film.Flood{Strength: maths.RGB(.025, .035, .065)})
	}
	for _, row := range []struct {
		mesh     *siteBlocks
		material *thing.Material
	}{
		{&walls, &thing.Material{Name: "authored Keck connecting service wing", Colour: maths.RGB(.68, .7, .72)}},
		{&roofs, &thing.Material{Name: "Keck service wing roofs", Colour: maths.RGB(.18, .21, .24)}},
		{&trim, &thing.Material{Name: "Keck service and safety details", Colour: maths.RGB(.5, .53, .57)}},
		{&vents, &thing.Material{Name: "Keck recessed plant ventilation", Colour: maths.RGB(.025, .035, .04)}},
		{&glass, &thing.Material{Name: "Keck occupied support rooms", Glow: maths.RGB(.22, .15, .065)}},
		{&lights, &thing.Material{Name: "Keck interior work lights", Glow: maths.RGB(2, 1.15, .35)}},
	} {
		if err := row.mesh.placeSeparate(s, row.material); err != nil {
			return err
		}
	}
	return s.maunaKeaPrecinct()
}
