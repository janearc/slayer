package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The roles of the small parts a winnebago is drawn with at the demo
// level. What each is made of is in data. The mirrors' glass is of the
// glazing role.
const (
	// roleSeam is a seam between two body panels.
	roleSeam partRole = "seam"
	// roleTrim is the frame round a window.
	roleTrim partRole = "trim"
	// roleFitting is a fitting: a mirror's arm, the ladder, the bumper.
	roleFitting partRole = "fitting"
	// rolePlant is the air-conditioner's shroud.
	rolePlant partRole = "plant"
	// roleAwning is the awning's fabric.
	roleAwning partRole = "awning"
	// roleStep is the step before the entry door.
	roleStep partRole = "step"
	// roleArch is a wheel arch.
	roleArch partRole = "arch"
	// roleHeadlamp is the lens of a headlamp.
	roleHeadlamp partRole = "headlamp"
	// roleTaillamp is the lens of a tail lamp.
	roleTaillamp partRole = "taillamp"
)

// noseSizes are how the cab's front corners are rounded: the radius, and
// how many flat facets each corner is cut into.
type noseSizes struct {
	radius float64
	facets int
}

// seamSizes are the body panels' seams, belts and rails, in metres.
type seamSizes struct {
	width, proud, panel, beltAt, belt, railHeight, railProud float64
}

// trimSizes are the frames round the windows, the cab's and the rear's
// windows, and the entry door's handle and step, in metres.
type trimSizes struct {
	bar, depth                             float64
	cab, rear                              opening
	handleWidth, handleHeight, handleProud float64
	handleAt, stepDepth, stepTop           float64
}

// mirrorSizes are a mirror: its arm and its glass, in metres.
type mirrorSizes struct {
	arm, thickness, width, height, depth, at, forward float64
}

// ladderSizes are the ladder up the rear of the body, in metres.
type ladderSizes struct {
	width, offset, foot, topGap, rail, pitch float64
}

// conditionerSizes are the air-conditioner on the roof, in metres.
type conditionerSizes struct {
	width, length, height, base, place, fan, bar float64
}

// awningSizes are the roll-out awning on the kerb side, in metres.
type awningSizes struct {
	length, reach, top, drop, caseRadius, leaf, arm, armFoot, place float64
}

// lampSizes are one kind of lamp: its size, where it stands, and the
// heat and strength of the light it shines.
type lampSizes struct {
	width, height, depth, spread, at, heat, strength float64
}

// endSizes are the wheel arches, the front bumper and grille, and the
// lamps at each end.
type endSizes struct {
	archGap, archProud                  float64
	archFacets                          int
	bumperHeight, bumperProud, bumperAt float64
	grilleWidth, grilleHeight           float64
	head, tail                          lampSizes
}

// winnebagoFine is every measure of the demo level's small parts.
type winnebagoFine struct {
	nose        noseSizes
	seams       seamSizes
	trim        trimSizes
	mirror      mirrorSizes
	ladder      ladderSizes
	conditioner conditionerSizes
	awning      awningSizes
	ends        endSizes
}

// readWinnebagoFine is the demo level's measures from data, or why one is
// not there.
func readWinnebagoFine() (winnebagoFine, error) {
	reader := &choiceReader{}
	fine := winnebagoFine{
		nose: noseSizes{reader.of("nose-radius"), reader.whole("nose-facets")},
		seams: seamSizes{reader.of("seam-width"), reader.of("seam-proud"),
			reader.of("panel-length"), reader.of("belt-at"),
			reader.of("belt-height"), reader.of("roof-rail-height"),
			reader.of("roof-rail-proud")},
		trim: readTrim(reader), mirror: readMirror(reader),
		ladder: readLadder(reader), conditioner: readConditioner(reader),
		awning: readAwning(reader), ends: readEnds(reader),
	}
	return fine, reader.trouble
}

// readTrim is the frames', the extra windows', the handle's and the
// step's measures, read from data.
func readTrim(reader *choiceReader) trimSizes {
	thickness := reader.of("window-thickness")
	return trimSizes{bar: reader.of("window-bar"),
		depth: reader.of("trim-depth"),
		cab: opening{reader.of("cab-window-width"),
			reader.of("cab-window-height"), thickness,
			reader.of("cab-window-sill")},
		rear: opening{reader.of("rear-window-width"),
			reader.of("rear-window-height"), thickness,
			reader.of("rear-window-sill")},
		handleWidth:  reader.of("handle-width"),
		handleHeight: reader.of("handle-height"),
		handleProud:  reader.of("handle-proud"),
		handleAt:     reader.of("handle-at"),
		stepDepth:    reader.of("step-depth"), stepTop: reader.of("step-top")}
}

// readMirror is a mirror's measures, read from data.
func readMirror(reader *choiceReader) mirrorSizes {
	return mirrorSizes{arm: reader.of("mirror-arm"),
		thickness: reader.of("mirror-arm-thickness"),
		width:     reader.of("mirror-glass-width"),
		height:    reader.of("mirror-glass-height"),
		depth:     reader.of("mirror-glass-depth"),
		at:        reader.of("mirror-at"),
		forward:   reader.of("mirror-forward")}
}

