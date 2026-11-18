package searching

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// owlCampus builds an authored optical-engineering campus, not a surveyed
// building. Processing, storage, thermal plant, power and access each have
// separate footprints, connected by service routes and mechanical systems.
func (s *heardStage) owlCampus() error {
	var walls, roofs, frames, recesses, glass, slab, paint, lights siteBlocks
	put := func(b *siteBlocks, x, y, z, w, h, d float64) { b.box(maths.V(x, y, z), maths.V(w, h, d)) }
	// Mirror washing/coating hall: high bays, a lower clean-room wing and
	// individual roof services. The dark loading doors are on the yard face.
	hall := func(x, z, w, h, d float64, bays int) {
		put(&slab, x-2, -.5, z-2, w+4, .55, d+4)
		put(&walls, x, 0, z, w, h, d)
		put(&roofs, x-.5, h, z-.5, w+1, .45, d+1)
		for i := range bays {
			bx := x + float64(i)*w/float64(bays)
			put(&frames, bx-.15, 0, z-.2, .3, h, .35)
			put(&recesses, bx+1, 1, z-.12, w/float64(bays)-2, h*.58, .15)
			put(&frames, bx+1, h*.58+1, z-.4, w/float64(bays)-2, .35, .5)
			for j := range 16 {
				put(&frames, bx+1, 1+float64(j)*h*.034, z-.3, w/float64(bays)-2, .08, .15)
			}
			put(&glass, bx+2, h*.76, z-.14, w/float64(bays)-4, 1.2, .17)
			put(&roofs, bx+2, h+.45, z+4, 5, 1.8, 5)
			for j := range 9 {
				put(&recesses, bx+2.15, h+.6+float64(j)*.16, z+3.9, 4.7, .06, .12)
			}
		}
		for i := 0; i < int(d/4); i++ {
			zz := z + float64(i)*4
			put(&frames, x-.15, 0, zz, .3, h, .3)
			put(&frames, x+w-.15, 0, zz, .3, h, .3)
		}
		for i := range 3 {
			put(&frames, x, h+float64(i)*.45, z-.4, w, .08, .1)
		}
	}
	hall(-250, -145, 125, 22, 65, 7)
	hall(-245, -72, 90, 9, 28, 9)
	hall(85, -170, 114, 12, 36, 12)
	hall(140, 95, 54, 18, 48, 4)
	// Control rooms have smaller occupied windows and balconies; their
	// circulation is separate from the heavy mirror-transport loading bays.
	for floor := range 3 {
		for bay := range 25 {
			put(&glass, 87+float64(bay)*4.4, 1.8+float64(floor)*3.3, -170.15, 2.2, 1.6, .2)
			put(&frames, 87+float64(bay)*4.4, 1.3+float64(floor)*3.3, -171, 2.3, .14, .9)
		}
	}
	put(&slab, 82, 0, -185, 120, .3, 13)
	for i := range 25 {
		put(&frames, 84+float64(i)*4.8, .3, -183, .08, 1.15, .08)
	}
	put(&frames, 84, 1.45, -183, 115, .08, .08)
	// Segment logistics: unequal weatherproof cradles, handling dollies,
	// racks, container corners and lids, alongside a clear transport aisle.
	for row := range 6 {
		for col := range 14 {
			x := -246 + float64(col)*7
			z := -220 + float64(row)*8
			height := 1.7 + .7*math.Abs(math.Sin(float64(row*14+col)*1.37))
			put(&slab, x-.3, -.1, z-.3, 5.6, .15, 5.6)
			put(&roofs, x, .3, z, 5, height, 4.6)
			put(&frames, x-.08, height+.3, z-.08, 5.16, .12, 4.76)
			for _, dx := range []float64{0, 4.8} {
				for _, dz := range []float64{0, 4.4} {
					put(&frames, x+dx, .1, z+dz, .2, height+.4, .2)
				}
			}
			put(&paint, x+1, height+.44, z+.2, 2, .02, .4)
		}
	}
	// Cooling batteries, manifolds and return lines serve the coating hall
	// and enclosure. Each module carries its own fans, slats and valve stands.
	for row := range 3 {
		for col := range 6 {
			x := -95 + float64(col)*16
			z := -245 + float64(row)*18
			put(&slab, x-.5, -.15, z-.5, 13, .2, 13)
			put(&walls, x, 0, z, 12, 4.2, 12)
			for i := range 15 {
				put(&recesses, x+.2, .25+float64(i)*.23, z-.12, 11.6, .1, .15)
			}
			for i := range 3 {
				put(&roofs, x+1+float64(i)*3.4, 4.2, z+1, 2.6, .7, 9.5)
			}
			put(&frames, x+2, 0, z+12.2, .4, 2.3, .4)
		}
	}
	for i := range 4 {
		x := -100 + float64(i)*3
		if _, err := s.characterRod(maths.V(x, 1.1, -245), maths.V(x, 1.1, -35), .28, "OWL cooling manifold", maths.Light{}); err != nil {
			return err
		}
	}
	for i := range 6 {
		x := -240 + float64(i)*8
		if _, err := s.characterRod(maths.V(x, 0, -28), maths.V(x, 11, -28), 2.6, "OWL process storage vessel", maths.Light{}); err != nil {
			return err
		}
		put(&frames, x-2, 0, -31, .25, 8, .25)
		put(&frames, x+2, 0, -31, .25, 8, .25)
	}
	// Substation: transformers, radiator fins, busbars, insulators and a
	// perimeter fence. Tall structure is confined to the service side.
	put(&slab, 200, -.2, -120, 76, .25, 90)
	for row := range 3 {
		for col := range 4 {
			x := 205 + float64(col)*17
			z := -113 + float64(row)*25
			put(&walls, x, 0, z, 10, 3.8, 14)
			for i := range 14 {
				put(&frames, x-.4, 0, z+float64(i), .35, 3.4, .3)
			}
			for i := range 3 {
				put(&recesses, x+2+float64(i)*2.6, 3.8, z+3, .55, 2.4, .55)
			}
			put(&frames, x+2, 6.2, z+3, 6, .15, .15)
		}
	}
	for i := range 40 {
		z := -120 + float64(i)*2.3
		for _, x := range []float64{199, 277} {
			put(&frames, x, 0, z, .09, 2.2, .09)
		}
	}
	for level := range 5 {
		for _, x := range []float64{199, 277} {
			put(&frames, x, .4+float64(level)*.4, -120, .04, .04, 90)
		}
	}
	// Supported rear bridge crane: the earlier proof had floating chords.
	for _, x := range []float64{-65, 65} {
		put(&slab, x-5, -.8, 235, 10, .9, 20)
		for _, z := range []float64{239, 250} {
			put(&frames, x-1, 0, z, 2, 108, 2)
		}
		for level := range 12 {
			y := float64(level) * 8.5
			put(&frames, x-1, y, 239, 2, .35, 13)
			a, b := maths.V(x, y, 239), maths.V(x, y+8.5, 250)
			if _, err := s.characterRod(a, b, .25, "OWL crane portal diagonal", maths.Light{}); err != nil {
				return err
			}
		}
	}
	// Walkable maintenance gallery around the main azimuth base.
	for i := range 64 {
		a, b := float64(i)*2*math.Pi/64, float64(i+1)*2*math.Pi/64
		point := func(radius, y, angle float64) maths.Vec {
			return maths.V(radius*math.Cos(angle), y, radius*math.Sin(angle))
		}
		first := len(slab.vertices)
		slab.vertices = append(slab.vertices, point(80, 6, a), point(84, 6, a), point(84, 6, b), point(80, 6, b))
		slab.faces = append(slab.faces, [3]int{first, first + 2, first + 1}, [3]int{first, first + 3, first + 2})
		at := point(83, 0, a)
		put(&frames, at.X, 0, at.Z, .3, 7.1, .3)
		if _, err := s.characterRod(point(83, 7.1, a), point(83, 7.1, b), .055, "OWL azimuth gallery rail", maths.Light{}); err != nil {
			return err
		}
		if i%4 == 0 {
			put(&lights, at.X, 6.9, at.Z, .18, .16, .18)
		}
	}
	// Loading routes, cable trenches, retaining edges and safety bollards.
	put(&roofs, -265, -.02, -270, 550, .04, 18)
	for i := range 90 {
		x := -260 + float64(i)*6
		put(&paint, x, .04, -262, 2, .04, .18)
		if i%2 == 0 {
			put(&paint, x, .04, -247, .15, .9, .15)
		}
	}
	for i := range 60 {
		put(&frames, -117, .04, -240+float64(i)*5, 1.2, .08, 3.7)
	}
	for _, at := range []maths.Vec{maths.V(-180, 24, -165), maths.V(145, 18, -125)} {
		put(&frames, at.X, 0, at.Z, .5, at.Y, .5)
	}
	for _, row := range []struct {
		mesh     *siteBlocks
		material *thing.Material
	}{
		{&walls, &thing.Material{Name: "OWL campus insulated walls", Colour: maths.RGB(.38, .41, .43)}},
		{&roofs, &thing.Material{Name: "OWL campus roof and plant", Colour: maths.RGB(.13, .17, .21), Gloss: .12}},
		{&frames, &thing.Material{Name: "OWL campus steel and circulation", Colour: maths.RGB(.3, .34, .38), Gloss: .2}},
		{&recesses, &thing.Material{Name: "OWL campus recesses and ventilation", Colour: maths.RGB(.025, .035, .045)}},
		{&glass, &thing.Material{Name: "OWL occupied control and processing rooms", Colour: maths.RGB(.02, .03, .04), Glow: maths.RGB(.32, .23, .11)}},
		{&slab, &thing.Material{Name: "OWL campus foundations and galleries", Colour: maths.RGB(.25, .27, .29)}},
		{&paint, &thing.Material{Name: "OWL campus safety markings", Colour: maths.RGB(.68, .48, .12)}},
		{&lights, &thing.Material{Name: "OWL gallery practical lights", Glow: maths.RGB(2, 1.1, .35)}},
	} {
		if err := row.mesh.placeSeparate(s, row.material); err != nil {
			return err
		}
	}
	for _, at := range []maths.Vec{maths.V(-180, 24, -165), maths.V(145, 18, -125)} {
		s.world.Lamp(emit.Lit{At: at, Strength: maths.RGB(3200, 1900, 800)})
	}
	return nil
}
