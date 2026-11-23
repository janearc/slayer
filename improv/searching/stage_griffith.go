package searching

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// griffithArchitecture follows the supplied photographs: an elevated central
// rotunda, stepped Art Deco wings and copper domes above a landscaped public
// forecourt. Dimensions and small details are authored, not survey geometry.
func (s *heardStage) griffithArchitecture() error {
	var stone, cornice, copper, glazing, shadow, paving, grass, metal siteBlocks
	y := s.ground(0, 0)
	put := func(batch *siteBlocks, x, h, z, w, height, d float64) {
		batch.box(maths.V(x, y+h, z), maths.V(w, height, d))
	}
	put(&stone, -49, 0, -10, 98, 11, 25)
	put(&stone, -18, 0, -20, 36, 17, 40)
	for _, x := range []float64{-46, 29} {
		put(&stone, x, 11, -7, 17, 2, 18)
	}
	stone.round(maths.V(0, y+12, 0), 18, 18, 9, 96)
	copper.cap(maths.V(0, y+21, 0), 17, 11, 96, 16)
	for _, x := range []float64{-37, 37} {
		stone.round(maths.V(x, y+8, 2), 10.5, 10.5, 7, 64)
		copper.cap(maths.V(x, y+15, 2), 10.7, 7, 64, 12)
		// Paired shutter rails make the side domes read as observing rooms.
		for _, dx := range []float64{-.45, .45} {
			put(&metal, x+dx, 15, -8.6, .12, 5.8, 1)
		}
		for j := range 12 {
			put(&shadow, x-8.7, 9+float64(j)*.38, -3, .15, .08, 9)
		}
	}
	// Stepped courses, recesses and piers break the long facade into actual
	// architectural depth rather than printed light stripes.
	for i := range 22 {
		x := -47 + float64(i)*4.4
		if math.Abs(x) < 13 {
			continue
		}
		put(&stone, x, 0, -11.2, .8, 12, 1.2)
		put(&cornice, x-.35, 10.2, -11.65, 1.5, .55, 1.65)
		put(&shadow, x+1.25, 2, -10.12, 1.65, 6.8, .18)
		put(&glazing, x+1.45, 2.3, -10.22, 1.25, 6.2, .15)
		for j := range 5 {
			put(&metal, x+1.4, 2.4+float64(j)*1.15, -10.35, 1.35, .08, .1)
		}
	}
	for level := range 4 {
		h := 10.8 + float64(level)*.42
		put(&cornice, -49.4, h, -10.6, 98.8, .22, 26.2)
	}
	// Main entrance: deep vertical portal, stepped surround and narrow
	// rotunda glazing beneath the copper dome.
	put(&shadow, -5, 0, -20.1, 10, 11.8, .25)
	put(&metal, -4.3, 0, -20.3, 8.6, 10.6, .3)
	for i := range 7 {
		put(&glazing, -3.9+float64(i)*1.15, .7, -20.5, .65, 9.2, .2)
	}
	for i := range 4 {
		w := 12 + float64(i)*2.6
		h := 12 + float64(i)*1.2
		for _, sign := range []float64{-1, 1} {
			put(&stone, sign*w/2-.35, 0, -20.7+float64(i)*.6, .7, h, .65)
		}
		put(&cornice, -w/2, h, -20.7+float64(i)*.6, w, .55, .8)
	}
	for i := range 48 {
		a := float64(i) * 2 * math.Pi / 48
		if math.Sin(a) < -.4 && math.Abs(math.Cos(a)) < .65 {
			continue
		}
		p := maths.V(18.08*math.Cos(a), y+16, 18.08*math.Sin(a))
		glazing.box(p, maths.V(.4, 2.3, .4))
	}
	// Terrace, broad steps, balustrades and two lawns frame a central walk
	// and astronomers' monument, matching the recognisable public approach.
	put(&paving, -57, -.25, -29, 114, .4, 65)
	put(&paving, -53, -.12, -103, 106, .25, 76)
	put(&grass, -43, .15, -92, 34, .12, 50)
	put(&grass, 9, .15, -92, 34, .12, 50)
	put(&paving, -5, .2, -98, 10, .12, 72)
	for i := range 12 {
		put(&stone, -17, -float64(i)*.14, -28-float64(i)*.9, 34, .2, .95)
	}
	for _, x := range []float64{-55, 55} {
		for j := range 30 {
			put(&stone, x, 0, -27+float64(j)*2.1, .28, 1.15, .28)
		}
		put(&cornice, x-.1, 1.15, -27, .5, .18, 62)
	}
	for _, z := range []float64{-28, 34} {
		for i := range 48 {
			put(&stone, -55+float64(i)*2.3, 0, z, .27, 1.15, .27)
		}
		put(&cornice, -55, 1.15, z, 110, .18, .45)
	}
	for i := range 3 {
		put(&stone, -4-float64(i), .3+float64(2-i)*.45, -78-float64(i), 8+float64(i)*2, .45, 8+float64(i)*2)
	}
	stone.round(maths.V(0, y+1.8, -74), 1.5, .45, 14, 8)
	copper.cap(maths.V(0, y+15.8, -74), .6, .85, 12, 3)
	// Low architectural uplighting and path lamps stay below the eye.
	for _, x := range []float64{-24, 24} {
		s.world.Lamp(emit.Lit{At: maths.V(x, y+.5, -25), Strength: maths.RGB(460, 330, 190)})
	}
	for _, x := range []float64{-48, 48} {
		for j := range 8 {
			z := -96 + float64(j)*9
			put(&metal, x, .3, z, .13, 2.5, .13)
			put(&glazing, x-.25, 2.8, z-.25, .5, .48, .5)
		}
	}
	for _, row := range []struct {
		mesh *siteBlocks
		mat  *thing.Material
	}{
		{&stone, &thing.Material{Name: "authored Griffith limestone and Art Deco facade", Colour: maths.RGB(.69, .66, .57)}},
		{&cornice, &thing.Material{Name: "Griffith stepped cornices and balustrade coping", Colour: maths.RGB(.8, .77, .68)}},
		{&copper, &thing.Material{Name: "Griffith weathered copper observing domes", Colour: maths.RGB(.17, .19, .15), Gloss: .35}},
		{&glazing, &thing.Material{Name: "Griffith warm occupied windows and path lamps", Glow: maths.RGB(.48, .29, .13)}},
		{&shadow, &thing.Material{Name: "Griffith recessed windows and portal", Colour: maths.RGB(.018, .026, .029)}},
		{&paving, &thing.Material{Name: "Griffith forecourt and terrace paving", Colour: maths.RGB(.34, .33, .3)}},
		{&grass, &thing.Material{Name: "Griffith twin landscaped lawns", Colour: maths.RGB(.075, .16, .045)}},
		{&metal, &thing.Material{Name: "Griffith doors glazing bars and lamp posts", Colour: maths.RGB(.08, .105, .095)}},
	} {
		if err := row.mesh.placeSeparate(s, row.mat); err != nil {
			return err
		}
	}
	temp := &set{World: s.world, prototypes: map[string][]thing.Thing{}}
	for i := range 230 {
		a := float64(i) * 2.399963229728653
		r := 75 + 6*math.Sqrt(float64(i))
		x, z := r*math.Cos(a), r*math.Sin(a)-20
		if math.Abs(x) < 61 && z > -109 && z < 40 {
			continue
		}
		name := "chaparral-brush"
		size := .6 + .8*noise.Hash(2601, int64(i))
		if i%7 == 0 {
			name = "valley-oak"
			size = .65 + .35*noise.Hash(2602, int64(i))
		}
		if i%3 == 0 {
			name = "manzanita"
		}
		if err := temp.scenery(name, uint64(2600+i%8), maths.V(x, s.ground(x, z), z), size); err != nil {
			return err
		}
	}
	return s.losAngelesBasin()
}