// readLadder is the ladder's measures, read from data.
func readLadder(reader *choiceReader) ladderSizes {
	return ladderSizes{width: reader.of("ladder-width"),
		offset: reader.of("ladder-offset"), foot: reader.of("ladder-foot"),
		topGap: reader.of("ladder-top-gap"), rail: reader.of("ladder-rail"),
		pitch: reader.of("ladder-pitch")}
}

// readConditioner is the air-conditioner's measures, read from data.
func readConditioner(reader *choiceReader) conditionerSizes {
	return conditionerSizes{width: reader.of("ac-width"),
		length: reader.of("ac-length"), height: reader.of("ac-height"),
		base: reader.of("ac-base"), place: reader.of("ac-place"),
		fan: reader.of("fan-radius"), bar: reader.of("grille-bar")}
}

// readAwning is the awning's measures, read from data.
func readAwning(reader *choiceReader) awningSizes {
	return awningSizes{length: reader.of("awning-length"),
		reach: reader.of("awning-reach"), top: reader.of("awning-top"),
		drop:       reader.of("awning-drop"),
		caseRadius: reader.of("awning-case-radius"),
		leaf:       reader.of("awning-leaf"), arm: reader.of("awning-arm"),
		armFoot: reader.of("awning-arm-foot"),
		place:   reader.of("awning-place")}
}

// readEnds is the arches', bumper's, grille's and lamps' measures, read
// from data.
func readEnds(reader *choiceReader) endSizes {
	return endSizes{archGap: reader.of("arch-gap"),
		archProud: reader.of("arch-proud"), archFacets: reader.whole("arch-facets"),
		bumperHeight: reader.of("bumper-height"),
		bumperProud:  reader.of("bumper-proud"),
		bumperAt:     reader.of("bumper-at"),
		grilleWidth:  reader.of("grille-width"),
		grilleHeight: reader.of("grille-height"),
		head: lampSizes{reader.of("headlamp-width"),
			reader.of("headlamp-height"), reader.of("headlamp-depth"),
			reader.of("headlamp-spread"), reader.of("headlamp-at"),
			reader.of("headlamp-heat"), reader.of("headlamp-strength")},
		tail: lampSizes{reader.of("taillamp-width"),
			reader.of("taillamp-height"), reader.of("taillamp-depth"),
			reader.of("taillamp-spread"), reader.of("taillamp-at"),
			reader.of("taillamp-heat"), reader.of("taillamp-strength")}}
}

// winnebagoDress is a winnebago's sizes, its small parts' sizes and the
// materials it reads, for drawing it part by part.
type winnebagoDress struct {
	sizes   winnebagoSizes
	fine    winnebagoFine
	clothes *wardrobe
}

// WinnebagoAt is the winnebago drawn for a detail level. Below the demo
// level it is the winnebago as it was, whole. At the demo level the body
// has its panel seams, the cab a rounded nose, the windows frames, the
// door a handle and a step, and it has mirrors, a ladder, an
// air-conditioner, an awning, wheel arches, a bumper and lamps.
func WinnebagoAt(level Detail) (Design, error) {
	design, err := winnebago()
	if err != nil || level != demoLevel {
		return design, err
	}
	sizes, err := readWinnebago()
	if err != nil {
		return Design{}, err
	}
	fine, err := readWinnebagoFine()
	if err != nil {
		return Design{}, err
	}
	dress := winnebagoDress{sizes: sizes, fine: fine, clothes: &wardrobe{}}
	elements := dress.elements()
	if dress.clothes.trouble != nil {
		return Design{}, dress.clothes.trouble
	}
	design.Massing, design.Elements = dress.masses(), elements
	return design, nil
}

// elements is everything placed on the winnebago at the demo level: the
// wheels, the side windows and the entry door as they were, and the rest
// of the detail. The windscreen is drawn again, narrower, for the
// rounded nose.
func (dress winnebagoDress) elements() []placedElement {
	sizes := dress.sizes
	elements := append(sizes.wheelElements(), sizes.windowElements()...)
	elements = append(elements, sizes.entryElement())
	for _, more := range [][]placedElement{dress.screen(), dress.windowFrames(),
		dress.cabWindows(), dress.rearWindow(), dress.doorFittings(),
		dress.seams(), dress.arches(), dress.mirrors(), dress.ladder(),
		dress.conditioner(), dress.awning(), dress.lamps()} {
		elements = append(elements, more...)
	}
	return elements
}

// masses are the demo winnebago's two closed volumes: the body as it
// was, and the cab with its front corners rounded.
func (dress winnebagoDress) masses() []mass {
	masses := dress.sizes.masses()
	body := dress.sizes.body
	nose := polygon{Corners: dress.noseOutline()}
	masses[1].Form = prism{Bottom: nose, Top: nose,
		Height: body.cabRoof - body.clearance}
	masses[1].Place = standingAt(maths.V(0, body.clearance, 0))
	return masses
}

