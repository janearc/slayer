package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// The roles of a winnebago's parts. What each is made of is in data.
const (
	// roleBody is the box the people live in.
	roleBody partRole = "body"
	// roleCab is the lower box at the front, where it is driven from.
	roleCab partRole = "cab"
	// roleGlazing is the glass of the windscreen and the side windows.
	roleGlazing partRole = "glazing"
	// roleTyre is the rubber of a wheel.
	roleTyre partRole = "tyre"
	// roleEntry is the door people go in by.
	roleEntry partRole = "entry"
)

// quarterTurn is a quarter of a turn, in radians. It is how a wheel is
// laid on its side and how a pane is turned to face a side of the
// vehicle.
const quarterTurn = math.Pi / 2

// The two sides of the vehicle, as the sign of x on that side. The vehicle
// faces along positive z, so the kerb side, where it stops in traffic that
// keeps right, is the negative x side.
const (
	kerbSide = -1.0
	offSide  = 1.0
)

// pane is a flat box standing proud of a wall: a window or a door. It is
// centred across at its place and stands on it, half its thickness out of
// the wall, facing along its own positive z. What it is made of is its
// role.
type pane struct {
	Width, Height, Thickness float64
	Role                     partRole
}

// Rendered is the pane's box, with its place in its own space and the
// look of its role.
func (pane pane) Rendered(level Detail) (Rendered, error) {
	mesh, err := Box{pane.Width, pane.Height, pane.Thickness}.Mesh(level)
	if err != nil {
		return Rendered{}, err
	}
	material, err := material(pane.Role)
	if err != nil {
		return Rendered{}, err
	}
	mesh.Name = string(pane.Role)
	centred := maths.V(-pane.Width/2, 0, -pane.Thickness/2)
	return Rendered{Mesh: mesh, Material: material,
		Place: maths.NewSubspace(centred, maths.Identity, 0)}, nil
}

// wheel is a tyre: a circle prism laid on its side, its axle along x. In
// its own space its centre is the origin, and it reaches from there
// toward negative x by its width.
type wheel struct{ Radius, Width float64 }

// Rendered is the wheel's prism, turned on its side, in the look of the
// tyre role.
func (wheel wheel) Rendered(level Detail) (Rendered, error) {
	round := outlineCircle{Radius: wheel.Radius}
	mesh, err := prism{Bottom: round, Top: round,
		Height: wheel.Width}.Mesh(level)
	if err != nil {
		return Rendered{}, err
	}
	material, err := material(roleTyre)
	if err != nil {
		return Rendered{}, err
	}
	mesh.Name = string(roleTyre)
	onItsSide := maths.Rotation(maths.ZAxis, quarterTurn)
	return Rendered{Mesh: mesh, Material: material,
		Place: maths.NewSubspace(maths.Vec{}, onItsSide, 0)}, nil
}

// choiceReader reads named choices from the designs' data, and keeps the
// first it could not read. After that it reads nothing and gives nought,
// so a run of reads is checked once at the end.
type choiceReader struct{ trouble error }

// of is a named choice, as the data gives it.
func (reader *choiceReader) of(name string) float64 {
	if reader.trouble != nil {
		return 0
	}
	value, err := choice(name)
	reader.trouble = err
	return value
}

// whole is a named choice that is a count: a whole number, not below
// nought.
func (reader *choiceReader) whole(name string) int {
	value := reader.of(name)
	if reader.trouble == nil && (value != math.Trunc(value) || value < 0) {
		reader.trouble = fmt.Errorf("cities: %s is %v, and a count is a "+
			"whole number, not below nought", name, value)
	}
	return int(value)
}

// bodySizes are the sizes of the two boxes, in metres. Heights are above
// the ground.
type bodySizes struct {
	length, width, roof, clearance float64
	cabLength, cabWidth, cabRoof   float64
}

// wheelSizes are the sizes and places of the wheels, in metres, and how
// many axles the rear has.
type wheelSizes struct {
	radius, width, track               float64
	frontSetback, rearSetback, axleGap float64
	rearAxles                          int
}

// opening is a pane's size and how high its foot stands, in metres.
type opening struct{ width, height, thickness, sill float64 }

