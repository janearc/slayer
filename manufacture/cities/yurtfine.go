package cities

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// The roles of the small parts a yurt is drawn with at the demo level.
// What each is made of is in data.
const (
	// rolePole is a roof pole, a uni.
	rolePole partRole = "pole"
	// roleRope is a rope of hair that ties the cover down, and the darker
	// of its two strands.
	roleRope partRole = "rope"
	// roleHemp is the paler strand of a rope.
	roleHemp partRole = "hemp"
	// roleBand is a woven band round the roof or the wall.
	roleBand partRole = "band"
	// roleDoorFrame is the painted frame round the door.
	roleDoorFrame partRole = "doorframe"
	// rolePaint is the painted panels of the door.
	rolePaint partRole = "paint"
	// roleCanvas is the outer cover of the roof and the wall.
	roleCanvas partRole = "canvas"
	// roleFloor is the floor inside, and the mud that soils the canvas.
	roleFloor partRole = "floor"
	// roleCarpet is the rug on the floor.
	roleCarpet partRole = "carpet"
	// roleStove is the stove and its pipe.
	roleStove partRole = "stove"
	// roleVermilion is the red of the painted ornament.
	roleVermilion partRole = "vermilion"
	// roleSaffron is the orange of the painted ornament.
	roleSaffron partRole = "saffron"
	// roleIndigo is the blue of the painted ornament.
	roleIndigo partRole = "indigo"
)

// bothSides is two: the two ends of a chord, the two layers of a khana.
const bothSides = 2

// grainPale and grainDark are how much lighter and darker than a wood's
// own colour its early and its late wood are.
const (
	grainPale = 1.2
	grainDark = 0.6
)

// latticeSizes are the measures of the khana, the lattice wall: its slats
// and the rail round its top. Metres, but the rise, in degrees.
type latticeSizes struct {
	width, leaf, pitch, rise, railHeight, railProud float64
}

// poleSizes are the roof poles: how many and how thick.
type poleSizes struct {
	count     int
	thickness float64
}

// coverSizes are the felt and the canvas laid over the khana and the
// poles: each a ring of sectors, how many sectors a ring has and how
// many each side of the door are left out, how thick each cover is, how
// far the canvas stops above the ground, and how thick the floor is.
type coverSizes struct {
	sectors, gap               int
	felt, canvas, skirt, floor float64
}

// tieSizes are the ropes and the woven bands that tie the cover, and the
// band round the wall. Places up the roof are shares of its height.
type tieSizes struct {
	ropes, bands                                     int
	ropeFirst, ropeGap, ropeThickness                float64
	bandFirst, bandGap, bandWidth, bandProud, atWall float64
}

// toonoSizes are the crown ring's bars and spokes, in metres.
type toonoSizes struct {
	segments, spokes                  int
	bar, spokeWidth, spokeHeight, hub float64
}

// doorSizes are the frame round the door and the panels painted on it,
// in metres, and how far the leaf stands open, in degrees.
type doorSizes struct {
	bar, depth, inset, proud, open float64
	panels                         int
}

// paintSizes are the scales of the textures: the canvas's weave and mud,
// the rope's twist, the band's chevrons, the felt's fibre, the wood's
// grain and its paint, and the ornament's lozenges. Metres.
type paintSizes struct {
	canvasPitch, canvasDepth, soilHeight, soiling, streaks float64
	ropePitch, ropeLay, bandCell, fibre, grainRing, cover  float64
	doorCell, doorRing, crownCell, crownRing, rugCell      float64
}

// roomSizes are what is inside: the stove and its pipe, the rug, the
// beds and the chest. Metres, but the beds' and the chest's headings, in
// degrees from the door.
type roomSizes struct {
	stoveRadius, stoveHeight, pipeRadius, pipeRise, carpetRadius float64
	carpetHeight, bedLength, bedWidth, bedHeight, bedHeading     float64
	mattress, mattressInset                                      float64
	chestWidth, chestHeight, chestDepth, chestHeading            float64
}

// yurtFine is every measure of the demo level's small parts.
type yurtFine struct {
	lattice latticeSizes
	poles   poleSizes
	cover   coverSizes
	ties    tieSizes
	toono   toonoSizes
	door    doorSizes
	paint   paintSizes
	room    roomSizes
}