// arc is the corners of a quarter of a circle about a centre, from an
// angle, cut into facets, both ends included.
func arc(centre outlineCorner, radius, start float64, facets int) []outlineCorner {
	corners := make([]outlineCorner, 0, facets+1)
	for index := range facets + 1 {
		angle := start + quarterTurn*float64(index)/float64(facets)
		corners = append(corners, outlineCorner{centre.X + radius*math.Cos(angle),
			centre.Z + radius*math.Sin(angle)})
	}
	return corners
}

// noseOutline is the cab seen from above: straight at the back and sides,
// and rounded at the two front corners.
func (dress winnebagoDress) noseOutline() []outlineCorner {
	body, nose := dress.sizes.body, dress.fine.nose
	across, front := body.cabWidth/half, dress.sizes.frontFace()
	right := outlineCorner{across - nose.radius, front - nose.radius}
	left := outlineCorner{nose.radius - across, front - nose.radius}
	corners := []outlineCorner{{across, body.length / half}}
	corners = append(corners, arc(right, nose.radius, 0, nose.facets)...)
	corners = append(corners, arc(left, nose.radius, quarterTurn,
		nose.facets)...)
	return append(corners, outlineCorner{-across, body.length / half})
}

// belts are the belt seam along the middle of one side of the body, and
// the rail along its roof's edge.
func (dress winnebagoDress) belts(side float64) []placedElement {
	body, seam := dress.sizes.body, dress.fine.seams
	paint := dress.clothes.of(roleSeam)
	wall := side * body.width / half
	return []placedElement{
		part("belt", Box{bothSides * seam.proud, seam.belt, body.length},
			paint, standingAt(maths.V(wall-seam.proud, seam.beltAt,
				-body.length/half))),
		part("rail", Box{bothSides * seam.railProud, seam.railHeight,
			body.length}, paint, standingAt(maths.V(wall-seam.railProud,
			body.roof-seam.railHeight, -body.length/half)))}
}

// wallSeams are the seams on one side of the body: a vertical one
// between each pair of panels, the belt along the middle, and the rail
// along the roof's edge.
func (dress winnebagoDress) wallSeams(side float64) []placedElement {
	body, seam := dress.sizes.body, dress.fine.seams
	paint := dress.clothes.of(roleSeam)
	panels := int(math.Round(body.length / seam.panel))
	wall := side*body.width/half - seam.proud
	upright := Box{bothSides * seam.proud, body.roof - body.clearance,
		seam.width}
	seams := dress.belts(side)
	for index := 1; index < panels; index++ {
		along := float64(index)*body.length/float64(panels) -
			body.length/half - seam.width/half
		seams = append(seams, part("seam", upright, paint,
			standingAt(maths.V(wall, body.clearance, along))))
	}
	return seams
}

// seams are the body's panel seams: down each side and across the rear.
func (dress winnebagoDress) seams() []placedElement {
	body, seam := dress.sizes.body, dress.fine.seams
	paint := dress.clothes.of(roleSeam)
	behind := -body.length/half - seam.proud
	seams := append(dress.wallSeams(kerbSide), dress.wallSeams(offSide)...)
	return append(seams,
		part("belt", Box{body.width, seam.belt, bothSides * seam.proud},
			paint, standingAt(maths.V(-body.width/half, seam.beltAt, behind))),
		part("seam", Box{seam.width, body.roof - body.clearance,
			bothSides * seam.proud}, paint, standingAt(maths.V(-seam.width/half,
			body.clearance, behind))))
}

// archAt is the wall a wheel's arch stands on: the cab's, for an axle
// forward of the body's front, and the body's for the rest.
func (dress winnebagoDress) archAt(along float64) float64 {
	body := dress.sizes.body
	if along > body.length/half {
		return body.cabWidth / half
	}
	return body.width / half
}

// arches are the plates behind the top of each wheel: a half circle
// standing out of the wall, a little bigger than the tyre, so the tyre
// stands in front of a dark arch.
func (dress winnebagoDress) arches() []placedElement {
	ends, wheels := dress.fine.ends, dress.sizes.wheels
	radius := wheels.radius + ends.archGap
	section := make([]outlineCorner, 0, ends.archFacets+1)
	for index := range ends.archFacets + 1 {
		angle := math.Pi * float64(index) / float64(ends.archFacets)
		section = append(section, outlineCorner{radius * math.Cos(angle),
			radius * math.Sin(angle)})
	}
	plate := extrusion{Section: section, Length: ends.archProud,
		Ridge: ridgeAlongX}
	dark := dress.clothes.of(roleArch)
	var arches []placedElement
	for _, along := range dress.sizes.axles() {
		wall := dress.archAt(along)
		for _, side := range []float64{kerbSide, offSide} {
			foot := wall
			if side == kerbSide {
				foot = -wall - ends.archProud
			}
			arches = append(arches, part("arch", plate, dark,
				standingAt(maths.V(foot, wheels.radius, along))))
		}
	}
	return arches
}
