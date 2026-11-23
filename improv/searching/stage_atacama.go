package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// atacamaSite is authored dry high-desert scenery for the OWL concept,
// not a DEM or a claim that this fictional facility was built in Chile.
func (s *heardStage) atacamaSite() error {
	s.ground = func(x, z float64) float64 {
		grade := math.Max(math.Abs(x)/360, math.Abs(z-120)/450)
		blend := maths.Smooth(.92, 1.25, grade)
		hills := 100*noise.Fractal(921, x/650, 0, z/650, 4) + 65*math.Sin(x/1800)*math.Sin(z/1100)
		return blend * (hills - 25)
	}
	const side = 257
	const step = 50.0
	points := make([]maths.Vec, side*side)
	normals := make([]maths.Vec, len(points))
	faces := make([][3]int, 0, 2*(side-1)*(side-1))
	for row := range side {
		for column := range side {
			x, z := (float64(column)-128)*step, (float64(row)-128)*step
			index := row*side + column
			points[index] = maths.V(x, s.ground(x, z), z)
			normals[index] = maths.V((s.ground(x-5, z)-s.ground(x+5, z))/10, 1, (s.ground(x, z-5)-s.ground(x, z+5))/10)
			if row < side-1 && column < side-1 {
				a := row*side + column
				b := a + side
				faces = append(faces, [3]int{a, b + 1, a + 1}, [3]int{a, b, b + 1})
			}
		}
	}
	mesh, err := shape.NewMesh(points, faces, normals, true)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: mesh, Material: &thing.Material{Name: "authored dry-desert salts alluvium and volcanic strata", Texture: atacamaAlbedo{}}})
	s.world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Name: "authored low desert beyond site model", Colour: maths.RGB(.08, .09, .12)}, Motion: thing.Still(maths.NewSubspace(maths.V(0, -600, 0), maths.Identity, 1))})
	temp := &set{World: s.world, prototypes: map[string][]thing.Thing{}}
	for i := range 150 {
		angle := float64(i) * 2.399963229728653
		radius := 280 + 40*math.Sqrt(float64(i+1))
		x, z := radius*math.Cos(angle), radius*math.Sin(angle)
		if math.Abs(x) < 310 && z > -290 && z < 500 {
			continue
		}
		size := .7 + 3.5*math.Pow(math.Abs(math.Sin(float64(i)*1.79)), 3)
		if err := temp.scenery("boulder", uint64(940+i), maths.V(x, s.ground(x, z), z), size); err != nil {
			return err
		}
	}
	return s.desertServiceYard()
}

// atacamaAlbedo keeps large, resolvable lithology variation and fine grain.
// The pale pan, purple-grey volcanic ground and rust strata are invented.
type atacamaAlbedo struct{}

// At returns the surface colour of the Atacama ground at p.
func (atacamaAlbedo) At(p maths.Vec) maths.Light {
	pan := math.Exp(-math.Pow((p.X+850)/1150, 2) - math.Pow((p.Z-1300)/620, 2))
	rock := .5 + .5*noise.Fractal(931, p.X/180, 0, p.Z/200, 3)
	grain := .88 + .12*noise.Fractal(932, p.X*1.7, p.Y*1.7, p.Z*1.7, 3)
	dark := maths.RGB(.085, .09, .125).Scale(.7 + .6*rock)
	red := maths.RGB(.25, .14, .115)
	ground := dark.Scale(1 - rock*.65).Add(red.Scale(rock * .65))
	salt := maths.RGB(.68, .7, .65)
	return ground.Scale(1 - pan*.9).Add(salt.Scale(pan * .9)).Scale(grain)
}

// desertServiceYard adds reasons for the site to exist: cooling plant,
// electrical distribution, maintenance storage and a physically marked road.
func (s *heardStage) desertServiceYard() error {
	var pads, plant, doors, road siteBlocks
	pads.box(maths.V(-150, -.3, -92), maths.V(95, .35, 42))
	for i := range 4 {
		x := -142 + float64(i)*20
		plant.box(maths.V(x, 0, -85), maths.V(16, 5.5, 14))
		for j := range 9 {
			doors.box(maths.V(x+.4, .5+float64(j)*.45, -85.1), maths.V(15.2, .15, .15))
		}
		plant.box(maths.V(x+5, 5.5, -81), maths.V(6, 1.3, 6))
	}
	for i := range 8 {
		plant.box(maths.V(-145+float64(i)*8, 0, -60), maths.V(4, 2.4, 3))
	}
	for i := range 60 {
		z := -105 - float64(i)*9
		x := -55 + 28*math.Sin(z/180)
		road.box(maths.V(x, s.ground(x, z)+.03, z), maths.V(.18, .07, 3.5))
	}
	for _, row := range []struct {
		mesh     *siteBlocks
		material *thing.Material
	}{
		{&pads, &thing.Material{Name: "OWL service plant slab", Colour: maths.RGB(.27, .29, .3)}},
		{&plant, &thing.Material{Name: "OWL desert services", Colour: maths.RGB(.28, .33, .39)}},
		{&doors, &thing.Material{Name: "OWL plant ventilation", Colour: maths.RGB(.035, .05, .06)}},
		{&road, &thing.Material{Name: "OWL access road markings", Colour: maths.RGB(.65, .6, .35)}},
	} {
		if err := row.mesh.place(s, row.material); err != nil {
			return err
		}
	}
	return nil
}
