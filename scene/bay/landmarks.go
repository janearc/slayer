package bay

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/internal/beams"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
)

type building struct {
	name         string
	east, north  float64
	length, deep float64
	height, yaw  float64
}

var majorBuildings = []building{
	{"Cellhouse", 550786, 4186741, 142, 41, 14, -.67},
	{"Building 64 barracks", 550893, 4186746, 77, 19, 13, -.56},
	{"New Industries", 550628, 4186830, 103, 18, 10, -1.08},
	{"Model Industries", 550580, 4186892, 53, 23, 13, -.15},
	{"Powerhouse", 550676, 4186891, 42, 23, 14, -.67},
}

// landmarks is the island's major buildings, each on the ground height at
// its place, with the lighthouse and the water tower. It refuses a building
// that has no land under it.
func landmarks(raster *terrain.Raster, grid Grid) ([]thing.Thing, error) {
	var scenery scenery
	for _, building := range majorBuildings {
		base, err := landmarkHeight(raster, building.east, building.north)
		if err != nil {
			return nil, fmt.Errorf("alcatraz %s: %w", building.name, err)
		}
		site := buildingSite{
			made: &scenery,
			base: grid.at(building.east, building.north, base),
			turn: maths.Rotation(maths.Up, building.yaw),
		}
		switch building.name {
		case "Cellhouse":
			site.cellhouse()
		case "Building 64 barracks":
			site.barracks(building)
		default:
			site.building(building)
		}
	}
	lighthouse, err := landmarkHeight(raster, lighthouseEast, lighthouseNorth)
	if err != nil {
		return nil, fmt.Errorf("alcatraz lighthouse: %w", err)
	}
	scenery.lighthouse(grid.at(lighthouseEast, lighthouseNorth, lighthouse))
	const wardenEast, wardenNorth = 550863., 4186677.
	warden, err := landmarkHeight(raster, wardenEast, wardenNorth)
	if err != nil {
		return nil, fmt.Errorf("alcatraz warden house: %w", err)
	}
	wardenSite := buildingSite{&scenery, grid.at(wardenEast, wardenNorth, warden),
		maths.Rotation(maths.Up, -.65)}
	wardenSite.wardenHouse()
	const towerEast, towerNorth = 550709., 4186838.
	tower, err := landmarkHeight(raster, towerEast, towerNorth)
	if err != nil {
		return nil, fmt.Errorf("alcatraz water tower: %w", err)
	}
	scenery.waterTower(grid.at(towerEast, towerNorth, tower))
	// The landing apron lies on the northeast shore below the barracks.
	apron, err := beams.Span(grid.at(550918, 4186729, grid.SeaElevation),
		grid.at(550935, 4186755, grid.SeaElevation), 13, 2.7, .82, 1.1,
		pierConcrete)
	if err != nil {
		return nil, err
	}
	scenery.parts = append(scenery.parts, apron)
	return scenery.parts, scenery.err
}

// landmarkHeight is the ground height at an east and north. It refuses a
// place with no height or none above sea level.
func landmarkHeight(raster *terrain.Raster, east, north float64) (float64, error) {
	elevation, present, err := raster.Elevation(east, north)
	if err != nil {
		return 0, err
	}
	if !present || elevation <= 0 {
		return 0, fmt.Errorf("no positive land height at %.2f E %.2f N", east, north)
	}
	return elevation, nil
}

var concrete = &thing.Material{
	Name: "sun-weathered concrete", Colour: maths.RGB(.58, .56, .49),
}

var roof = &thing.Material{
	Name: "weathered roof and iron", Colour: maths.RGB(.30, .30, .28),
}

var pierConcrete = &thing.Material{
	Name: "grey concrete pier", Colour: maths.RGB(.40, .39, .34),
}

type scenery struct {
	parts []thing.Thing
	err   error
}

// add keeps a solid as a still part in a material. It keeps the first error
// it is given, and adds nothing after one.
func (made *scenery) add(solid *mesh.Mesh, at maths.Subspace, worn *thing.Material, err error) {
	if made.err != nil {
		return
	}
	if err != nil {
		made.err = err
		return
	}
	made.parts = append(made.parts, thing.Thing{
		Shape: solid, Material: worn, Motion: thing.Still(at),
	})
}

// block adds a box of a size at a centre, turned by an orientation, in a
// material.
func (made *scenery) block(at, size maths.Vec, turn maths.Orientation, worn *thing.Material) {
	half := size.Scale(.5)
	x, y, z := half.X, half.Y, half.Z
	points := []maths.Vec{
		maths.V(-x, -y, -z), maths.V(x, -y, -z),
		maths.V(x, y, -z), maths.V(-x, y, -z),
		maths.V(-x, -y, z), maths.V(x, -y, z),
		maths.V(x, y, z), maths.V(-x, y, z),
	}
	faces := [][3]int{{0, 3, 2}, {0, 2, 1}, {4, 5, 6}, {4, 6, 7},
		{0, 4, 7}, {0, 7, 3}, {1, 2, 6}, {1, 6, 5},
		{0, 1, 5}, {0, 5, 4}, {3, 7, 6}, {3, 6, 2}}
	solid, err := mesh.New(points, faces, nil, false)
	made.add(solid, maths.NewSubspace(at, turn, 1), worn, err)
}

