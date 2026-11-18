package searching

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// Positions retain the sourced study offsets; pads, access and small service
// structures are authored approximations guided by the supplied summit map.
var maunaKeaNeighbours = []struct {
	kind         string
	x, z, radius float64
}{
	{"subaru", -164.670, 67.512, 29},
	{"irtf", 255.825, -11.921, 19},
	{"cfht", 582.511, 95.551, 22},
	{"gemini-north", 564.620, 256.823, 23},
	{"uh-88", 524.161, 346.933, 18},
	{"ukirt", 430.727, 409.156, 18},
	{"jcmt", -267.436, 367.332, 25},
	{"sma-array", -370, 205, 43},
}

var maunaKeaRoads = [][]maths.Vec{
	{maths.V(-80, 0, -160), maths.V(-75, 0, -65), maths.V(-95, 0, 12), maths.V(-130, 0, 65), maths.V(-225, 0, 125), maths.V(-295, 0, 225), maths.V(-267, 0, 335)},
	{maths.V(-225, 0, 125), maths.V(-320, 0, 145), maths.V(-370, 0, 205)},
	{maths.V(-95, 0, 12), maths.V(-35, 0, 65), maths.V(65, 0, 100), maths.V(150, 0, 60), maths.V(255, 0, 20)},
	{maths.V(150, 0, 60), maths.V(230, 0, 190), maths.V(330, 0, 300), maths.V(430, 0, 385), maths.V(525, 0, 350), maths.V(560, 0, 260), maths.V(585, 0, 110)},
	{maths.V(330, 0, 300), maths.V(300, 0, 430), maths.V(365, 0, 540), maths.V(500, 0, 610)},
}