// openingSizes are the windows, the windscreen and the entry door: each
// one's size, and the places along the length that say where the windows
// and the door stand, from the middle of the body, forward positive.
type openingSizes struct {
	window, screen, entry           opening
	windowFirst, windowGap, entryAt float64
	windowCount                     int
}

// winnebagoSizes are everything a winnebago's shape takes from data.
type winnebagoSizes struct {
	body     bodySizes
	wheels   wheelSizes
	openings openingSizes
}

// Facts are what a winnebago is as a place to stay, for the plans that
// count a district by its people and its ground.
type Facts struct {
	// Sleeps is how many people it sleeps.
	Sleeps int
	// FloorArea is the floor inside, in square metres.
	FloorArea float64
	// PitchArea is the ground one pitch takes, the vehicle and its
	// clearance, in square metres.
	PitchArea float64
}

// WinnebagoFacts is how many a winnebago sleeps, its floor and its pitch,
// or why the data does not say.
func WinnebagoFacts() (Facts, error) {
	reader := &choiceReader{}
	facts := Facts{Sleeps: reader.whole("sleeps"),
		FloorArea: reader.of("floor-area"),
		PitchArea: reader.of("pitch-area")}
	if reader.trouble != nil {
		return Facts{}, reader.trouble
	}
	return facts, nil
}

// readBody is the two boxes' sizes, read from data.
func readBody(reader *choiceReader) bodySizes {
	return bodySizes{length: reader.of("body-length"),
		width: reader.of("body-width"), roof: reader.of("roof-height"),
		clearance: reader.of("ground-clearance"),
		cabLength: reader.of("cab-length"), cabWidth: reader.of("cab-width"),
		cabRoof: reader.of("cab-roof")}
}

// readWheels is the wheels' sizes and places, read from data.
func readWheels(reader *choiceReader) wheelSizes {
	return wheelSizes{radius: reader.of("tyre-radius"),
		width: reader.of("tyre-width"), track: reader.of("wheel-track"),
		frontSetback: reader.of("front-axle-setback"),
		rearSetback:  reader.of("rear-axle-setback"),
		axleGap:      reader.of("rear-axle-gap"),
		rearAxles:    reader.whole("rear-axles")}
}

// readOpenings is the windows', the windscreen's and the door's sizes
// and places, read from data. A window and the windscreen share a
// pane's thickness.
func readOpenings(reader *choiceReader) openingSizes {
	thickness := reader.of("window-thickness")
	return openingSizes{
		window: opening{reader.of("window-width"),
			reader.of("window-height"), thickness, reader.of("window-sill")},
		screen: opening{reader.of("screen-width"),
			reader.of("screen-height"), thickness, reader.of("screen-sill")},
		entry: opening{reader.of("entry-width"), reader.of("entry-height"),
			reader.of("entry-thickness"), reader.of("entry-sill")},
		windowFirst: reader.of("window-first"),
		windowGap:   reader.of("window-gap"),
		entryAt:     reader.of("entry-place"),
		windowCount: reader.whole("window-count"),
	}
}

// readWinnebago is the winnebago's sizes from its data, or why one is
// not there.
func readWinnebago() (winnebagoSizes, error) {
	reader := &choiceReader{}
	sizes := winnebagoSizes{body: readBody(reader),
		wheels: readWheels(reader), openings: readOpenings(reader)}
	if reader.trouble != nil {
		return winnebagoSizes{}, reader.trouble
	}
	return sizes, nil
}

// standingAt is the place a point in the vehicle's space, with no turn.
func standingAt(at maths.Vec) maths.Subspace {
	return maths.NewSubspace(at, maths.Identity, 0)
}

// masses are the winnebago's two closed volumes: the body, centred on
// the vehicle's origin in plan, and the lower cab standing against its
// front face. Both stand clear of the ground by the clearance.
func (sizes winnebagoSizes) masses() []mass {
	body := sizes.body
	return []mass{
		{Name: "body", Role: roleBody,
			Form: Box{body.width, body.roof - body.clearance, body.length},
			Place: standingAt(maths.V(-body.width/2, body.clearance,
				-body.length/2))},
		{Name: "cab", Role: roleCab,
			Form: Box{body.cabWidth, body.cabRoof - body.clearance,
				body.cabLength},
			Place: standingAt(maths.V(-body.cabWidth/2, body.clearance,
				body.length/2))},
	}
}