// readYurtFine is the demo level's measures from data, or why one is not
// there.
func readYurtFine() (yurtFine, error) {
	reader := &choiceReader{}
	fine := yurtFine{
		lattice: latticeSizes{reader.of("lattice-slat-width"),
			reader.of("lattice-slat-leaf"), reader.of("lattice-slat-pitch"),
			reader.of("lattice-slat-rise"),
			reader.of("lattice-rail-height"),
			reader.of("lattice-rail-proud")},
		poles: poleSizes{reader.whole("pole-count"),
			reader.of("pole-thickness")},
		cover: coverSizes{reader.whole("cover-sectors"),
			reader.whole("door-gap-sectors"), reader.of("felt-thickness"),
			reader.of("canvas-thickness"), reader.of("canvas-skirt"),
			reader.of("floor-height")},
		ties: readTies(reader), toono: readToono(reader),
		door: doorSizes{bar: reader.of("door-frame-bar"),
			depth: reader.of("door-frame-depth"),
			inset: reader.of("door-panel-inset"),
			proud: reader.of("door-panel-proud"),
			open:  reader.of("door-open"), panels: reader.whole("door-panels")},
		paint: readPaint(reader), room: readRoom(reader),
	}
	return fine, reader.trouble
}

// readTies is the ropes' and bands' measures, read from data.
func readTies(reader *choiceReader) tieSizes {
	return tieSizes{ropes: reader.whole("rope-count"),
		bands:     reader.whole("band-count"),
		ropeFirst: reader.of("rope-first"), ropeGap: reader.of("rope-gap"),
		ropeThickness: reader.of("rope-thickness"),
		bandFirst:     reader.of("band-first"),
		bandGap:       reader.of("band-gap"),
		bandWidth:     reader.of("band-width"),
		bandProud:     reader.of("band-proud"),
		atWall:        reader.of("wall-band-at")}
}

// readToono is the crown ring's measures, read from data.
func readToono(reader *choiceReader) toonoSizes {
	return toonoSizes{segments: reader.whole("toono-segments"),
		spokes: reader.whole("toono-spokes"), bar: reader.of("toono-bar"),
		spokeWidth:  reader.of("toono-spoke-width"),
		spokeHeight: reader.of("toono-spoke-height"),
		hub:         reader.of("toono-hub-radius")}
}

// readPaint is the textures' scales, read from data.
func readPaint(reader *choiceReader) paintSizes {
	return paintSizes{canvasPitch: reader.of("canvas-pitch"),
		canvasDepth: reader.of("canvas-depth"),
		soilHeight:  reader.of("canvas-soil-height"),
		soiling:     reader.of("canvas-soiling"),
		streaks:     reader.of("canvas-streaks"),
		ropePitch:   reader.of("rope-twist-pitch"),
		ropeLay:     reader.of("rope-twist-lay"),
		bandCell:    reader.of("band-cell"), fibre: reader.of("felt-fibre"),
		grainRing: reader.of("grain-ring"), cover: reader.of("paint-cover"),
		doorCell:  reader.of("ornament-cell"),
		doorRing:  reader.of("ornament-ring"),
		crownCell: reader.of("crown-ornament-cell"),
		crownRing: reader.of("crown-ornament-ring"),
		rugCell:   reader.of("rug-cell")}
}

// readRoom is the room's measures, read from data.
func readRoom(reader *choiceReader) roomSizes {
	return roomSizes{stoveRadius: reader.of("stove-radius"),
		stoveHeight: reader.of("stove-height"),
		pipeRadius:  reader.of("pipe-radius"), pipeRise: reader.of("pipe-rise"),
		carpetRadius:  reader.of("carpet-radius"),
		carpetHeight:  reader.of("carpet-height"),
		bedLength:     reader.of("bed-length"),
		bedWidth:      reader.of("bed-width"),
		bedHeight:     reader.of("bed-height"),
		bedHeading:    reader.of("bed-heading"),
		mattress:      reader.of("mattress-height"),
		mattressInset: reader.of("mattress-inset"),
		chestWidth:    reader.of("chest-width"),
		chestHeight:   reader.of("chest-height"),
		chestDepth:    reader.of("chest-depth"),
		chestHeading:  reader.of("chest-heading")}
}