// round is an independent closed cylindrical or tapered architectural part.
func (b *siteBlocks) round(at maths.Vec, low, high, height float64, count int) {
	first := len(b.vertices)
	for _, row := range [][2]float64{{low, 0}, {high, height}} {
		for i := range count {
			a := float64(i) * 2 * math.Pi / float64(count)
			b.vertices = append(b.vertices, at.Add(maths.V(row[0]*math.Cos(a), row[1], row[0]*math.Sin(a))))
		}
	}
	bottom := len(b.vertices)
	b.vertices = append(b.vertices, at, at.Add(maths.V(0, height, 0)))
	for i := range count {
		j := (i + 1) % count
		lo, lj, hi, hj := first+i, first+j, first+count+i, first+count+j
		b.faces = append(b.faces, [3]int{lo, hi, lj}, [3]int{lj, hi, hj}, [3]int{bottom, lo, lj}, [3]int{bottom + 1, hj, hi})
	}
}

// cap adds a domed cap of the given radius and rows of faces at a position.
func (b *siteBlocks) cap(at maths.Vec, radius, rise float64, count, rows int) {
	first := len(b.vertices)
	for row := range rows {
		a := float64(row) * math.Pi / (2 * float64(rows))
		for i := range count {
			angle := float64(i) * 2 * math.Pi / float64(count)
			b.vertices = append(b.vertices, at.Add(maths.V(radius*math.Cos(a)*math.Cos(angle), rise*math.Sin(a), radius*math.Cos(a)*math.Sin(angle))))
		}
	}
	pole := len(b.vertices)
	b.vertices = append(b.vertices, at.Add(maths.V(0, rise, 0)), at)
	for row := 0; row < rows-1; row++ {
		for i := range count {
			j := (i + 1) % count
			a, c := first+row*count+i, first+row*count+j
			b.faces = append(b.faces, [3]int{a, a + count, c}, [3]int{c, a + count, c + count})
		}
	}
	for i := range count {
		j := (i + 1) % count
		a, c := first+(rows-1)*count+i, first+(rows-1)*count+j
		b.faces = append(b.faces, [3]int{a, pole, c}, [3]int{pole + 1, first + i, first + j})
	}
}
