package seaside

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/internal/beams"
	"github.com/janearc/slayer/scene/thing"
)

// solid adds one surface to the study.
func (s *structures) solid(part thing.Thing, err error) {
	s.add([]thing.Thing{part}, err)
}

// house builds walls, glazing and a gabled roof above the deck.
func (s *structures) house(base maths.Vec, width, height, depth float64,
	wall, roof *thing.Material) {
	s.solid(beams.Box(base.Add(maths.V(0, height*.5, 0)),
		maths.V(width, height, depth), maths.Identity, wall))
	x, z, rise := width*.55, depth*.55, width*.21
	verts := []maths.Vec{
		maths.V(-x, 0, -z), maths.V(x, 0, -z), maths.V(0, rise, -z),
		maths.V(-x, 0, z), maths.V(x, 0, z), maths.V(0, rise, z),
	}
	faces := [][3]int{{0, 2, 1}, {3, 4, 5}, {0, 3, 5}, {0, 5, 2},
		{2, 5, 4}, {2, 4, 1}, {0, 1, 4}, {0, 4, 3}}
	solid, err := mesh.New(verts, faces, nil, false)
	s.solid(thing.Thing{Shape: solid, Material: roof, Motion: thing.Still(
		maths.NewSubspace(base.Add(maths.V(0, height, 0)), maths.Identity, 1))}, err)
	for _, side := range [...]float64{-1, 1} {
		for along := -depth * .34; along < depth*.41; along += 3.6 {
			at := base.Add(maths.V(side*(width*.5+.035), height*.56, along))
			s.solid(beams.Box(at, maths.V(.075, height*.41, 1.65), maths.Identity, glass))
		}
	}
}

// coaster stages a stylized sequence of track rises behind the Pacific Wheel.
func (s *structures) coaster(deck float64) {
	const spans = 56
	local := func(i int) maths.Vec {
		t := float64(i) / spans
		return maths.V(40+4*math.Sin(t*4*math.Pi),
			deck+3+10*math.Pow(.5+.5*math.Sin(t*5*math.Pi), 2),
			-169-160*t)
	}
	for i := range spans {
		p, q := local(i), local(i+1)
		for _, x := range [...]float64{-.67, .67} {
			offset := maths.V(x, 0, 0)
			s.solid(beams.Rod(p.Add(offset), q.Add(offset), .20, red))
		}
		if i%3 == 0 {
			s.solid(beams.Rod(maths.V(p.X, deck, p.Z), p, .27, blue))
		}
	}
}

type sand struct {
	shingle bool
}

// At shades the procedural shore from damp sand to backshore vegetation.
func (paint sand) At(p maths.Vec) maths.Light {
	low := maths.RGB(.49, .38, .24)
	high := maths.RGB(.69, .56, .38)
	if paint.shingle {
		low = maths.RGB(.40, .33, .27)
		high = maths.RGB(.53, .44, .36)
	}
	wet := maths.Smooth(.05, 2.1, p.Y)
	bank := maths.Smooth(4, 13, p.Y)
	base := low.Scale(1 - wet).Add(high.Scale(wet))
	base = base.Scale(1 - bank).Add(maths.RGB(.35, .39, .21).Scale(bank))
	return base.Scale(.92 + .12*noise.Gradient(318, p.X*1.2, p.Z*1.2, .2))
}

// shore makes the compositional beach relief behind the architecture study.
func (s *structures) shore(name string) {
	const across, down = 301, 151
	points := make([]maths.Vec, 0, across*down)
	faces := make([][3]int, 0, 2*(across-1)*(down-1))
	for row := range down {
		z := -455 + 12*float64(row)
		for col := range across {
			x := -3600 + 24*float64(col)
			shoreline := 5*math.Sin(x/122) + 10*math.Sin(x/307)
			inland := z - shoreline
			y := inland * .068
			if inland < -65 {
				y -= math.Pow((-65-inland)/31, 1.7)
			}
			if inland > 55 {
				y = 3.74 + .023*(inland-55) +
					4*math.Max(0, math.Sin((inland-55)/105))*
						(.4+.6*math.Pow(math.Cos(x/282), 2))
			}
			points = append(points, maths.V(x, y, z))
			if row > 0 && col > 0 {
				i := row*across + col
				faces = append(faces, [3]int{i - across - 1, i, i - across},
					[3]int{i - across - 1, i - 1, i})
			}
		}
	}
	solid, err := mesh.New(points, faces, nil, false)
	s.solid(thing.Thing{Shape: solid, Motion: thing.Still(maths.Local()),
		Material: &thing.Material{Name: "coastal study beach",
			Texture: sand{shingle: name == "brighton"}}}, err)
}