// roofRadius is the radius of the roof's cone at a height above the
// ground, between the wall's top and the crown.
func (sizes yurtSizes) roofRadius(height float64) float64 {
	rise := (height - sizes.wallHeight) / (sizes.roofTop - sizes.wallHeight)
	return sizes.wallRadius + rise*(sizes.crownRadius-sizes.crownLap-
		sizes.wallRadius)
}

// yurtLooks are the materials of the demo level's parts, each textured.
type yurtLooks struct {
	canvas, felt, rope, band, pole, leaf, frame, crown, carpet *thing.Material
}

// yurtDress is a yurt's sizes, its small parts' sizes and the materials it
// reads, for drawing it part by part.
type yurtDress struct {
	sizes   yurtSizes
	fine    yurtFine
	clothes *wardrobe
	looks   yurtLooks
}

// YurtAt is the yurt drawn for a detail level. Below the demo level it is
// the yurt as it was, whole. At the demo level the khana stands inside
// felt and a canvas cover, with ropes and bands over them, the roof's poles
// run from the wall's top to a crown ring open to the sky with a stovepipe
// through it, and the open door shows a floor with a rug, a stove, beds and
// a chest. Each material is textured.
func YurtAt(level Detail) (Design, error) {
	design, err := yurt()
	if err != nil || level != demoLevel {
		return design, err
	}
	sizes, err := readSizes()
	if err != nil {
		return Design{}, err
	}
	fine, err := readYurtFine()
	if err != nil {
		return Design{}, err
	}
	dress := yurtDress{sizes: sizes, fine: fine, clothes: &wardrobe{}}
	dress.looks = dress.dressedLooks()
	parts := dress.parts()
	if dress.clothes.trouble != nil {
		return Design{}, dress.clothes.trouble
	}
	design.Massing = dress.masses()
	design.Elements = parts
	return design, nil
}

// grain is wood of a role's colour: pale early wood and dark late wood in
// rings, the grain running up.
func (dress yurtDress) grain(role partRole) texture.Texture {
	colour := dress.clothes.colour(role)
	return texture.Wood{Early: colour.Scale(grainPale),
		Late: colour.Scale(grainDark), Along: maths.YAxis,
		Ring: dress.fine.paint.grainRing}
}

// painted is wood of a role thinly painted in another's colour.
func (dress yurtDress) painted(wood, paint partRole) texture.Texture {
	return texture.Stained{Grain: dress.grain(wood),
		Paint: dress.clothes.colour(paint), Cover: dress.fine.paint.cover}
}

// inks are the three colours of the ornament, from its middle out.
func (dress yurtDress) inks() [3]maths.Light {
	return [3]maths.Light{dress.clothes.colour(roleVermilion),
		dress.clothes.colour(roleSaffron), dress.clothes.colour(roleIndigo)}
}

