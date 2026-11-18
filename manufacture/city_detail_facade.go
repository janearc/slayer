package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// stack models one block of massing: a core set back by the reveals, a
// slab edge at each floor, a strip on each face of each floor, corner
// piers for what the bays leave, and a cornice and parapet unless a
// gable sits on it.
func (building *cityDetail) stack(stack cityStack) {
	storey, design := building.design.storey(), building.design
	across, along := stack.planes()
	reveal := cityValue("reveal-depth")
	low, high := float64(stack.from)*storey, float64(stack.to)*storey
	building.box(cityWhole*(across-reveal), high-low,
		cityWhole*(along-reveal), maths.Vec{Y: low}, design.wall)
	band := storey - cityValue("module-height")
	for floor := stack.from; floor < stack.to; floor++ {
		building.box(stack.width, band, stack.depth,
			maths.Vec{Y: float64(floor) * storey}, design.wall)
		for face := range cityFaces {
			building.facade(stack, face, floor)
		}
	}
	building.corners(stack, low, high)
	if !design.gable {
		building.cornice(stack, high)
	}
}

// facade places one face's strip for one floor, an entrance on the
// ground floor's front, and glazes its openings.
func (building *cityDetail) facade(stack cityStack, face, floor int) {
	frame, bays, out := building.frame(stack, face, floor)
	key := cityModule{what: cityStripModule,
		family: building.design.family(), bays: bays}
	if stack.ground && floor == 0 && out.Z > spaceHalf {
		key.what = cityEntranceModule
	}
	building.batch.share(building.design.wall, building.batch.module(key),
		frame.Within(building.place))
	building.glaze(key, frame, face, floor)
}

// frame is a strip's frame on a face: on the wall plane, at the top of
// the floor's slab edge, centred along the face, facing out. It also
// says how many bays the face holds and which way it faces.
func (building *cityDetail) frame(stack cityStack, face,
	floor int) (maths.Subspace, int, maths.Vec) {
	turn := maths.About(maths.YAxis, float64(face)*math.Pi*spaceHalf)
	out := turn.Out(maths.ZAxis)
	side := math.Abs(out.X) > spaceHalf
	across, along := stack.planes()
	bays := stack.bays(side)
	foot := float64(floor+1)*building.design.storey() -
		cityValue("module-height")
	run := float64(bays) * cityValue("bay-pitch") * spaceHalf
	centre := maths.Vec{X: out.X * across, Y: foot, Z: out.Z * along}
	origin := centre.Add(turn.Out(maths.Vec{X: -run}))
	return maths.NewSubspace(origin, turn, spaceOne), bays, out
}

// glaze puts a pane in each opening of a placed strip, just before the
// core, in the material its seeded draw gives at night.
func (building *cityDetail) glaze(key cityModule, frame maths.Subspace,
	face, floor int) {
	z := cityValue("pane-offset") - cityValue("reveal-depth")
	for bay, opening := range key.openings() {
		role := building.role(opening, face, floor, bay)
		pane := cityQuad(
			maths.Vec{X: opening.left, Y: opening.foot, Z: z},
			maths.Vec{X: opening.right, Y: opening.foot, Z: z},
			maths.Vec{X: opening.right, Y: opening.head, Z: z},
			maths.Vec{X: opening.left, Y: opening.head, Z: z}, maths.ZAxis)
		building.panes[role] = append(building.panes[role],
			pane.placed(frame)...)
	}
}

// role is a pane's material: dark glass by day; by night a lit lobby
// behind a door, and each window's own seeded draw.
func (building *cityDetail) role(opening cityOpening, face, floor,
	bay int) slot {
	switch {
	case !building.night:
		return cityGlass
	case opening.door:
		return cityWarmWindow
	}
	return cityPaneRole(building.own, face, floor, bay)
}

// corners are four piers over the stack's height that fill what each
// face's bays leave at its ends and close the strips' ends. Each
// reaches past the reveal so it never shares the core's corner.
func (building *cityDetail) corners(stack cityStack, low, high float64) {
	reveal := cityValue("reveal-depth")
	across, along := stack.planes()
	wide := math.Max(stack.spare(false), reveal) + reveal*spaceHalf
	deep := math.Max(stack.spare(true), reveal) + reveal*spaceHalf
	for _, x := range []float64{-1, 1} {
		for _, z := range []float64{-1, 1} {
			building.box(wide, high-low, deep, maths.Vec{
				X: x * (across - wide*spaceHalf), Y: low,
				Z: z * (along - deep*spaceHalf)}, building.design.wall)
		}
	}
}

// cornice caps a stack with a projecting coping and a parapet on it.
func (building *cityDetail) cornice(stack cityStack, high float64) {
	height := cityValue("cornice-height")
	building.box(stack.width, height, stack.depth, maths.Vec{Y: high},
		cityRoof)
	across, along := stack.planes()
	building.solid.add(cityParapet{across, along},
		maths.NewSubspace(maths.Vec{Y: high + height}, maths.Identity,
			spaceOne), building.design.wall)
}

// box adds one of the building's own closed boxes, centred over a foot.
func (building *cityDetail) box(wide, tall, deep float64, foot maths.Vec,
	material slot) {
	districtBox(&building.solid, wide, tall, deep, foot, material)
}

// share places a module standing at a point in the building's frame.
func (building *cityDetail) share(material slot, key cityModule,
	at maths.Vec) {
	place := maths.NewSubspace(at, maths.Identity, spaceOne)
	building.batch.share(material, building.batch.module(key),
		place.Within(building.place))
}

// parts are the building's own solids and its panes, panes by material
// in the order the night city gives them.
func (building *cityDetail) parts() []part {
	parts := building.solid.object.parts
	for _, role := range []slot{cityGlass, cityWarmWindow, cityCoolWindow} {
		if len(building.panes[role]) > 0 {
			parts = append(parts, part{shape: cityPaneSurface{
				building.panes[role]}, space: maths.Local(), of: role})
		}
	}
	return parts
}

// cityParapet is a closed rectangular ring wall whose outer faces stand
// on the wall planes' half-width and half-depth.
type cityParapet struct{ across, along float64 }

// skin is the ring's outer and inner faces, its top and its foot.
func (ring cityParapet) skin(cut) surface {
	thick, tall := cityValue("parapet-thickness"), cityValue("parapet-height")
	corner := func(index int, inset, y float64) maths.Vec {
		x, z := ring.across-inset, ring.along-inset
		signs := [cityFaces][2]float64{{1, 1}, {1, -1}, {-1, -1}, {-1, 1}}
		sign := signs[index%cityFaces]
		return maths.Vec{X: sign[0] * x, Y: y, Z: sign[1] * z}
	}
	var skin surface
	for index := range cityFaces {
		next := index + 1
		out := corner(index, 0, 0).Add(corner(next, 0, 0))
		skin = append(skin, cityQuad(corner(index, 0, 0), corner(next, 0, 0),
			corner(next, 0, tall), corner(index, 0, tall), out)...)
		skin = append(skin, cityQuad(corner(index, thick, 0),
			corner(next, thick, 0), corner(next, thick, tall),
			corner(index, thick, tall), out.Neg())...)
		for _, y := range []float64{0, tall} {
			skin = append(skin, cityQuad(corner(index, 0, y),
				corner(next, 0, y), corner(next, thick, y),
				corner(index, thick, y), maths.Vec{Y: y - tall*spaceHalf})...)
		}
	}
	return skin
}