// building adds a plain building: a block of its length, depth and height
// with a flat roof slab on top.
func (site buildingSite) building(building building) {
	height := building.height
	site.block(maths.V(0, height/2-1, 0),
		maths.V(building.length, height+2, building.deep), concrete)
	site.block(maths.V(0, height+.4, 0),
		maths.V(building.length+1.0, .8, building.deep+1.0), roof)
}

var lighthouseMasonry = &thing.Material{
	Name: "pale lighthouse masonry", Colour: maths.RGB(.79, .77, .69),
}

var lantern = &thing.Material{
	Name: "dark lighthouse lantern", Colour: maths.RGB(.095, .13, .14),
}

// lighthouse adds the lighthouse on a base point: a footing, a tapering
// tower, its gallery, roof and lantern.
func (made *scenery) lighthouse(base maths.Vec) {
	made.frustum(base.Add(maths.V(0, -1, 0)), 3.5, 3.5, 2.2, concrete)
	made.frustum(base.Add(maths.V(0, 1.2, 0)), 2.75, 1.65, 22.2, lighthouseMasonry)
	made.frustum(base.Add(maths.V(0, 23.4, 0)), 1.8, 2.65, .7, lighthouseMasonry)
	made.frustum(base.Add(maths.V(0, 24.1, 0)), 2.65, 2.65, .55, roof)
	made.frustum(base.Add(maths.V(0, 24.65, 0)), 1.76, 1.76, 2.55, lantern)
	made.frustum(base.Add(maths.V(0, 27.2, 0)), 2.22, .18, 1.8, roof)
}

// waterTower adds the water tower on a base point: four legs, braced, under
// a tank.
func (made *scenery) waterTower(base maths.Vec) {
	corners := [...]maths.Vec{
		maths.V(-6, 0, -6), maths.V(6, 0, -6),
		maths.V(6, 0, 6), maths.V(-6, 0, 6),
	}
	legAt := func(corner maths.Vec, height float64) maths.Vec {
		return base.Add(corner.Scale(1 - .28*height/22)).Add(maths.V(0, height, 0))
	}
	for index, corner := range corners {
		next := corners[(index+1)%len(corners)]
		made.post(legAt(corner, -1.5), legAt(corner, 22), .78, roof)
		for lower := 1.; lower < 20; lower += 7 {
			upper := lower + 7
			made.post(legAt(corner, lower), legAt(next, upper), .42, roof)
			made.post(legAt(next, lower), legAt(corner, upper), .42, roof)
			made.post(legAt(corner, upper), legAt(next, upper), .54, roof)
		}
	}
	made.frustum(base.Add(maths.V(0, 19.8, 0)), 3.45, 6.3, 3.2, roof)
	made.frustum(base.Add(maths.V(0, 23, 0)), 6.3, 6.3, 4.55, roofMembrane)
	made.frustum(base.Add(maths.V(0, 23, 0)), 6.47, 6.47, .46, roof)
	made.frustum(base.Add(maths.V(0, 27.55, 0)), 6.52, .24, 1.1, roof)
}

// frustum adds an eight-sided cone with its top cut off: a lower and an
// upper radius over a height, standing on a base point.
func (made *scenery) frustum(base maths.Vec, lower, upper, height float64, worn *thing.Material) {
	const sides = 8
	points := []maths.Vec{{}, maths.V(0, height, 0)}
	for corner := range sides {
		theta := 2 * math.Pi * float64(corner) / sides
		x, z := math.Cos(theta), math.Sin(theta)
		points = append(points, maths.V(lower*x, 0, lower*z),
			maths.V(upper*x, height, upper*z))
	}
	var faces [][3]int
	for corner := range sides {
		a, b := 2+2*corner, 2+2*((corner+1)%sides)
		faces = append(faces, [3]int{a, a + 1, b + 1}, [3]int{a, b + 1, b},
			[3]int{0, a, b}, [3]int{1, b + 1, a + 1})
	}
	solid, err := mesh.New(points, faces, nil, false)
	made.add(solid, maths.NewSubspace(base, maths.Identity, 1), worn, err)
}

// post adds a square post of a width from one point to another.
func (made *scenery) post(from, to maths.Vec, width float64, worn *thing.Material) {
	way := to.Sub(from)
	made.block(from.Add(to).Scale(.5), maths.V(width, way.Len(), width),
		maths.Upright(way), worn)
}