// dressedLooks is every material the demo level's parts are made of,
// with its texture: canvas and felt as cloth, ropes twisted, bands woven,
// wood grained under paint, and the ornament over the paint.
func (dress yurtDress) dressedLooks() yurtLooks {
	sizes, fine, cloth := dress.sizes, dress.fine, dress.clothes
	paint := fine.paint
	round := func(point maths.Vec) (float64, float64) {
		return math.Atan2(point.X, point.Z) * sizes.crownRadius, point.Y
	}
	plan := func(point maths.Vec) (float64, float64) { return point.X, point.Z }
	return yurtLooks{
		canvas: cloth.dressed(roleCanvas, "weave", texture.Canvas{
			Cloth: cloth.colour(roleCanvas), Soil: cloth.colour(roleFloor),
			Pitch: paint.canvasPitch, Depth: paint.canvasDepth,
			Radius: sizes.wallRadius, SoilHeight: paint.soilHeight,
			Soiling: paint.soiling, Streaks: paint.streaks}),
		felt: cloth.dressed(roleCover, "fibre", texture.Fibre{
			Wool: cloth.colour(roleCover), Fleck: cloth.colour(roleRope),
			Scale: paint.fibre}),
		rope: cloth.dressed(roleRope, "twist", texture.Twist{
			A: cloth.colour(roleRope), B: cloth.colour(roleHemp),
			Pitch: paint.ropePitch, Lay: paint.ropeLay}),
		band: cloth.dressed(roleBand, "chevrons", texture.Woven{
			Ground: cloth.colour(roleBand), Ink: cloth.colour(roleSaffron),
			Edge: cloth.colour(roleIndigo), Cell: paint.bandCell,
			Rise: fine.ties.bandWidth}),
		pole: cloth.dressed(rolePole, "painted grain",
			dress.painted(rolePole, roleDoorFrame)),
		leaf: cloth.dressed(roleDoor, "painted grain",
			dress.painted(roleFrame, roleDoor)),
		frame: cloth.dressed(roleDoorFrame, "painted grain",
			dress.painted(roleFrame, roleDoorFrame)),
		crown: cloth.dressed(roleCrown, "ornament", texture.Ornament{
			Under: dress.painted(roleCrown, roleDoor), Inks: dress.inks(),
			Cell: paint.crownCell, Ring: paint.crownRing, Coords: round}),
		carpet: cloth.dressed(roleCarpet, "ornament", texture.Ornament{
			Under: texture.Fibre{Wool: cloth.colour(roleCarpet),
				Fleck: cloth.colour(roleRope), Scale: paint.fibre},
			Inks: dress.inks(), Cell: paint.rugCell, Ring: paint.doorRing,
			Coords: plan}),
	}
}

// parts is everything placed on the yurt at the demo level.
func (dress yurtDress) parts() []placedElement {
	return slices.Concat(dress.lattice(), dress.rail(), dress.covers(),
		dress.poles(), dress.ties(), dress.toono(), dress.doorway(),
		dress.interior())
}

// masses is the demo yurt's one closed volume: the floor.
func (dress yurtDress) masses() []mass {
	round := outlineCircle{Radius: dress.sizes.wallRadius}
	return []mass{{Name: "floor", Role: roleFloor, Place: maths.Local(),
		Form: prism{Bottom: round, Top: round, Height: dress.fine.cover.floor}}}
}

// stand is how far the felt stands off the cone the poles lie along: the
// rail's proud at the wall's top, and half a pole's thickness at the
// crown, straight between.
func (dress yurtDress) stand(height float64) float64 {
	sizes, fine := dress.sizes, dress.fine
	share := (height - sizes.wallHeight) / (sizes.roofTop - sizes.wallHeight)
	return fine.lattice.railProud +
		share*(fine.poles.thickness/half-fine.lattice.railProud)
}

// roofLine is the radius of the inside of a layer on the roof, a distance
// above the felt's inside, at a height.
func (dress yurtDress) roofLine(above float64) func(float64) float64 {
	return func(height float64) float64 {
		return dress.sizes.roofRadius(height) + dress.stand(height) + above
	}
}

// wallLine is the radius of the inside of a layer on the wall, a distance
// above the felt's inside.
func (dress yurtDress) wallLine(above float64) func(float64) float64 {
	return func(float64) float64 {
		return dress.sizes.wallRadius + dress.fine.lattice.railProud + above
	}
}

// sector is one piece of a ring that wraps the yurt: a piece of a hollow
// cone or cylinder, a thickness through, from a height to another, drawn
// at heading nought and turned to where it goes. Its inside is at a radius
// that says where it stands at each height.
func (dress yurtDress) sector(name string, material *thing.Material,
	heading, low, high, thick float64, inner func(float64) float64) placedElement {
	step := fullTurn / float64(dress.fine.cover.sectors)
	corner := func(radius, turn, up float64) maths.Vec {
		return maths.V(radius*math.Sin(turn), up, radius*math.Cos(turn))
	}
	quad := func(height, up float64) []maths.Vec {
		in, out := inner(height), inner(height)+thick
		return []maths.Vec{corner(in, -step/half, up),
			corner(in, step/half, up), corner(out, step/half, up),
			corner(out, -step/half, up)}
	}
	return part(name, shell{Bottom: quad(low, 0), Top: quad(high, high-low)},
		material, maths.NewSubspace(maths.V(0, low, 0),
			maths.Rotation(maths.YAxis, heading), 0))
}