// maunaKeaPrecinct adds the summit neighbour sites and the authored plant.
func (s *heardStage) maunaKeaPrecinct() error {
	var slab, white, dark, metal, glass, markings siteBlocks
	// Retained neighbouring instruments establish the actual summit campus,
	// rather than inventing a row of identical domes beside the Kecks.
	for _, site := range maunaKeaNeighbours {
		at := maths.V(site.x, s.ground(site.x, site.z), site.z)
		if err := s.factory(site.kind, at, 1, 0); err != nil {
			return fmt.Errorf("summit %s: %w", site.kind, err)
		}
		r := site.radius
		slab.box(at.Add(maths.V(-r, -.22, -r)), maths.V(2*r, .24, 2*r))
	}
	// Graded service forecourt. Foundation skirts reach the actual slope,
	// while occupied rooms and rooftop plant are built on the shared pad.
	hall := func(x, z, w, h, d float64, bays int) {
		y := s.ground(x+w/2, z+d/2)
		slab.box(maths.V(x-1, y-.45, z-1), maths.V(w+2, .45, d+2))
		white.box(maths.V(x, y, z), maths.V(w, h, d))
		dark.box(maths.V(x-.25, y+h, z-.25), maths.V(w+.5, .3, d+.5))
		for i := range bays {
			bx := x + float64(i)*w/float64(bays)
			metal.box(maths.V(bx, y, z-.12), maths.V(.15, h, .2))
			dark.box(maths.V(bx+.7, y+.5, z-.1), maths.V(w/float64(bays)-1.4, h*.62, .12))
			for j := range 12 {
				metal.box(maths.V(bx+.7, y+.5+float64(j)*h*.048, z-.2), maths.V(w/float64(bays)-1.4, .05, .08))
			}
			glass.box(maths.V(bx+.9, y+h*.78, z-.16), maths.V(w/float64(bays)-1.8, .55, .2))
			white.box(maths.V(bx+.8, y+h+.3, z+2), maths.V(2.2, 1.1, 2))
			for j := range 7 {
				dark.box(maths.V(bx+.9, y+h+.4+float64(j)*.11, z+1.95), maths.V(2, .045, .1))
			}
		}
		for _, zz := range []float64{z, z + d} {
			for i := 0; i < int(w/2); i++ {
				xx := x + float64(i)*2
				ground := s.ground(xx, zz)
				slab.box(maths.V(xx, math.Min(ground, y)-.15, zz), maths.V(1.98, math.Abs(y-ground)+.2, .4))
			}
		}
	}
	hall(-16, -18, 27, 5.8, 14, 6)
	hall(-44, -12, 22, 4.6, 10, 4)
	// Loading apron, occupied-room facade details, mechanical cooling and
	// electrical distribution on the summit pad, not a fictional lower camp.
	for i := range 6 {
		x := -28 + float64(i)*5
		y := s.ground(x, -22)
		markings.box(maths.V(x, y+.4, -26), maths.V(.12, .04, 8))
	}
	for i := range 7 {
		x := -17 + float64(i)*3.4
		z := 10.0
		y := s.ground(x, z)
		slab.box(maths.V(x-.2, y, z-.2), maths.V(2.8, .2, 2.8))
		white.box(maths.V(x, y+.2, z), maths.V(2.4, 1.8, 2.4))
		for j := range 10 {
			dark.box(maths.V(x+.15, y+.35+float64(j)*.14, z-.06), maths.V(2.1, .06, .12))
		}
		metal.box(maths.V(x+.3, y+2, z+.3), maths.V(1.8, .16, 1.8))
	}
	// Dome panel seams, mounted access ladders, service hatches, plinth vents
	// and curb railings follow the real shell envelope rather than floating.
	for _, at := range []maths.Vec{maths.V(-28.669, 0, 18.299), maths.V(22.005, 0, -49.972)} {
		base := s.ground(at.X, at.Z)
		for i := range 64 {
			a := float64(i) * 2 * math.Pi / 64
			x, z := at.X+15.05*math.Cos(a), at.Z+15.05*math.Sin(a)
			dark.box(maths.V(x, base+.55, z), maths.V(.12, 1.3, .12))
			if i%4 == 0 {
				p := maths.V(at.X+17*math.Cos(a), base+.12, at.Z+17*math.Sin(a))
				metal.box(p, maths.V(.1, .95, .1))
			}
		}
		for _, x := range []float64{-7, 7} {
			for i := range 40 {
				y := 4 + float64(i)*.48
				dz := -math.Sqrt(math.Max(.1, 225-x*x-(y-13)*(y-13))) - .22
				metal.box(at.Add(maths.V(x-.55, base+y, dz)), maths.V(1.1, .07, .09))
				for _, dx := range []float64{-.55, .55} {
					metal.box(at.Add(maths.V(x+dx, base+y, dz)), maths.V(.09, .52, .09))
				}
			}
		}
		for ring := range 5 {
			y := 5 + float64(ring)*4
			r := math.Sqrt(225-(y-13)*(y-13)) + .035
			for i := range 128 {
				a := float64(i) * 2 * math.Pi / 128
				// Leave the open observing slit free of crossbars.
				if math.Sin(a) < -.86 {
					continue
				}
				metal.box(at.Add(maths.V(r*math.Cos(a), base+y, r*math.Sin(a))), maths.V(.16, .035, .16))
			}
		}
	}
	temp := &Set{World: s.world, prototypes: map[string][]thing.Thing{}}
	for i, name := range []string{"service-truck", "service-truck", "mobile-crane"} {
		x, z := -35+float64(i)*11, -28.0
		if err := temp.scenery(name, uint64(1400+i), maths.V(x, s.ground(x, z)+.4, z), 1); err != nil {
			return err
		}
	}
	for _, row := range []struct {
		mesh     *siteBlocks
		material *thing.Material
	}{
		{&slab, &thing.Material{Name: "authored summit foundation and retaining edges", Colour: maths.RGB(.3, .31, .32)}},
		{&white, &thing.Material{Name: "summit service architecture and mechanical plant", Colour: maths.RGB(.62, .66, .7)}},
		{&dark, &thing.Material{Name: "recessed summit doors and vents", Colour: maths.RGB(.025, .035, .05)}},
		{&metal, &thing.Material{Name: "Keck mounted mechanical details", Colour: maths.RGB(.33, .38, .43)}},
		{&glass, &thing.Material{Name: "occupied summit control and workshop windows", Glow: maths.RGB(.36, .21, .075)}},
		{&markings, &thing.Material{Name: "summit forecourt paint and markers", Colour: maths.RGB(.75, .72, .62)}},
	} {
		if err := row.mesh.placeSeparate(s, row.material); err != nil {
			return err
		}
	}
	return nil
}