// axles are the places along the length of each axle, the front first.
func (sizes winnebagoSizes) axles() []float64 {
	body, wheels := sizes.body, sizes.wheels
	axles := []float64{body.length/2 + body.cabLength - wheels.frontSetback}
	first := -body.length/2 + wheels.rearSetback
	for index := range wheels.rearAxles {
		axles = append(axles, first+float64(index)*wheels.axleGap)
	}
	return axles
}

// wheelElements is a wheel at each end of each axle, on the ground. The
// wheel reaches toward negative x from its place, so the one on the off
// side is turned half a turn about the vertical to reach outward there.
func (sizes winnebagoSizes) wheelElements() []placedElement {
	wheels := sizes.wheels
	var placed []placedElement
	for _, along := range sizes.axles() {
		for _, side := range []float64{kerbSide, offSide} {
			turn := maths.Rotation(maths.YAxis, (1+side)*quarterTurn)
			at := maths.V(side*wheels.track/2, wheels.radius, along)
			placed = append(placed, placedElement{
				Element: wheel{Radius: wheels.radius, Width: wheels.width},
				Place:   maths.NewSubspace(at, turn, 0)})
		}
	}
	return placed
}

// onSide is where a pane stands on a side of the body: on the wall, at a
// place along the length and a height, turned to face outward.
func (sizes winnebagoSizes) onSide(side, sill, along float64) maths.Subspace {
	turn := maths.Rotation(maths.YAxis, side*quarterTurn)
	return maths.NewSubspace(
		maths.V(side*sizes.body.width/2, sill, along), turn, 0)
}

// paneOf is a pane of an opening's size and a role.
func paneOf(shape opening, role partRole) pane {
	return pane{Width: shape.width, Height: shape.height,
		Thickness: shape.thickness, Role: role}
}

// windowElements is the side windows: the same number each side, at the
// same places along the length.
func (sizes winnebagoSizes) windowElements() []placedElement {
	openings := sizes.openings
	var placed []placedElement
	for index := range openings.windowCount {
		along := openings.windowFirst + float64(index)*openings.windowGap
		for _, side := range []float64{kerbSide, offSide} {
			placed = append(placed, placedElement{
				Element: paneOf(openings.window, roleGlazing),
				Place:   sizes.onSide(side, openings.window.sill, along)})
		}
	}
	return placed
}

// frontFace is how far forward the cab's front face stands.
func (sizes winnebagoSizes) frontFace() float64 {
	return sizes.body.length/2 + sizes.body.cabLength
}

// screenElement is the windscreen, on the cab's front face.
func (sizes winnebagoSizes) screenElement() placedElement {
	openings := sizes.openings
	return placedElement{Element: paneOf(openings.screen, roleGlazing),
		Place: standingAt(maths.V(0, openings.screen.sill,
			sizes.frontFace()))}
}

// entryElement is the entry door, on the kerb side of the body.
func (sizes winnebagoSizes) entryElement() placedElement {
	openings := sizes.openings
	return placedElement{Element: paneOf(openings.entry, roleEntry),
		Place: sizes.onSide(kerbSide, openings.entry.sill, openings.entryAt)}
}

// frontElements is the windscreen, on the cab's front face, and the
// entry door, on the kerb side of the body.
func (sizes winnebagoSizes) frontElements() []placedElement {
	return []placedElement{sizes.screenElement(), sizes.entryElement()}
}

// winnebago is a motorhome of the common American kind as a design: a
// tall box body with a lower box cab in front of it, six wheels, a
// windscreen, windows down each side and an entry door on the kerb side.
// It faces along positive z and stands on the ground at its origin's
// height. Every dimension, count and colour is data, and picked.
func winnebago() (Design, error) {
	sizes, err := readWinnebago()
	if err != nil {
		return Design{}, err
	}
	elements := append(sizes.wheelElements(), sizes.windowElements()...)
	elements = append(elements, sizes.frontElements()...)
	return Design{Name: "winnebago", Massing: sizes.masses(),
		Elements: elements}, nil
}