// ring is a sector at each heading round the yurt. Where it is asked to
// leave the doorway open, it leaves out the sectors at the door and the
// gap each side of it.
func (dress yurtDress) ring(name string, material *thing.Material,
	low, high, thick float64, inner func(float64) float64,
	open bool) []placedElement {
	cover := dress.fine.cover
	step := fullTurn / float64(cover.sectors)
	var ring []placedElement
	for index := range cover.sectors {
		if open && min(index, cover.sectors-index) <= cover.gap {
			continue
		}
		ring = append(ring, dress.sector(name, material,
			float64(index)*step, low, high, thick, inner))
	}
	return ring
}

// covers are the felt on the wall and the roof, to the ground, and the
// canvas over it, which stops short of the ground by the skirt so that
// the felt shows beneath. Each leaves the doorway open.
func (dress yurtDress) covers() []placedElement {
	sizes, cover := dress.sizes, dress.fine.cover
	top := sizes.wallHeight
	return slices.Concat(
		dress.ring("felt", dress.looks.felt, 0, top, cover.felt,
			dress.wallLine(0), true),
		dress.ring("canvas", dress.looks.canvas, cover.skirt, top,
			cover.canvas, dress.wallLine(cover.felt), true),
		dress.ring("felt", dress.looks.felt, top, sizes.roofTop, cover.felt,
			dress.roofLine(0), false),
		dress.ring("canvas", dress.looks.canvas, top, sizes.roofTop,
			cover.canvas, dress.roofLine(cover.felt), false))
}

// gapAngle is how far each side of the door the wall is left open, in
// radians.
func (dress yurtDress) gapAngle() float64 {
	cover := dress.fine.cover
	return (float64(cover.gap) + half) * fullTurn / float64(cover.sectors)
}

// lattice is the khana: at stations round the wall, two slats cross, one
// leaning each way, the second in front of the first. Each climbs the
// wall's height, less its own width. The stations in the doorway are left
// out.
func (dress yurtDress) lattice() []placedElement {
	sizes, slat := dress.sizes, dress.fine.lattice
	rise := slat.rise * math.Pi / halfTurn
	count := int(math.Round(fullTurn * sizes.wallRadius / slat.pitch))
	size := Box{slat.width, (sizes.wallHeight - slat.width) / math.Sin(rise),
		slat.leaf}
	wood := dress.clothes.of(roleFrame)
	var slats []placedElement
	for index := range count {
		heading := fullTurn * float64(index) / float64(count)
		if math.Abs(math.Remainder(heading, fullTurn)) < dress.gapAngle() {
			continue
		}
		for layer, lean := range []float64{quarterTurn - rise,
			rise - quarterTurn} {
			frame := radial(heading, sizes.wallRadius+float64(layer)*slat.leaf,
				sizes.wallHeight/half)
			slats = append(slats, part("slat", size, wood,
				mounted(size, maths.Rotation(maths.ZAxis, lean), frame)))
		}
	}
	return slats
}

// rail is the ring round the top of the khana, where the poles seat.
func (dress yurtDress) rail() []placedElement {
	sizes, lattice := dress.sizes, dress.fine.lattice
	return dress.ring("rail", dress.clothes.of(roleFrame),
		sizes.wallHeight-lattice.railHeight, sizes.wallHeight,
		lattice.railProud, func(float64) float64 { return sizes.wallRadius },
		false)
}

// poles are the uni: each runs from the khana's top to the crown ring, and
// no further past the wall's top. They are painted over their grain.
func (dress yurtDress) poles() []placedElement {
	sizes, fine := dress.sizes, dress.fine.poles
	foot := maths.V(0, sizes.wallHeight, sizes.wallRadius)
	head := maths.V(0, sizes.roofTop, sizes.crownRadius-sizes.crownLap)
	pole := []placedElement{rod("pole", foot, head, fine.thickness,
		dress.looks.pole)}
	var poles []placedElement
	for index := range fine.count {
		poles = append(poles,
			spun(pole, fullTurn*float64(index)/float64(fine.count))...)
	}
	return poles
}

