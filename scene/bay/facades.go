package bay

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

var wallTrim = &thing.Material{
	Name: "pale concrete cornice and pilasters", Colour: maths.RGB(.70, .67, .59),
}

var windowGrille = &thing.Material{
	Name: "dark green window grilles", Colour: maths.RGB(.105, .135, .115),
}

var roofMembrane = &thing.Material{
	Name: "warm grey flat roofing", Colour: maths.RGB(.43, .42, .38),
}

type buildingSite struct {
	made *scenery
	base maths.Vec
	turn maths.Orientation
}

// block adds a box of a size about a centre given in the building's own
// frame, turned and placed by the site.
func (site buildingSite) block(center, size maths.Vec, material *thing.Material) {
	site.made.block(site.base.Add(site.turn.Out(center)), size, site.turn, material)
}

// cellhouse adds the cellblock: a long block, a wider entrance end, and a
// lower terrace along the bay side.
func (site buildingSite) cellhouse() {
	// Long cellblock, wider entrance end, and lower terrace along the bay side.
	site.block(maths.V(-14, 5.75, 0), maths.V(114, 15.5, 39), concrete)
	site.block(maths.V(56, 6.15, 0), maths.V(30, 16.3, 41), concrete)
	site.block(maths.V(25, -.5, 21.5), maths.V(87, 3, 5), concrete)
	site.block(maths.V(70, .3, 2), maths.V(8, 2.6, 29), concrete)

	site.southWall(-71, 41, 19.5, 13.5, 20)
	site.southWall(41, 71, 20.5, 14.3, 5)
	site.block(maths.V(-15, 13.6, 0), maths.V(112.5, .2, 38.5), roofMembrane)
	site.block(maths.V(56, 14.4, 0), maths.V(30, .2, 40.5), roofMembrane)

	// Five bays mark the end facing the lighthouse. The middle opening descends
	// to the terrace, while the four outer openings keep the two window levels.
	for bay := -2; bay <= 2; bay++ {
		z := 7.1 * float64(bay)
		site.block(maths.V(71.05, 9.5, z), maths.V(.3, 4.1, 3.2), windowGrille)
	}
	for _, z := range []float64{-14.2, -7.1, 7.1, 14.2} {
		site.block(maths.V(71.05, 3.8, z), maths.V(.3, 3.3, 3.1), windowGrille)
	}
	site.block(maths.V(71.15, 3, 0), maths.V(.45, 5.6, 4.3), windowGrille)
	site.block(maths.V(71.25, 6.1, 0), maths.V(.9, .7, 6.2), wallTrim)
	site.block(maths.V(71.15, 13.6, 0), maths.V(.7, 1.3, 41.4), wallTrim)
	site.block(maths.V(-70.95, 13, 0), maths.V(.7, 1.2, 39.3), wallTrim)
}

// southWall adds a wall face from start to end along the face, to a height:
// three courses of trim, and between them a number of bays, each with two
// window grilles and a pilaster.
func (site buildingSite) southWall(start, end, face, height float64, bays int) {
	length := end - start
	middle := start + length/2
	for _, course := range []struct{ elevation, height float64 }{
		{height - .55, 1.2},
		{6.2, .45},
		{.3, 1.3},
	} {
		site.block(maths.V(middle, course.elevation, face+.16),
			maths.V(length, course.height, .6), wallTrim)
	}

	spacing := length / float64(bays)
	for bay := range bays {
		x := start + spacing*(float64(bay)+.5)
		site.block(maths.V(x, 9.45, face+.04), maths.V(3.15, 3.75, .24), windowGrille)
		site.block(maths.V(x, 3.8, face+.04), maths.V(2.65, 2.8, .24), windowGrille)
		site.block(maths.V(x+spacing*.5, 7, face+.14), maths.V(.58, 11.8, .6), wallTrim)
	}
}

// wardenHouse adds the burnt-out warden's house: open walls, with piers and
// beams that frame the void.
func (site buildingSite) wardenHouse() {
	// The fire left open concrete walls. Piers and beams frame the void through
	// the former rooms and keep the sky visible through the upper openings.
	const length, depth = 19., 13.
	site.block(maths.V(0, .15, 0), maths.V(length, 2.1, depth), concrete)
	for _, z := range []float64{-depth / 2, depth / 2} {
		for _, x := range []float64{-9, -6.2, -2.3, 2.2, 6, 9} {
			site.block(maths.V(x, 4.8, z), maths.V(1, 8.8, .9), concrete)
		}
		for _, y := range []float64{3.2, 6.1, 9.2} {
			site.block(maths.V(0, y, z), maths.V(length, .78, 1.1), wallTrim)
		}
	}
	for _, x := range []float64{-9, 9} {
		site.block(maths.V(x, 4.8, 0), maths.V(.85, 8.8, 1.8), concrete)
		for _, y := range []float64{3.2, 6.1, 9.2} {
			site.block(maths.V(x, y, 0), maths.V(.9, .78, depth), concrete)
		}
	}
}

var brickCasemate = &thing.Material{
	Name: "weathered brick casemates", Colour: maths.RGB(.36, .27, .22),
}

// barracks adds a barracks: a brick casemate under a walled storey, with a
// hipped roof.
func (site buildingSite) barracks(building building) {
	length, depth, height := building.length, building.deep, building.height
	site.block(maths.V(0, -.15, 0), maths.V(length, 4, depth), brickCasemate)
	site.block(maths.V(0, (height+1.85)/2, 0), maths.V(length, height-1.85, depth), concrete)
	site.hipRoof(length+1, depth+1, height, 3)

	for _, y := range []float64{3.7, 7.15, 10.6} {
		for bay := -8; bay <= 8; bay++ {
			x := 4.22 * float64(bay)
			for _, z := range []float64{-depth / 2, depth / 2} {
				site.block(maths.V(x, y, z), maths.V(1.25, 2.4, .35), windowGrille)
			}
		}
		for _, z := range []float64{-5.5, 0, 5.5} {
			site.block(maths.V(length/2, y, z), maths.V(.35, 2.4, 1.25), windowGrille)
		}
	}
	site.block(maths.V(0, 2.2, -depth/2-.65), maths.V(length, .55, 2.1), wallTrim)
	site.block(maths.V(length/2+.65, 2.2, 0), maths.V(2.1, .55, depth), wallTrim)
}

// hipRoof adds a hipped roof of a length and depth at height y, its ridge
// rising by rise.
func (site buildingSite) hipRoof(length, depth, y, rise float64) {
	x, z := length/2, depth/2
	ridge := x - z
	points := []maths.Vec{
		maths.V(-x, 0, -z), maths.V(x, 0, -z),
		maths.V(x, 0, z), maths.V(-x, 0, z),
		maths.V(-ridge, rise, 0), maths.V(ridge, rise, 0),
	}
	faces := [][3]int{
		{0, 1, 2}, {0, 2, 3},
		{0, 4, 5}, {0, 5, 1},
		{1, 5, 2}, {2, 5, 4}, {2, 4, 3}, {3, 4, 0},
	}
	solid, err := mesh.New(points, faces, nil, false)
	base := site.base.Add(maths.V(0, y, 0))
	site.made.add(solid, maths.NewSubspace(base, site.turn, 1), roofMembrane, err)
}
