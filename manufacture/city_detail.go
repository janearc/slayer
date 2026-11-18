package manufacture

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// cityStreetLevel is the city's level for looking from the street. It
// is manufacture's finest level, which the city refused before.
const cityStreetLevel = "demo"

// cityPlant is the material slot of rooftop plant and crown louvres.
const cityPlant slot = 14

// cityFaces is a building's four faces.
const cityFaces = 4

// cityPlantRowGaps is the gaps two rows of plant need: behind the back
// row, between the rows and in front of the front row.
const cityPlantRowGaps = 3

// cityLevels is the city's representations, coarsest first.
func cityLevels() []string {
	return []string{"hull", "quick", cityStreetLevel}
}

// cityLevel reports whether the city makes a representation.
func cityLevel(detail string) bool {
	return slices.Contains(cityLevels(), detail)
}

// cityStack is one block of a building's massing: its outer footprint,
// the floors it holds, and whether it stands on the ground.
type cityStack struct {
	width, depth float64
	from, to     int
	ground       bool
}

// cityDetail gathers one street-level building: its design and seed,
// where it stands, whether its windows are lit, the batch its shared
// modules go to, its own solids, and its panes by material.
type cityDetail struct {
	design cityDesign
	own    seed
	place  maths.Subspace
	night  bool
	batch  *cityBatch
	solid  instrumentAssembly
	panes  map[slot]surface
}

// addBuilding puts one lot's building in the batch at a level. Hull and
// quick take the path they always took, unchanged.
func (batch *cityBatch) addBuilding(building CityBuilding, detail string,
	night bool) {
	if detail != cityStreetLevel {
		batch.add(districtBuildingParts(building, detail, night),
			building.Place)
		return
	}
	own := sow(building.Seed)
	batch.detailed(cityBuildingDesign(building.Kind, own), own,
		building.Place, night)
}

// detailed models one building for the street: its stacks, their
// facades, and the roof of the highest.
func (batch *cityBatch) detailed(design cityDesign, own seed,
	place maths.Subspace, night bool) {
	building := &cityDetail{design: design, own: own, place: place,
		night: night, batch: batch, panes: map[slot]surface{}}
	stacks := design.stacks()
	for _, stack := range stacks {
		building.stack(stack)
	}
	building.roof(stacks[len(stacks)-1])
	batch.add(building.parts(), place)
}

// share places a module where a building stands, or nothing once the
// batch has failed.
func (batch *cityBatch) share(material slot, made *mesh.Mesh,
	place maths.Subspace) {
	if batch.trouble != nil {
		return
	}
	batch.members[material] = append(batch.members[material],
		shape.MeshMember{Mesh: made, Space: place})
	batch.counts[material]++
}

// family is the design's family name.
func (design cityDesign) family() string {
	switch {
	case design.gable:
		return "city-rowhouse"
	case design.tower:
		return "city-tower"
	}
	return "city-apartment"
}

// storey is the design's height of one floor.
func (design cityDesign) storey() float64 {
	return design.height / float64(design.floors)
}

// stacks is the design's massing. A rowhouse is one stack under its
// gable. A flat roof's top storey is its plant storey. A tower sets
// back from its podium to its shaft, and a tall apartment sets back
// for its highest windowed floor.
func (design cityDesign) stacks() []cityStack {
	w, d, top := design.width, design.depth, design.floors-1
	ground := cityStack{width: w, depth: d, to: design.floors, ground: true}
	switch {
	case design.gable:
		return []cityStack{ground}
	case design.tower:
		podium := max(1, int(math.Round(float64(design.floors)*
			cityValue("podium-height-share"))))
		shaft := cityValue("shaft-width-share")
		ground.to = podium
		return []cityStack{ground, {width: w * shaft, depth: d * shaft,
			from: podium, to: top}}
	case design.floors < int(cityValue("setback-floors")):
		ground.to = top
		return []cityStack{ground}
	}
	inset := cityWhole * cityValue("setback-depth")
	ground.to = top - 1
	return []cityStack{ground, {width: w - inset, depth: d - inset,
		from: top - 1, to: top}}
}

// planes is the half-width and half-depth of a stack's wall planes,
// set in so slab edges end at the stack's footprint.
func (stack cityStack) planes() (float64, float64) {
	inset := cityValue("facade-projection")
	return stack.width*spaceHalf - inset, stack.depth*spaceHalf - inset
}

// bays is how many whole bays a face of the stack holds: a side face
// runs along the depth, the front and back along the width.
func (stack cityStack) bays(side bool) int {
	return int(cityWhole * stack.half(side) / cityValue("bay-pitch"))
}

// spare is what a face's bays leave at each of its ends.
func (stack cityStack) spare(side bool) float64 {
	return stack.half(side) - float64(stack.bays(side))*
		cityValue("bay-pitch")*spaceHalf
}

// half is half the length of a face's wall plane.
func (stack cityStack) half(side bool) float64 {
	across, along := stack.planes()
	if side {
		return along
	}
	return across
}