// ties are the ropes and the woven bands that hold the cover down, over
// the canvas, and the band round the wall.
func (dress yurtDress) ties() []placedElement {
	sizes, fine := dress.sizes, dress.fine
	roof := sizes.roofTop - sizes.wallHeight
	over := dress.roofLine(fine.cover.felt + fine.cover.canvas)
	var ties []placedElement
	for index := range fine.ties.ropes {
		low := sizes.wallHeight + roof*(fine.ties.ropeFirst+
			float64(index)*fine.ties.ropeGap)
		ties = append(ties, dress.ring("rope", dress.looks.rope, low,
			low+fine.ties.ropeThickness, fine.ties.ropeThickness, over,
			false)...)
	}
	for index := range fine.ties.bands {
		low := sizes.wallHeight + roof*(fine.ties.bandFirst+
			float64(index)*fine.ties.bandGap)
		ties = append(ties, dress.ring("band", dress.looks.band, low,
			low+fine.ties.bandWidth, fine.ties.bandProud, over, false)...)
	}
	low := sizes.wallHeight * fine.ties.atWall
	return append(ties, dress.ring("band", dress.looks.band, low,
		low+fine.ties.bandWidth, fine.ties.bandProud,
		dress.wallLine(fine.cover.felt+fine.cover.canvas), true)...)
}

// toono is the crown ring: straight bars set end to end round the ring's
// radius, painted with the ornament, with the spokes across it. There is
// nothing over it: it is open to the sky.
func (dress yurtDress) toono() []placedElement {
	sizes, fine := dress.sizes, dress.fine.toono
	chord := bothSides * sizes.crownRadius * math.Tan(math.Pi/float64(
		fine.segments))
	bar := Box{chord, sizes.crownSeat + sizes.crownRise, fine.bar}
	var ring []placedElement
	for index := range fine.segments {
		heading := fullTurn * float64(index) / float64(fine.segments)
		ring = append(ring, part("toono", bar, dress.looks.crown, seated(bar,
			radial(heading, sizes.crownRadius-fine.bar/half,
				sizes.roofTop-sizes.crownSeat))))
	}
	return append(ring, dress.spokes()...)
}

// spokes are the bars across the crown ring from its hub to its bars,
// with their tops level with the ring's.
func (dress yurtDress) spokes() []placedElement {
	sizes, fine := dress.sizes, dress.fine.toono
	outer := sizes.crownRadius - fine.bar
	size := Box{fine.spokeWidth, fine.spokeHeight, outer - fine.hub}
	level := sizes.roofTop + sizes.crownRise - fine.spokeHeight
	round := outlineCircle{Radius: fine.hub}
	spokes := []placedElement{part("hub", prism{Bottom: round, Top: round,
		Height: fine.spokeHeight}, dress.looks.crown, lifted(level))}
	for index := range fine.spokes {
		heading := fullTurn * float64(index) / float64(fine.spokes)
		spokes = append(spokes, part("spoke", size, dress.looks.crown,
			seated(size, radial(heading, (fine.hub+outer)/half, level))))
	}
	return spokes
}

// doorway is the frame round the doorway, two jambs and a lintel, and the
// leaf, standing open into the yurt on its hinge, with the panels painted
// on it.
func (dress yurtDress) doorway() []placedElement {
	sizes, fine := dress.sizes, dress.fine.door
	across, back := sizes.doorWidth, -fine.depth/half
	jamb := Box{fine.bar, sizes.doorHeight + fine.bar, fine.depth}
	lintel := Box{across, fine.bar, fine.depth}
	frame := dress.looks.frame
	parts := []placedElement{
		part("jamb", jamb, frame,
			standingAt(maths.V(-across/half-fine.bar, 0, back))),
		part("jamb", jamb, frame, standingAt(maths.V(across/half, 0, back))),
		part("lintel", lintel, frame,
			standingAt(maths.V(-across/half, sizes.doorHeight, back)))}
	return append(within(parts, sizes.doorPlace()), dress.leaf()...)
}

// leaf is the door itself, swung open about its hinge on the left of the
// doorway, and the panels on it, painted with the ornament in the leaf's
// own plane.
func (dress yurtDress) leaf() []placedElement {
	sizes, fine := dress.sizes, dress.fine.door
	hinge := maths.NewSubspace(maths.V(-sizes.doorWidth/half, 0, 0),
		maths.Rotation(maths.YAxis, fine.open*math.Pi/halfTurn), 0).
		Within(sizes.doorPlace())
	origin, along := hinge.Out(maths.Vec{}), hinge.Orientation.Out(maths.XAxis)
	ornament := dress.clothes.dressed(rolePaint, "ornament", texture.Ornament{
		Under: dress.painted(roleFrame, roleDoor), Inks: dress.inks(),
		Cell: dress.fine.paint.doorCell, Ring: dress.fine.paint.doorRing,
		Coords: func(point maths.Vec) (float64, float64) {
			return point.Sub(origin).Dot(along), point.Y
		}})
	size := Box{sizes.doorWidth, sizes.doorHeight, sizes.doorThickness}
	leaf := part("door", size, dress.looks.leaf, standingAt(
		maths.V(0, 0, -sizes.doorThickness/half)))
	return within(append([]placedElement{leaf}, dress.panels(ornament)...), hinge)
}

// panels are the painted panels up the door, inset from its edges and
// from each other, standing out of its face. They are in the leaf's frame,
// which has its origin at the hinge.
func (dress yurtDress) panels(paint *thing.Material) []placedElement {
	sizes, fine := dress.sizes, dress.fine.door
	each := (sizes.doorHeight - float64(fine.panels+1)*fine.inset) /
		float64(fine.panels)
	size := Box{sizes.doorWidth - bothSides*fine.inset, each, fine.proud}
	var panels []placedElement
	for index := range fine.panels {
		foot := fine.inset + float64(index)*(each+fine.inset)
		panels = append(panels, part("panel", size, paint, standingAt(
			maths.V(fine.inset, foot, sizes.doorThickness/half))))
	}
	return panels
}

// interior is what the open door shows: the rug, the stove with its pipe
// up through the crown, the beds round the wall, and the chest at the back.
func (dress yurtDress) interior() []placedElement {
	sizes, room, cover := dress.sizes, dress.fine.room, dress.fine.cover
	round := outlineCircle{Radius: room.carpetRadius}
	iron := outlineCircle{Radius: room.stoveRadius}
	pipe := outlineCircle{Radius: room.pipeRadius}
	top := sizes.roofTop + sizes.crownRise + room.pipeRise
	base := cover.floor + room.stoveHeight
	black := dress.clothes.of(roleStove)
	parts := []placedElement{
		part("carpet", prism{Bottom: round, Top: round,
			Height: room.carpetHeight}, dress.looks.carpet, lifted(cover.floor)),
		part("stove", prism{Bottom: iron, Top: iron, Height: room.stoveHeight},
			black, lifted(cover.floor)),
		part("pipe", prism{Bottom: pipe, Top: pipe, Height: top - base},
			black, lifted(base))}
	for _, heading := range []float64{room.bedHeading, -room.bedHeading} {
		parts = append(parts, dress.bed(heading*math.Pi/halfTurn)...)
	}
	return append(parts, dress.chest())
}

// bed is a bed against the wall at a heading: a wooden frame, and a
// mattress with a woven blanket on it.
func (dress yurtDress) bed(heading float64) []placedElement {
	sizes, room, cover := dress.sizes, dress.fine.room, dress.fine.cover
	out := sizes.wallRadius - dress.fine.lattice.railProud - room.bedWidth/half
	frame := Box{room.bedLength, room.bedHeight, room.bedWidth}
	blanket := Box{room.bedLength - bothSides*room.mattressInset,
		room.mattress, room.bedWidth - bothSides*room.mattressInset}
	return []placedElement{
		part("bed", frame, dress.looks.frame,
			seated(frame, radial(heading, out, cover.floor))),
		part("blanket", blanket, dress.looks.band, seated(blanket,
			radial(heading, out, cover.floor+room.bedHeight)))}
}

// chest is the painted chest at the back of the yurt, against the wall
// opposite the door.
func (dress yurtDress) chest() placedElement {
	sizes, room, cover := dress.sizes, dress.fine.room, dress.fine.cover
	size := Box{room.chestWidth, room.chestHeight, room.chestDepth}
	out := sizes.wallRadius - dress.fine.lattice.railProud - room.chestDepth/half
	return part("chest", size, dress.looks.crown, seated(size, radial(
		room.chestHeading*math.Pi/halfTurn, out, cover.floor)))
}
